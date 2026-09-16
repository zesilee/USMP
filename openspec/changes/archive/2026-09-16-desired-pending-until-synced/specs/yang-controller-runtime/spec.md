## MODIFIED Requirements

### Requirement: YR-02 diff-then-push（不删除）

`GenericReconciler.Reconcile` SHALL 反射 diff 出 `[]Change`，经 `DeviceClient.Set` 下发（edit-config + commit）。desired 为 nil 时 SHALL 视为 no-op，SHALL NOT 删除设备已有配置；该 no-op SHALL 以 `Result.NoDesired=true` 返回，controller SHALL NOT 为其记录任何结局（SHALL NOT 记 `Converged`，SHALL NOT 覆盖上一次真实对账结局），仅 `Forget` 退避计数。

#### Scenario: 检测漂移并下发
- **WHEN** desired 与回读的 actual 有差异
- **THEN** SHALL 产出 Change 并 edit-config+commit 到设备

#### Scenario: 期望态为空不删除
- **WHEN** desired 为 nil
- **THEN** SHALL no-op（Changes=0），SHALL NOT 下发删除

#### Scenario: 期望态为空不冒充收敛
- **WHEN** 上一次结局为 `error`（或从未对账），随后 desired 读空触发对账
- **THEN** 状态 SHALL 保持 `error`（或 `unknown`），SHALL NOT 变为 `converged`；退避计数 SHALL 被 `Forget`

### Requirement: YR-04 失败重投带退避

`process` 处理 Result：error 或 Requeue SHALL `AddRateLimited`（指数退避 + 令牌桶）或按 `RequeueAfter` `AddAfter`；收敛成功 SHALL `Forget`。`Result.Terminal=true` 的 error SHALL 记 `error` 结局后 `Forget`，SHALL NOT 重投。

#### Scenario: 失败退避重投
- **WHEN** Reconcile 返回 error
- **THEN** SHALL 以指数退避重新入队

#### Scenario: 终态错误不重投
- **WHEN** Reconcile 返回 `Terminal=true` 且带 error
- **THEN** SHALL 记 `error`（含原因），SHALL `Forget`，队列 SHALL NOT 再出现该 Request

## ADDED Requirements

### Requirement: YR-09 desired 生命周期：送达确认起算过期、超限放弃

`InMemoryConfigStore.Set` SHALL 以待同步态写入 desired（不计 TTL）。`GenericReconciler` SHALL 在 `Reconcile` 开头、**读取 desired 值之前**取写代 `gen` 快照（写入落在快照与读值之间时快照偏旧，标记/放弃 SHALL 一律失败关闭）；仅当 diff 结果为零变更（含 YR-05 复验）时 SHALL 调用 `MarkSynced(gen)`，成功后该条目自此刻起按存储 TTL 过期；写代不匹配时 SHALL 保持待同步。条目待同步时长超过放弃上限（默认 30 分钟，SHALL 可经 `USMP_DESIRED_ABANDON_AFTER` 配置，非法值回退默认）时，无论本轮是**对账失败**（回读/diff/下发任一）还是**下发成功但仍有变更**（设备接受下发而回读不收敛），SHALL 删除该 desired、返回 `Terminal=true` 的 error（消息含「已放弃」与等待时长），SHALL NOT 静默变为收敛；待同步起点未知（零值）时 SHALL NOT 放弃。`ConfigStore` 不实现 `SyncTracker` 时 SHALL 退回既有行为（兼容替身）。

#### Scenario: 重试跨过原 TTL 仍能送达
- **WHEN** 写入 desired 后设备首次下发失败，经过超过存储 TTL 的时间后重试，设备恢复
- **THEN** 重试 SHALL 读到原 desired 并下发成功，SHALL 记 `Drifted` 后复验 `Converged`

#### Scenario: 送达确认后过期
- **WHEN** 复验 `Changes==0` 触发 `MarkSynced` 成功，再经过存储 TTL+ε
- **THEN** desired SHALL 读空；此后周期对账 SHALL NOT 改写最近一次 `converged` 结局

#### Scenario: 对账中途被改写不误标
- **WHEN** 对账已读入 desired（gen=1）尚未完成，用户再次写入同 key（gen=2），随后对账复验 0 变更调用 `MarkSynced(1)`
- **THEN** 标记 SHALL 失败，条目 SHALL 仍为待同步，下一次对账 SHALL 以 gen=2 的值下发

#### Scenario: 永远送不到则放弃
- **WHEN** 设备持续失败，条目待同步时长超过放弃上限后再次对账失败
- **THEN** desired SHALL 被删除，结局 SHALL 为 `error` 且消息含「已放弃」，SHALL NOT 重投，SHALL NOT 触碰设备配置

#### Scenario: 未超限的失败照常重投
- **WHEN** 设备失败但待同步时长未超放弃上限
- **THEN** SHALL 按 YR-04 退避重投，desired SHALL 保留

#### Scenario: 送得到但永远不收敛则放弃
- **WHEN** 设备每轮都接受下发（`Changes>0`、无 error）但复验回读始终与 desired 不等，待同步时长超过放弃上限
- **THEN** SHALL 删除 desired、返回 `Terminal=true` 的 error（消息含「已放弃」「不收敛」），SHALL NOT 继续 push→复验 循环；未超限时 SHALL 照常上报 `Changes` 等待复验

#### Scenario: 写入落在快照与读值之间不误标
- **WHEN** 对账取到写代快照 gen=1 后、读取 desired 前，用户写入新值（gen=2），本轮以新值对账得零变更
- **THEN** `MarkSynced(1)` SHALL 失败，新值 SHALL 仍为待同步

#### Scenario: 放弃上限环境变量
- **WHEN** 设置 `USMP_DESIRED_ABANDON_AFTER=200ms` 启动
- **THEN** 放弃判定 SHALL 以 200ms 为上限；设为非法字符串时 SHALL 回退 30 分钟并记 warning
