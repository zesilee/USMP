## MODIFIED Requirements

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

## ADDED Requirements

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
