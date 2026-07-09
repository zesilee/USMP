# device-store delta — snd-vendor-registry

## MODIFIED Requirements

### Requirement: DS-01 共享连接信息注册表

系统 SHALL 提供 Manager 级、进程内存的 `DeviceStore`，键为 DeviceID（裸 IP，与 ConfigStore desired 同键），值为完整 `DeviceConnectionInfo`（含 `Vendor` 厂商标识）。`Manager` 接口 SHALL 暴露 `GetDeviceStore()`。实现 SHALL 无 DB（R03），并发 `Put/Get/Delete/List` SHALL 无数据竞态（R09）。

#### Scenario: Vendor 随连接信息透传
- **WHEN** `Put(id, info)` 时 `info.Vendor="huawei"`，随后 `Get(id)`
- **THEN** 返回的 `DeviceConnectionInfo.Vendor` SHALL 为 `"huawei"`

#### Scenario: Vendor 零值缺省语义（边界）
- **WHEN** 存入的 `DeviceConnectionInfo.Vendor` 为空字符串（存量数据）
- **THEN** 消费方 SHALL 按 `huawei` 缺省解读并 SHALL NOT 失败（R08 降级）

### Requirement: DS-03 设备注册写入库

DeviceHandler SHALL 把种子设备与 `AddDevice`/`RemoveDevice` 写入/删除 DeviceStore（含 `Vendor`，未指定时缺省 `huawei`）；`/api/devices`（含在线探活）与配置回读 SHALL 从库读取。设备连接信息 SHALL NOT 再散落于各处私有结构。

#### Scenario: 种子设备携带缺省厂商
- **WHEN** 后端启动
- **THEN** DeviceStore 中种子设备的 `Vendor` SHALL 为 `huawei`
