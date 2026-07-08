# shared-device-store — design（共享设备连接信息注册表）

## Context

设备连接信息在 `config → manager → predicate → reconcile → client` 链路中被逐段丢弃，导致 #100（缺协议/凭据）、#101（周期源无设备）两批症状。现状：

- 凭据唯一存放处：`internal/api/device_handler.go` 的私有 `devices map`（种子 `192.168.1.1` admin/admin/830），未导出、不共享。
- `Manager` 接口（`pkg/yang-runtime/manager/manager.go`）暴露 ClientPool/ConfigStore/RunningCache/ReconcileStatus/AuditStore/PluginManager/Schema，**无 device registry**。
- `reconcile.Request` 只带 `DeviceID string`；`deviceClient` 用字符串解析 `ip` / `ip:port` / `user:pass@ip:port` 三种形式补连接信息，生产只传裸 IP → 凭据空。
- 连接池按 IP 缓存 client，掩盖了缺参数问题（时好时坏）。

## Goals / Non-Goals

**Goals**
- 单一可信来源：DeviceID → 完整 `client.DeviceConnectionInfo`。
- reconciler / config 回读 / 周期源都从该源解析建连参数与设备列表。
- 删除 #100 的 `admin/admin` 兜底与 `user:pass@ip:port` 字符串解析。
- 内存实现，协程安全，无 DB（R03/R09）。

**Non-Goals**
- 不做设备信息持久化（§8 本地 JSON 元信息，留后续增量）。
- 不引入真机 gNMI（模拟器仍纯 NETCONF）。
- 不改 `reconcile.Request` 结构（仍只带 DeviceID；连接信息由库解析，避免污染事件层）。

## Decisions

### D-1 DeviceStore 形状与 Manager 接线
- 新增接口 `DeviceStore`：`Get(id) (DeviceConnectionInfo, bool)` / `Put(id, info)` / `Delete(id)` / `List() []string`，`sync.RWMutex` 保护（R09）。
- 值复用 `client.DeviceConnectionInfo`（已含 IP/Port/Username/Password/Protocol/Timeout），避免另造类型。
- `Manager` 接口加 `GetDeviceStore() DeviceStore`；`Manager` 实现持有一个默认内存 store。
- 键 = DeviceID = 裸 IP（与 `ConfigStore` 的 desired 键一致，保证 desired 与连接信息同键）。

### D-2 reconciler / config 从库解析连接信息
- `deviceClient`（ifm/vlan/system）持有 `DeviceStore`（经 reconciler 构造注入）；`Get/Set` 用 `req.DeviceID` 查库拿 `DeviceConnectionInfo` 再 `pool.Get(info)`。
- **删除** `reconciler.go` 的 `lastAt/splitHostPort` 字符串解析分支（`user:pass@ip:port`）——那是测试专用、掩盖缺陷。查库未命中 → 降级：用 `{IP:id, Protocol:AUTO}` 兜底一次并记 warning（R08，避免直接崩），但不再硬编码凭据。
- `config_handler.fetchFromDevice` 同样查库取 Port/凭据/Protocol。

### D-3 周期源设备列表来自库
- `main.go` 构造 `NewPeriodicSource(interval, mgr.GetDeviceStore().List(), path)`；或让周期源持有 store 引用、每 tick 动态取 `List()`（更实时，推荐——新增设备无需重启）。取 D-3b 动态方案。

### D-4 删除凭据兜底
- 连接信息补齐后，删 `netconf.go` 的 `if info.Username=="" { admin }` 兜底（#100）及其测试；改由 DeviceStore 提供真实凭据。
- 保留 `NewNETCONFClient` 的 Port/Timeout 零值兜底（与凭据无关）。

## Risks / Trade-offs

- **集成测试重构**：现有 ifm/vlan/system 集成测试用 `user:pass@ip:port` DeviceID；需改为「先 `DeviceStore.Put(id, info)` 再用纯 id 触发」。工作量集中但机械。
- **连接池按 IP 缓存**：库解析后 info 更完整，但池仍按 IP 缓存——不同凭据同 IP 不支持（当前无此场景，记为约束）。
- **未命中降级**：库里没有的 DeviceID → 用 AUTO 兜底一次（保持 R08 不崩），但会认证失败并记 error；这是"设备未注册"的正确表现，非回归。
- **迁移期双写**：阶段①旧 map 与新 store 并存双写，避免一次性重写（§5.3）；切换完成后删旧 map。
- **持久化缺位**：进程重启后仅剩种子设备（AddDevice 的动态设备丢失）——与现状一致（现状也在内存 map），持久化留后续增量，不在本次退化。
