## MODIFIED Requirements

### Requirement: DP-02 协议选择（AUTO 按端口）

client factory SHALL 依据 `DeviceConnectionInfo.Protocol` 选择协议：`NETCONF`→NETCONF、`AUTO`→按端口判定（端口 0 或 830→NETCONF、9339→显式未实现错误、其余默认 NETCONF）。`GNMI` 与 `AUTO`+9339 SHALL 返回明确的「gNMI 尚未实现（规划能力）」错误，SHALL NOT 建立伪装成功的空壳连接；未知协议 SHALL 返回错误。

#### Scenario: AUTO 按端口落 NETCONF
- **WHEN** `Protocol=AUTO` 且端口为 0 或 830（或其他非 9339 端口）
- **THEN** SHALL 建 NETCONF client（端口 0 补 830）

#### Scenario: gNMI 显式未实现（负路径）
- **WHEN** `Protocol=GNMI`（或 `AUTO` 且端口 9339）
- **THEN** factory SHALL 返回含「gNMI」与「未实现」语义的明确错误，SHALL NOT 返回 client；上层探活 SHALL 如实呈现离线

#### Scenario: 未知协议报错（负路径）
- **WHEN** `Protocol` 非 NETCONF/GNMI/AUTO
- **THEN** factory SHALL 返回 "unsupported protocol" 错误，不建连

### Requirement: DP-06 契约缺口（已知未实现/降级）

本层部分能力当前为占位或降级实现，SHALL 记录为已知缺口，SHALL NOT 被上层当作已生效功能依赖：gNMI 为规划能力（client 空壳已删，工厂显式错误，见 DP-02）、NETCONF `Subscribe` 未实现、`CloseAll` 仅返回最后一个错误（吞掉其余）。

#### Scenario: NETCONF Subscribe 未实现（负路径）
- **WHEN** 调用 NETCONF `Subscribe`
- **THEN** SHALL 返回 "subscription not implemented for NETCONF" 错误，不 panic

#### Scenario: CloseAll 吞错（负路径）
- **WHEN** `CloseAll` 关闭多个连接、其中多个 `Close()` 报错
- **THEN** SHALL 关闭全部连接并清空池，仅返回最后一个错误（其余错误被吞、`Errors` 计数递增）
