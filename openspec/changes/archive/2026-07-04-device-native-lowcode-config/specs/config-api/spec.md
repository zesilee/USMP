## ADDED Requirements

### Requirement: 通用 path↔ygot 配置编解码

`POST /api/v1/config/:ip/*path` SHALL 采用通用的 path→ygot 类型编解码，将请求 JSON 绑定到由 path 定位的 ygot 强类型结构，支持任意已加载 YANG 路径，替换仅认 system/ifm/vlan 三条 path 的硬编码 `convertToTypedStruct`。编解码 SHALL 以 ygot 生成结构为准，不新增手写 YANG 结构体，不以裸 `interface{}` 承载已知类型（R04）。

#### Scenario: 已加载路径通用编解码
- **WHEN** 向一个已加载但非旧硬编码三条之一的 YANG 路径 POST 合法配置
- **THEN** 系统 SHALL 将其编解码为对应 ygot 结构并写入 ConfigStore，触发 reconcile

#### Scenario: 与旧路径语义等价（双路径验证）
- **WHEN** 对 system/ifm/vlan 三条旧硬编码路径分别经新编解码与旧 `convertToTypedStruct` 处理相同输入
- **THEN** 两者产出的 desired 配置 SHALL 语义等价

#### Scenario: 未知路径不静默截断
- **WHEN** POST 一个 schema 树中不存在的路径
- **THEN** 系统 SHALL 回退处理并记录告警日志，而非静默丢弃或 panic（R08）

#### Scenario: 声明式语义不变
- **WHEN** POST 成功
- **THEN** 响应 SHALL 表示配置已接受（ACCEPTED），实际下发由 Reconciler 异步完成，契约与迁移前一致
