# frontend — 左树模块级展开 delta

## MODIFIED Requirements

### Requirement: FE-19 模型驱动 rpc 渲染与执行

前端 SHALL 把模块的 rpc 与顶层配置容器**平级**呈现在左侧导航树的模块叶下（导航层级：模块 → container 与 rpc 并列平铺，LT-03）；模块控制台 `/module/:module` 的 Tab 栏 SHALL NOT 再出现 rpc Tab（rpc 入口唯一收敛到左树）。点击某 rpc 节点 SHALL 路由 `/module/:module/rpc/:rpcName`，右侧内容区 SHALL 仅渲染该 rpc 的执行面板：input 由 schema 的 FieldDef 渲染（复用既有渲染管线，含 leafref 下拉、mandatory 校验、单位后缀），rpc 名与 input 叶标签 SHALL 按 UI-03 本地化，执行后 SHALL 回显 rpc-reply 结果或错误。rpc 路由页 SHALL 沿用全局设备上下文与面包屑骨架（配置/厂商/模块/rpc 名）。`rpcName` 不存在于该模块 schema 时 SHALL 展示明确错误提示且不崩（R08）。渲染 SHALL 由 schema 驱动，SHALL NOT 为具体 rpc 硬编码表单。

#### Scenario: rpc 与 container 平级呈现于左树

- **WHEN** 展开某含 rpc 的模块（huawei-ifm）左树叶
- **THEN** 该模块的 rpc（如「按接口名清除统计」）SHALL 与配置容器节点（「通用接口」）平级出现在左树中
- **AND** `/module/ifm` 控制台 Tab 栏 SHALL NOT 含任何 rpc Tab

#### Scenario: rpc 直达路由渲染执行面板

- **WHEN** 打开 `/module/ifm/rpc/reset-if-counters-by-name`
- **THEN** 内容区 SHALL 仅渲染该 rpc 执行面板，if-name SHALL 渲染为接口名下拉（leafref 驱动）
- **AND** 缺 mandatory input 时执行按钮 SHALL 被校验拦截（不执行）

#### Scenario: 执行回显结果

- **WHEN** 选合法 input 并执行
- **THEN** 前端 SHALL 调用执行 API 并回显 rpc-reply 结果；失败时回显错误（R08）

#### Scenario: 未知 rpc 名降级（负路径）

- **WHEN** 打开 `/module/ifm/rpc/no-such-rpc`
- **THEN** 内容区 SHALL 展示明确错误提示，SHALL NOT 崩溃或空白
