# frontend delta — nce-fidelity-polish

## MODIFIED Requirements

### Requirement: FE-01 schema 驱动渲染

前端 SHALL 将 schema（CRD `OpenAPIV3Schema` 或后端 YANG nested schema）经 `crdSchemaParser` 逐属性映射为 `Field[]`，类型映射为 boolean→switch、number→input-number、object→group；enum SHALL 按选项数与必填性细分：**必填且选项 ≤3 → segmented 分段控件，其余（可选或 >3 选项）→ select 下拉**（可选枚举 SHALL 保留清空能力，清空即该键不入 payload）。映射经 `DynamicForm`/`FieldRenderer` 渲染为 Element Plus 控件（R05）。SHALL NOT 手写固定表单。

#### Scenario: 类型到控件的自动映射

- **WHEN** `parseCRDSchemaToFields(schema)` 或 `getYangSchema(module, 'nested')` 返回带类型的属性
- **THEN** SHALL 生成对应 `Field[]`，并按类型渲染对应控件（boolean→switch、number→input-number、object→分组）

#### Scenario: 必填短枚举分段控件

- **WHEN** enum 字段 `required=true` 且选项数 ≤3
- **THEN** SHALL 渲染分段控件展示全部选项，选中 SHALL 触发值更新；readonly/禁用态 SHALL 透传为控件禁用

#### Scenario: 可选或长枚举保持下拉（边界）

- **WHEN** enum 字段可选（`required=false`）或选项数 >3
- **THEN** SHALL 渲染 select 下拉；可选枚举 SHALL 可清空，清空后该键 SHALL NOT 进入下发 payload

#### Scenario: 无有效 schema

- **WHEN** schema 拉取失败或为空
- **THEN** SHALL NOT 崩溃（R08），页面继续可用，仅不渲染该模块字段
