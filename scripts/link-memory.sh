#!/usr/bin/env bash
# USMP 记忆归档链接器 — 让 AI 记忆随仓库进 git，不再只躺在某台机器上
#
# 背景：AI 的项目记忆默认写在 ~/.claude/projects/<编码路径>/memory/，在仓库之外。
# 机器一换/一下线，所有「踩坑记录」「架构拍板」全丢。本脚本把那个目录换成指向
# docs/memory/ 的软链接，于是记忆的唯一真身在仓库里，跟着 PR 推到远端。
#
# 用法:
#   scripts/link-memory.sh          建立/修复链接（幂等，可反复跑）
#   scripts/link-memory.sh --check  只检查不改动，不健康时退出码非 0
#
# 安全保证：绝不删除有内容的目录——存量记忆先并入 docs/memory，原目录整体改名备份。
set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[0;33m'; CYAN='\033[0;36m'; NC='\033[0m'

MODE="link"
case "${1:-}" in
  --check) MODE="check" ;;
  "") ;;
  *) echo "用法: $0 [--check]" >&2; exit 2 ;;
esac

# ── 定位主仓根 ───────────────────────────────────────────
# 用 --git-common-dir 而非 --git-dir：从 worktree 里跑也能拿到主仓，
# 保证所有记忆都汇聚到同一个 docs/memory，不会散落在各 worktree 副本里。
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
GIT_COMMON="$(git -C "$SCRIPT_DIR" rev-parse --git-common-dir 2>/dev/null || true)"
if [ -z "$GIT_COMMON" ]; then
  echo -e "${RED}✗ 不在 git 仓库内，无法定位 docs/memory${NC}" >&2
  exit 2
fi
case "$GIT_COMMON" in
  /*) ;;
  *) GIT_COMMON="$(cd "$SCRIPT_DIR" && cd "$GIT_COMMON" && pwd)" ;;
esac
MAIN_ROOT="$(dirname "$GIT_COMMON")"
CANONICAL="$MAIN_ROOT/docs/memory"

CFG="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"
PROJECTS="$CFG/projects"

# harness 把项目绝对路径编码成目录名：/ . _ 三种字符统一转成 -
encode_path() { printf '%s' "$1" | sed 's|[/._]|-|g'; }

UNHEALTHY=0
LINKED=0
CONFLICT_TOTAL=0

# 需要收编的路径 = 主仓 + 所有 worktree（worktree 有独立记忆目录，
# 不一并链接的话，在 worktree 里写的记忆会随 worktree 删除而蒸发）
collect_paths() {
  echo "$MAIN_ROOT"
  git -C "$MAIN_ROOT" worktree list --porcelain 2>/dev/null \
    | sed -n 's|^worktree ||p' | grep -v "^${MAIN_ROOT}$" || true
}

# 把存量真实记忆目录并入 docs/memory：同名同内容跳过，同名异内容保留仓库版并告警
merge_existing() {
  local src="$1" conflicts=0
  shopt -s nullglob dotglob
  for f in "$src"/*; do
    local base; base="$(basename "$f")"
    if [ -e "$CANONICAL/$base" ]; then
      cmp -s "$f" "$CANONICAL/$base" || { conflicts=$((conflicts + 1)); echo -e "    ${YELLOW}冲突${NC} $base（保留仓库版本，本地版本在备份里）"; }
    else
      cp -a "$f" "$CANONICAL/$base"
      echo -e "    ${GREEN}并入${NC} $base"
    fi
  done
  shopt -u nullglob dotglob
  CONFLICT_TOTAL=$((CONFLICT_TOTAL + conflicts))
}

process_one() {
  local proj_path="$1"
  local enc; enc="$(encode_path "$proj_path")"
  local memdir="$PROJECTS/$enc/memory"
  local canon_abs; canon_abs="$(readlink -f "$CANONICAL" 2>/dev/null || echo "$CANONICAL")"
  local label; label="$(basename "$proj_path")"

  if [ -L "$memdir" ]; then
    local got; got="$(readlink -f "$memdir" 2>/dev/null || true)"
    if [ -n "$got" ] && [ "$got" = "$canon_abs" ]; then
      echo -e "  ${GREEN}✓${NC} ${label} 已链接"
      return
    fi
    if [ "$MODE" = "check" ]; then
      echo -e "  ${RED}✗${NC} ${label} 链接指向错误或已断链"; UNHEALTHY=1; return
    fi
    rm -f "$memdir"
    echo -e "  ${YELLOW}⟳${NC} ${label} 链接失效，重建"
  elif [ -d "$memdir" ]; then
    if [ "$MODE" = "check" ]; then
      echo -e "  ${RED}✗${NC} ${label} 仍是本地真实目录（未纳入仓库）"; UNHEALTHY=1; return
    fi
    echo -e "  ${YELLOW}⟳${NC} ${label} 发现存量记忆目录，迁移中"
    merge_existing "$memdir"
    local backup="$PROJECTS/$enc/memory.backup-$(date +%Y%m%d-%H%M%S)"
    mv "$memdir" "$backup"
    echo -e "    ${CYAN}备份${NC} $backup"
  elif [ -e "$memdir" ]; then
    if [ "$MODE" = "check" ]; then
      echo -e "  ${RED}✗${NC} ${label} 记忆路径被普通文件占用"; UNHEALTHY=1; return
    fi
    mv "$memdir" "$memdir.backup-$(date +%Y%m%d-%H%M%S)"
  else
    if [ "$MODE" = "check" ]; then
      echo -e "  ${RED}✗${NC} ${label} 尚未链接"; UNHEALTHY=1; return
    fi
  fi

  mkdir -p "$PROJECTS/$enc"
  ln -s "$CANONICAL" "$memdir"
  LINKED=$((LINKED + 1))
  echo -e "  ${GREEN}✓${NC} ${label} → docs/memory"
}

# ── 主流程 ───────────────────────────────────────────────
if [ "$MODE" = "check" ]; then
  echo -e "${CYAN}检查记忆归档链接…${NC}"
  if [ ! -d "$CANONICAL" ]; then
    echo -e "  ${RED}✗${NC} docs/memory 不存在"
    exit 1
  fi
else
  echo -e "${CYAN}🔗 归档 AI 记忆到 docs/memory${NC}"
  mkdir -p "$CANONICAL"
fi

while IFS= read -r p; do
  [ -n "$p" ] && process_one "$p"
done <<< "$(collect_paths)"

echo ""
if [ "$MODE" = "check" ]; then
  if [ "$UNHEALTHY" -eq 1 ]; then
    echo -e "${RED}✗ 记忆未完全归档到仓库，运行: make memory-link${NC}"
    exit 1
  fi
  echo -e "${GREEN}✅ 记忆归档链接健康${NC}"
  exit 0
fi

echo -e "${GREEN}✅ 完成${NC}（新建/修复 ${LINKED} 处，记忆真身: docs/memory）"
if [ "$CONFLICT_TOTAL" -gt 0 ]; then
  echo -e "${YELLOW}⚠ 有 ${CONFLICT_TOTAL} 个同名文件冲突，仓库版本已保留，本地版本见备份目录${NC}"
fi
echo -e "${CYAN}提示${NC}: 之后写的记忆会直接出现在 git status，记得随 PR 提交。"
