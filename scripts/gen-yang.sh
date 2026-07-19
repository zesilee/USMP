#!/usr/bin/env sh
# gen-yang.sh — ygot YANG→Go 生成管线（厂商 manifest 驱动，CG-01）
#
# 扫描 backend/internal/generated/*/gen.conf，对每个厂商包执行：
#   ygot generator（版本由 backend/go.mod 锁定）→ genfix 后处理（跨平台，CG-02）
#   → 格式化收尾：单文件模式 gofmt；拆分模式 go tool goimports（ygot -output_dir
#     给每个文件写同一份 import 块，须剪未用 import 才能编译，版本同由 go.mod 锁定）
# 输出双模式（gen.conf 可选键 split_count 控制）：
#   未设置 → 单文件 all.gen.go（向后兼容，小厂商包零迁移）
#   split_count=N → -output_dir 拆分为 structs-*.go/enum*.go/union.go/schema.go，
#                   structs 按 N 分桶（blob 独占 schema.go，struct/blob diff 分离）
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

    if [ -n "$split_count" ]; then
        case "$split_count" in
        *[!0-9]* | 0*)
            echo "gen-yang: $conf 的 split_count 须为正整数: $split_count" >&2
            exit 1
            ;;
        esac
        echo "gen-yang: 生成 $pkg（modules: $modules，split_count=$split_count）"
        # 拆分模式：生成前清理新旧两种布局的产物（幂等，防 N 缩小残留旧分片），
        # 仅删生成物文件名，不动 doc.go/gen.conf。
        # $modules 依赖空格分词展开为多个模块参数，勿加引号
        (
            cd "$ROOT/backend" &&
                rm -f "internal/generated/$pkg/all.gen.go" \
                    "internal/generated/$pkg"/structs-*.go \
                    "internal/generated/$pkg"/enum*.go \
                    "internal/generated/$pkg/union.go" \
                    "internal/generated/$pkg/schema.go" &&
                go run github.com/openconfig/ygot/generator \
                    -path="$(echo "$yang_path" | awk -F, '{ for (i=1;i<=NF;i++) printf "%s../%s", (i>1?",":""), $i }')" \
                    -output_dir="internal/generated/$pkg" \
                    -structs_split_files_count="$split_count" \
                    -package_name="$pkg" \
                    -generate_fakeroot="$generate_fakeroot" \
                    -compress_paths="$compress_paths" \
                    -ignore_unsupported=true \
                    $modules &&
                go run ./tools/genfix "internal/generated/$pkg"/*.go &&
                go tool goimports -w "internal/generated/$pkg"
        )
    else
        echo "gen-yang: 生成 $pkg（modules: $modules）"
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
echo "✅ gen-yang 完成（生成物勿手改，改 YANG/gen.conf 后重跑 make gen-yang）"
