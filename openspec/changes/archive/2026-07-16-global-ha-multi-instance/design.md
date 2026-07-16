# design — global-ha-multi-instance

## Context

存量审计结论（2026-07-16 explore）：

- `device.Store`（`pkg/yang-runtime/device/store.go`）= `map[IP]client.DeviceConnectionInfo` + RWMutex，接口仅 Get/Put/Delete/List——替换缝极窄。值类型含 `Password` 明文与不可序列化 `*tls.Config`。消费方 15+ 处：9 个 reconciler 各自内联「未注册→AUTO+空凭据兜底」（reconcile-conninfo-debt 病灶）、intent 2PC（`tx.go:170-181`）、config handler 读/删、5 个周期 source（DeviceLister）、device API 探活。`NewDeviceHandler` 硬编码种子 `192.168.1.1 admin/admin`（`device_handler.go:37-45`）。
- 选主：`intent/leader.go` 用 client-go LeaseLock；`leaderGatedSource` 非 leader 从不启动内部 Source，OnStartedLeading 才 `inner.Start`。仅 intent 一处使用（`register.go:74`）。5 个原生控制器在 `main.go:53-109` 逐个注册 `NewPeriodicSourceWithLister(5min, deviceStore, path)`，无 gate。`GenericReconciler` diff 为空即短路（幂等），多副本危害 = N 倍 Get + drift 竞态 Set。
- audit：`audit.NewStore(path, 1000)` 内存数组 + 每条记录全量重写 `data/audit.json`；写点仅 API 层 2 处（POST 下发 `config_handler.go:253`、DELETE `:1123`）；读取方 `GET /logs`（`audit_handler.go:59-108`，与 reconcile-status live-join）。
- 可复用设施：`crdsource/register.go` 的 controller-runtime cache/client 接入模板 + `ctrlcfg.GetConfig()` 无集群优雅降级；`backend/api/core/v1` scheme/deepcopy 脚手架；`tools/crdgen` + `hack/crd-injector` manifest 管线；`intent/leader.go` 选主参数（Lease 15s/10s/2s，ReleaseOnCancel）。

约束：SC-02（持久元信息仅 CRD，CRD 仅当载体不当架构通道）、SC-06（禁本地持久、控制器须选主接缝）、R08（无集群必须降级不崩溃）、R09（并发安全）、存量改造军规（旧路径并行→双路径验证→切换→删除）、TM04（每 PR ≤1000 行）。

## Goals / Non-Goals

**Goals:**

- 设备连接元信息跨副本共享、跨重启存活；凭据不明文进 etcd（Secret 引用）。
- 消费方建连解析收敛到单一 helper，删除 9+ 处重复兜底（根治 conninfo-debt）。
- 5 个原生周期控制器多副本仅 leader 产生 reconcile 事件。
- 审计记录迁 CRD，多实例可见，退役本地 JSON；`GET /logs` 契约不变。
- 单机/无集群开发体验零回退：所有新路径在无 kubeconfig 时自动降级回现有内存行为。

**Non-Goals:**

- 不做归属硬锁（另任务 ownership-hard-lock）、不退役旧 BusinessVlan 桥接（另任务）。
- 不合并意图面与原生面的 Lease；不改意图控制器选主实现（仅让它复用泛化后的代码）。
- 不动运行配置缓存（副本各自 TTL+LRU 本就合规）；不做设备凭据轮转/加密网关。
- 不实现前端「添加设备」表单（API 已具备、前端未接是既有事实，不在本变更范围）。
- 不复活 `BusinessSwitch`/`NativeDeviceConfig` 遗留 CRD（biz 层概念/无控制器接线，新建 core 层 Device 类型更干净）。

## Decisions

**D1 Device CR 数据模型：可序列化投影 + secretRef，不搬 `DeviceConnectionInfo` 原样。**
`core.usmp.io/v1 Device`：spec = `{managementIP, port, protocol(netconf|gnmi|auto), timeoutSeconds, vendor, credentialsSecretRef}`。`*tls.Config` 不进 CR（现状仅测试用，运行路径未依赖持久 TLS 配置；留 `tlsSkipVerify`/字段扩展给未来）。凭据存同 namespace Secret（`username`/`password` 两 key），CR 只存引用。写入顺序：先 upsert Secret 再 upsert CR；删除反向（先 CR 后 Secret），中途失败以 CR 为准对账（有 CR 无 Secret = 设备存在但凭据缺失，Get 返回空凭据并记日志，R08 不崩）。备选「凭据放 CR 明文」被否：etcd 明文 + RBAC 无法单独收紧。

**D2 crdStore 形态：watch 驱动的本地镜像缓存实现 `device.Store` 接口，读零 RTT。**
新 `device.NewCRDStore(cache, client, secretReader)`：内部仍是 map+RWMutex 镜像，由 controller-runtime cache watch Device CR 增量维护（Secret 凭据随 Get 惰性解析+缓存失效跟随 watch）；`Put/Delete` 写穿 CR+Secret（同步阻塞，失败向调用方返回错误——devices-api BR-06「连接失败仍保存」语义保持，但「apiserver 不可达」是新的可失败点，返回 5xx 信封）。`Get/List` 只读镜像——周期 source 每 5min List 与 reconciler 高频 Get 不打 apiserver。备选「每次 Get 直读 apiserver」被否：reconcile 热路径引入网络 RTT 与 apiserver 负载。接口不变 ⇒ 15+ 消费方零改动。

**D3 装配与降级：main.go 按 `ctrlcfg.GetConfig()` 成败选 store，注入 Manager。**
`manager.New` 增加 `WithDeviceStore(device.Store)` option（缺省仍 `device.NewStore()` 内存版）。有集群 → crdStore；无集群 → 内存版 + 日志提示（复用 intent/crdsource 的降级模式）。种子设备从 handler 构造函数移除，改为可选 env `USMP_SEED_DEVICE`（仅内存降级模式生效，供本地开发/E2E；集群模式设备来自 CR）。E2E/staging compose 起的是单进程无集群模式，行为经 env 保持。

**D4 兜底收敛：`device` 包提供 `ResolveConn(store, ip) (info, registered)` 单一 helper。**
语义与现状一致（未注册 → `{IP, Protocol: AUTO}` + 空凭据 + 统一日志），9 个 reconciler、intent tx、config handler、device API 全部改调它，删除各自内联副本。这是纯收敛重构（行为不变），放在波次①，回归靠现有 B1/B2 测试面。

**D5 选主泛化：`leaderGatedSource` 提升为 `pkg/yang-runtime/leader.GateSources(cfg, name, sources...)`，单全局 Lease。**
从 intent 包提取（Lease 参数照搬：15s/10s/2s、ReleaseOnCancel、hostname identity），参数化 Lease name。原生面用 `usmp-native-controllers` 一把锁罩 5 个周期 source——它们同进程同生死，细分收益存疑（备选「每控制器一 Lease」被否：对象膨胀、观测复杂）。开关 `USMP_NATIVE_LEADER_ELECTION`（缺省关=现行为透传；与意图面 `USMP_INTENT_LEADER_ELECTION` 独立）。intent 包改为复用泛化实现，删除本地副本（行为等价，回归靠 leader_test.go 迁移）。非 leader 副本 API 读路径不受影响：config 读走缓存/直连设备，不依赖周期 source。

**D6 audit CRD：每条记录一个 `AuditRecord` CR，控制器侧按上限清理。**
`core.usmp.io/v1 AuditRecord`：spec 承载 Record 字段（deviceIP/path/summary/triggered/actor/timestamp），name = `audit-<unix-nano>-<rand4>`（避免 Date 冲突），label `usmp.io/device-ip` 供筛选。`audit.Store` 抽成接口（`Record/List/Flush`），现内存+文件实现改名保留为降级路径（去掉文件写 = 纯内存），新增 CRD 实现：写 = create CR（异步 fire-and-forget + 失败日志，与现状「写失败只 log 不阻断」一致）；读 = 镜像缓存 List（同 D2 watch 模式），排序分页在内存做——1000 条量级无压力。超上限清理：写入方 create 后检查镜像条数，超 1000 删最旧（leader 不需要——写点在 API 层任意副本，清理幂等且删 IsNotFound 容忍）。`GET /logs` handler 只依赖 `List()`，契约零变化。备选「单 CR 环形缓冲」被否：每条记录全量重写整个对象，写放大与 1.5MB 上限双输；「K8s Events」被否：1h TTL 丢历史，审计语义不合格。

**D7 CRD manifest 与 RBAC 走现有管线。**
`deploy/crds/` 新增 device/auditrecord manifest（手写或 crdgen 视波次①实操定，Device 非 YANG 源生成物，倾向 kubebuilder 注释 + controller-gen 手写 manifest 对齐现有风格）；`deploy/rbac/` 扩：Device/AuditRecord CRUD、Secret get/create/update/delete（限定 label 选择或专用 namespace 内）、Lease（已有 intent 条目，加 `usmp-native-controllers`）。安装顺序沿用 BIC-02（CRD 先于应用滚动）。

**D8 波次切分（每波次一个 PR，≤1000 行）：**

| 波次 | 内容 | 依赖 |
|------|------|------|
| W1 | D4 兜底收敛 + 种子设备迁 env（纯重构，行为不变） | 无 |
| W2 | Device CRD + crdStore + 装配降级（D1-D3）+ deploy 物料 | W1 |
| W3 | 选主泛化 + 原生面 gate + intent 复用（D5） | 无（与 W2 并行可，但串行提交避免 main.go 冲突） |
| W4 | audit 接口化 + CRD 后端 + 退役 WithAuditFile（D6） | 无（同上） |

## Risks / Trade-offs

- [Secret/CR 双写非原子] → 写序固定（先 Secret 后 CR）+ 以 CR 为存在性权威 + 凭据缺失降级空凭据（clean fail，与 netconf 层现状一致）；Put 返回错误让 API 呈现 5xx。
- [watch 镜像有陈旧窗口] → 设备增删本就低频；写穿后本地镜像同步先行更新（write-through + watch 兜底收敛），避免「刚 Put 立刻 Get 不到」。
- [非 leader 副本缓存冷] → 只影响周期对账链路（本就仅 leader 跑）；API 读路径独立建连不受影响。leader 切换后首轮 5min tick 内完成全量对账，可接受。
- [audit create CR 每操作一次 apiserver 写] → 用户操作驱动、量级低；失败不阻断主流程（与现状一致）。清理并发竞态 → 删除容忍 IsNotFound，多删无害（下限不破坏正确性）。
- [E2E/本地无集群行为漂移] → 所有新路径默认降级回现行为 + `USMP_SEED_DEVICE` 保种子；staging smoke 全绿作为双路径验证的一半，envtest 集成测试作为另一半。
- [W2 体量逼近 1000 行] → CRD types+deepcopy 为生成物可在 PR 体检中豁免口径（沿用 R04 生成物门禁先例）；仍超则 deploy 物料拆小尾巴 PR。

## Migration Plan

1. W1（无部署影响）→ W2：先 `kubectl apply` 新 CRD/RBAC 再滚动镜像（BIC-02 顺序）；回滚 = 回滚镜像即可（内存路径仍在，CR 留存无害）。
2. W3：滚动后设 `USMP_NATIVE_LEADER_ELECTION=1` 观察单 leader 日志；异常回滚 = 置 0（回到全副本轮询，行为退化但正确）。
3. W4：滚动后 `USMP_AUDIT_FILE` 弃用（保留一版兼容期：设了就打警告日志走内存降级，不再写文件）；确认 `GET /logs` 正常后删除文件路径代码。

## Open Questions

- Device CR 的 namespace 是否复用 `USMP_INTENT_NAMESPACE` 还是独立 env？（倾向复用，W2 实操定）
- crdgen 是否值得为非 YANG 源 CRD 扩展，或直接 controller-gen？（W2 首任务 spike 半天定）
