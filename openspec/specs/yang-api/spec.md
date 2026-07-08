# yang-api — YANG 动态表单 Schema 北向接口

## Purpose

yang-api 是 Stack B 北向 REST 接口，向前端供给**由 YANG 模型驱动**的动态表单元数据：模块列表（`GET /api/v1/yang/modules`）与单模块表单 Schema（`GET /api/v1/yang/schema/:module`，含扁平/嵌套两态）。Schema 从 ygot 内嵌的 goyang `yang.Entry` 树**动态生成**（R04：派生自 ygot 生成模型，非手写、非硬编码），并透出 YANG 业务约束（when/must/pattern/range/default）与 choice/case 分组，供前端 100% 数据驱动渲染（R05）。连接/加载能力由 Manager 的 schema 树决定。

## Requirements

### Requirement: BR-01 模块列表（已加载）

`GET /api/v1/yang/modules` 当 Manager 已加载 YANG 模块时 SHALL 遍历 `Schema.Modules()` 返回实际模块列表（每项含 `name`/`title`/`vendor`/`path`/`description`/`type`）。

#### Scenario: 有已加载模块
- **WHEN** Manager 的 schema 树已加载模块
- **THEN** SHALL 从 `Schema.Modules()` 动态遍历返回实际模块列表

### Requirement: BR-02 模块列表降级（无已加载）

`GET /api/v1/yang/modules` 当无已加载模块时 SHALL 返回一份最小示例模块列表（huawei-ifm + huawei-vlan）以保证页面可用，SHALL NOT panic（R08）。

#### Scenario: 无已加载模块
- **WHEN** Manager 尚无已加载模块
- **THEN** SHALL 返回最小示例模块列表，接口 SHALL NOT 失败

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

`?form=nested` schema SHALL 将 YANG `choice`/`case` 呈现为 `FieldDef{type:"choice", cases:[{name,label,fields}]}` 分组节点。choice/case 元数据 SHALL 直接取自 ygot 内嵌的 goyang `Entry` 树（`huawei.Schema()` 由编译进二进制的 gzip schema blob 重建，运行期不读 `.yang`；该树完整保留 `IsChoice()`/`IsCase()` 与嵌套），SHALL NOT 依赖构建期生成器或运行期解析原始 `.yang`。分组内子字段的 `path` SHALL 为其**扁平** YANG 数据路径（剥除 choice/case 段，保留真实 container/list 段，如 `/ifm/interfaces/interface/bandwidth`、嵌套 `…/damp/manual/suppress`），以保证 NETCONF 写入链路不受影响。schema 中无 choice 时该模块 SHALL 退化为原扁平字段（R08）。

#### Scenario: 恢复 choice 分组
- **WHEN** 已加载模块的 schema 树含 `choice`（如 IFM `choice bandwidth-type` 的 `bandwidth-mbps`/`bandwidth-kbps` 两 case）
- **THEN** SHALL 输出 `type:"choice"` 节点，其 `cases[]` 分组对应各 `case` 的子字段，子字段 `path` 为扁平数据路径（不含 choice/case 段）

#### Scenario: 嵌套 choice
- **WHEN** `case` 内嵌套 `container` 且其内再含 `choice`（如 IFM `damping→damp→level`）
- **THEN** SHALL 递归呈现嵌套 `type:"choice"` 节点，各层子字段 `path` 均剥除本层 choice/case 段、保留 container 段

#### Scenario: 无 choice 的模块
- **WHEN** 模块 schema 树不含任何 `choice`
- **THEN** SHALL 退化为原有扁平/嵌套字段输出，接口 SHALL NOT 失败（R08）

### Requirement: BR-07 呈现扩展透出（support-filter / operation-exclude）

`GET /yang/schema/{module}`（含 `?form=nested`）SHALL 把 YANG 扩展 `support-filter` 与
`operation-exclude` 从 schema 树透出到 `FieldDef`：叶级 `supportFilter`(bool)、
`operationExclude`([]string，按 `|`/`,` 切分、小写归一)；list/container 节点上的
`operation-exclude` SHALL 透出在对应 group/list FieldDef 上；list key 叶 SHALL 以
`isKey=true` 标注。匹配 SHALL 按扩展关键字本名（前缀无关）。无扩展的字段 SHALL
省略这些键（omitempty）。

#### Scenario: 真实模型透出

- **WHEN** 请求 `huawei-ifm` 的 nested schema
- **THEN** `interfaces/interface` 的 `class`、`type` 叶 SHALL 带 `supportFilter=true`
- **AND** `class`、`number`、`parent-name`、`router-type` 叶 SHALL 带
  `operationExclude=["update","delete"]`，`name` 叶 SHALL 带 `isKey=true`

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

## 数据模型

**FieldDef**（动态表单字段；omitempty 字段缺省即省略）：

| 字段 | 类型 | 说明 |
|------|------|------|
| path | string | 字段数据路径（扁平 YANG path，末段为数据键） |
| type | string | `string`/`number`/`boolean`/`enum`/`group`/`list`/`leaf-list`/`choice` |
| label | string | 字段标签（取 YANG 节点名） |
| required | bool | 是否必填（YANG mandatory） |
| default | any | 默认值 |
| options | Option[] | enum 选项（`{label,value}`） |
| group | string | 扁平形态下的分组名 |
| minimum/maximum | int | 显式数值 `range` 边界 |
| pattern | string | string `pattern` 正则 |
| when | string | `when` XPath 表达式（BR-05） |
| must | MustRule[] | `must` 约束 `[{expr,message}]`（BR-05） |
| fields | FieldDef[] | `group`/`list` 的嵌套子字段（nested 形态） |
| cases | CaseDef[] | `choice` 的互斥分支（BR-06） |
| supportFilter | bool | 厂商 `support-filter` 扩展：可作查询条件（BR-07） |
| operationExclude | string[] | 厂商 `operation-exclude` 扩展（小写归一，BR-07） |
| presence | bool | YANG presence 容器（type=group 时有意义，BR-08） |
| isKey | bool | list key 叶标注（BR-07） |

**CaseDef**：`{name, label, fields: FieldDef[]}` —— 一个 `case` 分支及其子字段（子字段 path 为扁平数据路径）。
