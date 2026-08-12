#!/usr/bin/env bash
# scripts/check-binary-files.sh 的行为测试（纯 bash，无外部测试框架依赖）
#
# 每个用例都在临时 git 仓库沙箱里跑真脚本、真 git 暂存区/提交区间——
# 因为这个脚本唯一的价值就是正确复用 git 的二进制判定。
#
# 运行: ./scripts/test/check-binary-files_test.sh   或   make binary-guard-test
set -uo pipefail

SCRIPT_UNDER_TEST="$(cd "$(dirname "$0")/.." && pwd)/check-binary-files.sh"

PASS=0
FAIL=0
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[0;33m'; NC='\033[0m'

ok()   { PASS=$((PASS + 1)); echo -e "  ${GREEN}✓${NC} $1"; }
ng()   { FAIL=$((FAIL + 1)); echo -e "  ${RED}✗${NC} $1"; [ $# -gt 1 ] && echo "      $2"; }
case_() { echo -e "${YELLOW}$1${NC}"; }

# 沙箱：临时 git 仓库（含一个初始提交，使 --cached / 区间 diff 都可用）
new_sandbox() {
  SANDBOX="$(mktemp -d)"
  (
    cd "$SANDBOX"
    git init -q
    git config user.email t@t && git config user.name t
    git config core.hooksPath /dev/null   # 沙箱内不受本仓库钩子影响
    echo seed > seed.txt
    git add . && git commit -qm seed
  )
}
cleanup_sandbox() { rm -rf "$SANDBOX"; }

# 生成一个必被 git 判为二进制的文件（含 NUL 字节）
make_binary() { printf 'BIN\0DATA\0%s' "$RANDOM" > "$1"; }

run_cached() { (cd "$SANDBOX" && "$SCRIPT_UNDER_TEST" --cached); }
run_range()  { (cd "$SANDBOX" && "$SCRIPT_UNDER_TEST" "$1"); }

# ── 用例 1：暂存二进制文件 → 拦截 ─────────────────────
case_ "用例1: 暂存二进制文件被拦截"
new_sandbox
make_binary "$SANDBOX/blob.gz"
(cd "$SANDBOX" && git add blob.gz)
if ! out=$(run_cached); then
  echo "$out" | grep -q "blob.gz" && ok "拦截并列出 blob.gz" || ng "拦截了但未列出文件名" "$out"
else
  ng "二进制文件未被拦截"
fi
cleanup_sandbox

# ── 用例 2：纯文本变更 → 放行 ─────────────────────────
case_ "用例2: 纯文本变更放行"
new_sandbox
echo "plain text" > "$SANDBOX/note.txt"
(cd "$SANDBOX" && git add note.txt)
run_cached >/dev/null && ok "文本文件放行" || ng "文本文件被误拦"
cleanup_sandbox

# ── 用例 3：docs/ 配图白名单 → 放行 ───────────────────
case_ "用例3: docs/ 文档配图白名单放行"
new_sandbox
mkdir -p "$SANDBOX/docs/research/assets"
make_binary "$SANDBOX/docs/research/assets/shot.png"
(cd "$SANDBOX" && git add docs)
run_cached >/dev/null && ok "docs/ 下 png 放行" || ng "docs/ 配图被误拦"
cleanup_sandbox

# ── 用例 4：docs/ 外的图片 → 拦截 ─────────────────────
case_ "用例4: docs/ 之外的二进制图片仍拦截"
new_sandbox
mkdir -p "$SANDBOX/frontend/assets"
make_binary "$SANDBOX/frontend/assets/logo.png"
(cd "$SANDBOX" && git add frontend)
if ! run_cached >/dev/null; then ok "docs/ 外 png 被拦截"; else ng "docs/ 外 png 未被拦截"; fi
cleanup_sandbox

# ── 用例 5：删除已入库二进制 → 放行（清债不受阻） ──────
case_ "用例5: 删除存量二进制放行"
new_sandbox
make_binary "$SANDBOX/legacy.bin"
(cd "$SANDBOX" && git add legacy.bin && git commit -qm add-bin && git rm -q legacy.bin)
run_cached >/dev/null && ok "二进制删除放行" || ng "删除二进制被误拦（清债会被卡死）"
cleanup_sandbox

# ── 用例 6：提交区间模式（CI 口径） → 拦截 ────────────
case_ "用例6: 区间模式检出已提交的二进制"
new_sandbox
BASE_BRANCH="$(cd "$SANDBOX" && git symbolic-ref --short HEAD)"
(cd "$SANDBOX" && git checkout -qb feature)
make_binary "$SANDBOX/artifact.tar"
(cd "$SANDBOX" && git add artifact.tar && git commit -qm add-artifact)
if ! out=$(run_range "${BASE_BRANCH}...feature"); then
  echo "$out" | grep -q "artifact.tar" && ok "区间模式拦截已提交二进制" || ng "拦截了但未列出文件名" "$out"
else
  ng "区间模式未检出二进制"
fi
cleanup_sandbox

# ── 用例 7：空暂存区 → 放行 ───────────────────────────
case_ "用例7: 空暂存区不误报"
new_sandbox
run_cached >/dev/null && ok "空暂存区放行" || ng "空暂存区误报"
cleanup_sandbox

echo ""
if [ "$FAIL" -gt 0 ]; then
  echo -e "${RED}❌ $FAIL 个用例失败（$PASS 通过）${NC}"
  exit 1
fi
echo -e "${GREEN}✅ 全部 $PASS 个用例通过${NC}"
