#!/bin/sh
# USMP 发布包一键启动脚本（随 usmp-release-*.zip 分发）
#
# 用法：解压 zip 后执行  sh usmp/start.sh
#   前端 http://localhost:${USMP_WEB_PORT:-3002}    后端 http://localhost:8080/api/v1
#
# 设计约束：
#   - POSIX sh 编写：目标 docker 基础镜像（如 alpine）可能没有 bash
#   - 自带静态站服务（bin/usmp-web），目标镜像无需预装 nginx/node
#   - 可直接作为容器 ENTRYPOINT：前台守护，任一子进程退出则整体退出（交给编排重启）
#   - 环境变量透传：USMP_SEED_DEVICE（种子设备 ip[:port],user,pass[,vendor]）、
#     USMP_WEB_PORT（前端端口，默认 3002；后端固定 :8080）
set -eu

DIR=$(cd "$(dirname "$0")" && pwd)
WEB_PORT="${USMP_WEB_PORT:-3002}"

# 部分解压工具（如 python -m zipfile）不保留可执行位，兜底补上
chmod +x "$DIR/bin/usmp-backend" "$DIR/bin/usmp-web" 2>/dev/null || true

[ -x "$DIR/bin/usmp-backend" ] || { echo "[usmp] ✗ 缺少 bin/usmp-backend，发布包不完整"; exit 1; }
[ -x "$DIR/bin/usmp-web" ]     || { echo "[usmp] ✗ 缺少 bin/usmp-web，发布包不完整"; exit 1; }
[ -f "$DIR/web/index.html" ]   || { echo "[usmp] ✗ 缺少 web/index.html，发布包不完整"; exit 1; }

echo "[usmp] 启动后端控制器 (:8080)..."
"$DIR/bin/usmp-backend" &
BACKEND_PID=$!

echo "[usmp] 启动前端静态站 (:$WEB_PORT)..."
USMP_WEB_ROOT="$DIR/web" USMP_WEB_PORT="$WEB_PORT" "$DIR/bin/usmp-web" &
WEB_PID=$!

stop_all() {
    echo "[usmp] 收到停止信号，退出..."
    kill "$BACKEND_PID" "$WEB_PID" 2>/dev/null || true
    wait "$BACKEND_PID" "$WEB_PID" 2>/dev/null || true
    exit 0
}
trap stop_all TERM INT

# 就绪探测：优先 curl，退而 wget（busybox 自带），都没有则跳过探测直接守护
probe() {
    if command -v curl >/dev/null 2>&1; then
        curl -fsS -o /dev/null "$1" 2>/dev/null
    elif command -v wget >/dev/null 2>&1; then
        wget -q -O /dev/null "$1" 2>/dev/null
    else
        return 0
    fi
}

i=0
while [ "$i" -lt 30 ]; do
    if probe "http://127.0.0.1:8080/api/v1/yang/modules" && probe "http://127.0.0.1:$WEB_PORT/healthz"; then
        echo "[usmp] 就绪 ✓  前端 http://localhost:$WEB_PORT    后端 http://localhost:8080/api/v1"
        break
    fi
    i=$((i + 1))
    sleep 1
done

# 前台守护：任一子进程退出则整体以非零退出（容器语义，交给编排系统重启）
while :; do
    if ! kill -0 "$BACKEND_PID" 2>/dev/null; then
        echo "[usmp] ✗ 后端进程退出，停止服务"
        kill "$WEB_PID" 2>/dev/null || true
        exit 1
    fi
    if ! kill -0 "$WEB_PID" 2>/dev/null; then
        echo "[usmp] ✗ 前端静态站进程退出，停止服务"
        kill "$BACKEND_PID" 2>/dev/null || true
        exit 1
    fi
    sleep 2
done
