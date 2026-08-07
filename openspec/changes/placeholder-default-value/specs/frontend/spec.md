# Delta: frontend

## MODIFIED Requirements

### Requirement: FE-22 NCE 表单控件规范（三列栅格/key 标识/约束占位/字段级清除）

详情编辑区与表单 Tab 的表单 SHALL 按三列栅格布局（窄视口 SHALL 降为 2/1 列；choice、leaf-list、嵌套子表格 SHALL 占整行；when 隐藏字段 SHALL NOT 占位）。key 叶 SHALL 呈现钥匙标识（真实图标，R12）且编辑态只读。未携带 `dynamicDefault` 的字段 SHALL 由 schema 契约携带的约束元数据合成 placeholder（数值 range→`整数 合法范围: <范围>`；字符串 length→`合法长度: <范围>`；携带 `default` 时 SHALL 在范围/长度段后追加`，默认值: <值>`，仅有 default 时 SHALL 单独展示`默认值: <值>`，enum 下拉空值 SHALL 同规展示默认值占位，default 值本身 SHALL 原样展示不本地化；元数据由后端契约透出；`dynamicDefault` 字段保持 FE-15 「系统自动分配」占位优先，显式 placeholder 优先级最高）。每个可编辑且已有值的字段旁 SHALL 提供清除控件：对基线（设备实际态）有值的字段，清除 SHALL 记录为该叶的删除意图（随条目入变更集，提交时经叶级删除报文生效，CS-05），tooltip SHALL 明示「提交后将从设备删除该配置项」；对基线无值的字段，清除 SHALL 仅置空本地值（该键不入 payload）。必填字段清除后 SHALL 触发必填校验拦截「确定」。

#### Scenario: 约束合成占位

- **WHEN** 数值字段携带 range `[60, 1000000]` / 字符串字段携带 length `[1..31]`
- **THEN** 输入框空值时 SHALL 展示 `整数 合法范围: [60, 1000000]` / `合法长度: [1..31]` 占位；`dynamicDefault` 字段 SHALL 仍展示「系统自动分配」；显式 placeholder SHALL 优先于合成占位

#### Scenario: 默认值并入合成占位（NCE waterMark 对齐）

- **WHEN** 数值字段携带 range `[10..600]` 且 `default=300`
- **THEN** 占位 SHALL 为 `整数 合法范围: [10, 600]，默认值: 300`

#### Scenario: 仅默认值字段占位（边界）

- **WHEN** 字段无 range/length 但携带 `default`（含 enum 下拉）
- **THEN** 空值时 SHALL 展示 `默认值: <值>` 占位，值原样不本地化

#### Scenario: dynamicDefault 优先于 default（边界）

- **WHEN** 字段同时携带 `dynamicDefault=true` 与 `default`
- **THEN** SHALL 展示「系统自动分配」占位，SHALL NOT 展示默认值段
