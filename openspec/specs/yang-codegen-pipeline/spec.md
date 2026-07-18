# yang-codegen-pipeline — ygot 生成管线

## Purpose

ygot YANG→Go 生成管线（R04 的可执行形态）：厂商 manifest（`backend/internal/generated/*/gen.conf`）驱动的可复现生成入口 `make gen-yang` + 跨平台后处理（`backend/tools/genfix`）+ 生成物漂移 CI/本地门禁（regen-and-diff，取代无条件冻结）。新增厂商 = 新增目录 + gen.conf，是异构多设备 SND（P5）加厂商路径的构建期一环（运行期对应 device-driver-registry）。

## Requirements

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

### Requirement: CG-02 跨平台后处理（枚举标识符 + 确定性 schema 规范化）

生成管线 SHALL 使用 Go 实现的后处理器（`backend/tools/genfix`）对 ygot 生成物做两类幂等、机器无关的后处理，以满足 CG-01「同一仓库状态下重复执行输出字节一致」。后处理器 SHALL 接受**一个或多个**文件路径，对生成文件集逐一处理——单文件模式作用于 `all.gen.go`，拆分模式作用于目录下全部生成 `*.go`；对不含相应构造的文件 SHALL 为 no-op（无 `ySchema` blob 的文件不做 schema 规范化、无非法枚举标识符的文件不做枚举修复）：

1. **枚举标识符修复**：修复含非法字符 `|` 的枚举标识符（如 `PortType_50|100GE` → `PortType_50_OR_100GE`），SHALL 在 Linux 与 macOS 上行为一致（不依赖平台 sed 方言）；SHALL NOT 改动枚举标识符之外的内容（含 YANG 原值字符串映射）。拆分模式下该修复作用于承载枚举定义的文件（`enum.go`）。
2. **确定性 schema 规范化**：ygot 内嵌的 gzip schema blob（`var ySchema = []byte{…}`）序列化了 `yang.Entry` 的**无序集合数组**（首要为各节点的 `Augmented`——同一目标被多模块 augment 时，goyang 以非确定 map 迭代序应用，致数组元素顺序逐次不同、gzip 字节漂移）。后处理器 SHALL 在承载该 blob 的文件（拆分模式下为 `schema.go`）中解压 blob、以稳定规则对**语义无序**的数组（`Augmented` 等，按元素规范化内容排序）与对象键做确定性重排、以固定参数（无时间戳、固定压缩级别）重新压缩并回填，使 blob 字节在重复生成间稳定。规范化 SHALL 保持 schema 语义等价：`ygot.GzipToSchema` 解出的 schema 与规范化前**结构与内容一致**（键集合、节点、类型、约束不变），SHALL NOT 重排语义有序的构造，SHALL NOT 改动数字/字符串字面量的值。

两类后处理 SHALL 幂等：对已处理或无匹配的输入（含文件集中任一文件）执行为 no-op。

#### Scenario: 非法字符修复
- **WHEN** 生成物包含标识符 `HuaweiIfm_PortType_50|100GE`
- **THEN** 后处理后 SHALL 为 `HuaweiIfm_PortType_50_OR_100GE`，且对应 YANG 原值映射字符串（如 `"50|100GE"`）SHALL 保持原样

#### Scenario: 文件集中无 blob 文件的 no-op
- **WHEN** 后处理器处理拆分文件集中不含 `var ySchema` 的文件（如 `structs-3.go`/`union.go`）
- **THEN** 该文件的 schema 规范化步骤 SHALL 为 no-op（内容不变）

#### Scenario: 多-augment 闭包 schema 确定性
- **WHEN** 生成集含多模块 augment 同一目标节点的闭包（如 huawei-bgp 拉入 network-instance/ifm 被 tunnel-management/ethernet/bfd 等多方 augment），连续两次 `make gen-yang`
- **THEN** 两次生成物（含承载 `ySchema` blob 的文件）SHALL 字节一致（满足 CG-01 / CG-03 regen-and-diff）

#### Scenario: schema 规范化语义等价
- **WHEN** 对规范化前后的 `ySchema` 分别 `ygot.GzipToSchema`
- **THEN** 两者解出的 schema SHALL 结构与内容一致（节点键集合、类型、约束不变），仅无序集合数组的元素顺序被规整为稳定序

#### Scenario: 幂等 no-op
- **WHEN** 对同一文件集执行后处理两次
- **THEN** 第二次执行 SHALL 不产生任何变更

### Requirement: CG-03 生成物漂移 CI 门禁（R04 可验证形态）

CI SHALL 以 regen-and-diff 验证生成物：当 PR 变更触及 `backend/internal/generated/**`、生成脚本/后处理器或 `snd/ce6866p-yang/**` 模型源时，SHALL 重跑 `make gen-yang` 并断言 `git diff --exit-code backend/internal/generated/` 为空——生成物改动合法当且仅当可由管线复现（取代无条件冻结 `generated/` 的旧检查）。未触及上述路径的 PR SHALL 跳过该验证。CI SHALL NOT 含任何 submodule 初始化步骤。本地 pre-commit 钩子 SHALL 以同口径对称拦截（T09）：暂存触及生成物/manifest（纯文档除外）时本地 regen + diff 校验。

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
