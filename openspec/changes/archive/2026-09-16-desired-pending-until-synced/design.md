# Design: desired-pending-until-synced

## Context

- desired 存储 = `cache.TTLLRUCache`（TTL 1min，`manager.go:163`）包在 `InMemoryConfigStore` 里，实现 `reconcile.ConfigStore`（Get/Set/Delete/List/ListDevices）。
- 写入方三条：`POST /config`（异步：写 desired → 触发对账）、`/config/changeset/commit`（同步 2PC 推设备成功后才写 desired → 触发复核对账）、意图层 `internal/intent`（5 分钟 resync 重写）。
- 读取方：`GenericReconciler.Reconcile`（对账开头一次性读入本地变量）、changeset preview 合并基底、周期对账。
- controller `process()`：`Requeue/Error` → `AddRateLimited`；`Changes>0` → 复验；否则 `Forget` 并记 `Converged`。desired 读空走的正是最后一支。
- 硬约束：R03（仅内存缓存）、R08（读空绝不删设备配置，`reconcile_desired_absent_test.go` 锁定）、R09（-race 全绿）、Go 1.22。

## Goals / Non-Goals

**Goals:**
- desired 在送达确认前不因时间流逝丢失；送达确认后按既有 TTL 释放（口径 A：送到就撒手）。
- 永远送不到的条目有界（放弃上限）且结局显式（`error` + 「已放弃」），不再假收敛。
- 读空不冒充收敛、不覆盖上一次真实结局。
- `reconcile.ConfigStore` 接口与 `status.Outcome` 枚举零破坏，模块 reconciler 与前端零改动。

**Non-Goals:**
- 已同步后持续盯漂移（口径 B）——漂移纠正归意图层（CRD 落盘 + resync）。
- desired 跨实例共享 / 持久化（多实例各持各的内存，本 change 不改）。
- 改 `POST /config` 为同步下发。
- 新增前端状态展示。

## Decisions

### D1 两态放在缓存原语层，而不是 ConfigStore 自建 map
`TTLLRUCache` 条目增加 `gen uint64`（写代）与 `pending bool`：
- `SetPending(key, v)`：写入，`pending=true`，`gen++`，不计 TTL。
- `MarkExpiring(key, gen) bool`：`gen` 匹配才把 `pending=false`、`createdAt=now`（TTL 从此刻起算）。
- `Track(key) (gen, pendingSince, pending, ok)`：只读查询。
- `Get/GetWithAge/Keys/ClearExpired`：`pending` 条目一律视为未过期。
- 既有 `Set` 语义不变（写入即计 TTL），运行缓存/状态快照不受影响。
- **备选**：`InMemoryConfigStore` 另建 map 记元数据 → 两份状态要对齐同一把锁，反而更易竞态；否决。

### D2 经可选接口接入，不改 `reconcile.ConfigStore`
```go
type SyncTracker interface {
    Track(deviceID, path string) (gen uint64, pendingSince time.Time, pending, ok bool)
    MarkSynced(deviceID, path string, gen uint64) bool   // gen 不匹配返回 false
    Abandon(deviceID, path string, gen uint64) bool      // gen 不匹配返回 false
}
```
`GenericReconciler` 对 `configStore` 做类型断言；不实现则退回旧行为（测试替身与第三方实现零改动）。`InMemoryConfigStore.Set` 改用 `SetPending`。
- **备选**：直接扩 `ConfigStore` 接口 → 破坏所有 fake，且意图层 `WithPush` 传入的也是同一实现，无收益；否决。

### D3 何时标记「已同步」
只在**复验确认** `len(changes)==0` 时 `MarkSynced(gen)`。首轮下发 `Changes>0` 不标记，由 YR-05 复验入队再确认。`gen` 取自本次 `Reconcile` 开头 `Track` 的值：中途用户再写入 → `gen` 变 → 标记失败 → 新值继续待同步。
- changeset commit：设备已成功，写 desired 后触发对账，复核 0 change 即标记；与 POST 路径统一，不加特判。

### D4 放弃判定与终态
`Reconcile` 在 `deviceClient.Get/Set`、`Diff` 任一失败分支，**以及下发成功但 `Changes>0` 的分支**（评审 🟡-2：设备接受下发而回读永不相等时，否则 push→复验 无限循环、每 30s 打一次设备；改前 1 分钟 TTL 至少会停），统一经 `abandonIfOverdue`：若 `Track` 得 `pending && pendingSince 非零 && time.Since(pendingSince) > AbandonAfter` → `Abandon(gen)` 成功则返回 `Result{Terminal:true, Error: ReconcileError{Err: ErrDesiredAbandoned 与原因双 %w 包装}}`。零值 pendingSince 视为未知不放弃（R08，第三方 tracker 兜底）。
快照时机（评审 🟡-1）：`trackLifecycle` 在 `configStore.Get` **之前**——写入落在两者之间时快照 gen 偏旧，MarkSynced/Abandon 都失败关闭（保守）；反过来先 Get 后 Track 会让新值凭旧值的零变更被标已同步（TOCTOU）。
`Result` 新增 `Terminal bool`；controller：`Error!=nil && Terminal` → 记 `error`（消息带「已放弃」+ 等待时长）、`Forget`、不重投。
`AbandonAfter` 默认 30 分钟，进程启动读 `USMP_DESIRED_ABANDON_AFTER`（Go duration），非法值回退默认并 warning。放在 `reconcile` 包级（`SetAbandonAfter` 供测试与 main 注入），6 个模块 reconciler 零改动。
- **备选**：复用 `NumRequeues` 按次数放弃 → 次数与设备慢的程度无关，30 秒封顶的退避下 60 次≈30 分钟，不如时间直观；否决。

### D5 读空不记录
`Reconcile` desired==nil 分支返回 `Result{NoDesired:true}`（新增字段）；controller 对 `NoDesired` 分支：不调 `recorder.Record`，`Forget`。上一次真实结局（converged/error/已放弃）保留，符合「状态 = 最近一次**真实**对账」。`status.Outcome` 不加新值，前端类型 `api.gen.ts` 不漂。
- **备选**：新增 `OutcomeNoDesired` → 前端契约漂移、progress 状态机要改，收益小；否决。

### D6 LRU 与内存边界
待同步条目仍受容量 1000 LRU 淘汰（极端下被淘汰 = 读空 → 不记录），放弃上限是主边界。文档明示。

## Risks / Trade-offs

- [已同步后 1 分钟即释放，周期对账对原生配置依旧空转] → 口径 A 已拍板；YR-07 注记明确「漂移纠正归意图层」。
- [`Terminal` 误用导致该重投的不重投] → 仅放弃分支置位，controller 单测覆盖 Terminal 与非 Terminal 两路。
- [gen 守卫遗漏导致新值被标已同步] → reconcile 单测「Get 与 MarkSynced 之间插入 Set」断言标记失败且条目仍 pending。
- [30 分钟内设备持续离线的条目占内存] → 有界（≤1000 条 × 单模块结构），可接受。
- [依赖 `time.Now` 的用例 flaky] → 缓存/对账用例用 20–50ms 级 TTL 与显式 `time.Sleep`，或注入 `now func()`；-race 弱机 ×5 估上界。

## Migration Plan

- 单 PR，无数据迁移（内存态）。回滚 = revert。
- 默认值保守（30 分钟），环境变量可调；不改任何 API 契约。

## Open Questions

（无——放弃上限 30 分钟与口径 A 已按默认处理，若需调整改环境变量即可。）
