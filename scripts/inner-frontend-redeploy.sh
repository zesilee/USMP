#!/usr/bin/env bash
# 内网前端一键重部署（迭代快路径）：拉码 → 宿主构建 dist → prebuilt 镜像 →
# kind load → 滚动重启 frontend。全栈首次部署仍用 scripts/kind-deploy.sh。
# 背景：目视验收迭代期每轮手敲 4 条命令且易漏（构建没跑完就 deploy、忘强刷）。
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CLUSTER="${USMP_KIND_CLUSTER:-usmp}"
NS="${USMP_NS:-usmp-system}"

log() { printf '\033[1;33m[inner-fe] %s\033[0m\n' "$*"; }

log "git pull"
git -C "$ROOT" pull --ff-only

log "宿主构建 dist（缺省口径=eview+inula 交付产物）"
(cd "$ROOT/frontend" && npm run build)

log "构建 prebuilt 镜像并载入 kind(${CLUSTER})"
docker build -t usmp-frontend:latest -f "$ROOT/frontend/Dockerfile.prebuilt" "$ROOT/frontend"
kind load docker-image usmp-frontend:latest --name "$CLUSTER"

log "滚动重启 usmp-frontend 并等待就绪"
kubectl -n "$NS" rollout restart deployment usmp-frontend
kubectl -n "$NS" rollout status deployment usmp-frontend --timeout=120s

log "完成。浏览器强制刷新（Ctrl+Shift+R）后验收。"
