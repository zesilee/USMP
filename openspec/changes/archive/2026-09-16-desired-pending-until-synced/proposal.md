# desired-pending-until-synced — desired 存活到送达为止

## Why

desired 存储（`InMemoryConfigStore`）目前是「写入后 1 分钟硬过期」，而对账重试链（1/2/4/8/16/30s 退避 + 单次设备操作最长 60s）轻易跨过 1 分钟。设备慢或短暂不可达时，第二三次重试读到的 desired 已为空，`GenericReconciler` 按「无事可做」返回，controller 记 `Converged` 并 `Forget`——**用户的改动被静默丢弃，界面却显示已收敛**（假成功）。周期对账（5 分钟）也因此对原生配置几乎永远空转。

由墙上时钟决定放不放弃是错的；应由「有没有送到设备」决定。

## What Changes

- desired 条目引入两态生命周期：**待同步**（写入后不计 TTL，直到对账确认设备与意图一致）→ **已同步**（自确认时刻起计 TTL，到期清除，之后设备为唯一真相，产品口径 A）。
- 再次写入同一 key SHALL 退回待同步态并递增写代（generation）；对账只在写代未变时才把条目标记为已同步（防「读旧值→用户改新值→标旧值已同步」竞态）。
- 待同步条目设**放弃上限**（默认 30 分钟，`USMP_DESIRED_ABANDON_AFTER` 可调）：超限仍未送达则删除 desired、记 `error`（消息含「已放弃」）、不再重投；不得静默变为收敛。
- desired 读空（已同步后过期 / 被放弃 / LRU 淘汰 / 重启）时 SHALL NOT 记 `Converged`，SHALL NOT 覆盖上一次真实对账结局，仅 `Forget` 退避计数。
- 运行配置读缓存（30s）、状态快照缓存、意图层 5 分钟 resync 行为不变。

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `config-cache`: CC-02 增加「待同步条目不计 TTL」的例外；新增 CC-08 待同步/已同步两态与写代跟踪原语。
- `yang-controller-runtime`: YR-02 补「desired 读空不记收敛、不覆盖结局」；YR-04 补「终态错误不重投」；新增 YR-09 desired 生命周期（同步确认才起 TTL、写代守卫、放弃上限）。

## Impact

- 后端：`internal/cache/ttl_lru.go`（两态条目）、`pkg/yang-runtime/manager/manager.go`（`InMemoryConfigStore` 实现可选扩展接口）、`pkg/yang-runtime/reconcile/reconcile.go`（同步确认 / 放弃判定 / `Result.Terminal`）、`pkg/yang-runtime/controller/controller.go`（读空不记录、终态错误 Forget）。
- 接口：`reconcile.ConfigStore` **不变**（新能力经可选接口 `reconcile.SyncTracker` 类型断言接入，测试替身零改动）；`status.Outcome` 枚举不变，前端无改动。
- 6 个模块 reconciler（vlan/ifm/system/bgp/networkinstance/plainmodule）与意图层 reconciler 经 `GenericReconciler` 自动获得新行为，零改动。
- 内存：待同步条目不再自动过期，仍受 LRU 容量（1000）与放弃上限双重约束。
- 文档：`docs/memory/frontend-landing-risklog.md` 「desired 1min 过期」条目收口。
