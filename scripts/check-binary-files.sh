#!/usr/bin/env sh
# check-binary-files.sh — R18: 禁止二进制文件入库
#
# 构建产物（如 schema.ir.gz）应构建期生成不入库；唯一白名单是 docs/ 下的
# 文档配图（截图是文档内容，不是产物；2026-08-12 用户拍板）。
# 判定复用 git 自身的二进制启发（numstat 行数列为 "-"），零自造 magic 检测。
#
# 用法:
#   scripts/check-binary-files.sh --cached           # pre-commit: 检查暂存区
#   scripts/check-binary-files.sh <base>...<head>    # CI: 检查提交区间
# 违规时列出文件并 exit 1。删除二进制不拦（--diff-filter=d 排除删除）。
set -eu

range="${1:---cached}"
if [ "$range" = "--cached" ]; then
    numstat=$(git diff --cached --numstat --no-renames --diff-filter=d)
else
    numstat=$(git diff --numstat --no-renames --diff-filter=d "$range")
fi

offenders=$(printf '%s\n' "$numstat" \
    | awk -F'\t' '$1 == "-" && $2 == "-" { print $3 }' \
    | grep -vE '^docs/.*\.(png|jpe?g|gif|svg|webp)$' || true)

if [ -n "$offenders" ]; then
    echo "[R18] 检测到二进制文件（禁止入库）:"
    printf '%s\n' "$offenders" | sed 's/^/  /'
    echo "构建产物请改为构建期生成（参考 make gen-schema-ir）；文档配图请放 docs/ 下。"
    exit 1
fi
exit 0
