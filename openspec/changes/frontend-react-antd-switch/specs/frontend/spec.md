## MODIFIED Requirements

### Requirement: FE-01 schema 驱动渲染

前端 SHALL 将后端 YANG nested schema 经 `crdSchemaParser` 逐属性映射为 `Field[]`，类型映射为 boolean→「打开/关闭」radio 单选组（i18n 文案，值仍为 true/false；可选 boolean SHALL 支持不选=不入 payload）、number→input-number、object→group；enum SHALL 按选项数与必填性细分：**必填且选项 ≤3 → segmented 分段控件，其余（可选或 >3 选项）→ select 下拉**（可选枚举 SHALL 保留清空能力，清空即该键不入 payload）。映射经 `FieldRenderer` 渲染为 **UI 适配层（`src/ui`）导出的控件**（R05），SHALL NOT 直接依赖具体组件库（见 `frontend-ui-adapter` FA-01）。SHALL NOT 手写固定表单。

#### Scenario: 类型到控件的自动映射
- **WHEN** `getYangSchema(module, 'nested')` 返回带类型的属性
- **THEN** SHALL 生成对应 `Field[]`，并按类型渲染对应控件（boolean→打开/关闭 radio、number→input-number、object→分组）

#### Scenario: boolean radio 值语义
- **WHEN** boolean 字段选中「打开」
- **THEN** payload 中该叶 SHALL 为 true；「关闭」SHALL 为 false；可选 boolean 未选时该键 SHALL NOT 入 payload

#### Scenario: 必填短枚举分段控件
- **WHEN** enum 字段 `required=true` 且选项数 ≤3
- **THEN** SHALL 渲染分段控件展示全部选项，选中 SHALL 触发值更新；readonly/禁用态 SHALL 透传为控件禁用

#### Scenario: 可选或长枚举保持下拉（边界）
- **WHEN** enum 字段可选（`required=false`）或选项数 >3
- **THEN** SHALL 渲染 select 下拉；可选枚举 SHALL 可清空，清空后该键 SHALL NOT 进入下发 payload

#### Scenario: 无有效 schema
- **WHEN** schema 拉取失败或为空
- **THEN** SHALL NOT 崩溃（R08），页面继续可用，仅不渲染该模块字段

#### Scenario: 控件映射结论与组件库实现无关（换库锚点）
- **WHEN** 底层组件库实现发生替换
- **THEN** 上述全部类型→控件映射结论 SHALL 保持不变，派生黄金（GD-01）SHALL 零漂移

## ADDED Requirements

### Requirement: FE-27 表单键存在性即节点存在性

表单状态 SHALL 以「键是否存在」表达 YANG 节点是否存在：presence 容器关闭、choice 非激活分支成员、动态缺省叶留空、字段级清除等场景，对应键 SHALL 从表单数据中真正移除，SHALL NOT 仅置为空值（`undefined`/`null`）。下发 payload SHALL NOT 包含这些键。

#### Scenario: presence 容器关闭后键消失
- **WHEN** 用户关闭 presence 容器开关
- **THEN** 表单数据中该容器键 SHALL 不存在（以「键枚举」判定，非「取值为空」判定），且 SHALL NOT 进入下发 payload

#### Scenario: choice 切换分支清空非激活成员
- **WHEN** 用户从 case A 切换到 case B
- **THEN** case A 全部成员键 SHALL 从表单数据中移除，payload SHALL 只含 case B 成员

#### Scenario: 动态缺省叶留空不下发（负路径）
- **WHEN** 带 `dynamicDefault` 的叶被清空
- **THEN** 该键 SHALL NOT 进入 payload，SHALL NOT 以空串或 null 形式下发覆盖设备缺省
