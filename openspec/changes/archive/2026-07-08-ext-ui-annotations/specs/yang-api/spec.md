# yang-api delta — ext-ui-annotations

## ADDED Requirements

### Requirement: BR-09 原生呈现元数据透出（config-false→readonly / units）

`GET /api/v1/yang/schema?module=<m>` SHALL 从内嵌 goyang Entry 透出原生呈现元数据：
`config false` 节点（含其整棵子树，遵循 YANG config 继承性）SHALL 置 `FieldDef.readonly=true`
（容器/list/叶一致生效）；叶携带 `units`（`Type.Units` 或 `Entry.Units`）时 SHALL 透出
`FieldDef.units`。config true 且无 units 的字段 SHALL 省略两字段（omitempty）。

#### Scenario: config false 子树整体只读

- **WHEN** 请求 ifm 模块 schema（`remote-interfaces` 等子树在模型中为 `config false`）
- **THEN** 该子树根及全部后代 FieldDef SHALL `readonly=true`
- **AND** config true 的兄弟字段 SHALL NOT 携带 readonly

#### Scenario: units 透出

- **WHEN** 叶在模型中声明 `units "bit/s"`
- **THEN** 对应 FieldDef SHALL 含 `units: "bit/s"`

#### Scenario: 无元数据时省略（边界）

- **WHEN** 叶既非只读也无 units
- **THEN** 序列化结果 SHALL NOT 出现 `readonly`/`units` 键

### Requirement: BR-10 dynamic-default 扩展透出

`GET /api/v1/yang/schema?module=<m>` 对携带厂商 `dynamic-default` 扩展的叶 SHALL 透出 `FieldDef.dynamicDefault=true`（前缀无关按本名匹配，与 BR-07 同规）；无该扩展 SHALL 省略。
提取 SHALL 容忍带 `default-value` 子句与无子句两种形态（仅取布尔存在性，R08 不解析表达式）。

#### Scenario: 动态缺省叶标记

- **WHEN** ifm `admin-status` 叶携带 `ext:dynamic-default`
- **THEN** 其 FieldDef SHALL 含 `dynamicDefault: true`

#### Scenario: 负路径——其他扩展不误报

- **WHEN** 叶仅携带 `support-filter`/`operation-exclude` 等其他扩展
- **THEN** FieldDef SHALL NOT 含 `dynamicDefault`

## MODIFIED Requirements

### Requirement: BR-01 模块列表（已加载）

`GET /api/v1/yang/modules` 当 Manager 已加载 YANG 模块时 SHALL 遍历 `Schema.Modules()` 返回实际模块列表（每项含 `name`/`title`/`vendor`/`path`/`description`/`type`）。模块在源 YANG 中声明模块级 `ext:task-name` 时 SHALL 附带 `category` 字段，取值来自构建期生成的任务域映射表（键=模块根容器名，与 `name` 一致）；无映射 SHALL 省略 `category`（omitempty，R08 不失败）。

#### Scenario: 有已加载模块
- **WHEN** Manager 的 schema 树已加载模块
- **THEN** SHALL 从 `Schema.Modules()` 动态遍历返回实际模块列表

#### Scenario: 任务域 category 附带
- **WHEN** 已加载模块（如 ifm）在构建期映射表中存在 task-name
- **THEN** 该模块项 SHALL 含 `category`，且键匹配按模块根容器名进行
- **WHEN** 模块无映射
- **THEN** 该模块项 SHALL NOT 含 `category`，接口 SHALL NOT 失败
