# frontend-ui-adapter — UI 组件库适配层

## Purpose

`src/ui` 是前端业务代码与具体 UI 组件库之间的唯一隔离层：业务代码只从适配层导入控件/图标/反馈 API/主题令牌，组件库成为单点可替换依赖（已于 2026-08 完成 antd→EviewUI 切换实证：生产=EviewUI 桥经 `@ui-backend` 别名，外网测试=antd 镜像 `src/ui/antd-backend`；Tree/Tabs/Popover 桥内自绘）。适配层是薄转发（控件重导出 + API 差异抹平 + 图标与令牌收口），不承载业务语义。

## Requirements

### Requirement: FA-01 适配层为组件库唯一入口

前端业务代码（页面、业务组件、composable/hook）SHALL 只从 UI 适配层（`src/ui`）导入界面控件，SHALL NOT 直接 import 具体组件库包名。适配层 SHALL 是组件库的唯一引用点，使组件库成为单点可替换依赖（当前生产实现为 EviewUI 桥，antd 镜像仅作外网测试后端，经 `@ui-backend` 别名切换）。违反 SHALL 由静态检查与守护测试拦截。

#### Scenario: 业务代码直接引用组件库被拦截
- **WHEN** 业务代码中出现直接 import 具体组件库包名的语句（`src/ui` 目录自身除外）
- **THEN** 静态检查 SHALL 报错、守护测试 SHALL 失败，改动 SHALL NOT 通过门禁

#### Scenario: 经适配层引用正常通过
- **WHEN** 业务代码从 `src/ui` 导入控件
- **THEN** 检查 SHALL 通过，控件 SHALL 正常渲染

#### Scenario: 组件库替换只改适配层（能力锚点）
- **WHEN** 替换底层组件库实现
- **THEN** 改动范围 SHALL 限于 `src/ui` 目录，业务代码 SHALL NOT 需要修改 import 语句

### Requirement: FA-02 适配层导出面收敛且为受控桥接层

适配层 SHALL 仅导出项目实际使用的控件，SHALL NOT 整包透传组件库全部导出。适配层职责为：控件重导出与 API 差异抹平（含**受控性桥接**——底层控件为半受控实现时，适配层 SHALL 对外呈现真受控语义）、图标与主题令牌收口；SHALL NOT 承载业务语义（YANG 字段派生、校验规则生成、下发链路逻辑）。

#### Scenario: 新增控件须先进适配层
- **WHEN** 业务需要一个适配层尚未导出的控件
- **THEN** SHALL 先在适配层补充导出，再由业务代码引用

#### Scenario: 整包透传被拒
- **WHEN** 适配层以整包再导出（如 `export * from` 组件库根）的方式实现
- **THEN** 守护测试 SHALL 失败——导出面 SHALL 逐项显式声明

#### Scenario: 半受控底层不泄漏到业务面
- **WHEN** 业务代码以受控形态使用适配层控件（传 value/checked 与 onChange）且父级拒绝回写
- **THEN** 控件展示 SHALL 还原为父级值，SHALL NOT 停留在内部自改状态（适配层以事件拦截/受控回写/key 重挂三档兜底实现）

### Requirement: FA-03 命令式反馈 API 收口

轻提示与确认框 SHALL 由适配层统一导出为可在任意函数中调用的形式：提示为 `toast()`，确认为返回 Promise 的 `confirm()`。业务代码 SHALL 以 `await confirm(...)` 表达二次确认，SHALL NOT 各自持有组件库的实例句柄。

#### Scenario: 确认框以 Promise 返回用户选择
- **WHEN** 业务代码调用 `await confirm(...)` 且用户点击确认
- **THEN** SHALL 返回真值并继续后续操作；用户取消 SHALL 返回假值且 SHALL NOT 执行后续操作

#### Scenario: 非组件上下文中调用不崩溃
- **WHEN** 在普通函数（非组件渲染上下文）中调用 `toast()`/`confirm()`
- **THEN** SHALL 正常弹出且 SHALL NOT 抛错（R08）

### Requirement: FA-04 图标与主题令牌收口

界面图标 SHALL 由适配层统一导出，SHALL NOT 使用 emoji 代替图标（R12）；无对应图标时 SHALL 使用规范占位符。主题色板与间距 SHALL 经适配层的令牌收口，业务组件 SHALL NOT 硬编码色值。

#### Scenario: emoji 充当图标被拦截
- **WHEN** 界面代码以 emoji 字符充当图标
- **THEN** 门禁 SHALL 失败（R12）

#### Scenario: 缺失图标降级占位
- **WHEN** 所需图标在组件库中不存在
- **THEN** 适配层 SHALL 提供规范占位图标，界面 SHALL NOT 空白或崩溃（R08）

### Requirement: FA-05 测试锚点跨库稳定

`data-test` 测试锚点语义 SHALL 跨组件库保持：底层组件支持属性透传时直传；不支持时适配层 SHALL 以外包 wrapper 元素或组件 `id` prop 承载同名锚点。锚点被底层静默吞掉 SHALL 被守护测试拦截，SHALL NOT 出现"传了 data-test 但 DOM 上不存在"的静默失效。

#### Scenario: 不透传组件的锚点落点
- **WHEN** 业务经适配层给不支持透传的控件传 `data-test`
- **THEN** 渲染结果中 SHALL 存在可被 `[data-test=...]` 选择器命中的元素（wrapper 或 id 映射）

#### Scenario: 锚点静默失效被拦（负路径）
- **WHEN** 适配层某控件未实现锚点落点
- **THEN** 守护测试 SHALL 失败

### Requirement: FA-06 表单项外壳（FormItemShell）

表单项的 label/必填星/错误态展示 SHALL 由适配层自有的 FormItemShell 提供（受控 `error` 文案入参），SHALL NOT 使用组件库表单容器的内部 store 与校验器（校验权威在自研约束引擎）。

#### Scenario: 受控错误态渲染
- **WHEN** 校验引擎判定字段非法并传入 error 文案
- **THEN** FormItemShell SHALL 显示错误文案与错误样式；error 清空 SHALL 即时消除

#### Scenario: 不接组件库校验器（负路径）
- **WHEN** 适配层实现向底层控件传递 validator/required/rules 类校验属性
- **THEN** 守护测试 SHALL 失败
