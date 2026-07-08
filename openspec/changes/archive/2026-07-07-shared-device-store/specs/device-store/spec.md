## ADDED Requirements

### Requirement: 共享设备连接信息注册表

系统 SHALL 提供一个 Manager 级、进程内存的 `DeviceStore`，作为设备连接信息（IP、Port、Username、Password、Protocol、Timeout）的**单一可信来源**，键为 DeviceID（裸 IP，与 ConfigStore desired 同键）。`Manager` 接口 SHALL 暴露 `GetDeviceStore()`。DeviceStore SHALL 为内存实现（R03：无 DB），并发读写 SHALL 安全（R09）。

#### Scenario: 注册后可解析完整连接信息
- **WHEN** 向 DeviceStore `Put(id, info)` 注册设备后 `Get(id)`
- **THEN** SHALL 返回完整 `DeviceConnectionInfo`（含凭据与协议），`ok=true`

#### Scenario: 未注册设备
- **WHEN** `Get` 一个未注册的 DeviceID
- **THEN** SHALL 返回 `ok=false`（调用方据此降级，SHALL NOT panic，R08）

#### Scenario: 并发读写安全
- **WHEN** 多协程并发 `Put`/`Get`/`Delete`/`List`
- **THEN** SHALL 无数据竞态（`-race` 通过，R09）
