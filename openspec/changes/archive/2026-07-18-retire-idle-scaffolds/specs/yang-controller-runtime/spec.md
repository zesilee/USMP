## MODIFIED Requirements

### Requirement: YR-07 事件源驱动漂移检测

事件源 SHALL 支持周期轮询 / K8s CRD watch / 文件变更（gNMI 订阅源已随 gNMI 空壳清除移除，协议为规划能力）。`PeriodicSource` SHALL 按提供的设备列表（生产为共享 DeviceStore 的 `List()`，动态取）逐设备入队对账，实现持续 out-of-band 漂移检测；设备列表为空 SHALL 不入队、SHALL NOT panic。

#### Scenario: 周期按库中设备发对账
- **WHEN** 周期 tick 且 DeviceStore 中有 N 个设备
- **THEN** SHALL 为每个设备就配置路径入队一个对账事件

#### Scenario: 空设备列表不空转报错
- **WHEN** 设备列表为空
- **THEN** SHALL 不入队任何事件，SHALL NOT panic
