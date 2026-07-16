# global-ha-multi-instance — 全局 HA：device store 共享化、全控制器选主、audit 迁出本地文件

## Why

USMP 以 K8s 内多实例（≥2 副本）PaaS 底座组件形态部署（SC-06），但存量三处仍是单实例假设：设备连接信息注册表纯内存（重启即丢、副本间不可见）、5 个原生周期控制器无 leader election（N 副本对同一设备 N 倍 NETCONF Get + drift 时竞态并发 Set）、操作审计写实例本地 JSON 文件（违反 SC-06 禁本地持久）。business-network-config 收官时把意图层做到了多实例就绪，本变更把剩余存量收敛，USMP 才是真 HA。

## What Changes

- **① device store 共享化**：`device.Store` 接口不变，新增 CRD 后端实现——设备连接元信息（IP/Port/Protocol/Timeout/Vendor）持久化为 `Device` CR，凭据（Username/Password）经 `secretRef` 存 K8s Secret；无可达集群时自动降级回现有内存实现（R08）。副本间经 CR watch 同步设备集合；重启后设备列表从 CR 完整恢复。顺手：收敛 9 个 reconciler 各自内联的「未注册→AUTO 兜底」为单一共享 helper（根治 reconcile-conninfo-debt），移除 `DeviceHandler` 构造函数硬编码种子设备 `192.168.1.1 admin/admin`。
- **② 全控制器 leader election**：把 `internal/intent/leader.go` 的 `leaderGatedSource`（非 leader 不启动内部事件源）泛化提升到 `pkg/yang-runtime`，以**单一全局 Lease**（`usmp-native-controllers`，独立于意图面 `usmp-business-intent`）罩住 vlan/ifm/system/bgp/network-instance 全部 5 个原生周期 source；无集群/env 关闭时透传（现行为零变化）。
- **③ audit 迁出本地文件**：`audit.Store` 接口化，新增 CRD 后端——每条审计记录一个 `AuditRecord` CR（label 索引设备/时间），超上限按时间清理；`GET /logs` 契约（筛选/分页/reconcile live-join）不变；无集群降级为纯内存（现行为）。退役 `WithAuditFile` 本地 JSON 路径。
- 渐进替换军规：每条线旧实现保留并行 → 双路径验证 → 切换 → 删除旧路径；三条线分波次独立 PR（TM04 ≤1000 行）。

## Capabilities

### New Capabilities

- `operation-audit`: 操作审计（配置下发/删除记录）的存储与查询契约——CRD 持久化、多实例可见、容量清理、无集群降级、`GET /logs` 查询语义。现状无 spec，借本变更立契约。

### Modified Capabilities

- `device-store`: DS-01「进程内存」升级为「CRD 持久 + 内存降级」；新增凭据 Secret 引用、跨副本可见、重启恢复、无集群降级要求；新增兜底收敛要求（消费方统一经共享解析 helper，不再各自内联 AUTO 兜底）。
- `devices-api`: 新增设备注册跨实例重启存活的可观测行为（集群模式下）。
- `yang-controller-runtime`: 新增周期事件源 leader election 门控要求（多副本仅 leader 产生 reconcile 事件；关闭/无集群透传）。
- `system-architecture`: SC-06 从「意图层就绪、存量另立任务」收敛为「全控制器具备选主、无任何本地持久文件」。

## Impact

- **后端**：`pkg/yang-runtime/device`（新 CRD store）、`pkg/yang-runtime/manager`（audit 装配、选主 option）、`pkg/yang-runtime/audit`（接口化+CRD 后端）、`pkg/yang-runtime/source` 或新 `pkg/yang-runtime/leader`（泛化 gate）、`backend/api/core/v1`（新 Device/AuditRecord 类型）、`internal/intent/leader.go`（改为复用泛化实现）、9 个 `internal/controller/*/reconciler.go` 的 `resolveConn`、`internal/api/{device,config,audit}_handler.go`、`backend/main.go` 装配。
- **部署**：`deploy/crds/` 新增 2 个 CRD manifest、`deploy/rbac/` 扩 Device/AuditRecord/Secret/Lease 权限、README 环境变量表更新。
- **不动**：前端（`GET /logs`、devices API 契约不变）；意图面选主（独立 Lease 不合并）；运行配置缓存（TTL+LRU 语义不变，副本各自缓存本就合规）。
- **风险**：`DeviceConnectionInfo` 含不可序列化 `*tls.Config`，CR spec 需定义可序列化投影；Secret/CR 双资源写入的原子性（先 Secret 后 CR，删除反向）；audit 每条一 CR 的 etcd 对象数量（上限+清理兜底）。
