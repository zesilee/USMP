#!/usr/bin/env bash
# scripts/link-memory.sh 的行为测试（纯 bash，无外部测试框架依赖）
#
# 每个用例都在临时 git 仓库沙箱里跑真脚本、真 git worktree、真软链接，
# 不打桩——因为这个脚本唯一的价值就是正确操作文件系统。
#
# 运行: ./scripts/test/link-memory_test.sh   或   make memory-test
set -uo pipefail

SCRIPT_UNDER_TEST="$(cd "$(dirname "$0")/.." && pwd)/link-memory.sh"

PASS=0
FAIL=0
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[0;33m'; NC='\033[0m'

ok()   { PASS=$((PASS + 1)); echo -e "  ${GREEN}✓${NC} $1"; }
ng()   { FAIL=$((FAIL + 1)); echo -e "  ${RED}✗${NC} $1"; [ $# -gt 1 ] && echo "      $2"; }
case_() { echo -e "${YELLOW}$1${NC}"; }

assert_link_to() {
  local link="$1" want="$2" msg="$3"
  if [ ! -L "$link" ]; then ng "$msg" "不是软链接: $link"; return; fi
  local got; got="$(readlink -f "$link" 2>/dev/null || true)"
  local wantabs; wantabs="$(readlink -f "$want" 2>/dev/null || true)"
  if [ -n "$got" ] && [ "$got" = "$wantabs" ]; then ok "$msg"; else ng "$msg" "指向 $got，期望 $wantabs"; fi
}

assert_file_has() {
  local f="$1" want="$2" msg="$3"
  if [ -f "$f" ] && grep -qF "$want" "$f"; then ok "$msg"; else ng "$msg" "文件缺失或内容不含 '$want': $f"; fi
}

# 建一个沙箱：临时 git 仓库 + 临时 CLAUDE_CONFIG_DIR，导出 REPO / CFG / PROJECTS
setup_sandbox() {
  SANDBOX="$(mktemp -d)"
  REPO="$SANDBOX/my_repo"          # 故意带下划线，验证编码规则
  CFG="$SANDBOX/cfghome/.claude"
  PROJECTS="$CFG/projects"
  mkdir -p "$REPO" "$PROJECTS"
  git -C "$REPO" init -q -b main
  mkdir -p "$REPO/scripts" "$REPO/docs/memory"
  cp "$SCRIPT_UNDER_TEST" "$REPO/scripts/link-memory.sh"
  chmod +x "$REPO/scripts/link-memory.sh"
  echo "seed" > "$REPO/docs/memory/MEMORY.md"
  git -C "$REPO" add -A >/dev/null
  git -C "$REPO" -c user.email=t@t -c user.name=t commit -qm init
}

teardown_sandbox() { [ -n "${SANDBOX:-}" ] && rm -rf "$SANDBOX"; }

# 把绝对路径编码成 harness 的项目目录名（/ . _ 全部转 -）
encode_path() { echo "$1" | sed 's|[/._]|-|g'; }

run_link() { (cd "$REPO" && CLAUDE_CONFIG_DIR="$CFG" ./scripts/link-memory.sh "$@" 2>&1); }

# ──────────────────────────────────────────────
case_ "T1 全新环境：记忆目录不存在 → 建立软链接"
setup_sandbox
run_link >/dev/null
assert_link_to "$PROJECTS/$(encode_path "$REPO")/memory" "$REPO/docs/memory" "主仓记忆目录已链到 docs/memory"
teardown_sandbox

# ──────────────────────────────────────────────
case_ "T2 幂等：重复执行不报错、不改变结果"
setup_sandbox
run_link >/dev/null
OUT2="$(run_link)"; RC2=$?
[ "$RC2" -eq 0 ] && ok "第二次执行退出码 0" || ng "第二次执行退出码 0" "实际 $RC2"
assert_link_to "$PROJECTS/$(encode_path "$REPO")/memory" "$REPO/docs/memory" "链接仍然正确"
BAK_COUNT=$(find "$PROJECTS" -maxdepth 2 -name 'memory.backup-*' 2>/dev/null | wc -l)
[ "$BAK_COUNT" -eq 0 ] && ok "幂等执行不产生多余备份" || ng "幂等执行不产生多余备份" "产生了 $BAK_COUNT 个备份"
teardown_sandbox

# ──────────────────────────────────────────────
case_ "T3 存量迁移：已有真实记忆目录 → 内容并入 docs/memory 且原目录备份"
setup_sandbox
MEMDIR="$PROJECTS/$(encode_path "$REPO")/memory"
mkdir -p "$MEMDIR"
echo "旧记忆内容" > "$MEMDIR/legacy-note.md"
run_link >/dev/null
assert_file_has "$REPO/docs/memory/legacy-note.md" "旧记忆内容" "存量记忆已并入 docs/memory"
assert_link_to "$MEMDIR" "$REPO/docs/memory" "原目录已被软链接取代"
BAK=$(find "$PROJECTS" -maxdepth 2 -name 'memory.backup-*' | head -1)
if [ -n "$BAK" ] && [ -f "$BAK/legacy-note.md" ]; then ok "原目录已完整备份（不做破坏性删除）"; else ng "原目录已完整备份（不做破坏性删除）" "未找到备份"; fi
teardown_sandbox

# ──────────────────────────────────────────────
case_ "T4 冲突保护：同名文件内容不同 → 仓库版本优先，原版留在备份里"
setup_sandbox
echo "仓库版本" > "$REPO/docs/memory/dup.md"
MEMDIR="$PROJECTS/$(encode_path "$REPO")/memory"
mkdir -p "$MEMDIR"
echo "本地版本" > "$MEMDIR/dup.md"
OUT="$(run_link)"
assert_file_has "$REPO/docs/memory/dup.md" "仓库版本" "仓库版本未被覆盖"
BAK=$(find "$PROJECTS" -maxdepth 2 -name 'memory.backup-*' | head -1)
assert_file_has "$BAK/dup.md" "本地版本" "本地版本保留在备份中"
echo "$OUT" | grep -qi "冲突" && ok "冲突有明确告警" || ng "冲突有明确告警" "输出未提示冲突"
teardown_sandbox

# ──────────────────────────────────────────────
case_ "T5 断链修复：软链接指向不存在的路径 → 重建"
setup_sandbox
MEMPARENT="$PROJECTS/$(encode_path "$REPO")"
mkdir -p "$MEMPARENT"
ln -s "$SANDBOX/nowhere" "$MEMPARENT/memory"
run_link >/dev/null
assert_link_to "$MEMPARENT/memory" "$REPO/docs/memory" "断链已修复"
teardown_sandbox

# ──────────────────────────────────────────────
case_ "T6 错指修复：软链接指向别处 → 重新指向 docs/memory"
setup_sandbox
MEMPARENT="$PROJECTS/$(encode_path "$REPO")"
mkdir -p "$MEMPARENT" "$SANDBOX/elsewhere"
ln -s "$SANDBOX/elsewhere" "$MEMPARENT/memory"
run_link >/dev/null
assert_link_to "$MEMPARENT/memory" "$REPO/docs/memory" "已重新指向 docs/memory"
teardown_sandbox

# ──────────────────────────────────────────────
case_ "T7 worktree 覆盖：worktree 的记忆目录也链回主仓 docs/memory"
setup_sandbox
WT="$REPO/.claude/worktrees/feat_x"
git -C "$REPO" worktree add -q -b feat-x "$WT" >/dev/null 2>&1
run_link >/dev/null
assert_link_to "$PROJECTS/$(encode_path "$WT")/memory" "$REPO/docs/memory" "worktree 记忆目录链回主仓（不随 worktree 删除而丢失）"
teardown_sandbox

# ──────────────────────────────────────────────
case_ "T8 --check 健康 → 退出码 0"
setup_sandbox
run_link >/dev/null
run_link --check >/dev/null; RC=$?
[ "$RC" -eq 0 ] && ok "健康时 --check 退出码 0" || ng "健康时 --check 退出码 0" "实际 $RC"
teardown_sandbox

# ──────────────────────────────────────────────
case_ "T9 --check 不健康 → 退出码非 0 且不修改任何东西"
setup_sandbox
run_link --check >/dev/null; RC=$?
[ "$RC" -ne 0 ] && ok "未链接时 --check 退出码非 0" || ng "未链接时 --check 退出码非 0" "实际 $RC"
if [ ! -e "$PROJECTS/$(encode_path "$REPO")/memory" ]; then ok "--check 是只读的，未擅自创建链接"; else ng "--check 是只读的，未擅自创建链接" "--check 产生了副作用"; fi
teardown_sandbox

# ──────────────────────────────────────────────
case_ "T10 路径编码：/ . _ 三种字符都转成 -"
setup_sandbox
WT="$REPO/.claude/worktrees/a_b"
git -C "$REPO" worktree add -q -b a-b "$WT" >/dev/null 2>&1
run_link >/dev/null
ENC="$(encode_path "$WT")"
echo "$ENC" | grep -q '[/._]' && ng "编码后不含 / . _" "编码结果: $ENC" || ok "编码后不含 / . _"
[ -L "$PROJECTS/$ENC/memory" ] && ok "按编码规则命中真实 harness 目录名" || ng "按编码规则命中真实 harness 目录名" "未找到 $PROJECTS/$ENC/memory"
teardown_sandbox

# ──────────────────────────────────────────────
echo ""
if [ "$FAIL" -gt 0 ]; then
  echo -e "${RED}link-memory 测试失败: ${PASS} 通过 / ${FAIL} 失败${NC}"
  exit 1
fi
echo -e "${GREEN}link-memory 测试全绿: ${PASS} 通过${NC}"
