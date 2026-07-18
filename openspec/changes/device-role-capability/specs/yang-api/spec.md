# yang-api — delta（device-role-capability）

## ADDED Requirements

### Requirement: BR-12 按设备协商的模块列表

`GET /api/v1/yang/modules` SHALL 支持可选 `device=<id>` 查询参数：给定时按 device-capability-negotiation CN-02 返回该设备协商子集并携带 `negotiated` 标记；未给定时行为与既有 BR-01/BR-02 完全一致（全量，向后兼容）。模块项 SHALL 按 CN-03 附带 `blacklisted` 注解（omitempty）。

#### Scenario: 带 device 参数
- **WHEN** 请求 `GET /api/v1/yang/modules?device=<已注册设备>`
- **THEN** SHALL 返回该设备 hello 能力协商后的模块子集 + `negotiated` 标记

#### Scenario: 不带 device 参数（兼容）
- **WHEN** 请求不带 `device`
- **THEN** 返回 SHALL 与既有 BR-01/BR-02 行为一致，SHALL NOT 因协商能力引入新失败路径
