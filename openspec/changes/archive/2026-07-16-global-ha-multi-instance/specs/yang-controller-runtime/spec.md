# yang-controller-runtime delta — 周期事件源 leader election 门控

## ADDED Requirements

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
