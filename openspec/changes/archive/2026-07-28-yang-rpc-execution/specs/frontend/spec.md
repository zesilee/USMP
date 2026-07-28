## ADDED Requirements

### Requirement: FE-19 模型驱动 rpc 渲染与执行

前端 SHALL 在模块控制台内把该模块的 rpc 与顶层配置节点**平级**呈现（导航层级：模块 → container 与 rpc 并列）。点击某 rpc SHALL 打开其执行面板：input 由 schema 的 FieldDef 渲染（复用既有渲染管线，含 leafref 下拉、mandatory 校验、单位后缀），执行后 SHALL 回显 rpc-reply 结果或错误。渲染 SHALL 由 schema 驱动，SHALL NOT 为具体 rpc 硬编码表单。

#### Scenario: rpc 与 container 平级呈现

- **WHEN** 进入某含 rpc 的模块（huawei-ifm）控制台
- **THEN** 该模块的 rpc（如「按接口名清除统计」）SHALL 与配置容器平级出现在导航中

#### Scenario: input 由 schema 渲染 + 校验拦截

- **WHEN** 打开 `reset-if-counters-by-name` 执行面板
- **THEN** if-name SHALL 渲染为接口名下拉（leafref 驱动）
- **AND** 缺 mandatory input 时执行按钮 SHALL 被校验拦截（不执行）

#### Scenario: 执行回显结果

- **WHEN** 选合法 input 并执行
- **THEN** 前端 SHALL 调用执行 API 并回显 rpc-reply 结果；失败时回显错误（R08）

### Requirement: FE-20 高危 rpc 执行确认

前端 SHALL 在执行任一 rpc 前弹确认（展示 rpc 名、input 值、目标设备）。对高危 rpc（highRisk 标记，如 `restart-if`）SHALL 升级为更醒目的警示确认。用户未确认时 SHALL NOT 执行。

#### Scenario: 普通 rpc 基础确认

- **WHEN** 执行一个非高危 rpc
- **THEN** 前端 SHALL 先弹确认展示 rpc 名/input/目标设备，确认后才执行

#### Scenario: 高危 rpc 升级警示

- **WHEN** 执行高危 rpc（highRisk，如 restart-if）
- **THEN** 前端 SHALL 展示升级的高危警示确认
- **AND** 用户取消时 SHALL NOT 向设备下发
