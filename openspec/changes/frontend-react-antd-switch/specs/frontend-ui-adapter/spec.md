## ADDED Requirements

### Requirement: FA-01 适配层为组件库唯一入口

前端业务代码（页面、业务组件、composable/hook）SHALL 只从 UI 适配层（`src/ui`）导入界面控件，SHALL NOT 直接 import 具体组件库包名。适配层 SHALL 是组件库的唯一引用点，使组件库成为单点可替换依赖（当前实现为 antd，后继可能为公司内部组件库）。违反 SHALL 由静态检查与守护测试拦截。

#### Scenario: 业务代码直接引用组件库被拦截
- **WHEN** 业务代码中出现直接 import 具体组件库包名的语句（`src/ui` 目录自身除外）
- **THEN** 静态检查 SHALL 报错、守护测试 SHALL 失败，改动 SHALL NOT 通过门禁

#### Scenario: 经适配层引用正常通过
- **WHEN** 业务代码从 `src/ui` 导入控件
- **THEN** 检查 SHALL 通过，控件 SHALL 正常渲染

#### Scenario: 组件库替换只改适配层（能力锚点）
- **WHEN** 替换底层组件库实现
- **THEN** 改动范围 SHALL 限于 `src/ui` 目录，业务代码 SHALL NOT 需要修改 import 语句

### Requirement: FA-02 适配层导出面收敛且为薄转发

适配层 SHALL 仅导出项目实际使用的控件，SHALL NOT 整包透传组件库全部导出。适配层 SHALL 为薄转发层，职责限于：控件重导出、已知 API 差异抹平、图标与主题令牌收口；SHALL NOT 承载业务语义（如 YANG 字段派生、校验规则生成、下发链路逻辑）。

#### Scenario: 新增控件须先进适配层
- **WHEN** 业务需要一个适配层尚未导出的控件
- **THEN** SHALL 先在适配层补充导出，再由业务代码引用

#### Scenario: 整包透传被拒
- **WHEN** 适配层以整包再导出（如 `export * from` 组件库根）的方式实现
- **THEN** 守护测试 SHALL 失败——导出面 SHALL 逐项显式声明

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
