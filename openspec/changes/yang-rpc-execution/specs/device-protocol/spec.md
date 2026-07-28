## ADDED Requirements

### Requirement: DP-10 NETCONF `<rpc>` 自定义操作执行

device-protocol SHALL 提供执行任意模块 rpc 的能力：把 rpc 名与 input 值编码为 NETCONF `<rpc>` payload（命名空间取模块 namespace）下发，解析 `<rpc-reply>` 返回结果（`<ok/>` / 数据 / `<rpc-error>`）。断线重试、超时、连接复用语义 SHALL 与既有读写（DP-03 get-config / edit-config）一致。既有 get/edit-config/commit 行为 SHALL 不变。

#### Scenario: 执行 rpc 并返回 ok

- **WHEN** 以 input（if-name=X）执行 `reset-if-counters-by-name`
- **THEN** device-protocol SHALL 发送含该 input 的 `<rpc>`，并在设备返回 `<ok/>` 时返回成功结果

#### Scenario: 解析 rpc-error

- **WHEN** 设备对 rpc 返回 `<rpc-error>`
- **THEN** device-protocol SHALL 把错误结构解析并返回，SHALL NOT panic（R08）

#### Scenario: 读写路径不受影响

- **WHEN** rpc 执行能力引入后执行常规 get-config / edit-config
- **THEN** 其行为 SHALL 与引入前完全一致
