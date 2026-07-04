## ADDED Requirements

### Requirement: 从 schema 树动态生成 YANG schema

`GET /api/v1/yang/schema/:module` SHALL 从 Manager 的 schema 树动态生成前端字段定义（FieldDef），不再使用 handler 内硬编码的 IFM/VLAN 预定义 schema。字段类型映射 SHALL 覆盖 YANG 基本类型（leaf/leaf-list/container/list/enumeration/boolean/number/string）。

#### Scenario: 已加载模块返回真实 schema
- **WHEN** 请求一个 schema 树中已加载的模块（如 huawei-vlan）
- **THEN** 响应 SHALL 是从该模块 YANG 结构生成的字段定义，覆盖其可配置属性，而非 2 字段占位

#### Scenario: 未知模块返回明确结果
- **WHEN** 请求一个 schema 树中不存在的模块
- **THEN** 响应 SHALL 明确指示模块未知（而非静默回退到通用 2 字段桩）

### Requirement: 模块列表反映已加载 YANG 模型

`GET /api/v1/yang/modules` SHALL 从 schema 树枚举实际已加载的 YANG 模块（名称、根节点、厂商），不再在无模块时回退硬编码示例列表。

#### Scenario: 列出已加载模块
- **WHEN** schema 树已加载 huawei/openconfig 模块后请求模块列表
- **THEN** SHALL 返回这些模块的元信息，`vendor` 依模型命名空间/来源判定

#### Scenario: 契约保持
- **WHEN** 前端按既有响应结构消费该端点
- **THEN** 响应的 JSON 结构（字段名/形态）SHALL 与迁移前保持兼容
