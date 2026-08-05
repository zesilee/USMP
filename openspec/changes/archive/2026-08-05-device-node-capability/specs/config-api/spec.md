## ADDED Requirements

### Requirement: BR-12 节点不支持快速失败与结构化错误

`GET /config/:ip/*path` 命中该设备节点级不支持集（CN-04）时，系统 SHALL 不向设备发起请求，直接返回统一响应格式错误并携带 `reason:"node-unsupported"`。`force_refresh=true` SHALL 绕过快速失败真实读取设备：成功 SHALL 移除该路径标记并返回数据；仍失败 SHALL 保留标记。`POST /config` 及变更集提交命中不支持路径的项 SHALL 拒绝且不打设备，错误同样携带 `reason:"node-unsupported"`。学习触发（设备真实返回 unknown-element）的那次请求 SHALL 返回同款结构化错误（前端可即时转占位态）。

#### Scenario: 已学习路径快速失败
- **WHEN** 设备 D 的 `devm:devm/devm:cards` 已在不支持集，请求 `GET /config/D/devm:devm/devm:cards`
- **THEN** SHALL 不发起设备请求，响应错误含 `reason:"node-unsupported"`

#### Scenario: force_refresh 重试并恢复
- **WHEN** 已标记路径以 `force_refresh=true` 请求且设备本次成功返回数据
- **THEN** SHALL 返回数据且移除该路径标记，后续常规读恢复正常链路

#### Scenario: 写路径门禁
- **WHEN** 变更集包含落在不支持路径下的配置项并提交
- **THEN** 该项 SHALL 被拒绝且不向设备下发，错误含 `reason:"node-unsupported"`

#### Scenario: 首次学习即结构化（负路径）
- **WHEN** 未标记路径读取时设备返回 unknown-element
- **THEN** 本次响应 SHALL 已含 `reason:"node-unsupported"`（无需第二次请求才转态）
