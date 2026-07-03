# Kind 集群完整部署实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 USMP 平台在 Kind 集群的一键部署，包含前端、Controller、NETCONF 模拟器三大组件，支持开发调试和功能演示。

**Architecture:** NodePort 极简方案 - 所有服务通过 NodePort 暴露，前端直接连接 K8s API Server，无需 Ingress 或 LoadBalancer 组件。Kustomize 统一管理部署清单。

**Tech Stack:** Kind, Kustomize, Kubernetes Manifests, Docker, Makefile

---

## 前置检查

- [ ] 确认 Kind 已安装：`kind --version`
- [ ] 确认 Docker 正在运行：`docker info`
- [ ] 确认 kubectl 已安装：`kubectl version --client`
- [ ] 确认 backend 目录下有 Makefile 和 Dockerfile

---

## 第一阶段：目录结构和基础配置

### Task 1: 创建部署目录结构

**Files:**
- Create: `backend/deploy/manifests/kustomization.yaml`
- Create: `backend/deploy/manifests/namespace.yaml`
- Create: `backend/deploy/kind-cluster.yaml` (copy from existing)

- [ ] **Step 1: 创建目录结构**

```bash
mkdir -p backend/deploy/manifests/{controller,frontend,netconf-simulator}
```

- [ ] **Step 2: 创建 namespace.yaml**

```yaml
# backend/deploy/manifests/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: usmp-system
  labels:
    app: usmp
    environment: development
```

- [ ] **Step 3: 复制 Kind 集群配置**

```bash
cp backend/test/e2e/config/kind-cluster.yaml backend/deploy/kind-cluster.yaml
```

修改 `name: usmp-e2e` 为 `name: usmp-dev`

- [ ] **Step 4: 创建基础 kustomization.yaml**

```yaml
# backend/deploy/manifests/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: usmp-system

resources:
  - namespace.yaml

commonLabels:
  app: usmp
  environment: development
```

- [ ] **Step 5: 验证目录结构**

```bash
find backend/deploy -type f | sort
```

Expected:
```
backend/deploy/kind-cluster.yaml
backend/deploy/manifests/kustomization.yaml
backend/deploy/manifests/namespace.yaml
```

- [ ] **Step 6: Commit**

```bash
git add backend/deploy
git commit -m "feat(deploy): 创建 Kind 部署目录结构和基础配置"
```

---

## 第二阶段：Controller 部署配置

### Task 2: Controller 部署清单

**Files:**
- Create: `backend/deploy/manifests/controller/service_account.yaml`
- Create: `backend/deploy/manifests/controller/role.yaml`
- Create: `backend/deploy/manifests/controller/role_binding.yaml`
- Create: `backend/deploy/manifests/controller/deployment.yaml`
- Create: `backend/deploy/manifests/controller/service.yaml`
- Modify: `backend/deploy/manifests/kustomization.yaml`

- [ ] **Step 1: 创建 service_account.yaml**

```yaml
# backend/deploy/manifests/controller/service_account.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: usmp-controller
  namespace: usmp-system
```

- [ ] **Step 2: 复制并创建 role.yaml**

从 `backend/config/rbac/role.yaml` 复制全部内容，确保 namespace 为 usmp-system

- [ ] **Step 3: 创建 role_binding.yaml**

```yaml
# backend/deploy/manifests/controller/role_binding.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: usmp-controller-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: usmp-controller-role
subjects:
- kind: ServiceAccount
  name: usmp-controller
  namespace: usmp-system
```

- [ ] **Step 4: 创建 deployment.yaml**

```yaml
# backend/deploy/manifests/controller/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: usmp-controller
  namespace: usmp-system
spec:
  replicas: 1
  selector:
    matchLabels:
      control-plane: controller-manager
  template:
    metadata:
      labels:
        control-plane: controller-manager
    spec:
      serviceAccountName: usmp-controller
      containers:
      - name: manager
        image: usmp-controller:latest
        imagePullPolicy: IfNotPresent
        args:
        - "--zap-log-level=debug"
        - "--zap-stacktrace-level=error"
        ports:
        - containerPort: 8080
          name: api
          protocol: TCP
        - containerPort: 8081
          name: health
          protocol: TCP
        resources:
          limits:
            cpu: 500m
            memory: 256Mi
          requests:
            cpu: 100m
            memory: 128Mi
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8081
          initialDelaySeconds: 15
          periodSeconds: 20
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8081
          initialDelaySeconds: 5
          periodSeconds: 10
      terminationGracePeriodSeconds: 10
```

- [ ] **Step 5: 创建 service.yaml (NodePort)**

```yaml
# backend/deploy/manifests/controller/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: usmp-controller
  namespace: usmp-system
spec:
  type: NodePort
  selector:
    control-plane: controller-manager
  ports:
  - name: api
    port: 8080
    targetPort: 8080
    nodePort: 30080
    protocol: TCP
  - name: health
    port: 8081
    targetPort: 8081
    protocol: TCP
```

- [ ] **Step 6: 更新 kustomization.yaml 添加 controller 资源**

```yaml
resources:
  - namespace.yaml
  # Controller
  - controller/service_account.yaml
  - controller/role.yaml
  - controller/role_binding.yaml
  - controller/deployment.yaml
  - controller/service.yaml
```

- [ ] **Step 7: 验证 YAML 语法**

```bash
cd backend && kubectl kustomize deploy/manifests/
```

Expected: 无语法错误，输出所有资源

- [ ] **Step 8: Commit**

```bash
git add backend/deploy/manifests/controller
git add backend/deploy/manifests/kustomization.yaml
git commit -m "feat(deploy): 添加 Controller 部署清单 (SA/RBAC/Deployment/Service)"
```

---

## 第三阶段：前端部署配置

### Task 3: 前端 Dockerfile 和 Nginx 配置

**Files:**
- Create: `frontend/Dockerfile`
- Create: `frontend/nginx.conf`

- [ ] **Step 1: 创建前端 Dockerfile**

```dockerfile
# frontend/Dockerfile
# 构建阶段
FROM node:20-alpine AS builder

WORKDIR /app

# 安装依赖
COPY package*.json ./
RUN npm install --registry=https://registry.npmmirror.com

# 复制源码
COPY . .

# 构建
RUN npm run build

# 运行阶段
FROM nginx:alpine

# 复制 Nginx 配置
COPY nginx.conf /etc/nginx/nginx.conf

# 复制构建产物
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
```

- [ ] **Step 2: 创建 nginx.conf**

```nginx
# frontend/nginx.conf
events {
    worker_connections 1024;
}

http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;

    log_format  main  '$remote_addr - $remote_user [$time_local] "$request" '
                      '$status $body_bytes_sent "$http_referer" '
                      '"$http_user_agent" "$http_x_forwarded_for"';

    access_log  /var/log/nginx/access.log  main;

    sendfile        on;
    keepalive_timeout  65;

    server {
        listen       80;
        server_name  localhost;
        root         /usr/share/nginx/html;
        index        index.html;

        # SPA 路由支持
        location / {
            try_files $uri $uri/ /index.html;
        }

        # 静态资源缓存
        location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg)$ {
            expires 1y;
            add_header Cache-Control "public, immutable";
        }

        # 健康检查
        location /healthz {
            access_log off;
            return 200 "healthy\n";
            add_header Content-Type text/plain;
        }
    }
}
```

- [ ] **Step 3: 验证 Dockerfile 可构建**

```bash
cd frontend && docker build -t usmp-frontend:test .
```

Expected: 构建成功

- [ ] **Step 4: 清理测试镜像**

```bash
docker rmi usmp-frontend:test
```

- [ ] **Step 5: Commit**

```bash
git add frontend/Dockerfile frontend/nginx.conf
git commit -m "feat(deploy): 添加前端 Dockerfile 和 Nginx 配置"
```

---

### Task 4: 前端 K8s 部署清单

**Files:**
- Create: `backend/deploy/manifests/frontend/service_account.yaml`
- Create: `backend/deploy/manifests/frontend/cluster_role.yaml`
- Create: `backend/deploy/manifests/frontend/cluster_role_binding.yaml`
- Create: `backend/deploy/manifests/frontend/configmap.yaml`
- Create: `backend/deploy/manifests/frontend/deployment.yaml`
- Create: `backend/deploy/manifests/frontend/service.yaml`
- Modify: `backend/deploy/manifests/kustomization.yaml`

- [ ] **Step 1: 创建 service_account.yaml**

```yaml
# backend/deploy/manifests/frontend/service_account.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: usmp-frontend
  namespace: usmp-system
```

- [ ] **Step 2: 创建 cluster_role.yaml**

```yaml
# backend/deploy/manifests/frontend/cluster_role.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: usmp-frontend-role
rules:
# 业务 CRD 完整读写权限
- apiGroups: ["biz.usmp.io"]
  resources: ["businessvlans", "businessinterfaces", "businessroutes", "businessswitches"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# 原生配置 CRD 完整读写权限
- apiGroups: ["core.usmp.io"]
  resources: ["nativedeviceconfigs"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# CRD 定义读取权限（用于 Schema 解析）
- apiGroups: ["apiextensions.k8s.io"]
  resources: ["customresourcedefinitions"]
  verbs: ["get", "list", "watch"]
```

- [ ] **Step 3: 创建 cluster_role_binding.yaml**

```yaml
# backend/deploy/manifests/frontend/cluster_role_binding.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: usmp-frontend-binding
subjects:
- kind: ServiceAccount
  name: usmp-frontend
  namespace: usmp-system
roleRef:
  kind: ClusterRole
  name: usmp-frontend-role
  apiGroup: rbac.authorization.k8s.io
```

- [ ] **Step 4: 创建 configmap.yaml (环境配置)**

```yaml
# backend/deploy/manifests/frontend/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: usmp-frontend-config
  namespace: usmp-system
data:
  K8S_API_SERVER: "https://kubernetes.default.svc"
  K8S_NAMESPACE: "usmp-system"
```

- [ ] **Step 5: 创建 deployment.yaml**

```yaml
# backend/deploy/manifests/frontend/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: usmp-frontend
  namespace: usmp-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: usmp-frontend
  template:
    metadata:
      labels:
        app: usmp-frontend
    spec:
      serviceAccountName: usmp-frontend
      containers:
      - name: frontend
        image: usmp-frontend:latest
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 80
          name: http
          protocol: TCP
        envFrom:
        - configMapRef:
            name: usmp-frontend-config
        resources:
          limits:
            cpu: 200m
            memory: 128Mi
          requests:
            cpu: 50m
            memory: 64Mi
        livenessProbe:
          httpGet:
            path: /healthz
            port: 80
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /healthz
            port: 80
          initialDelaySeconds: 3
          periodSeconds: 5
```

- [ ] **Step 6: 创建 service.yaml (NodePort)**

```yaml
# backend/deploy/manifests/frontend/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: usmp-frontend
  namespace: usmp-system
spec:
  type: NodePort
  selector:
    app: usmp-frontend
  ports:
  - name: http
    port: 80
    targetPort: 80
    nodePort: 30081
    protocol: TCP
```

- [ ] **Step 7: 更新 kustomization.yaml 添加前端资源**

```yaml
resources:
  - namespace.yaml
  # Controller
  - controller/service_account.yaml
  - controller/role.yaml
  - controller/role_binding.yaml
  - controller/deployment.yaml
  - controller/service.yaml
  # Frontend
  - frontend/service_account.yaml
  - frontend/cluster_role.yaml
  - frontend/cluster_role_binding.yaml
  - frontend/configmap.yaml
  - frontend/deployment.yaml
  - frontend/service.yaml
```

- [ ] **Step 8: 验证 YAML 语法**

```bash
cd backend && kubectl kustomize deploy/manifests/ | grep -A5 Deployment
```

Expected: 显示 frontend 和 controller 两个 Deployment

- [ ] **Step 9: Commit**

```bash
git add backend/deploy/manifests/frontend
git add backend/deploy/manifests/kustomization.yaml
git commit -m "feat(deploy): 添加前端部署清单 (SA/ClusterRole/ConfigMap/Deployment/Service)"
```

---

## 第四阶段：NETCONF 模拟器部署

### Task 5: NETCONF 模拟器部署清单

**Files:**
- Create: `backend/deploy/manifests/netconf-simulator/configmap.yaml`
- Create: `backend/deploy/manifests/netconf-simulator/deployment.yaml`
- Create: `backend/deploy/manifests/netconf-simulator/service.yaml`
- Modify: `backend/deploy/manifests/kustomization.yaml`

- [ ] **Step 1: 复制现有模拟器配置并调整 namespace**

从 `backend/test/e2e/config/netconf-simulator.yaml` 拆分到三个文件，将 usmp-e2e 改为 usmp-system

- [ ] **Step 2: 创建 configmap.yaml**

```yaml
# backend/deploy/manifests/netconf-simulator/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: netconf-simulator-config
  namespace: usmp-system
data:
  initial-config.xml: |
    <config>
      <vlans xmlns="urn:huawei:params:xml:ns:yang:huawei-vlan">
        <vlan>
          <id>1</id>
          <name>default</name>
          <description>Default VLAN</description>
          <admin-status>1</admin-status>
        </vlan>
      </vlans>
      <interfaces xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm">
        <interface>
          <name>GigabitEthernet0/0/1</name>
          <description>Uplink Port</description>
          <admin-status>1</admin-status>
          <mtu>1500</mtu>
        </interface>
      </interfaces>
      <system xmlns="urn:huawei:params:xml:ns:yang:huawei-system">
        <system-info>
          <sysName>USMP-Dev-Switch</sysName>
          <sysContact>neteng@company.com</sysContact>
          <sysLocation>Dev-Lab</sysLocation>
        </system-info>
      </system>
    </config>
```

- [ ] **Step 3: 创建 deployment.yaml**

复制 e2e 配置中的 Deployment 部分，调整 namespace 为 usmp-system

- [ ] **Step 4: 创建 service.yaml**

```yaml
# backend/deploy/manifests/netconf-simulator/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: netconf-simulator
  namespace: usmp-system
  labels:
    app: netconf-simulator
spec:
  type: NodePort
  selector:
    app: netconf-simulator
  ports:
  - name: netconf
    port: 830
    targetPort: 830
    nodePort: 30830
    protocol: TCP
---
apiVersion: v1
kind: Service
metadata:
  name: netconf-simulator-internal
  namespace: usmp-system
spec:
  type: ClusterIP
  selector:
    app: netconf-simulator
  ports:
  - name: netconf
    port: 830
    targetPort: 830
    protocol: TCP
```

- [ ] **Step 5: 更新 kustomization.yaml 添加模拟器资源**

```yaml
resources:
  # ... existing ...
  # NETCONF Simulator
  - netconf-simulator/configmap.yaml
  - netconf-simulator/deployment.yaml
  - netconf-simulator/service.yaml
```

- [ ] **Step 6: 添加 CRD 资源引用**

```yaml
resources:
  # CRD Definitions
  - ../../config/crd/bases
  # ... existing ...
```

- [ ] **Step 7: 完整验证 Kustomize**

```bash
cd backend && kubectl kustomize deploy/manifests/ > /tmp/usmp-full.yaml && wc -l /tmp/usmp-full.yaml
```

Expected: > 200 lines, 包含所有资源类型

- [ ] **Step 8: Commit**

```bash
git add backend/deploy/manifests/netconf-simulator
git add backend/deploy/manifests/kustomization.yaml
git commit -m "feat(deploy): 添加 NETCONF 模拟器部署清单 (ConfigMap/Deployment/Service)"
```

---

## 第五阶段：Makefile 自动化

### Task 6: 添加 Kind 部署相关 Makefile 目标

**Files:**
- Modify: `backend/Makefile`

- [ ] **Step 1: 在 Makefile 末尾添加 Kind 部署目标**

```makefile
# =========================
# Kind 开发环境相关命令
# =========================

## kind-up: 创建集群并部署完整开发环境
kind-up: kind-cluster kind-load-images kind-deploy
	@echo ""
	@echo "✅ USMP Kind 开发环境部署完成！"
	@echo ""
	@echo "访问地址："
	@echo "  前端界面: http://localhost:30081"
	@echo "  后端 API: http://localhost:30080"
	@echo "  NETCONF 模拟器: localhost:30830"
	@echo ""
	@echo "查看状态: make kind-status"
	@echo "查看日志: make kind-logs"
	@echo "清理环境: make kind-clean"

## kind-cluster: 创建 Kind 集群
kind-cluster:
	@echo "Creating Kind cluster for USMP development..."
	kind create cluster --name usmp-dev --config deploy/kind-cluster.yaml
	@echo "Kind cluster created successfully"

## kind-delete: 删除 Kind 集群 (别名)
kind-delete: kind-clean

## kind-clean: 删除 Kind 集群
kind-clean:
	@echo "Cleaning up Kind cluster..."
	kind delete cluster --name usmp-dev
	@echo "Kind cluster cleaned up"

## kind-deploy: 部署/更新所有组件到 Kind 集群
kind-deploy:
	@echo "Deploying resources to Kind cluster..."
	kubectl --context=kind-usmp-dev apply -k deploy/manifests/
	@echo "Waiting for pods to be ready..."
	kubectl --context=kind-usmp-dev -n usmp-system wait --for=condition=Ready pods --all --timeout=120s || true
	@echo "Deployment completed"

## kind-redeploy: 重新部署（先删除再应用）
kind-redeploy:
	@echo "Redeploying all resources..."
	kubectl --context=kind-usmp-dev delete -k deploy/manifests/ || true
	kubectl --context=kind-usmp-dev apply -k deploy/manifests/
	@echo "Redeployment completed"

## kind-status: 查看 Kind 集群状态
kind-status:
	@echo "=== USMP Kind Development Cluster Status ==="
	@echo ""
	@echo "Cluster info:"
	@kubectl --context=kind-usmp-dev cluster-info 2>/dev/null || echo "Cluster not found or not accessible"
	@echo ""
	@echo "Namespace usmp-system resources:"
	@kubectl --context=kind-usmp-dev -n usmp-system get all 2>/dev/null || echo "Namespace usmp-system not found"
	@echo ""
	@echo "CRD Instances:"
	@kubectl --context=kind-usmp-dev -n usmp-system get businessswitches,businessvlans,businessinterfaces,businessroutes,nativedeviceconfigs 2>/dev/null || echo "No CRD instances found"

## kind-logs: 查看 Controller 日志
kind-logs:
	@echo "Controller manager logs:"
	@kubectl --context=kind-usmp-dev -n usmp-system logs -l control-plane=controller-manager --tail=50 -f 2>/dev/null || echo "No controller pod found or log error"

## kind-frontend-logs: 查看前端日志
kind-frontend-logs:
	@echo "Frontend logs:"
	@kubectl --context=kind-usmp-dev -n usmp-system logs -l app=usmp-frontend --tail=50 -f 2>/dev/null || echo "No frontend pod found or log error"

## kind-simulator-logs: 查看模拟器日志
kind-simulator-logs:
	@echo "NETCONF Simulator logs:"
	@kubectl --context=kind-usmp-dev -n usmp-system logs -l app=netconf-simulator --tail=50 -f 2>/dev/null || echo "No simulator pod found or log error"

## kind-load-images: 加载本地镜像到 Kind 集群
kind-load-images: docker-build docker-build-frontend
	@echo "Loading Docker images to Kind cluster..."
	kind load docker-image usmp-controller:latest --name usmp-dev
	kind load docker-image usmp-frontend:latest --name usmp-dev
	@echo "Docker images loaded to Kind cluster"

## docker-build-frontend: 构建前端 Docker 镜像
docker-build-frontend:
	@echo "Building frontend Docker image..."
	cd ../frontend && docker build -t usmp-frontend:latest .
	@echo "Frontend Docker image built successfully"

## kind-shell: 在 Controller Pod 中打开 Shell
kind-shell:
	@echo "Opening shell in controller pod..."
	@kubectl --context=kind-usmp-dev -n usmp-system exec -it deployment/usmp-controller -- sh 2>/dev/null || echo "Failed to open shell"

## kind-port-forward: 端口转发（用于本地调试）
kind-port-forward:
	@echo "Starting port forwarding..."
	@echo "API: 8080 -> 30080, Frontend: 30081 -> 30081"
	kubectl --context=kind-usmp-dev -n usmp-system port-forward service/usmp-controller 8080:8080 &
	kubectl --context=kind-usmp-dev -n usmp-system port-forward service/usmp-frontend 30081:80 &
	@echo "Port forwarding started. Press Ctrl+C to stop."
	wait
```

- [ ] **Step 2: 验证 Makefile 语法**

```bash
cd backend && make help | grep kind
```

Expected: 显示所有 kind-* 目标

- [ ] **Step 3: Commit**

```bash
git add backend/Makefile
git commit -m "feat(deploy): 添加 Kind 部署相关 Makefile 目标 (15+ commands)"
```

---

## 第六阶段：端到端测试

### Task 7: 完整部署测试

**Files:**
- None (test commands only)

- [ ] **Step 1: 构建所有镜像**

```bash
cd backend && make docker-build && make docker-build-frontend
```

Expected: 两个镜像构建成功

- [ ] **Step 2: 创建 Kind 集群**

```bash
cd backend && make kind-cluster
```

Expected: Kind 集群 usmp-dev 创建成功

- [ ] **Step 3: 加载镜像**

```bash
cd backend && make kind-load-images
```

Expected: 镜像加载成功

- [ ] **Step 4: 部署所有组件**

```bash
cd backend && make kind-deploy
```

Expected: 所有资源创建成功

- [ ] **Step 5: 检查集群状态**

```bash
cd backend && make kind-status
```

Expected: 3 个 Pod 全部 Running

- [ ] **Step 6: 验证前端可访问**

```bash
curl -s http://localhost:30081/healthz
```

Expected: 返回 "healthy"

- [ ] **Step 7: 验证后端 API 可访问**

```bash
curl -s http://localhost:30080/healthz
```

Expected: 返回健康检查响应

- [ ] **Step 8: 清理测试环境**

```bash
cd backend && make kind-clean
```

- [ ] **Step 9: 更新 README 添加部署说明**

在 README.md 的"快速开始"部分添加 Kind 部署说明

- [ ] **Step 10: Commit README 更新**

```bash
git add README.md
git commit -m "docs: 添加 Kind 集群部署说明到 README"
```

---

## 完整验收清单

- [ ] 运行 `make kind-up` 一键部署成功
- [ ] 前端 http://localhost:30081 可访问
- [ ] 后端 API http://localhost:30080 健康检查正常
- [ ] NETCONF 模拟器 localhost:30830 可连接
- [ ] `make kind-status` 正确显示集群状态
- [ ] `make kind-logs` 可查看 Controller 日志
- [ ] `make kind-clean` 正确清理集群
- [ ] 所有 3 个 Pod 处于 Running 状态
- [ ] kubectl kustomize 无语法错误
