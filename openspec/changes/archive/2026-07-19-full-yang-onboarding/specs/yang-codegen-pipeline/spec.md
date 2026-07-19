# yang-codegen-pipeline — delta（full-yang-onboarding）

## ADDED Requirements

### Requirement: CG-04 本地 deviation 豁免机制

生成闭包 SHALL 支持纳入仓库本地 deviation 模块（`backend/internal/yang/deviations/`）：
`gen.conf` 的 `yang_path` SHALL 支持逗号分隔多目录，deviation 模块与厂商模型同闭包
生成。deviation SHALL 仅用于豁免 ygot 生成器不支持的个别节点（如 bits 类型默认值、
anydata、binary key list、穿 choice/case augment 的 leafref），每条 SHALL 注明豁免
原因与影响面；SHALL NOT 修改 snd 子模块内的模型本体（只读源）。生成器 SHALL 开启
`-ignore_unsupported`（解析期跳过不支持语句）。无法经 deviation 豁免的模块（解析期
致命错误，如跨模块 submodule typedef 引用）SHALL 显式记录为延期项而非静默缺失。

#### Scenario: deviation 豁免后模块可生成

- **WHEN** huawei-syslog 的 bits 叶默认值经 deviation `delete default` 豁免后执行生成
- **THEN** 生成 SHALL 成功，syslog 根容器进入闭包；被豁免叶仍存在（仅失去默认值）

#### Scenario: not-supported 剔除非配置面节点

- **WHEN** cfg 的 anydata 节点 / qos 的 binary-key 查询列表经 `deviate not-supported` 豁免
- **THEN** 生成 SHALL 成功且该节点不出现在生成物中，模块其余配置面不受影响

#### Scenario: 延期项显式记录（负路径）

- **WHEN** 某模块存在 deviation 无法豁免的解析期错误（如 huawei-pic 的
  `devm:switch-status-type` 跨模块 submodule typedef 引用）
- **THEN** 该模块 SHALL 不入 `modules` 清单并在 gen.conf 注释中记录原因，左树对应叶
  保持 `available:false` 占位

## MODIFIED Requirements

### Requirement: CG-01 厂商 manifest 驱动的可复现生成

系统 SHALL 提供 `make gen-yang` 生成入口：扫描 `backend/internal/generated/*/gen.conf`（每厂商包一份声明式生成配置：YANG 模型路径、模块列表、fakeroot/compress 选项，及**可选 `split_count`**），对每包执行 ygot generator（版本由 go.mod 锁定）→ 跨平台后处理 → 格式化收尾（单文件模式 gofmt；拆分模式 goimports——`-output_dir` 给每个文件写同一份 import 块，须剪除未用 import 方可编译，goimports 版本同由 go.mod `tool` 指令锁定），输出该包生成物。华为包的 `yang_path` SHALL 以入库目录 `snd/ce6866p-yang` 为首目录（SP-01，无 submodule 依赖），并 MAY 以逗号追加仓库本地 deviation 目录（CG-04）。生成物布局由 `split_count` 决定：**未设置**时输出单文件 `all.gen.go`；**设置为 N** 时输出 `-output_dir` 拆分文件集（`structs-0..(N-1).go` + `enum.go`/`enum_map.go`/`union.go`/`schema.go`），使**单文件规模可控**（避免单包生成物随模型集成无限膨胀）。文件命名 SHALL 由 generator 确定性给定。`make gen-yang VENDOR=<pkg>` SHALL 仅重生成指定包。新增厂商 SHALL 只需新增目录 + `gen.conf`，零脚本/Makefile 改动。管线 SHALL 可复现且机器无关：同一仓库状态下重复执行输出字节一致（拆分模式下每个生成文件的内容与 struct→文件分配均确定），生成物 SHALL NOT 包含生成机器特定内容（如生成器绝对路径头部注释——由后处理规范化）。拆分 SHALL 语义等价于单文件：同包类型集合、导出符号（`Schema()`/`UnzipSchema()`/`SchemaTree`/`Unmarshal` 等）、schema 内容不变，下游 import 路径与消费无改动。

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
