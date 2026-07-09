#!/usr/bin/env sh
# gen-yang.sh — ygot YANG→Go 生成管线（厂商 manifest 驱动，CG-01）
#
# 扫描 backend/internal/generated/*/gen.conf，对每个厂商包执行：
#   ygot generator（版本由 backend/go.mod 锁定）→ genfix 后处理（跨平台，CG-02）→ gofmt
# 输出固定为该包目录下的 all.gen.go，package 名 = 目录名。
# 新增厂商 = 新增目录 + gen.conf，本脚本与 Makefile 零改动。
#
# 用法: scripts/gen-yang.sh [<pkg>]   缺省全量；<pkg> 为 backend/internal/generated/ 下目录名
set -eu

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GEN_DIR="$ROOT/backend/internal/generated"
ONLY="${1:-}"

found=0
for conf in "$GEN_DIR"/*/gen.conf; do
    [ -f "$conf" ] || continue
    pkg="$(basename "$(dirname "$conf")")"
    if [ -n "$ONLY" ] && [ "$pkg" != "$ONLY" ]; then
        continue
    fi
    found=1

    yang_path=""
    modules=""
    generate_fakeroot=true
    compress_paths=false
    while IFS='=' read -r key val; do
        case "$key" in
        yang_path) yang_path="$val" ;;
        modules) modules="$val" ;;
        generate_fakeroot) generate_fakeroot="$val" ;;
        compress_paths) compress_paths="$val" ;;
        '' | \#*) ;;
        *)
            echo "gen-yang: $conf 含未知键: $key" >&2
            exit 1
            ;;
        esac
    done <"$conf"

    if [ -z "$yang_path" ] || [ -z "$modules" ]; then
        echo "gen-yang: $conf 缺少 yang_path 或 modules" >&2
        exit 1
    fi

    # 前置校验：YANG 模型目录必须存在且非空（yang-models 是仅构建期 submodule）
    if [ ! -d "$ROOT/$yang_path" ] || [ -z "$(ls -A "$ROOT/$yang_path" 2>/dev/null)" ]; then
        echo "gen-yang: YANG 模型目录不存在或为空: $yang_path" >&2
        echo "  若为 yang-models submodule，请先执行: git submodule update --init yang-models" >&2
        exit 1
    fi

    echo "gen-yang: 生成 $pkg（modules: $modules）"
    # $modules 依赖空格分词展开为多个模块参数，勿加引号
    (
        cd "$ROOT/backend" &&
            go run github.com/openconfig/ygot/generator \
                -path="../$yang_path" \
                -output_file="internal/generated/$pkg/all.gen.go" \
                -package_name="$pkg" \
                -generate_fakeroot="$generate_fakeroot" \
                -compress_paths="$compress_paths" \
                $modules &&
            go run ./tools/genfix "internal/generated/$pkg/all.gen.go" &&
            gofmt -w "internal/generated/$pkg/all.gen.go"
    )
done

if [ "$found" = 0 ]; then
    if [ -n "$ONLY" ]; then
        echo "gen-yang: 未找到厂商包 '$ONLY' 的 gen.conf（backend/internal/generated/$ONLY/gen.conf）" >&2
    else
        echo "gen-yang: 未找到任何 gen.conf" >&2
    fi
    exit 1
fi
echo "✅ gen-yang 完成（生成物勿手改，改 YANG/gen.conf 后重跑 make gen-yang）"
