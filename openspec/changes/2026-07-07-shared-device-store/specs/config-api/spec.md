## MODIFIED Requirements

### Requirement: 设备注册以共享 DeviceStore 为单一来源

DeviceHandler SHALL 把种子设备与 `AddDevice` 写入共享 `DeviceStore`（替代私有 `devices` map）；`/api/devices`（含在线探活）SHALL 从库读取。设备连接信息 SHALL NOT 再散落于各处私有结构。

#### Scenario: 种子设备进库
- **WHEN** 后端启动
- **THEN** DeviceStore SHALL 含种子设备 `192.168.1.1`，字段完整（Port=830、admin/admin、Protocol=AUTO）

#### Scenario: 新增设备进库
- **WHEN** POST `/api/devices` 注册新设备
- **THEN** SHALL 写入 DeviceStore，后续对账/回读 SHALL 能从库解析其连接信息

### Requirement: 配置回读从 DeviceStore 解析连接信息

`fetchFromDevice` SHALL 用 DeviceID 查 DeviceStore 取 Port/凭据/Protocol 建连，替代仅传 `{IP, Protocol:AUTO}`。库未命中 SHALL 降级（AUTO 兜底并记 warning，SHALL NOT 崩，R08）。

#### Scenario: 已注册设备回读带凭据
- **WHEN** 回读一个已注册设备的运行配置
- **THEN** 建连 `DeviceConnectionInfo` SHALL 含库中的 Port 与凭据（不再依赖连接池按 IP 缓存侥幸命中）
