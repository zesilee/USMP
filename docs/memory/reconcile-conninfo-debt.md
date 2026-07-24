---
name: reconcile-conninfo-debt
description: "对账/回读连接信息(协议+凭据)无共享来源的架构债；PR#100最小兜底解封，根治=共享device store"
metadata: 
  node_type: memory
  type: project
  originSessionId: 014f7a79-ea21-4958-af4f-6af2e70396f4
---

**病根**：设备连接信息（Protocol + Username/Password + Port）**没有单一可信来源**，多个子系统各自用残缺的 `DeviceConnectionInfo` 建连，跨子系统传递时逐段丢弃。种子凭据 admin/admin/830 只存在 `internal/api/device_handler.go:35` 的 handler 私有 map，没进连接池、没共享给 reconciler/config_handler。Manager 接口(`pkg/yang-runtime/manager/manager.go:103-115`)无 device store。靠连接池按 IP 缓存(`client/pool.go:56`)偶发命中 → 表现「时好时坏」。

**暴露的两 bug**（2026-07-07 WSL 部署联调，「新增接口只填 name」）：
- A：回读 `config_handler.go:43` fetchFromDevice + `device_handler.go:67` probeOnline 漏传 Protocol → 工厂 default 分支 `unsupported protocol:`(force_refresh 回读 500)。
- B：对账链 `config→manager.TriggerReconcile→predicate.Event→reconcile.Request→ifm/reconciler` 全程只带裸 DeviceID(IP)，无凭据字段 → NETCONF 空 user/pass → SSH 仅 none 认证被模拟器(要 admin/admin,`simulator.go:80-91`)拒。配置只 ACCEPTED 进 store、未真正下发。

**已交付(PR#100, 分支 fix-config-push-conn-info)**：最小兜底解封——两处补 `Protocol: client.ProtocolAUTO`；`netconf.go NewNETCONFClient` 空凭据兜底 admin/admin(带 TODO)。3 个 TDD 回归测试(捕获型 fakePool 断 Protocol、netconf 空凭据兜底/显式保留)。

**后续(PR#101, 分支 fix-drift-status-and-config-list)**：修「下发后一直漂移」+「接口列表看不到」两个终态 bug。根因：(1) controller 下发有变更(Changes>0)后不复验，OutcomeDrifted 永不翻(周期源 deviceIDs=nil 空转,不触发第二轮);(2) fetchFromDevice 回读返裸 XML(base64),前端 extractRows 取不出行。修法：controller.process 在 Changes>0 时 AddRateLimited 复验(不改 reconcile.Result 契约,reconciler 测试零影响);fetchFromDevice→decodeRunningConfig(XML→ygot→EmitJSON RFC7951→map, 形状 {"interface":[{"name":...}]} 对齐前端 listKey)。注意坑:ygot 结构直接 json.Marshal 是大写 Go 字段名(`Interface`/`Name`)前端认不得,必须走 RFC7951 emit。**残留债**:周期源 deviceIDs 仍是 nil(无持续 out-of-band 漂移检测),需 shared device store 提供设备列表。

**根治(未做,应另开 PR)**：Manager 加 `GetDeviceStore()`,把 DeviceHandler.devices 提升为共享注册表;reconciler(ifm/vlan/system)+config_handler+device_handler 都从它按 DeviceID 取全套连接信息(IP+Port+User+Pass+Protocol)。届时删掉 netconf.go 的 admin/admin 兜底。涉及 `reconcile.Request`/`predicate.Event` 可能需加携带连接信息或让 client 层查 store。相关：[[dual-stack-migration]] [[reconcile-convergence-3rootcauses]] [[deploy-wsl-huawei-proxy]]
