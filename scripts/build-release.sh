#!/usr/bin/env bash
# USMP 一键式编译打包脚本
#
# 用法：./scripts/build-release.sh
#   产出 release/usmp-release-<git短sha>.zip，内含：
#     usmp/bin/usmp-backend   后端控制器（静态编译，:8080）
#     usmp/bin/usmp-web       前端静态站服务（自带 SPA fallback，免 nginx）
#     usmp/web/               前端构建产物（vite dist）
#     usmp/start.sh           一键启动脚本（解压后 sh usmp/start.sh 即启动全部服务）
#     usmp/VERSION            版本信息（git sha / 构建时间 / 目标平台）
#
# 目标平台缺省 linux/amd64（docker 镜像内使用），可用 USMP_GOOS/USMP_GOARCH 覆盖。
# 打包依赖 zip 或 python3 之一（python3 路径会保留可执行位）。
#
# Go 工具链：交付要求钉死 1.22，缺省 GOTOOLCHAIN=go1.22.12（本机没有会自动下载）。
# 离线且已装 1.22.x 的构建机可用 USMP_GOTOOLCHAIN=local 跳过下载。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GOOS="${USMP_GOOS:-linux}"
GOARCH="${USMP_GOARCH:-amd64}"
export GOTOOLCHAIN="${USMP_GOTOOLCHAIN:-go1.22.12}"
OUT="${USMP_RELEASE_OUT:-$ROOT/release}"
STAGE="$OUT/stage"
PKG="$STAGE/usmp"
VERSION="$(cd "$ROOT" && git rev-parse --short HEAD 2>/dev/null || echo dev)"
ZIP_PATH="$OUT/usmp-release-${VERSION}.zip"

fail() { echo "❌ $*" >&2; exit 1; }
step() { echo; echo "==> $*"; }

command -v go >/dev/null 2>&1 || fail "缺少 go 工具链"
command -v npm >/dev/null 2>&1 || fail "缺少 npm（前端构建需要 Node >= 20.12）"
command -v zip >/dev/null 2>&1 || command -v python3 >/dev/null 2>&1 || fail "打包需要 zip 或 python3 之一"

rm -rf "$STAGE"
mkdir -p "$PKG/bin" "$PKG/web"

# ──────────────────────────────────────────────
step "1/5 编译后端（GOOS=$GOOS GOARCH=$GOARCH，静态编译）"
# ──────────────────────────────────────────────
(
    cd "$ROOT/backend"
    # schema.ir.gz 为构建期产物不入库（R18）：go:embed 编译前置
    (cd tools && go run ./schemagen -repo_root=../.. -output=../internal/yangschema/schema.ir.gz)
    CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
        go build -trimpath -ldflags='-s -w' -o "$PKG/bin/usmp-backend" .
    CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
        go build -trimpath -ldflags='-s -w' -o "$PKG/bin/usmp-web" ./cmd/staticserver
)

# ──────────────────────────────────────────────
step "2/5 校验后端产物"
# ──────────────────────────────────────────────
for bin in usmp-backend usmp-web; do
    [ -s "$PKG/bin/$bin" ] || fail "$bin 编译产物缺失或为空"
    # go version -m 读二进制内嵌构建信息，确认目标平台无误
    go version -m "$PKG/bin/$bin" | grep -q "GOOS=$GOOS" || fail "$bin 目标平台不是 $GOOS"
    go version -m "$PKG/bin/$bin" | grep -q "GOARCH=$GOARCH" || fail "$bin 目标架构不是 $GOARCH"
    echo "  ✓ $bin ($(du -h "$PKG/bin/$bin" | cut -f1)) $GOOS/$GOARCH"
done

# ──────────────────────────────────────────────
step "3/5 编译前端（vite build）"
# ──────────────────────────────────────────────
(
    cd "$ROOT/frontend"
    [ -d node_modules ] || npm ci --prefer-offline --no-audit --fund=false
    # 组 8 口径：交付构建=EviewUI 真身，需 node_modules/@nce/* 在场（真包不出
    # 内网、不在任何 npm 源——npm ci 装不到，须内网环境预置）。缺包快速失败。
    [ -d node_modules/@nce/eview-react ] || fail "交付构建需 @nce/eview-react 真包（内网预置，npm 装不到）"
    npm run build
)

# ──────────────────────────────────────────────
step "4/5 校验前端产物并组装发布目录"
# ──────────────────────────────────────────────
[ -s "$ROOT/frontend/dist/index.html" ] || fail "前端产物缺失: dist/index.html"
[ -n "$(ls "$ROOT/frontend/dist/assets" 2>/dev/null)" ] || fail "前端产物缺失: dist/assets 为空"
cp -r "$ROOT/frontend/dist/." "$PKG/web/"
cp "$ROOT/scripts/release/start.sh" "$PKG/start.sh"
chmod +x "$PKG/start.sh" "$PKG/bin/"*
cat > "$PKG/VERSION" <<EOF
version=$VERSION
built_at=$(date '+%Y-%m-%d %H:%M:%S %z')
platform=$GOOS/$GOARCH
EOF
echo "  ✓ web/ ($(du -sh "$PKG/web" | cut -f1))  start.sh  VERSION"

# ──────────────────────────────────────────────
step "5/5 打包并校验 zip"
# ──────────────────────────────────────────────
rm -f "$ZIP_PATH"
if command -v zip >/dev/null 2>&1; then
    (cd "$STAGE" && zip -qr "$ZIP_PATH" usmp)
else
    # python3 兜底：ZipInfo.from_file 保留 unix 可执行位
    python3 - "$ZIP_PATH" "$STAGE" <<'PY'
import os, sys, zipfile
zip_path, stage = sys.argv[1], sys.argv[2]
with zipfile.ZipFile(zip_path, "w", zipfile.ZIP_DEFLATED) as zf:
    for base, _, files in os.walk(os.path.join(stage, "usmp")):
        for name in sorted(files):
            full = os.path.join(base, name)
            info = zipfile.ZipInfo.from_file(full, os.path.relpath(full, stage))
            info.compress_type = zipfile.ZIP_DEFLATED
            with open(full, "rb") as fh:
                zf.writestr(info, fh.read())
PY
fi

# zip 完整性 + 必备条目双重校验
if command -v python3 >/dev/null 2>&1; then
    python3 - "$ZIP_PATH" <<'PY'
import sys, zipfile
with zipfile.ZipFile(sys.argv[1]) as zf:
    bad = zf.testzip()
    if bad is not None:
        sys.exit("zip 损坏: %s" % bad)
    names = set(zf.namelist())
    required = ["usmp/start.sh", "usmp/bin/usmp-backend", "usmp/bin/usmp-web",
                "usmp/web/index.html", "usmp/VERSION"]
    missing = [r for r in required if r not in names]
    if missing:
        sys.exit("zip 缺少必备条目: %s" % ", ".join(missing))
PY
else
    unzip -tq "$ZIP_PATH" >/dev/null || fail "zip 完整性校验失败"
    for entry in usmp/start.sh usmp/bin/usmp-backend usmp/bin/usmp-web usmp/web/index.html usmp/VERSION; do
        unzip -l "$ZIP_PATH" | grep -q " $entry\$" || fail "zip 缺少必备条目: $entry"
    done
fi

echo
echo "✅ 发布包已生成并通过校验: $ZIP_PATH ($(du -h "$ZIP_PATH" | cut -f1))"
echo "   使用方式: unzip usmp-release-${VERSION}.zip && sh usmp/start.sh"
echo "   （前端 http://localhost:3002  后端 http://localhost:8080/api/v1；"
echo "    环境变量: USMP_SEED_DEVICE 注入种子设备、USMP_WEB_PORT 改前端端口）"
