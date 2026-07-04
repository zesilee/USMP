## Why

架构优化 P2 退役了 BusinessVlan/Interface 的 Actor 下发路径，但 `pkg/yang-runtime/actor`（4709 行、违反 R01）仍有两处生产使用，被误判为「D8 System 翻译阻塞」。复查发现：**BusinessSwitch 不是配置意图 CRD，而是设备注册/生命周期 CRD**（Spec=IP/厂商/凭据/端口），其 `probeDevice` 只是**借用一个 VLAN Actor 做连接探活**（`StatusQueryCmd`），并不翻译 System 配置——无需 System ygot 翻译。另一处 `internal/controller/vlan/actor_reconciler.go` 未接入任何 main.go、无外部引用，是**死代码**。因此消除 Actor 的生产使用（满足 R01、实质完成迁移债 D2）只需：用 ClientPool 直接探活替换 Switch 的 Actor 探针，并删除死的 actor_reconciler。

## What Changes

- **BusinessSwitch 探活去 Actor**：`probeDevice` 改用 `ClientPool.Get(connInfo)`（NETCONF 立即连接，失败即离线）+ `IsConnected()` 判定在线，替换借用的 `ModelActor`+`StatusQueryCmd`。行为等价（连接成功=在线）；原「简化处理」的 uptime 明细不再从 Actor 取（可后续经 `Client.Get` 补，记为差异）。
- **删除死代码** `internal/controller/vlan/actor_reconciler.go`（未接线、无引用）。
- **结果**：`pkg/yang-runtime/actor` 无任何生产/框架使用（仅其自身测试），R01 违规实质消除、D2 达成。
- **BREAKING（内部）无对外契约变化**：BusinessSwitch CRD 接口不变；在线判定机制替换。
- **物理删除 `pkg/yang-runtime/actor` 暂缓**：受 pr-size 上限约束（`model_actor.go` 1089 行 > 800 单 PR 限，13 文件 ≤20 无法进 3000 档），与 `datastore.go`/`yang-schema.ts` 同类，留作机械清理批次（后续）。

## Capabilities

### Modified Capabilities
- `business-crd`: BusinessSwitch 设备探活机制从「借用 Actor」改为「ClientPool 直连探活」；在线状态判定语义不变。
- `actor-transaction`: Actor 子系统**生产使用清零**（R01 实质满足、D2 达成）；物理包删除暂缓（pr-size）。

## Impact

- **后端**：`controllers/businessswitch_controller.go`（去 Actor 探针）、删 `internal/controller/vlan/actor_reconciler.go`；`pkg/yang-runtime/actor` 物理保留但无生产使用。
- **测试**：Switch 探活单测（ClientPool 桩：连接成功=在线、失败=离线，R08 降级）；全量 `go test ./...` 绿（actor 自身测试仍在，随后续批次删）。
- **红线**：R01（Actor 不再生产在用）、R08（探活失败降级为离线）、R06（TDD）。
- **不在范围**：物理删 actor 包（pr-size 批次，后续）；BusinessSwitch/Route 迁 Stack B 与退 `cmd/controller`（device-registry 收编，后续）；gNMI/plugin/多厂商翻译（P3 其余）。
