# device-protocol delta — netconf-get-state-read

## ADDED Requirements

### Requirement: DP-09 NETCONF `<get>` 状态读

NETCONF 客户端 `Get` SHALL 支持 `WithStateData()` GetOption：置位时 SHALL 发送 `<get>` RPC（携与 get-config 相同规则构造的 subtree filter），返回设备的配置+状态合并数据；缺省（未置位）SHALL 保持 DP-03 行为（`<get-config>` running）。`<get>` 读路径的断线自愈语义 SHALL 与 DP-05 一致：传输层错误 SHALL 标记连接失效、重连并重试一次（`<get>` 幂等，重试安全）。

#### Scenario: WithStateData 发 get RPC
- **WHEN** 调用方以 `client.WithStateData()` 调用 `Get(ctx, path)`
- **THEN** 客户端 SHALL 发送 `<get>` 携 subtree filter，返回含 config=false 状态子树的 XML 数据

#### Scenario: 缺省仍走 get-config
- **WHEN** 调用方不带 `WithStateData()` 调用 `Get(ctx, path)`
- **THEN** 客户端 SHALL 发送 `<get-config source=running>`，行为与 DP-03 完全一致

#### Scenario: get 读断线自愈
- **WHEN** `<get>` 请求遇传输层错误（EOF/连接重置）
- **THEN** 客户端 SHALL 标记连接失效、重连并重试一次；重试仍失败 SHALL 返回错误且不 panic（R08）
