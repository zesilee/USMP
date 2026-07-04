# retire-actor-usage — design（P3 破 D8 误判，消除 Actor 生产使用）

> change：`retire-actor-usage` | 依赖：`proposal.md`

## Context

延续已归档 `crd-intent-source-stackb`（P2）的遗留：整包退役 Actor 曾被记为「D8 System 翻译阻塞」。复查纠正：BusinessSwitch 是**设备注册/生命周期** CRD（非配置意图），其探活借用 VLAN Actor；`vlan/actor_reconciler.go` 为死代码。故消除 Actor 生产使用无需 System 翻译。

## Goals / Non-Goals

**Goals:**
- 消除 `pkg/yang-runtime/actor` 的一切生产/框架使用 → 满足 R01（禁 Actor）、达成迁移债 D2。
- BusinessSwitch 在线判定行为等价（连接成功=在线，失败=离线降级）。

**Non-Goals:**
- 物理删除 actor 包（pr-size 上限，后续机械批次）。
- BusinessSwitch/Route 迁 Stack B、退 `cmd/controller`（device-registry 收编，后续）。
- System/多厂商/Route ygot 翻译（本变更不需要）。

## Decisions

### D-1 ClientPool 直连探活替换 Actor 探针
- `probeDevice`：构造 `client.DeviceConnectionInfo{IP, Port(默认830), Username, Password}` → `r.ClientPool.Get(info)`。`NewNETCONFClient` 立即连接，失败即 `err`（离线）；成功后 `client.IsConnected()==true`（在线）。
- 移除 `actor.NewReflectTranslator`/`NewModelActor`/`StatusQueryCmd`/`time.Sleep`。
- uptime 明细（原「简化处理」）不再从 Actor 取；如需可后续 `client.Get(ctx, systemPath)`，本次记为差异（不影响在线判定）。

### D-2 删除死代码 actor_reconciler
- `internal/controller/vlan/actor_reconciler.go` 未接入任何入口、无外部引用 → 直接删除。

### D-3 物理包删除暂缓
- `pkg/yang-runtime/actor` 保留物理文件（无生产使用）。pr-size 限：`model_actor.go`(1089)/`device_actor.go`(615) 单文件即超 800，13 文件 ≤20 无法进 3000 档；需按 leaf 顺序分批 gut/删（≥7 PR）。与 `datastore.go`/`yang-schema.ts` 同类，记为机械清理债。

## Risks / Trade-offs

- **探活语义变化**：Actor 探针曾返回 uptime 等明细；ClientPool 探活以「连接成功」为在线判据，丢 uptime 明细。多数场景在线/离线判定足够；明细可后续补。记为可接受差异。
- **actor 包物理残留**：R01 关注「生产在用」，使用清零即实质合规；物理残留是清理债，不再违规（无 live 路径）。其自身测试仍跑（`go test ./...` 绿），随后续批次删。
- **连接副作用**：`pool.Get` 会缓存连接（每设备一条），探活即建立长连——与既有 reconciler 用法一致，ClientPool 负责复用/重连。
