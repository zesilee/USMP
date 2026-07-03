#!/usr/bin/env bash
# USMP L2 命令拦截 — Claude Code PreToolUse hook
# 在 AI Agent 执行 Bash 命令前拦截违规操作
# 退出码 0 = 允许执行, 1 = 阻止执行
set -euo pipefail

INPUT="${1:-}"

# 提取命令内容
CMD=$(echo "$INPUT" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('command',''))" 2>/dev/null || echo "$INPUT")

# ──────────────────────────────────────────────
# 1. R13: 禁止直接 push main
# ──────────────────────────────────────────────
if echo "$CMD" | grep -qE 'git\s+push\s+.*\bmain\b'; then
  if ! echo "$CMD" | grep -qE '(hotfix|hot-fix)'; then
    echo "[L2 拦截 R13] 禁止直接 push main，使用 PR 合入 (TEAM_HANDBOOK.md §7)"
    exit 1
  fi
fi

# ──────────────────────────────────────────────
# 2. W07: 禁止 force push
# ──────────────────────────────────────────────
if echo "$CMD" | grep -qE 'git\s+push.*--force'; then
  echo "[L2 拦截 W07] 禁止 force push，Hotfix 除外需 Maintainer 确认"
  exit 1
fi

# ──────────────────────────────────────────────
# 3. R04: 禁止直接编辑 generated 目录
# ──────────────────────────────────────────────
if echo "$CMD" | grep -qE '(vi|nano|vim|code|sed|echo.*>).*internal/generated/'; then
  echo "[L2 拦截 R04] 禁止手动编辑 generated/ 目录，使用 yang-ygot-generate 技能重新生成"
  exit 1
fi

# ──────────────────────────────────────────────
# 4. 禁止在 main 分支直接开发
# ──────────────────────────────────────────────
if echo "$CMD" | grep -qE 'git\s+checkout\s+main\b'; then
  echo "[L2 拦截 W01] 禁止在 main 分支开发，使用 EnterWorktree 创建隔离环境 (CLAUDE.md §6)"
  exit 1
fi

# ──────────────────────────────────────────────
# 5. 破坏性命令拦截
# ──────────────────────────────────────────────
if echo "$CMD" | grep -qE 'rm\s+-rf\s+/'; then
  echo "[L2 拦截] 禁止递归删除根路径"
  exit 1
fi

# ──────────────────────────────────────────────
# 6. R16: 敏感文件写入拦截
# ──────────────────────────────────────────────
if echo "$CMD" | grep -qE '(echo|cat|tee).*\>\s*.*\.env'; then
  echo "[L2 拦截 R16] 禁止写入 .env 文件，使用环境变量或配置管理"
  exit 1
fi

exit 0
