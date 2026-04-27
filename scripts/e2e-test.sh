#!/bin/bash
# E2E 测试启动脚本 - 同时启动后端测试服务器和前端 dev 服务器

set -e

echo "="
echo "启动 E2E 测试环境..."
echo "="

# 进入后端目录
cd "$(dirname "$0")/.."

# 检查端口
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo "端口 8080 已被占用，正在清理..."
    lsof -Pi :8080 -sTCP:LISTEN -t | xargs kill -9 2>/dev/null || true
    sleep 1
fi

# 启动后端测试服务器
echo "启动后端测试服务器..."
cd backend
go run ./cmd/test-server/main.go > /tmp/e2e-backend.log 2>&1 &
BACKEND_PID=$!

# 等待后端启动
sleep 3

# 检查后端是否启动成功
if curl -s http://localhost:8080/api/v1/devices > /dev/null 2>&1; then
    echo "✅ 后端测试服务器启动成功 (PID: $BACKEND_PID)"
else
    echo "❌ 后端启动失败"
    cat /tmp/e2e-backend.log
    kill $BACKEND_PID 2>/dev/null || true
    exit 1
fi

# 启动前端并运行 E2E 测试
echo ""
echo "启动前端并运行 E2E 测试..."
echo "="

cd ../frontend

# 运行 Playwright 测试
npx playwright test "$@"
TEST_EXIT_CODE=$?

# 清理后端进程
echo ""
echo "="
echo "清理测试环境..."
kill $BACKEND_PID 2>/dev/null || true
echo "后端服务已停止 (PID: $BACKEND_PID)"
echo "测试结果: $TEST_EXIT_CODE"
echo "="

exit $TEST_EXIT_CODE
