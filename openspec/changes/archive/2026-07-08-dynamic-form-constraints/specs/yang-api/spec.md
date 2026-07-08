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

`?form=nested` schema SHALL 将 YANG `choice`/`case` 呈现为 `FieldDef{type:"choice", cases:[{name,label,fields}]}` 分组节点。choice/case 元数据 SHALL 直接取自 ygot 内嵌的 goyang `Entry` 树（`huawei.Schema()` 由编译进二进制的 gzip schema blob 重建，运行期不读 `.yang`；该树完整保留 `IsChoice()`/`IsCase()` 与嵌套），SHALL NOT 依赖构建期生成器或运行期解析原始 `.yang`。分组内子字段的 `path` SHALL 为其**扁平** YANG 数据路径（剥除 choice/case 段，保留真实 container/list 段，如 `/ifm/interfaces/interface/bandwidth`、嵌套 `…/damp/manual/suppress`），以保证 NETCONF 写入链路不受影响。schema 中无 choice 时该模块 SHALL 退化为原扁平字段（R08）。

> 说明：先前设计基于「ygot 拍平 choice/case、须构建期生成 choice-map」的判断；实测 `huawei.Schema()` 内嵌 schema 已完整保留 choice/case，故改为直接从内嵌 schema 恢复，去除生成器与 `.yang` 运行期依赖。

#### Scenario: 恢复 choice 分组
- **WHEN** 已加载模块的 schema 树含 `choice`（如 IFM `choice bandwidth-type` 的 `bandwidth-mbps`/`bandwidth-kbps` 两 case）
- **THEN** SHALL 输出 `type:"choice"` 节点，其 `cases[]` 分组对应各 `case` 的子字段，子字段 `path` 为扁平数据路径（不含 choice/case 段）

#### Scenario: 嵌套 choice
- **WHEN** `case` 内嵌套 `container` 且其内再含 `choice`（如 IFM `damping→damp→level`）
- **THEN** SHALL 递归呈现嵌套 `type:"choice"` 节点，各层子字段 `path` 均剥除本层 choice/case 段、保留 container 段

#### Scenario: 无 choice 的模块
- **WHEN** 模块 schema 树不含任何 `choice`
- **THEN** SHALL 退化为原有扁平/嵌套字段输出，接口 SHALL NOT 失败（R08）
