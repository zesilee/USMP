# devices-api — delta（device-role-capability）

## ADDED Requirements

### Requirement: BR-14 设备网络角色（role）

设备注册（POST）与更新 SHALL 接受可选 `role` 字段（自由字符串标签，常用值 DCGW/EOR/TOR/BORDER；SHALL 校验 ≤32 字符且仅含字母数字与 `-_`，非法值 400）；列表与详情 SHALL 透传 `role`（omitempty）。role SHALL 持久化于 Device CRD `spec.role`（无集群模式内存后备，语义一致）。role SHALL NOT 影响模块协商/裁剪（仅展示与策略标签）。

#### Scenario: 注册携带 role
- **WHEN** POST /api/v1/devices 带 `role:"DCGW"`
- **THEN** 列表与详情 SHALL 返回该 role，Device CR `spec.role` SHALL 为 `DCGW`

#### Scenario: 非法 role（负路径）
- **WHEN** `role` 超长或含非法字符
- **THEN** SHALL 返回 400 明确错误，SHALL NOT 落库

#### Scenario: 未指定 role
- **WHEN** 注册未带 role
- **THEN** 响应 SHALL NOT 含 `role` 键（omitempty），功能不受影响
