# config-api delta — netconf-get-state-read

## MODIFIED Requirements

### Requirement: BR-01 配置读取（缓存优先）

`GET /api/v1/config/:ip/*path` SHALL 优先返回运行缓存（§8 TTL 30s）中的新鲜配置；缓存未命中时 SHALL 经共享 DeviceStore 解析的连接从设备回读（NETCONF `<get>`，含 config=false 状态数据），并回填缓存。回读结果 SHALL 为 RFC7951 结构（如 `{"interface":[{"name":…}]}`），可被前端列表化，而非裸 XML 字节；设备返回的 config=false 状态子树（如接口 `dynamic`、VLAN `status`）SHALL 原样包含在回读结果中——有则带出，无则不构造占位。响应 SHALL 携带 `cached` / `cache_age_seconds` / `ttl_seconds` / `source`（`cache`\|`device`）。

#### Scenario: 缓存命中
- **WHEN** 距上次读取 < TTL 且未带 `force_refresh`
- **THEN** SHALL 返回缓存数据，`source="cache"`、`cached=true`，不访问设备

#### Scenario: 缓存未命中回读设备
- **WHEN** 缓存过期/无 且设备已在 DeviceStore 注册
- **THEN** SHALL 用库中凭据回读设备，返回 RFC7951 结构，`source="device"`，并回填缓存

#### Scenario: 回读含状态数据
- **WHEN** 设备 `<get>` 回读返回含 config=false 子树（如接口 `dynamic`）的数据
- **THEN** 回读结果 SHALL 含对应状态字段（RFC7951 结构），前端只读控件可回显；写路径 payload 仍不含状态字段

#### Scenario: 设备无状态数据
- **WHEN** 设备回读仅返回配置数据（无状态子树）
- **THEN** 回读结果 SHALL 与改动前等值，SHALL NOT 构造空状态占位节点
