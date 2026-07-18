# yang-codegen-pipeline — delta（adopt-snd-baseline）

## MODIFIED Requirements

### Requirement: CG-01 厂商 manifest 驱动的可复现生成

系统 SHALL 提供 `make gen-yang` 生成入口：扫描 `backend/internal/generated/*/gen.conf`（每厂商包一份声明式生成配置：YANG 模型路径、模块列表、fakeroot/compress 选项，及**可选 `split_count`**），对每包执行 ygot generator（版本由 go.mod 锁定）→ 跨平台后处理 → 格式化收尾（单文件模式 gofmt；拆分模式 goimports——`-output_dir` 给每个文件写同一份 import 块，须剪除未用 import 方可编译，goimports 版本同由 go.mod `tool` 指令锁定），输出该包生成物。华为包的 `yang_path` SHALL 指向入库目录 `snd/ce6866p-yang`（SP-01，无 submodule 依赖）。生成物布局由 `split_count` 决定：**未设置**时输出单文件 `all.gen.go`；**设置为 N** 时输出 `-output_dir` 拆分文件集（`structs-0..(N-1).go` + `enum.go`/`enum_map.go`/`union.go`/`schema.go`），使**单文件规模可控**（避免单包生成物随模型集成无限膨胀）。文件命名 SHALL 由 generator 确定性给定。`make gen-yang VENDOR=<pkg>` SHALL 仅重生成指定包。新增厂商 SHALL 只需新增目录 + `gen.conf`，零脚本/Makefile 改动。管线 SHALL 可复现且机器无关：同一仓库状态下重复执行输出字节一致（拆分模式下每个生成文件的内容与 struct→文件分配均确定），生成物 SHALL NOT 包含生成机器特定内容（如生成器绝对路径头部注释——由后处理规范化）。拆分 SHALL 语义等价于单文件：同包类型集合、导出符号（`Schema()`/`UnzipSchema()`/`SchemaTree`/`Unmarshal` 等）、schema 内容不变，下游 import 路径与消费无改动。

#### Scenario: 全量重生成零漂移
- **WHEN** 在干净工作区执行 `make gen-yang`
- **THEN** `git diff backend/internal/generated/` SHALL 为空（生成物与仓库一致）

#### Scenario: 单厂商重生成
- **WHEN** 执行 `make gen-yang VENDOR=huawei`
- **THEN** SHALL 仅重生成 `backend/internal/generated/huawei/` 下该包生成物（`split_count` 设置时为拆分文件集，未设置时为 `all.gen.go`），其他包不动

#### Scenario: 拆分模式确定性与规模可控
- **WHEN** 某包 `gen.conf` 设 `split_count=N` 并连续两次 `make gen-yang VENDOR=<pkg>`
- **THEN** 两次输出的拆分文件集 SHALL 字节一致（含 struct→文件分配），且每个 `structs-*.go` 规模受 N 控制

#### Scenario: 拆分语义等价
- **WHEN** 将某包从单文件切换为 `split_count=N` 重生成
- **THEN** 拆分后包 SHALL `go build` 通过、`Schema()` round-trip 成功、类型集合与 `SchemaTree` 键集合与拆分前一致，下游消费方零改动

#### Scenario: 模型源目录缺失时可操作报错
- **WHEN** `gen.conf` 的 `yang_path` 目录不存在或为空时执行 `make gen-yang`
- **THEN** SHALL 以非零码退出并输出指明缺失目录的修复指引（入库目录应随仓库存在，请检查 checkout 完整性），SHALL NOT 产生半成品输出，SHALL NOT 提示任何 submodule 操作

### Requirement: CG-03 生成物漂移 CI 门禁（R04 可验证形态）

CI SHALL 以 regen-and-diff 验证生成物：当 PR 变更触及 `backend/internal/generated/**`、生成脚本/后处理器或 `snd/ce6866p-yang/**` 模型源时，SHALL 重跑 `make gen-yang` 并断言 `git diff --exit-code backend/internal/generated/` 为空——生成物改动合法当且仅当可由管线复现（取代无条件冻结 `generated/` 的旧检查）。未触及上述路径的 PR SHALL 跳过该验证。CI SHALL NOT 含任何 submodule 初始化步骤。本地 pre-commit 钩子 SHALL 以同口径对称拦截（T09）：暂存触及生成物/manifest/模型源（纯文档除外）时本地 regen + diff 校验。

#### Scenario: 手改生成物被拦截
- **WHEN** PR 直接手工编辑 `all.gen.go` 而未经管线生成
- **THEN** CI regen-and-diff SHALL fail

#### Scenario: 管线产物合法通过
- **WHEN** PR 通过修改 `gen.conf` 并执行 `make gen-yang` 提交生成物变更
- **THEN** CI regen-and-diff SHALL pass

#### Scenario: 模型源变更触发验证
- **WHEN** PR 变更 `snd/ce6866p-yang/**` 下任一模型文件
- **THEN** CI SHALL 重跑 regen-and-diff（模型源与生成物必须原子一致）

#### Scenario: 无关 PR 跳过
- **WHEN** PR 未触及生成物、生成脚本与 `snd/ce6866p-yang/**`
- **THEN** SHALL 跳过 regen 验证（不消耗生成耗时）
