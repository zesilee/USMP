#!/usr/bin/env bash
# USMP kind 一键部署（WSL 友好）—— 全局 HA 双副本验收环境
#
#   构建三镜像 → kind 集群（NodePort 直通宿主 8080/3002）→ CRD/RBAC →
#   simulator + backend×2（双选主开关开）+ frontend → 经 API 注册模拟网元
#   （集群模式：落 Device CR + 凭据 Secret）→ 打印 HA 验收清单。
#
# 用法：
#   ./scripts/kind-deploy.sh            # 部署/更新（幂等）
#   ./scripts/kind-deploy.sh down       # 删除 kind 集群
#   USMP_KIND_STOP_STAGING=1 ./scripts/kind-deploy.sh   # 8080/3002 被 compose
#                                       # staging 占用时先自动 staging down
#
# WSL/内网注意：
#   - 需要 docker（WSL2 内 daemon 或 Docker Desktop 集成）。
#   - 走代理时导出 HTTP_PROXY/HTTPS_PROXY/NO_PROXY 即可：镜像构建经预定义
#     build-arg 透传；kind 节点镜像拉取走 docker daemon 自身代理配置。
#   - 华为内网 TLS 拦截环境的 Go/apk 证书 hack 见 docs/CICD.md（勿提交 main）。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CLUSTER="${USMP_KIND_CLUSTER:-usmp}"
BIN_DIR="${HOME}/.local/bin"
KIND_VERSION="v0.24.0"
KUBECTL_VERSION="v1.31.0"
BACKEND_HOST_PORT=8080   # 前端 bundle 硬编码 http://localhost:8080/api/v1，勿改
FRONTEND_HOST_PORT=3002

log()  { printf '\033[0;32m[kind-deploy]\033[0m %s\n' "$*"; }
warn() { printf '\033[0;33m[kind-deploy]\033[0m %s\n' "$*"; }
die()  { printf '\033[0;31m[kind-deploy]\033[0m %s\n' "$*" >&2; exit 1; }

# ---------- down ----------
if [ "${1:-}" = "down" ]; then
  kind delete cluster --name "$CLUSTER"
  exit 0
fi

# ---------- 0. 预检 ----------
command -v docker >/dev/null || die "需要 docker（WSL2 内启动 daemon 或开启 Docker Desktop WSL 集成）"
docker info >/dev/null 2>&1 || die "docker daemon 不可达"

arch="$(uname -m)"; case "$arch" in x86_64) arch=amd64 ;; aarch64) arch=arm64 ;; esac
mkdir -p "$BIN_DIR"; export PATH="$BIN_DIR:$PATH"

if ! command -v kind >/dev/null; then
  log "安装 kind ${KIND_VERSION} → ${BIN_DIR}"
  curl -fsSLo "$BIN_DIR/kind" "https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-linux-${arch}"
  chmod +x "$BIN_DIR/kind"
fi
if ! command -v kubectl >/dev/null; then
  log "安装 kubectl ${KUBECTL_VERSION} → ${BIN_DIR}"
  curl -fsSLo "$BIN_DIR/kubectl" "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/${arch}/kubectl"
  chmod +x "$BIN_DIR/kubectl"
fi

# 宿主端口冲突（compose staging 同样占 8080/3002）
port_busy() { (exec 3<>"/dev/tcp/127.0.0.1/$1") 2>/dev/null && { exec 3>&- 3<&-; return 0; } || return 1; }
if ! kind get clusters 2>/dev/null | grep -qx "$CLUSTER"; then
  for p in "$BACKEND_HOST_PORT" "$FRONTEND_HOST_PORT"; do
    if port_busy "$p"; then
      if [ "${USMP_KIND_STOP_STAGING:-}" = "1" ] && docker ps --format '{{.Names}}' | grep -q '^usmp-staging-'; then
        log "端口 :$p 被 compose staging 占用，按 USMP_KIND_STOP_STAGING=1 停止 staging"
        (cd "$ROOT" && docker compose -p usmp-staging down)
        break
      fi
      die "端口 :$p 已被占用（compose staging？）。先 make staging-down，或 USMP_KIND_STOP_STAGING=1 重跑"
    fi
  done
fi

# ---------- 1. kind 集群（幂等） ----------
if ! kind get clusters 2>/dev/null | grep -qx "$CLUSTER"; then
  log "创建 kind 集群 ${CLUSTER}（NodePort 30080→:${BACKEND_HOST_PORT}，30002→:${FRONTEND_HOST_PORT}）"
  kind create cluster --name "$CLUSTER" --config=- <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    extraPortMappings:
      - containerPort: 30080
        hostPort: ${BACKEND_HOST_PORT}
      - containerPort: 30002
        hostPort: ${FRONTEND_HOST_PORT}
EOF
else
  log "集群 ${CLUSTER} 已存在，复用"
fi
kubectl config use-context "kind-${CLUSTER}" >/dev/null

# ---------- 2. 构建并载入镜像 ----------
build_args=()
for v in HTTP_PROXY HTTPS_PROXY NO_PROXY http_proxy https_proxy no_proxy; do
  [ -n "${!v:-}" ] && build_args+=(--build-arg "$v=${!v}")
done
log "构建镜像（simulator / controller / frontend）"
docker build "${build_args[@]}" -t usmp-simulator:latest  -f "$ROOT/backend/Dockerfile.simulator" "$ROOT/backend"
docker build "${build_args[@]}" -t usmp-controller:latest -f "$ROOT/backend/Dockerfile"           "$ROOT/backend"
docker build "${build_args[@]}" -t usmp-frontend:latest   -f "$ROOT/frontend/Dockerfile"          "$ROOT/frontend"
log "载入镜像到 kind 节点"
kind load docker-image usmp-simulator:latest usmp-controller:latest usmp-frontend:latest --name "$CLUSTER"

# ---------- 3. CRD + RBAC（BIC-02：CRD 先于应用） ----------
log "安装 CRD 与 RBAC"
kubectl apply -f "$ROOT/deploy/crds/"
kubectl create namespace usmp-system --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f "$ROOT/deploy/rbac/"

# ---------- 4. 应用编排 ----------
# simulator 在 default（与 Device CR/Lease 同 namespace 便于观察，非硬性）；
# backend 双副本在 usmp-system（SA usmp），双选主开关开=全局 HA 验收形态。
log "部署 simulator + backend×2 + frontend"
kubectl apply -f - <<'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: netconf-sim
  namespace: default
spec:
  replicas: 1
  selector: { matchLabels: { app: netconf-sim } }
  template:
    metadata: { labels: { app: netconf-sim } }
    spec:
      containers:
        - name: sim
          image: usmp-simulator:latest
          imagePullPolicy: Never
          ports: [{ containerPort: 830 }]
---
apiVersion: v1
kind: Service
metadata:
  name: netconf-sim
  namespace: default
spec:
  selector: { app: netconf-sim }
  ports: [{ port: 830, targetPort: 830 }]
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: usmp-backend
  namespace: usmp-system
spec:
  replicas: 2
  selector: { matchLabels: { app: usmp-backend } }
  template:
    metadata: { labels: { app: usmp-backend } }
    spec:
      serviceAccountName: usmp
      containers:
        - name: backend
          image: usmp-controller:latest
          imagePullPolicy: Never
          ports: [{ containerPort: 8080 }]
          env:
            # 意图 CR / Device CR / AuditRecord / Lease 所在 namespace
            - { name: USMP_INTENT_NAMESPACE,       value: "default" }
            # 全局 HA 验收形态：双选主开关全开（SC-06）
            - { name: USMP_INTENT_LEADER_ELECTION, value: "1" }
            - { name: USMP_NATIVE_LEADER_ELECTION, value: "1" }
          readinessProbe:
            httpGet: { path: /api/v1/yang/modules, port: 8080 }
            initialDelaySeconds: 5
            periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: usmp-backend
  namespace: usmp-system
spec:
  type: NodePort
  selector: { app: usmp-backend }
  ports: [{ port: 8080, targetPort: 8080, nodePort: 30080 }]
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: usmp-frontend
  namespace: usmp-system
spec:
  replicas: 1
  selector: { matchLabels: { app: usmp-frontend } }
  template:
    metadata: { labels: { app: usmp-frontend } }
    spec:
      containers:
        - name: frontend
          image: usmp-frontend:latest
          imagePullPolicy: Never
          ports: [{ containerPort: 80 }]
---
apiVersion: v1
kind: Service
metadata:
  name: usmp-frontend
  namespace: usmp-system
spec:
  type: NodePort
  selector: { app: usmp-frontend }
  ports: [{ port: 80, targetPort: 80, nodePort: 30002 }]
EOF

log "等待滚动就绪"
kubectl -n default     rollout status deploy/netconf-sim   --timeout=180s
kubectl -n usmp-system rollout status deploy/usmp-backend  --timeout=300s
kubectl -n usmp-system rollout status deploy/usmp-frontend --timeout=180s

# ---------- 5. 注册模拟网元（集群模式：API → Device CR + 凭据 Secret） ----------
log "经 API 注册模拟网元 netconf-sim.default:830"
for i in $(seq 1 30); do
  if curl -fsS -X POST "http://localhost:${BACKEND_HOST_PORT}/api/v1/devices" \
       -H 'Content-Type: application/json' \
       -d '{"ip":"netconf-sim.default","port":830,"username":"admin","password":"admin"}' >/dev/null 2>&1; then
    break
  fi
  [ "$i" = 30 ] && warn "设备注册重试超限——可稍后手动 POST /api/v1/devices"
  sleep 2
done

# ---------- 6. 验收快照 ----------
echo
log "===== 全局 HA 验收快照（SC-06）====="
echo "— Lease 持有者（两把锁应各有唯一 holder）:"
kubectl -n default get lease usmp-business-intent usmp-native-controllers \
  -o custom-columns='LEASE:.metadata.name,HOLDER:.spec.holderIdentity' 2>/dev/null || true
echo "— Device CR（集群模式设备来源）与凭据 Secret（CR 无明文）:"
kubectl -n default get devices.core.usmp.io 2>/dev/null || true
kubectl -n default get secrets -l usmp.io/device-ip 2>/dev/null || true
echo "— leader 日志:"
kubectl -n usmp-system logs deploy/usmp-backend --all-pods=true 2>/dev/null | grep -E "leader election|CRD-backed" | head -8 || \
  kubectl -n usmp-system logs -l app=usmp-backend --tail=200 --prefix 2>/dev/null | grep -E "leader election|CRD-backed" | head -8 || true
echo
log "入口：后端 http://localhost:${BACKEND_HOST_PORT}/api/v1  前端 http://localhost:${FRONTEND_HOST_PORT}"
cat <<'DRILL'

HA 演练（部署时验收项，逐条执行观察）:
  1. 单 leader：      kubectl -n default get lease -w        # holder 唯一且稳定
  2. 杀 leader 接管： kubectl -n usmp-system delete pod <holder 所在 pod>
                      # ~15s 内 holder 切到另一副本，日志出现 "leader election won"
  3. 跨副本可见：     curl -X POST localhost:8080/api/v1/devices ...（任一副本接的请求）
                      kubectl -n default get devices.core.usmp.io   # CR 即共享事实
  4. 审计跨重启：     做一次配置下发 → kubectl -n default get auditrecords.core.usmp.io
                      → 重建任一 pod → GET /logs 历史仍在
  5. 无本地持久：     kubectl -n usmp-system exec <pod> -- ls data/ 2>&1  # 应无 audit.json
DRILL
