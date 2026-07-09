# device-store — 共享设备连接信息注册表

## Purpose

device-store 是设备连接信息（IP/Port/Username/Password/Protocol/Timeout/Vendor）的**单一可信来源**，供 reconciler、config 回读、周期源、SND 驱动选择统一按 DeviceID 解析建连参数与设备列表。内存实现（R03：无 DB），并发安全（R09）。取代此前散落各子系统的残缺连接信息与 `admin/admin` 兜底、`user:pass@ip:port` 字符串解析。

## Requirements

### Requirement: DS-01 共享连接信息注册表

系统 SHALL 提供 Manager 级、进程内存的 `DeviceStore`，键为 DeviceID（裸 IP，与 ConfigStore desired 同键），值为完整 `DeviceConnectionInfo`（含 `Vendor` 厂商标识）。`Manager` 接口 SHALL 暴露 `GetDeviceStore()`。实现 SHALL 无 DB（R03），并发 `Put/Get/Delete/List` SHALL 无数据竞态（R09）。

#### Scenario: 注册后解析完整连接信息
- **WHEN** `Put(id, info)` 后 `Get(id)`
- **THEN** SHALL 返回完整 `DeviceConnectionInfo`（含凭据与协议），`ok=true`

#### Scenario: Vendor 随连接信息透传
- **WHEN** `Put(id, info)` 时 `info.Vendor="huawei"`，随后 `Get(id)`
- **THEN** 返回的 `DeviceConnectionInfo.Vendor` SHALL 为 `"huawei"`

#### Scenario: Vendor 零值缺省语义（边界）
- **WHEN** 存入的 `DeviceConnectionInfo.Vendor` 为空字符串（存量数据）
- **THEN** 消费方 SHALL 按 `huawei` 缺省解读并 SHALL NOT 失败（R08 降级）

#### Scenario: 未注册设备
- **WHEN** `Get` 一个未注册 DeviceID
- **THEN** SHALL 返回 `ok=false`（调用方据此降级，SHALL NOT panic，R08）

#### Scenario: 并发读写安全
- **WHEN** 多协程并发 `Put`/`Get`/`Delete`/`List`
- **THEN** SHALL 无数据竞态（`-race` 通过，R09）

### Requirement: DS-02 设备列表驱动周期对账

`DeviceStore.List()` SHALL 返回当前已注册的全部 DeviceID，供周期源动态取用逐设备触发对账（持续 out-of-band 漂移检测）。新增/删除设备 SHALL 即时反映在 `List()`，无需重启。

#### Scenario: 列出已注册设备
- **WHEN** 已注册 N 个设备后调用 `List()`
- **THEN** SHALL 返回该 N 个 DeviceID

#### Scenario: 删除后不再列出
- **WHEN** `Delete(id)` 后 `List()`
- **THEN** SHALL NOT 含该 id

### Requirement: DS-03 设备注册写入库

DeviceHandler SHALL 把种子设备与 `AddDevice`/`RemoveDevice` 写入/删除 DeviceStore（含 `Vendor`，未指定时缺省 `huawei`）；`/api/devices`（含在线探活）与配置回读 SHALL 从库读取。设备连接信息 SHALL NOT 再散落于各处私有结构。

#### Scenario: 种子设备进库
- **WHEN** 后端启动
- **THEN** DeviceStore SHALL 含种子设备（字段完整：Port=830、admin/admin、Protocol=AUTO）

#### Scenario: 种子设备携带缺省厂商
- **WHEN** 后端启动
- **THEN** DeviceStore 中种子设备的 `Vendor` SHALL 为 `huawei`
