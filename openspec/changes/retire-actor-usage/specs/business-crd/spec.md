## MODIFIED Requirements

### Requirement: BusinessSwitch 设备探活去 Actor

BusinessSwitch（设备注册/生命周期 CRD）的在线探活 SHALL 经 `ClientPool` 直连（`NewNETCONFClient` 立即连接语义）判定在线/离线，替换此前借用 VLAN `ModelActor` + `StatusQueryCmd` 的机制。CRD 用户接口不变。

#### Scenario: 设备可达 → 在线
- **WHEN** BusinessSwitch 指向的设备可连接
- **THEN** `ClientPool.Get` 成功、`IsConnected()` 为真，Status.OnlineStatus SHALL 为 Online

#### Scenario: 设备不可达 → 离线降级
- **WHEN** 连接失败
- **THEN** SHALL 降级：返回错误由 `handleProbeError` 置 Offline + 重试退避（R08），SHALL NOT panic
