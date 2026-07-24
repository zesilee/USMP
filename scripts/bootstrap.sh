#!/usr/bin/env bash
# USMP Bootstrap — 克隆后一键激活全流程
# 用法: ./scripts/bootstrap.sh 或 make setup
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m'

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$PROJECT_ROOT"

echo -e "${CYAN}🔧 USMP Bootstrap — 激活开发环境${NC}"
echo ""

FAILED=0

# ──────────────────────────────────────────────
# 1. Git Hooks 激活
# ──────────────────────────────────────────────
echo -e "${YELLOW}[1/8] 激活 Git Hooks...${NC}"
git config core.hooksPath .githooks
chmod +x .githooks/pre-commit .githooks/commit-msg .githooks/pre-push 2>/dev/null || true
if [ -f .githooks/post-checkout ]; then
  chmod +x .githooks/post-checkout
fi
echo -e "${GREEN}  ✅ core.hooksPath = .githooks${NC}"

# ──────────────────────────────────────────────
# 2. 后端依赖
# ──────────────────────────────────────────────
echo -e "${YELLOW}[2/8] 安装后端依赖...${NC}"
if [ -d backend ]; then
  (cd backend && go mod download 2>&1)
  echo -e "${GREEN}  ✅ Go 依赖已安装${NC}"
else
  echo -e "${YELLOW}  ⚠ backend/ 目录不存在，跳过${NC}"
fi

# ──────────────────────────────────────────────
# 3. 前端依赖
# ──────────────────────────────────────────────
echo -e "${YELLOW}[3/8] 安装前端依赖...${NC}"
if [ -d frontend ] && [ -f frontend/package.json ]; then
  (cd frontend && npm install --silent 2>&1 || npm install 2>&1)
  echo -e "${GREEN}  ✅ 前端依赖已安装${NC}"
else
  echo -e "${YELLOW}  ⚠ frontend/ 目录不存在，跳过${NC}"
fi

# ──────────────────────────────────────────────
# 4. 基线测试
# ──────────────────────────────────────────────
echo -e "${YELLOW}[4/8] 运行基线测试...${NC}"
if [ -d backend ]; then
  if (cd backend && go test ./... -count=1 -timeout=120s 2>&1); then
    echo -e "${GREEN}  ✅ 基线测试全绿${NC}"
  else
    echo -e "${RED}  ❌ 基线测试失败（可稍后重试: make test）${NC}"
    FAILED=1
  fi
else
  echo -e "${YELLOW}  ⚠ 跳过测试（无 backend/）${NC}"
fi

# ──────────────────────────────────────────────
# 5. 验证拦截体系
# ──────────────────────────────────────────────
echo -e "${YELLOW}[5/8] 验证拦截体系...${NC}"
HOOKS_PATH=$(git config core.hooksPath 2>/dev/null || echo "")
if [ "$HOOKS_PATH" = ".githooks" ]; then
  PRE_COMMIT=$([ -x .githooks/pre-commit ] && echo '✅' || echo '❌')
  COMMIT_MSG=$([ -x .githooks/commit-msg ] && echo '✅' || echo '❌')
  PRE_PUSH=$([ -x .githooks/pre-push ] && echo '✅' || echo '❌')
  echo -e "  pre-commit:  ${PRE_COMMIT}"
  echo -e "  commit-msg:  ${COMMIT_MSG}"
  echo -e "  pre-push:    ${PRE_PUSH}"
else
  echo -e "${RED}  ❌ Git Hooks 未激活${NC}"
  FAILED=1
fi

# ──────────────────────────────────────────────
# 6. 环境摘要
# ──────────────────────────────────────────────
echo ""
echo -e "${YELLOW}[6/8] 环境摘要${NC}"
echo -e "  CLAUDE.md:    $(wc -l < CLAUDE.md 2>/dev/null || echo '?') 行（AI 执行规范）"
echo -e "  TEAM_HANDBOOK:$(wc -l < TEAM_HANDBOOK.md 2>/dev/null || echo '?') 行（开发协作指南）"
echo -e "  OpenSpec:     $(ls openspec/specs/ 2>/dev/null | wc -l | tr -d ' ') 个能力规格"
echo -e "  CI Workflows: $(ls .github/workflows/ 2>/dev/null | wc -l | tr -d ' ') 个"
echo -e "  Git Hooks:    $(ls .githooks/ 2>/dev/null | wc -l | tr -d ' ') 个"

# ──────────────────────────────────────────────
# 7. 任务目录
# ──────────────────────────────────────────────
echo -e "${YELLOW}[7/8] 初始化任务目录...${NC}"
mkdir -p openspec/tasks/archive
IN_PROGRESS=$(grep -rl 'status: in_progress' openspec/tasks/ 2>/dev/null | grep -v archive | wc -l | tr -d ' ' || true)
PENDING=$(grep -rl 'status: pending' openspec/tasks/ 2>/dev/null | grep -v archive | wc -l | tr -d ' ' || true)
IN_PROGRESS=${IN_PROGRESS:-0}
PENDING=${PENDING:-0}
TOTAL=$((IN_PROGRESS + PENDING))
if [ "$TOTAL" -gt 0 ]; then
  echo -e "  ✅ openspec/tasks/ 就绪 (${IN_PROGRESS} 进行中, ${PENDING} 待处理)"
  echo -e "  💡 运行 /task list 查看未完成任务"
else
  echo -e "  ✅ openspec/tasks/ 就绪 (无进行中任务)"
fi

# ──────────────────────────────────────────────
# 8. AI 记忆归档链接
# ──────────────────────────────────────────────
# 记忆默认写在仓库外的 ~/.claude/projects/<编码路径>/memory/，换机器即全丢。
# 这一步把它换成指向 docs/memory 的软链接，让记忆随仓库进 git。
echo -e "${YELLOW}[8/8] 归档 AI 记忆到 docs/memory...${NC}"
if ./scripts/link-memory.sh >/dev/null 2>&1; then
  echo -e "${GREEN}  ✅ 记忆已链接到 docs/memory（$(ls docs/memory/*.md 2>/dev/null | wc -l | tr -d ' ') 条）${NC}"
else
  echo -e "${YELLOW}  ⚠ 记忆链接失败，可稍后重试: make memory-link${NC}"
fi

# ──────────────────────────────────────────────
# 结果
# ──────────────────────────────────────────────
echo ""
if [ "$FAILED" -eq 1 ]; then
  echo -e "${YELLOW}⚠️  Bootstrap 完成（部分步骤失败，请检查上方输出）${NC}"
  exit 1
fi

echo -e "${GREEN}✅ USMP 开发环境已就绪！${NC}"
echo ""
echo -e "${CYAN}快速开始:${NC}"
echo -e "  新功能: /opsx:explore → /opsx:propose → /opsx:apply"
echo -e "  Bug修复: 创建分支 → TDD → PR"
echo -e "  完整流程: cat TEAM_HANDBOOK.md"
