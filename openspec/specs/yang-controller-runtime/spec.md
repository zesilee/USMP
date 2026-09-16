# yang-controller-runtime — 声明式配置对齐框架

## Purpose

yang-controller-runtime 是权威栈（R01）的声明式对齐框架（C1–C5）：把设备 actual 配置向 desired 对齐。用户仅实现 C3 `Reconciler`，框架承担连接/排队/限频/反射 diff/协议编解码。连接信息由共享 DeviceStore 按 DeviceID 解析（见 [[device-store]]）。

> 已知契约缺口（as-built）：schema 层运行时为空；`ConfigStore.List` 等为 stub。（plugin 脚手架已随 retire-idle-scaffolds 物理删除。）

## Requirements

### Requirement: YR-01 期望态触发对齐

已 `ConfigStore.Set(deviceID:path, desired)` 后，`Manager.TriggerReconcile` 或 `PeriodicSource` 到期 SHALL 产生事件，经 predicate 过滤入 `RateLimitingQueue`，worker SHALL 调用对应 `Reconciler.Reconcile`。

#### Scenario: 提交触发对账
- **WHEN** 调用 `TriggerReconcile(deviceID, path)` 且有匹配 Controller
- **THEN** 事件 SHALL 入队并被 worker 处理，返回 `triggered=true`

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

### Requirement: YR-03 从 DeviceStore 解析建连

Reconciler 的 `DeviceClient` SHALL 用 `req.DeviceID` 查共享 DeviceStore 取完整连接信息（IP/端口/凭据/协议）建连。设备未注册（或无 store）SHALL 降级为 AUTO/无凭据连接并记 warning，认证失败干净返回（R08），SHALL NOT panic，SHALL NOT 硬编码凭据。

#### Scenario: 已注册设备带凭据建连
- **WHEN** 以纯 DeviceID 触发、设备已在 DeviceStore 注册
- **THEN** SHALL 用库中凭据建 NETCONF 连接，SSH 以 password 认证（非 none）

#### Scenario: 未注册设备降级
- **WHEN** DeviceID 未在库中
- **THEN** SHALL 以 AUTO/无凭据建连、认证失败返回错误，SHALL NOT panic

### Requirement: YR-04 失败重投带退避

`process` 处理 Result：error 或 Requeue SHALL `AddRateLimited`（指数退避 + 令牌桶）或按 `RequeueAfter` `AddAfter`；收敛成功 SHALL `Forget`。`Result.Terminal=true` 的 error SHALL 记 `error` 结局后 `Forget`，SHALL NOT 重投。

#### Scenario: 失败退避重投
- **WHEN** Reconcile 返回 error
- **THEN** SHALL 以指数退避重新入队

#### Scenario: 终态错误不重投
- **WHEN** Reconcile 返回 `Terminal=true` 且带 error
- **THEN** SHALL 记 `error`（含原因），SHALL `Forget`，队列 SHALL NOT 再出现该 Request

### Requirement: YR-05 纠正后复验收敛

对账下发有变更（`Changes>0`、无 error）时 SHALL 记 `Drifted`，且 controller SHALL 入队一次复验；复验若无变更 SHALL 记 `Converged` 并 `Forget`，使状态自 `Drifted` 自愈为 `Converged`，避免"纠正后永久显示漂移"。

#### Scenario: 纠正后自愈为收敛
- **WHEN** 首轮下发 `Changes>0`（记 Drifted）
- **THEN** SHALL 入队复验；复验 `Changes==0` 时 SHALL 记 `Converged`

### Requirement: YR-06 每模块一控制器隔离

每 YANG 模块（vlan/ifm/system）经 `ControllerManagedBy(name).WithReconciler().WithSource().Build()` 注册，SHALL 有独立事件队列 + worker 池，模块间 SHALL 隔离（一模块阻塞不影响另一模块）。

#### Scenario: 模块隔离
- **WHEN** 注册 vlan/ifm/system 三控制器
- **THEN** 各自独立队列处理，互不干扰

### Requirement: YR-07 事件源驱动漂移检测

事件源 SHALL 支持周期轮询 / K8s CRD watch / 文件变更（gNMI 订阅源已随 gNMI 空壳清除移除，协议为规划能力）。`PeriodicSource` SHALL 按提供的设备列表（生产为共享 DeviceStore 的 `List()`，动态取）逐设备入队对账，实现持续 out-of-band 漂移检测；设备列表为空 SHALL 不入队、SHALL NOT panic。

#### Scenario: 周期按库中设备发对账
- **WHEN** 周期 tick 且 DeviceStore 中有 N 个设备
- **THEN** SHALL 为每个设备就配置路径入队一个对账事件

#### Scenario: 空设备列表不空转报错
- **WHEN** 设备列表为空
- **THEN** SHALL 不入队任何事件，SHALL NOT panic

### Requirement: YR-08 周期事件源选主门控

框架 SHALL 提供泛化的 leader-gated Source 包装（自 intent 面实现提升）：多副本部署下 SHALL 仅 leader 副本启动被包装的内部事件源（非 leader SHALL NOT 产生 reconcile 事件）；leader 丢失 SHALL 停止内部事件源，已入队事件由 worker 自然排空。原生周期控制器（vlan/ifm/system/bgp/network-instance）SHALL 以单一全局 Lease（`usmp-native-controllers`，独立于意图面 Lease）统一门控，开关 `USMP_NATIVE_LEADER_ELECTION`（缺省关）。开关关闭或无可达集群时 SHALL 透传内部事件源（现行为零变化，R08）。意图面 SHALL 复用同一泛化实现（行为等价，Lease 名与开关不变）。

#### Scenario: 仅 leader 产生周期对账事件
- **WHEN** 开启门控且两副本同时运行
- **THEN** 仅持有 `usmp-native-controllers` Lease 的副本 SHALL 周期入队对账事件，另一副本 SHALL NOT 入队

#### Scenario: leader 切换接管
- **WHEN** leader 副本终止
- **THEN** 另一副本 SHALL 在 Lease 过期后取得领导权并启动周期事件源，恢复对账

#### Scenario: 关闭开关透传
- **WHEN** `USMP_NATIVE_LEADER_ELECTION` 未设或非 `1`
- **THEN** 周期事件源 SHALL 直接启动（与门控引入前行为一致）

#### Scenario: 无集群降级透传
- **WHEN** 开启开关但无可达 kubeconfig
- **THEN** SHALL 记日志并透传启动事件源，SHALL NOT 崩溃（R08）

#### Scenario: 与意图面 Lease 互不干扰
- **WHEN** 原生面与意图面选主同时开启
- **THEN** 两者 SHALL 使用各自独立 Lease，任一面的 leader 变更 SHALL NOT 影响另一面

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
