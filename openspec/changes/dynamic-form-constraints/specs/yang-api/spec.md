<!--
change delta：yang-api。主 spec 本次一并迁移到 CLI 标准格式（见 tasks P0.1）。
MODIFIED 的 Requirement 标题必须与迁移后的主 spec 完全一致。
-->

## MODIFIED Requirements

### Requirement: BR-03 Schema 动态生成（已知模块）

`GET /api/v1/yang/schema/:module` 对**已加载**的 YANG 模块 SHALL 由 `buildYangSchema`/`buildYangSchemaNested` **从 ygot schema 树动态生成** `fields`（含类型/枚举/必填/默认/约束元数据），SHALL NOT 返回硬编码/预定义 schema。`?form=nested` SHALL 返回容器/列表/叶的嵌套树。

#### Scenario: 动态生成已加载模块 schema
- **WHEN** 请求模块名（如 `ifm`）在 Manager 已加载的 schema 树中
- **THEN** SHALL 遍历该模块 node model 动态产出 `YangSchema{fields,listCols}`，字段类型/枚举/必填/默认取自 ygot 生成的 `yang.Entry`

#### Scenario: nested 形态
- **WHEN** 带 `?form=nested`
- **THEN** SHALL 返回容器→列表→叶的嵌套 `FieldDef` 树（`fields` 递归）

### Requirement: BR-04 未加载模块降级

`GET /api/v1/yang/schema/:module` 对**未加载**到 schema 树的模块名 SHALL 降级返回一个最小通用 schema（不崩，R08），SHALL NOT 500。

#### Scenario: 未知/未加载模块
- **WHEN** 请求模块名不在已加载 schema 树中
- **THEN** SHALL 返回最小通用 schema（如仅 name/description 字段）或明确错误码，页面仍可用

## ADDED Requirements

### Requirement: BR-05 约束元数据透出（when/must/pattern/range/default）

`GET /api/v1/yang/schema/:module` 生成的 `FieldDef` SHALL 从 ygot 内嵌的 goyang `yang.Entry` 采集并透出以下 YANG 约束元数据：`when`（可见性 XPath 表达式）、`must`（`[{expr,message}]`，`message` 取叶 `description` 兜底、缺省生成通用提示）、`pattern`（string 正则）、`minimum`/`maximum`（数值 `range`）、`default`。缺失某项时对应字段 SHALL 省略（omitempty），SHALL NOT 崩溃。

#### Scenario: 透出 when/must
- **WHEN** 某 leaf 在 YANG 定义了 `when`（如 `../class='sub-interface'`）或 `must`（如 `(../suppress>../reuse)`）
- **THEN** 该字段的 `FieldDef.when` / `FieldDef.must[].expr` SHALL 携带原始 XPath 表达式

#### Scenario: 透出 pattern/range/default
- **WHEN** 某 leaf 定义了 `pattern`、`range`/`length` 或 `default`
- **THEN** SHALL 分别填充 `FieldDef.pattern` / `minimum`+`maximum` / `default`

#### Scenario: 无约束的字段
- **WHEN** leaf 无 when/must/pattern/range/default
- **THEN** 对应字段 SHALL 省略，schema 生成 SHALL NOT 因缺约束而报错（R08）

### Requirement: BR-06 choice/case 呈现分组

由于 ygot 拍平 `choice`/`case`，`?form=nested` schema SHALL 从原始 `.yang`（goyang 解析）恢复 choice/case 分组，以 `FieldDef{nodeKind:"choice", cases:[{name,label,fields}]}` 呈现；分组内子叶的 `path` SHALL 保持其真实**扁平** YANG 路径（不因分组而改变），以保证 NETCONF 写入链路不受影响。原始 `.yang` 不可得时 SHALL 降级为扁平字段（R08）。

#### Scenario: 恢复 choice 分组
- **WHEN** 模块含 `choice`（如 IFM `choice bandwidth-type`）且原始 `.yang` 可解析
- **THEN** SHALL 输出 `nodeKind:"choice"` 节点，其 `cases[]` 分组对应各 `case` 的子字段，子字段 `path` 不变

#### Scenario: 缺原始 yang 降级
- **WHEN** 原始 `.yang` 文件缺失或解析失败
- **THEN** SHALL 跳过 choice 分组、退化为扁平字段并记录告警，接口 SHALL NOT 失败
