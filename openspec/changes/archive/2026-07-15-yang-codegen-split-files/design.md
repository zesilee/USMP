# Design — yang-codegen-split-files

## 1. 背景与目标

单文件 `all.gen.go`（59,414 行）随全量模型集成持续膨胀，造成 IDE/diff/编译的真实工程成本。目标：**在不改任何语义、不改下游、不改运行期的前提下**，把单包生成物按文件拆小到可控规模（≤~6.5k 行/文件）。

非目标：减少总代码量（模型驱动固有）、压缩/删除 gzip blob（承重墙）、改运行期行为。

## 2. 去风险实证结论（explore spike，权威依据）

| 验证项 | 命令/方法 | 结论 |
|--------|-----------|------|
| 拆分可行 | `-output_dir -structs_split_files_count=8` | 12 文件，最大 6.5k 行（原 1/9） |
| 语义等价 | genfix+gofmt 后 `type` 计数 | **1082 == 1082** 精确一致 |
| 无体积膨胀 | 总行数对比 | 59,733 vs 59,414（+0.5%，仅包头） |
| blob 隔离 | `grep -l ySchema` | 独占 `schema.go`（struct/blob 分文件） |
| **确定性** | 独立生成两次 `diff -rq` | **字节完全一致** ✓（CG-01/CG-03 前提） |
| genfix 复用 | 读 `genfix/main.go` 用法 | 已支持 `<file>...`，无 blob 文件 no-op |
| 门禁复用 | 读 pr-size/compliance | 均路径匹配，覆盖新文件名，零改动 |

> spike 目录（scratchpad，不入库）：`split-spike/`、`split-spike2/`。

## 3. 生成物文件布局（拆分后）

ygot `-output_dir` 模式产出固定命名的文件集（huawei，`split_count=8`）：

| 文件 | 内容 | ~行数 |
|------|------|-------|
| `structs-0.go` … `structs-7.go` | struct 定义 + `Validate()` 方法（含包级 `Schema()`/`UnzipSchema()`/`SchemaTree` 落 `structs-0.go`） | 4.6k–6.3k |
| `enum.go` | 枚举类型定义（genfix `\|`→`_OR_` 作用点） | ~6.5k |
| `enum_map.go` | 枚举值映射表 | ~2.5k |
| `union.go` | union 类型 | ~0.6k |
| `schema.go` | `var ySchema`（gzip blob，genfix 确定性重排作用点） | ~6.5k |

文件命名由 ygot 固定，跨运行确定。`structs_split_files_count=N` 只控制 `structs-*.go` 的份数；`enum/union/schema` 恒各一份。

## 4. 关键设计决策

### D1. gen.conf 加 `split_count`，脚本双模式（向后兼容）

- `gen.conf` 新增可选键 `split_count=N`。
- `gen-yang.sh` 分支：
  - `split_count` 设置 → `-output_dir="internal/generated/$pkg" -structs_split_files_count=$split_count`（无 `-output_file`）。
  - 未设置 → 保持现状 `-output_file="internal/generated/$pkg/all.gen.go"`。
- **huawei**：`split_count=8`。**openconfig**：不设（907 行，拆分零收益、避免无谓 churn）。
- 理由：单一 target 覆盖两种厂商规模；小厂商零迁移；大厂商可调 N。

### D2. 后处理改为「输出目录 glob」

现状脚本对固定 `all.gen.go` 跑 genfix+gofmt。改为：拆分模式下 `for f in internal/generated/$pkg/*.go` 逐一 `go run ./tools/genfix "$f"` 再 `gofmt -w`。genfix 对无 `ySchema` 文件走 `fixSchemaBlob` 的 `m == nil → 原样返回`；对无 `|` 文件 enum 修复 no-op——**逐文件安全**，无需感知哪个文件含什么。

### D3. 旧产物清理（防陈留）

切换到拆分模式时，旧 `all.gen.go` 必须删除，否则与新 `structs-*.go` 并存导致重复声明编译失败。脚本在拆分分支生成前：`rm -f internal/generated/$pkg/all.gen.go` + 清理上一轮可能残留的 `structs-*.go`/`enum*.go`/`union.go`/`schema.go`（幂等，避免 N 缩小时残留旧 `structs-8.go`）。首次切换的 PR 用 `git rm all.gen.go` 记录删除。

### D4. genfix 保持多文件契约 + 补测

genfix 逻辑本身已多文件就绪（`for _, f := range os.Args[1:]`）。本 change 只**补齐测试**证明文件集下的正确性（见测试矩阵），不改核心算法。若发现 blob 正则 `ySchemaBlock` 在独立 `schema.go`（无 struct 上下文）中仍能定位——spike 已间接验证（genfix 逐文件跑后 gofmt 通过、二次生成零漂移），apply 期加显式单测锁定。

## 5. 下游影响面（零改动论证）

拆分不改 package 名（`huawei`）、不改导出符号、不改 import 路径。审计确认的消费点全部无感：

- `internal/drivers/huawei.go` → `huawei.SchemaTree[...]`、`huawei.Unmarshal`
- `internal/yangschema/load.go` → `huawei.Schema()`
- `internal/api/config_codec.go`、`xmlcodec/*`、`actor/model_actor.go` → `ygot.GoStruct` 及生成 struct 类型

符号定义位置在文件间移动，但同包内对 Go 编译/import 透明。

## 6. 测试矩阵（T05/T06，先行）

改动类型 = **后端纯逻辑（构建期工具）+ 生成物重排**，非 Reconciler/协议/新 YANG 模型，故**不触发** B2 集成/`yang-config-test-design` 矩阵；核心防线是 **regen-and-diff 等价性 + 编译 + 既有全量测试不回归**。

| 层 | 用例 | 断言 |
|----|------|------|
| B1 genfix | 无 blob 文件输入 | 内容不变（no-op） |
| B1 genfix | 仅 enum `\|` 文件 | 仅标识符 `\|`→`_OR_`，YANG 原值串不变 |
| B1 genfix | 仅 `ySchema` 文件 | blob 确定性重排、语义等价（`GzipToSchema` 前后结构一致） |
| B1 genfix | 文件集幂等 | 二次执行零变更 |
| 管线 | huawei 拆分重生成 ×2 | `diff -rq` 字节一致（CG-01 确定性） |
| 等价 | 拆分后包 | `go build ./...` 通过；`huawei.Schema()` round-trip 成功 |
| 等价 | 全量单测 + xmlcodec golden | 全绿（证 struct/Validate 语义未变） |
| 门禁 | CI 环境 regen | `git diff --exit-code backend/internal/generated/` 空 |
| 门禁 | pr-size | `generated/**` 排除生效，本 PR 逻辑行 ≤ 阈值 |

## 7. 备选方案（评估后不选/推迟）

| 方案 | 判断 |
|------|------|
| 维持单文件 | ✗ 膨胀不可逆，IDE/diff/编译成本随模型全量集成放大 |
| `-include_schema=false`（删 blob） | ✗ 运行期承重墙，删则约束引擎/渲染/xmlcodec 全失源 |
| blob 外置 `//go:embed schema.gz` | ⏸ 推迟——省 13%、边际收益，需改 `UnzipSchema()` 读取源，独立 change 评估 |
| 手工拆分维护多文件 | ✗ 违反 R04（禁手改 generated/），且历史已证手拆布局跨平台不可复现（snd-ygot-pipeline 教训） |
| 按模块语义拆包 | ✗ ygot `structs_split_files_count` 按计数分桶、非按模块；且拆包会改 import 路径，破坏下游零改动 |

## 8. 排期与并发约束

- **TM03 硬约束**：本 change 重生成 `internal/generated/huawei/` 整包，与在跑的 `huawei-network-instance-config` 改同一 package。**禁止并行**——待 network-instance 合入 main 后，以其扩充模块后的 `all.gen.go` 为基线启动本 change apply（届时拆分对象是含 network-instance/BGP 全闭包的更大单文件，拆分收益更高）。
- 越早拆越省：每晚一个模型波次，切换 PR 的生成物 churn 越大、迁移面越广——本 change 应排在 network-instance 之后、下一个大模型波次（BGP 2a peering）之前。
