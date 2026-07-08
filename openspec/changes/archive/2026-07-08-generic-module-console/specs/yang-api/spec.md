# yang-api — generic-module-console delta

## ADDED Requirements

### Requirement: BR-07 呈现扩展透出（support-filter / operation-exclude）

`GET /yang/schema/{module}`（含 `?form=nested`）SHALL 把 YANG 扩展 `support-filter` 与
`operation-exclude` 从 schema 树透出到 `FieldDef`：叶级 `supportFilter`(bool)、
`operationExclude`([]string，按 `|`/`,` 切分、小写归一)；list/container 节点上的
`operation-exclude` SHALL 透出在对应 group/list FieldDef 上。匹配 SHALL 按扩展关键字
本名（前缀无关）。无扩展的字段 SHALL 省略这些键（omitempty）。

#### Scenario: 真实模型透出

- **WHEN** 请求 `huawei-ifm` 的 nested schema
- **THEN** `interfaces/interface` 的 `class`、`type` 叶 SHALL 带 `supportFilter=true`
- **AND** `class`、`number`、`parent-name`、`router-type` 叶 SHALL 带
  `operationExclude=["update","delete"]`

#### Scenario: 前缀与参数容错

- **WHEN** 扩展以任意 import 前缀出现（如 `hw:support-filter`），或参数含大小写/混合分隔符
- **THEN** 采集 SHALL 按本名匹配并归一化；参数缺失或无法解析时 SHALL 视为无扩展（降级，不报错）

### Requirement: BR-08 presence 容器与容器级 when/must 透出

nested schema 中 presence 容器 SHALL 以 `presence=true` 标注在其 group FieldDef 上；
容器级 `when`/`must`（goyang `Entry.Extra`）SHALL 与叶级同构透出（`when` 字符串、
`must` 为 `[{expr,message}]`）。

#### Scenario: IFM 全局冲突开关

- **WHEN** 请求 `huawei-ifm` 的 nested schema
- **THEN** `global/ipv4-conflict-enable` group SHALL 带 `presence=true` 且
  `must` 含 `../ipv4-ignore-primary-sub='false'`

#### Scenario: 非 presence 容器不受影响

- **WHEN** 容器无 presence 语句
- **THEN** 其 group FieldDef SHALL 不含 `presence` 键
