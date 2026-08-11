# yang-codegen-pipeline — 自研 yanggen 生成管线

## Purpose

自研 YANG→Go 生成管线（R04 的可执行形态，retire-ygot-runtime 后 ygot generator 仅存
businessdemo demo 路径）：厂商 manifest（`backend/internal/generated/*/gen.conf`）驱动的
可复现生成入口 `make gen-yang`（`backend/tools/yanggen`，tools 为独立 go module）+ 生成物
漂移 CI/本地门禁（regen-and-diff，取代无条件冻结）+ schema IR blob 联动刷新。新增厂商 =
新增目录 + gen.conf，是异构多设备 SND（P5）加厂商路径的构建期一环（运行期对应
device-driver-registry）。

## Requirements

### Requirement: CG-01 厂商 manifest 驱动的可复现生成

系统 SHALL 提供 `make gen-yang` 生成入口：扫描 `backend/internal/generated/*/gen.conf`（每厂商包一份声明式生成配置：YANG 模型路径、模块列表、fakeroot/compress 选项，及**可选 `split_count`**），对每包执行**自研生成器**（`backend/tools/yanggen`，命名以实现为准；构建期以 goyang 解析 YANG 源，版本由 go.mod 锁定）→ 格式化收尾（gofmt/goimports，版本由 go.mod `tool` 指令锁定），输出该包生成物。华为包的 `yang_path` SHALL 以入库目录 `snd/ce6866p-yang` 为首目录（SP-01，无 submodule 依赖），并 MAY 以逗号追加仓库本地 deviation 目录（CG-04）。生成物 SHALL 包含：结构体/枚举/union（结构约定按 yang-native-runtime YN-01 冻结，实现 `Object` 接口族，SHALL NOT import ygot/goyang）、per-type RFC7951 JSON 方法（YN-02）、Schema IR 产物（YN-03）。生成物布局由 `split_count` 决定：**未设置**时输出单文件 `all.gen.go`；**设置为 N** 时输出拆分文件集（`structs-0..(N-1).go` + `enum.go`/`enum_map.go`/`union.go`/`schema.go`），使**单文件规模可控**（避免单包生成物随模型集成无限膨胀）。文件命名 SHALL 由生成器确定性给定。`make gen-yang VENDOR=<pkg>` SHALL 仅重生成指定包。新增厂商 SHALL 只需新增目录 + `gen.conf`，零脚本/Makefile 改动。管线 SHALL 可复现且机器无关：同一仓库状态下重复执行输出字节一致（拆分模式下每个生成文件的内容与 struct→文件分配均确定；无序集合的确定性排序 SHALL 内建于生成器，不依赖后处理），生成物 SHALL NOT 包含生成机器特定内容（如生成器绝对路径头部注释）。拆分 SHALL 语义等价于单文件：同包类型集合、导出符号、Schema IR 内容不变，下游 import 路径与消费无改动。

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
- **THEN** 拆分后包 SHALL `go build` 通过、Schema IR round-trip 成功、类型集合与 IR 模块键集合与拆分前一致，下游消费方零改动

#### Scenario: 模型源目录缺失时可操作报错
- **WHEN** `gen.conf` 的 `yang_path` 目录不存在或为空时执行 `make gen-yang`
- **THEN** SHALL 以非零码退出并输出指明缺失目录的修复指引（入库目录应随仓库存在，请检查 checkout 完整性），SHALL NOT 产生半成品输出，SHALL NOT 提示任何 submodule 操作

#### Scenario: 生成物零 ygot/goyang import
- **WHEN** 审计任一包生成物的 import 块
- **THEN** SHALL NOT 含 `openconfig/ygot`/`openconfig/goyang`，仅依赖标准库与自研运行库（YN-05 守护测试同口径拦截）

#### Scenario: 非法字符枚举标识符内建合法化（承接原 CG-02）
- **WHEN** YANG 值域含 Go 非法标识符字符（如 `50|100GE`）
- **THEN** 生成器 SHALL 直接产出合法标识符（`..._50_OR_100GE`），YANG 原值字符串映射 SHALL 保持原样，Linux/macOS 行为一致


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

### Requirement: CG-04 本地 deviation 豁免机制

生成闭包 SHALL 支持纳入仓库本地 deviation 模块（`backend/internal/yang/deviations/`）：
`gen.conf` 的 `yang_path` SHALL 支持逗号分隔多目录，deviation 模块与厂商模型同闭包
生成。deviation SHALL 仅用于豁免**生成器**不支持的个别节点（如 bits 类型默认值、
anydata、binary key list、穿 choice/case augment 的 leafref），每条 SHALL 注明豁免
原因与影响面；SHALL NOT 修改 snd 子模块内的模型本体（只读源）。生成器 SHALL 支持
解析期跳过不支持语句（等价 `-ignore_unsupported` 语义）。自研生成器 SHALL 沿用既有
deviation 集合启动（存量豁免不因换生成器而失效）；新生成器原生支持的节点 MAY 逐条
摘除对应 deviation（每条摘除随 regen-and-diff 显形）。无法经 deviation 豁免的模块
（解析期致命错误，如跨模块 submodule typedef 引用）SHALL 显式记录为延期项而非静默缺失。

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
