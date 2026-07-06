#!/usr/bin/env bash
# USMP 本地 e2e smoke —— 与 e2e-staging 工作流等价的浏览器冒烟。
#
# 起本地 staging 全栈（simulator + backend + frontend，docker compose），等健康，
# 再用 Playwright 跑 tests/staging-smoke.spec.ts（chromium）。端口从 `docker compose port`
# 自动发现，兼容 docker-compose.override.yml 的端口重映射（如本机 8080 被占用改 18080）。
#
# 退出码即测试结果；被 .githooks/pre-push 在前端变更时调用（可 USMP_SKIP_E2E=1 显式跳过）。
set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[0;33m'; NC='\033[0m'
cd "$(git rev-parse --show-toplevel)"

if ! docker compose version >/dev/null 2>&1; then
  echo -e "${RED}[e2e-smoke] 需要 docker compose（未检测到）。安装后重试，或 git push --no-verify 紧急绕过。${NC}"
  exit 1
fi

echo -e "${YELLOW}[e2e-smoke] 构建并起本地 staging...${NC}"
docker compose up -d --build --remove-orphans

# 端口自动发现（尊重 override 的重映射；失败回退默认）
BE_PORT=$(docker compose port backend 8080 2>/dev/null | sed -E 's/.*:([0-9]+)$/\1/'); BE_PORT=${BE_PORT:-8080}
FE_PORT=$(docker compose port frontend 80 2>/dev/null | sed -E 's/.*:([0-9]+)$/\1/'); FE_PORT=${FE_PORT:-3002}
echo -e "${YELLOW}[e2e-smoke] backend :$BE_PORT  frontend :$FE_PORT${NC}"

echo -e "${YELLOW}[e2e-smoke] 等待健康...${NC}"
be_ok=0; for i in $(seq 1 40); do curl -fsS -o /dev/null "http://localhost:$BE_PORT/api/v1/yang/modules" && { be_ok=1; break; }; sleep 3; done
fe_ok=0; for i in $(seq 1 20); do curl -fsS -o /dev/null "http://localhost:$FE_PORT/healthz" && { fe_ok=1; break; }; sleep 3; done
if [ "$be_ok" != 1 ] || [ "$fe_ok" != 1 ]; then
  echo -e "${RED}[e2e-smoke] 后端/前端未就绪（be=$be_ok fe=$fe_ok）${NC}"; docker compose ps; exit 1
fi

# 预热种子设备（best-effort：让「设备在线」，失败不阻断——列表本就有种子设备）
curl -fsS -X POST "http://localhost:$BE_PORT/api/v1/devices" -H 'Content-Type: application/json' \
  -d '{"ip":"192.168.1.1","port":830,"username":"admin","password":"admin"}' >/dev/null 2>&1 || true

echo -e "${YELLOW}[e2e-smoke] 运行 Playwright staging smoke...${NC}"
cd frontend
npm ci --prefer-offline --no-audit --fund=false
npx playwright install chromium
PLAYWRIGHT_BASE_URL="http://localhost:$FE_PORT" \
  npx playwright test tests/staging-smoke.spec.ts --project=chromium --reporter=list

echo -e "${GREEN}[e2e-smoke] ✅ 通过（staging 仍在运行，make staging-down 可停止）${NC}"
