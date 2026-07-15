# yang-codegen-split-files — 生成物按文件拆分（消除单文件巨物）

## Why

ygot 生成物 `backend/internal/generated/huawei/all.gen.go` 已达 **59,414 行**单文件（1,082 struct + 1,637 Validate 方法 + 内嵌 gzip schema blob），且随全量 YANG 模型集成（BGP peering 2a/2b、routing、ACL、route-policy…）**必然继续膨胀**——单文件是模型驱动路线的固有产物，不是可压缩的冗余。

单文件巨物的**真实成本**（区别于伪成本）：

- **真成本**：IDE 打开/索引 59k 行文件卡顿；`git diff` 定位困难（struct 改动与 schema blob 漂移混在一个文件）；单 Go 文件无法并行编译。
- **伪成本**（无需处理）：代码审查负担——生成物不逐行看，走 R04 regen-and-diff（CG-03），genfix 已保证确定性。

**关键澄清（纠正常见误解）**：文件大的主因是 **struct+Validate 代码（87%）**，不是 gzip blob（仅 13%、7,663 行）。blob 是**承重墙不可删**——部署容器不带 `.yang` 源文件（yang-models 是仅构建期 submodule），运行期 `internal/drivers/huawei.go`、`pkg/yang-runtime/schema/entry.go`（消费 `Entry.Extra` 的 when/must/presence 与 `Entry.Exts` 的厂商扩展）**唯一** schema 元数据源就是该 blob。故本 change **不动 blob 语义**，只做「同包内按文件拆分」的物理重排。

**去风险实证（explore spike，2026-07-14，scratchpad 隔离目录）**——用 go.mod 锁定的 ygot v0.34.0 实跑 `-output_dir -structs_split_files_count=8`：

1. **拆为 12 文件**：`structs-0..7.go`（各 ~4.6k–6.3k 行）、`enum.go`(6521)、`enum_map.go`(2530)、`union.go`(600)、`schema.go`(6522)。**每文件 ≤6.5k 行**，最大文件缩到原 1/9。
2. **零语义变更**：genfix+gofmt 后总行数 59,733 ≈ 单文件 59,414（+0.5%，仅 12×包头开销）；`type` 数 **1082==1082 精确一致**。
3. **blob 自动隔离进 `schema.go`**——struct 改动与 blob 漂移从此在**不同文件**，diff 更干净（意外收益）。
4. **字节级确定性成立**：同参数独立生成两次，genfix+gofmt 后 `diff -rq` **完全一致**——CG-01/CG-03 regen-and-diff 门禁前提满足。
5. **genfix 已支持多文件入参**（`<file> [<file>...]`），无 blob 文件原样返回；enum `|` 修复落 `enum.go`、blob 确定性重排落 `schema.go`——**genfix 逻辑几乎不用改**，只需脚本把文件集全传入。
6. **CI/本地门禁零改动**：pr-size（`:(exclude)backend/internal/generated/**`）与 CG-03 R04 regen（`git diff --exit-code backend/internal/generated/`）均**路径匹配**，天然覆盖拆分后的新文件名。

结论：本 change 是**低风险、纯构建期物理重排**，无运行时行为变更，去风险已实证。

## What Changes

- **`gen.conf` 新增可选 `split_count=N`**：声明该厂商包生成物拆分文件数。设置时管线走 `-output_dir` + `-structs_split_files_count=N`（多文件）；未设置时保持现状 `-output_file`（单 `all.gen.go`，向后兼容）。**huawei 设 `split_count=8`**（每文件回落 ≤6.5k 行）；**openconfig 不设**（907 行无需拆，零 churn）。
- **`scripts/gen-yang.sh` 双模式**：读 `split_count` → 选 `-output_dir`（拆分）或 `-output_file`（单文件）分支；生成后对**输出目录下全部 `*.go`** 逐一 `genfix` + `gofmt`（glob，非硬编码单文件名）；拆分模式下先清理旧产物（`git rm` `all.gen.go` + 旧 `structs-*.go` 幂等清理，防陈留）。
- **重生成 huawei 包**：`all.gen.go`（1 文件）→ `structs-0..7.go`/`enum.go`/`enum_map.go`/`union.go`/`schema.go`（12 文件）。纯生成物 churn，语义等价（spike 实证）。
- **genfix 多文件健壮化 + 测试**：确认/补齐对「无 blob 文件 no-op」「enum 文件仅修 `|`」「schema 文件仅重排 blob」的表格驱动单测（B1）；genfix 幂等性对文件集成立。
- **明确不做**：不动 blob 语义、不改 `-include_schema`、不改任何运行期代码、不改 CI 门禁 YAML（路径匹配已覆盖）、不拆 openconfig。
- **明确排除（可选后续）**：blob 外置 `//go:embed schema.gz`（可再省 13%）——收益边际、需改 `UnzipSchema()` 读取源，本 change 不做，另 change 评估。

## Capabilities

### Modified Capabilities

- `yang-codegen-pipeline`：CG-01 生成入口从「输出唯一 `all.gen.go`」扩展为「按 `gen.conf` 的 `split_count` 输出单文件或拆分文件集」，并明确「每文件规模可控 + 拆分确定性」契约；CG-02 后处理器明确作用于「生成文件集」（blob 定位到含 `ySchema` 的文件、enum 修复作用于枚举文件）。CG-03 门禁（路径匹配）不变。

## Impact

- **构建工具链**：`backend/internal/generated/huawei/gen.conf`（+`split_count=8`）、`scripts/gen-yang.sh`（双模式 + glob 后处理 + 旧产物清理）、`backend/tools/genfix/*`（多文件健壮化 + 单测，逻辑改动极小）。
- **生成物**：`backend/internal/generated/huawei/` 1 文件 → 12 文件（纯 churn，语义等价、`type` 数一致、blob 字节一致；pr-size 已排除 `generated/**` 故不计体积）。
- **下游消费方**：**零改动**——同 package `huawei`、同 import 路径；`huawei.Schema()`/`huawei.SchemaTree`/`huawei.Unmarshal`/`ygot.GoStruct` 全部不变（`Schema()`/`UnzipSchema()` 落 `structs-0.go`，仍为包级导出）。
- **CI/本地门禁**：`compliance.yml` CG-03、`pr-size.yml`、pre-commit/pre-push 钩子**零改动**（路径匹配覆盖新文件名）——需在 apply 期实测确认（regen 二次生成零漂移 + pr-size 排除生效）。
- **版本**：`8.20.10/ne40e-x8x16`（gen.conf 现行目标）。
- **风险（低）**：① 拆分 struct→文件分配的跨机器确定性（spike 同机两次已一致，apply 期需在 CI 环境复验 CG-01 Scenario）；② `Validate` 方法 grep 计数 spike 中 1636 vs 单文件 1637 off-by-one——疑为 grep 命中差异（`type` 数精确一致佐证无真实语义差），apply 期需以「编译通过 + `Schema()` round-trip + 全量单测/xmlcodec golden 全绿」证实等价；③ genfix 对多文件的幂等边界。均由既有防线（regen-and-diff、golden 对拍、全量单测、B2 集成）兜底。
- **合规**：R04（禁手改 generated/、regen-and-diff）、R10、R17（本 proposal + delta 先于开发）、T05/T06（genfix 多文件 B1 测试先行）、worktree 隔离、≤500 行/commit（生成物 churn 不计逻辑行）。
- **排期约束（TM03）**：本 change 重生成 `internal/generated/huawei/` 整包，与在跑的 `huawei-network-instance-config`（同样重生成该包 + 改 `huawei.go`）**改同一 Go package，禁止并行 worktree**。**必须待 network-instance change 交付合入 main 后再启动本 change 的 apply**，届时以其新增模块后的生成物为基线重跑拆分。
