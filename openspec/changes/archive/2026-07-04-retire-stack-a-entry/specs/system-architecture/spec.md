## MODIFIED Requirements

### Requirement: 单一生产入口（退役 Stack A）

系统 SHALL 以 `backend/main.go` 为**唯一生产入口**（设备原生面 reconciler + CRD 意图源 + 北向 REST API + 设备注册/在线探活）。`cmd/controller`（controller-runtime 独立进程，Stack A 入口）SHALL 退役；其 3 个控制器（BusinessSwitch 探活、BusinessRoute/NativeDeviceConfig 空壳下发）已由 DeviceHandler 探活 + config-api/CRD 意图源承接。

#### Scenario: 单进程承载全部职责
- **WHEN** 启动生产
- **THEN** SHALL 仅需 `backend/main.go`；不再有 `cmd/controller` 进程

#### Scenario: 设备状态由 Stack B 提供
- **WHEN** 查询设备在线状态
- **THEN** SHALL 经 Stack B REST `/api/devices`（不再依赖 BusinessSwitch CR status）
