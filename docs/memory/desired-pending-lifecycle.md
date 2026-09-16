---
name: desired-pending-lifecycle
description: desired 存储两态生命周期（送达确认前不过期/确认后 1min 释放/30min 放弃/读空不记录）——改对账、ConfigStore、TTL、状态结局前必读
metadata:
  type: project
---

# desired 生命周期：送达确认才起算过期（change desired-pending-until-synced，2026-09-16）

**根因（已修）**：desired 存储原为写入后 1min 硬过期，对账重试链（1/2/4/8/16/30s 退避 + 60s 单次超时）轻易跨过；设备慢时第二三次重试读空 → `GenericReconciler` 按无事可做返回 → controller 记 **Converged 并 Forget**。用户改动静默丢失、界面显示已收敛、审计写着「已触发对账」。周期对账（5min）对原生配置因此长期空转。

**现行契约（YR-09 / CC-08）**：
- `InMemoryConfigStore.Set` = 待同步写入（`cache.SetPending`，不计 TTL）；每次写入递增写代 `gen`。
- `GenericReconciler` 开头 `Track` 取 `(gen, pendingSince, pending)` 快照；**仅零变更（含 YR-05 复验）时 `MarkSynced(gen)`**，此后按存储 TTL（1min）释放——口径 A「送到就撒手」，已同步后不盯漂移，漂移纠正归意图层（CRD + 5min resync）。
- 回读/diff/下发任一失败且 `pending` 超过放弃上限（默认 30min，`USMP_DESIRED_ABANDON_AFTER`，非法回退+warning）→ `Abandon(gen)` + `Result.Terminal` error（含「已放弃」）→ controller 记 error 后 **Forget 不重投**。
- desired 读空 → `Result.NoDesired` → controller **不记录任何结局**（不冒充收敛、不覆盖上次真实结局），只 Forget。`status.Outcome` 枚举未加值，前端零改动。
- 接入方式：`reconcile.SyncTracker` **可选接口**（类型断言），`reconcile.ConfigStore` 未改；不实现者行为与改前一致（替身零改动）。

**Why**：由投递结果而非墙上时钟决定放不放弃；写代守卫防「对账读旧值→用户改新值→旧值标已同步→新值随之过期」。

**How to apply**：
- 改 `Reconcile` 失败分支时保持三处失败都走 `fail()`（放弃判定单点）；`MarkSynced` 只对 pending 条目调用，别在周期收敛读里续期已同步条目。
- 新的 ConfigStore 实现若要享受生命周期，必须实现 `SyncTracker` 三方法且 gen 语义与 `TTLLRUCache` 一致。
- 时序用例用 20–50ms TTL + 显式 sleep；`SetAbandonAfter(0)` 重置为 env/默认，用 defer 复位。
- 攒批提交（前端唯一写路径）先同步 2PC 推设备再写 desired，本就不踩坑；踩坑的是直连 `POST /config` 与周期漂移纠正。
- 已知边界：待同步条目仍受 LRU 容量 1000 淘汰（淘汰=读空=不记录）；多实例各持各的内存 desired，本 change 不改。

相关：[[config-delete-semantics]]（读空绝不翻译成删设备配置）、[[frontend-landing-risklog]]（原「desired 1min 过期」条目已收口）。
