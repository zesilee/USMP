## Why

`新增接口` 一条链路连修三层（#100 连接层、#101 终态层），根因始终是**同一个**：设备连接信息（IP + Port + Username + Password + Protocol）**没有单一可信来源**。每个子系统各自拼一个残缺的 `DeviceConnectionInfo`，跨子系统传递时逐段丢字段：

- **reconciler** 只拿到裸 DeviceID（IP），无凭据 → SSH 仅 `none` 认证被拒（#100 用 `netconf.go` 兜底 admin/admin **临时绕过**，是 hack）。
- **config_handler 回读** 建连不传 Protocol/Port/凭据（#100 补了 Protocol，仍缺 Port/凭据，靠连接池按 IP 缓存侥幸命中）。
- **周期源** `deviceIDs=nil` → 空转，**没有持续 out-of-band 漂移检测**（#101 靠 controller 下发后复验兜住"新增后收敛"，但周期漂移检测仍缺）。

种子设备（`192.168.1.1` admin/admin/830）只存在 `DeviceHandler` 的私有 map，未共享；`Manager` 接口无任何 device registry。本提案建**共享 device store** 一次根治，并删掉 #100/#101 的两处兜底。

## What Changes

- **新增共享 DeviceStore**：内存注册表（R03：非 DB），键=DeviceID，值=完整连接信息（IP/Port/Username/Password/Protocol）。挂到 `Manager` 接口 `GetDeviceStore()`。
- **DeviceHandler 改写共享库**：种子设备与 `AddDevice` 写入 DeviceStore（替代私有 `devices` map）；`/api/devices` 从库读取。
- **reconciler 从库解析连接信息**：`deviceClient.Get/Set` 用 `req.DeviceID` 查库拿全套连接信息建连，**删除 `user:pass@ip:port` 字符串解析**（那只有集成测试用，掩盖了生产缺凭据的缺陷）。ifm/vlan/system 同构。
- **config_handler 回读从库解析**：`fetchFromDevice` 查库拿 Port/凭据/Protocol（替代只传 IP）。
- **周期源用库里的设备列表**：`main.go` 的 `NewPeriodicSource` 从 DeviceStore 取 deviceIDs → 真正的持续漂移检测。
- **删除两处兜底**：`netconf.go` 的 `admin/admin` 空凭据兜底（#100）；连接信息补齐后不再需要。
- **BREAKING（内部）**：`deviceClient` 不再接受 `user:pass@ip:port` 形式的 DeviceID；集成测试改为向 DeviceStore 注册设备后用纯 DeviceID 触发。

## Capabilities

### Added Capabilities
- `device-store`：Manager 级共享设备连接信息注册表（内存，R03）；单一来源供 reconciler / config-api / 周期源解析建连参数。

### Modified Capabilities
- `config-api`：DeviceHandler 读写共享 DeviceStore；`fetchFromDevice` 从库解析连接信息。
- `yang-controller-runtime`：reconciler 与周期源从 DeviceStore 解析连接信息 / 设备列表；删除裸-IP 字符串解析与凭据兜底。

## Impact

- **后端**：新增 `pkg/yang-runtime/device`（或同层）DeviceStore + 接口；`manager` 接口+实现加 `GetDeviceStore()`；`internal/api/device_handler.go`（写库）、`config_handler.go`（回读查库）；`internal/controller/{ifm,vlan,system}/reconciler.go`（deviceClient 查库、删字符串解析）；`main.go`（周期源喂库里设备）；`pkg/yang-runtime/client/netconf.go`（删 admin/admin 兜底）。
- **测试**：DeviceStore 单测（增删查/并发安全）；reconciler 从库解析凭据建连；周期源发出库中设备的事件；config 回读带凭据；调整/删除 #100 的兜底测试与集成测试的 `user:pass@ip:port` 用法。
- **红线**：R01（Stack B 权威）、R03（内存 store 无 DB）、R06（TDD）、R08（库中无该设备→降级明确报错，不崩）、R09（store 并发安全）。
- **迁移策略（§5.3 渐进）**：① 建 DeviceStore + DeviceHandler 双写（新路径，旧 map 并存）→ 验证 → ② reconciler/config/周期源切到读库（双路径验证）→ ③ 删裸-IP 解析 + admin/admin 兜底 + 旧私有 map（切换完成）。
- **不在范围**：真机 gNMI（模拟器仍纯 NETCONF，D3/D5 空转另计）；设备信息持久化到本地 JSON（§8 元信息，可作后续增量）。
