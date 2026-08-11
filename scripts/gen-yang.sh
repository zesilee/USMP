#!/usr/bin/env sh
# gen-yang.sh — YANG→Go 生成管线（厂商 manifest 驱动，CG-01 修订版）
#
# 扫描 backend/internal/generated/*/gen.conf：
#   常规包（huawei/business）→ 自研 yanggen 生成 internal/generated/native/<pkg>
#     （结构约定冻结自 ygot，确定性内建，零 genfix/goimports 后处理）
#   businessdemo（北向 demo 隔离锚点）→ 保留 ygot generator 单文件路径
#     （随 demo 生命周期退役，见 retire-ygot-runtime 任务6.2 拍板）
# 末尾联动重生成 schema IR blob（schemagen 直读 YANG 源）。
# package 名 = 目录名。新增厂商 = 新增目录 + gen.conf，本脚本与 Makefile 零改动。
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
    split_count=""
    while IFS='=' read -r key val; do
        case "$key" in
        yang_path) yang_path="$val" ;;
        modules) modules="$val" ;;
        generate_fakeroot) generate_fakeroot="$val" ;;
        compress_paths) compress_paths="$val" ;;
        split_count) split_count="$val" ;;
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

    # 前置校验：YANG 模型目录（逗号分隔多目录）必须存在且非空（模型源为入库目录）
    for dir in $(echo "$yang_path" | tr ',' ' '); do
        if [ ! -d "$ROOT/$dir" ] || [ -z "$(ls -A "$ROOT/$dir" 2>/dev/null)" ]; then
            echo "gen-yang: YANG 模型目录不存在或为空: $dir" >&2
            echo "  模型源为入库目录（如 snd/ce6866p-yang），请检查 checkout 完整性" >&2
            exit 1
        fi
    done

    if [ "$pkg" = "businessdemo" ]; then
        # businessdemo：北向 demo 隔离锚点，保留 ygot 单文件路径（任务6.2 拍板：
        # 随 demo 生命周期退役，不迁 native）。
        echo "gen-yang: 生成 $pkg（ygot demo 路径，modules: $modules）"
        # $modules 依赖空格分词展开为多个模块参数，勿加引号
        (
            cd "$ROOT/backend" &&
                go run github.com/openconfig/ygot/generator \
                    -path="$(echo "$yang_path" | awk -F, '{ for (i=1;i<=NF;i++) printf "%s../%s", (i>1?",":""), $i }')" \
                    -output_file="internal/generated/$pkg/all.gen.go" \
                    -package_name="$pkg" \
                    -generate_fakeroot="$generate_fakeroot" \
                    -compress_paths="$compress_paths" \
                    $modules &&
                go run ./tools/genfix "internal/generated/$pkg/all.gen.go" &&
                gofmt -w "internal/generated/$pkg/all.gen.go"
        )
    else
        native_split=""
        if [ -n "$split_count" ]; then
            native_split="-structs_split_files_count=$split_count"
        fi
        echo "gen-yang: 生成 native/$pkg（yanggen）"
        (
            cd "$ROOT/backend" &&
                rm -f "internal/generated/native/$pkg/all.gen.go" \
                    "internal/generated/native/$pkg"/structs-*.go \
                    "internal/generated/native/$pkg"/enum*.go \
                    "internal/generated/native/$pkg/union.go" \
                    "internal/generated/native/$pkg/registry.go" &&
                go run ./tools/yanggen \
                    -path="$(echo "$yang_path" | awk -F, '{ for (i=1;i<=NF;i++) printf "%s../%s", (i>1?",":""), $i }')" \
                    -output_dir="internal/generated/native/$pkg" \
                    -package_name="$pkg" \
                    $native_split \
                    $modules
        )
    fi
done

if [ "$found" = 0 ]; then
    if [ -n "$ONLY" ]; then
        echo "gen-yang: 未找到厂商包 '$ONLY' 的 gen.conf（backend/internal/generated/$ONLY/gen.conf）" >&2
    else
        echo "gen-yang: 未找到任何 gen.conf" >&2
    fi
    exit 1
fi
# schema IR blob 随生成物联动刷新（YN-03；ir_parity_test 兜底「忘刷新」）。
(cd "$ROOT/backend" && go run ./tools/schemagen -repo_root=.. -output=internal/yangschema/schema.ir.gz)

echo "✅ gen-yang 完成（生成物勿手改，改 YANG/gen.conf 后重跑 make gen-yang）"
