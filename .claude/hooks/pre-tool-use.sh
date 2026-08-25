#!/usr/bin/env bash
# USMP L2 命令拦截 — Claude Code PreToolUse hook
# 在 AI Agent 执行 Bash 命令前拦截违规操作
# 退出码 0 = 允许执行, 2 = 阻止执行（Claude Code 钩子协议：exit 2 才阻断，1 只是非阻断报错）
set -euo pipefail

# Claude Code 以 stdin JSON 传递钩子载荷（{"tool_name":...,"tool_input":{"command":...}}）；
# 保留 $1 入参兼容手动调试。历史 bug：旧版只读 $1（settings 传的 $TOOL_INPUT 环境变量并不存在），
# 导致命令恒为空、全部放行——修改本文件前先跑文末的自测命令。
INPUT="${1:-}"
if [ -z "$INPUT" ] && [ ! -t 0 ]; then
  INPUT="$(cat || true)"
fi

# 提取命令内容：优先 tool_input.command（钩子协议），兼容裸 {"command":...} 与纯文本
CMD=$(printf '%s' "$INPUT" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
except Exception:
    sys.exit(1)
ti = d.get('tool_input', d)
print(ti.get('command', '') if isinstance(ti, dict) else '')
" 2>/dev/null) || CMD="$INPUT"
[ -z "$CMD" ] && CMD="$INPUT"

# ──────────────────────────────────────────────
# 1. R13: 禁止直接 push main
# ──────────────────────────────────────────────
if echo "$CMD" | grep -qE 'git\s+push\s+.*\bmain\b'; then
  if ! echo "$CMD" | grep -qE '(hotfix|hot-fix)'; then
    echo >&2 "[L2 拦截 R13] 禁止直接 push main，使用 PR 合入 (TEAM_HANDBOOK.md §7)"
    exit 2
  fi
fi

# ──────────────────────────────────────────────
# 2. W07: 禁止 force push
# ──────────────────────────────────────────────
if echo "$CMD" | grep -qE 'git\s+push.*--force'; then
  echo >&2 "[L2 拦截 W07] 禁止 force push，Hotfix 除外需 Maintainer 确认"
  exit 2
fi

# ──────────────────────────────────────────────
# 3. R04: 禁止直接编辑 generated 目录
# ──────────────────────────────────────────────
if echo "$CMD" | grep -qE '(vi|nano|vim|code|sed|echo.*>).*internal/generated/'; then
  echo >&2 "[L2 拦截 R04] 禁止手动编辑 generated/ 目录，走 make gen-yang 生成管线 regen-and-diff"
  exit 2
fi

# ──────────────────────────────────────────────
# 4. 禁止在 main 分支直接开发
# ──────────────────────────────────────────────
if echo "$CMD" | grep -qE 'git\s+checkout\s+main\b'; then
  echo >&2 "[L2 拦截 W01] 禁止在 main 分支开发，使用 EnterWorktree 创建隔离环境 (CLAUDE.md §6)"
  exit 2
fi

# ──────────────────────────────────────────────
# 5. 破坏性命令拦截
# ──────────────────────────────────────────────
if echo "$CMD" | grep -qE 'rm\s+-rf\s+/'; then
  echo >&2 "[L2 拦截] 禁止递归删除根路径"
  exit 2
fi

# ──────────────────────────────────────────────
# 6. R16: 敏感文件写入拦截
# ──────────────────────────────────────────────
if echo "$CMD" | grep -qE '(echo|cat|tee).*\>\s*.*\.env'; then
  echo >&2 "[L2 拦截 R16] 禁止写入 .env 文件，使用环境变量或配置管理"
  exit 2
fi

exit 0

# 自测（改动本文件后必跑，期望：第一条 exit 2、第二条 exit 0）：
#   echo '{"tool_name":"Bash","tool_input":{"command":"git push origin main"}}' | .claude/hooks/pre-tool-use.sh; echo $?
#   echo '{"tool_name":"Bash","tool_input":{"command":"git status"}}' | .claude/hooks/pre-tool-use.sh; echo $?
