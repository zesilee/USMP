## ADDED Requirements

### Requirement: 设备在线状态经 REST 暴露

Stack B 的 `/api/devices`（DeviceHandler）SHALL 对每个设备返回在线状态，经 `ClientPool` 直连探活（`Get` + `IsConnected`）判定，替代 Stack A BusinessSwitch 控制器的 CR-status 探活。探活失败 SHALL 降级为离线（R08），SHALL NOT panic。

#### Scenario: 列设备含在线状态
- **WHEN** GET `/api/devices`
- **THEN** 每个设备 SHALL 含 `online` 布尔字段，由 `ClientPool.Get`+`IsConnected` 判定

#### Scenario: 设备不可达 → 离线
- **WHEN** 某设备连接失败
- **THEN** 该设备 `online` SHALL 为 false，其余设备与响应不受影响（R08）
