#!/usr/bin/env bash
# 本地全栈热循环 —— 免 docker，秒级迭代。
#
#   后端：go build 后直接运行（:8080）
#   前端：vite dev（:3000，HMR，/api 代理到 :8080）
#
# 设备接线说明：种子设备经 USMP_SEED_DEVICE 注入（DS-03，本脚本缺省注入
# 192.168.1.1:830,admin,admin 保持既有开发体验）。本地无 docker 自定义网桥，该地址
# 不可达 → 设备展示「离线」。这对前端/API 迭代无碍：页面渲染、路由、动态表单、
# YANG 契约、绝大多数 API 端点照常工作。
# 需要「设备在线」的端到端（配置读写对账、E2E）请用 docker 路径：make staging-up / make e2e-local
# （它把 simulator 固定到 192.168.1.1:830，正确对齐种子设备）。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$(mktemp -t usmp-dev-backend.XXXXXX)"

# 预检 :8080 是否已被占用（后端硬编码该端口，无法退让）
if (exec 3<>/dev/tcp/127.0.0.1/8080) 2>/dev/null; then
  exec 3>&- 3<&- 2>/dev/null || true
  echo "[dev] ✗ 端口 :8080 已被占用，后端无法启动。请先释放（lsof -i:8080 / ss -tlnp | grep 8080）。"
  exit 1
fi

cleanup() {
  echo
  echo "[dev] 停止..."
  [ -n "${BACK_PID:-}" ] && kill "$BACK_PID" 2>/dev/null || true
  rm -f "$BIN" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

echo "[dev] 编译后端..."
( cd "$ROOT/backend" && go build -o "$BIN" . )

echo "[dev] 启动后端（:8080）..."
( cd "$ROOT/backend" && USMP_SEED_DEVICE="${USMP_SEED_DEVICE:-192.168.1.1:830,admin,admin}" exec "$BIN" ) &
BACK_PID=$!

echo "[dev] 等待后端就绪..."
for _ in $(seq 1 60); do
  if curl -fsS -o /dev/null http://localhost:8080/api/v1/yang/modules 2>/dev/null; then
    echo "[dev] 后端就绪 ✓  http://localhost:8080/api/v1"
    break
  fi
  if ! kill -0 "$BACK_PID" 2>/dev/null; then
    echo "[dev] ✗ 后端启动失败，中止"
    exit 1
  fi
  sleep 1
done

echo "[dev] 启动前端 vite dev（:3000，HMR）..."
echo "[dev] → 前端 http://localhost:3000    后端 http://localhost:8080/api/v1"
echo "[dev] （Ctrl-C 同时停止前后端）"
cd "$ROOT/frontend"
[ -d node_modules ] || npm ci --prefer-offline --no-audit --fund=false
exec npm run dev
