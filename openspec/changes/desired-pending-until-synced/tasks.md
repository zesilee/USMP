# Tasks: desired-pending-until-synced

> 全程 TDD（T01/T05/T07）：每步先写红灯用例再写实现。改动类型 = Reconciler + 纯逻辑 → 必补 B1（表格驱动 + race）+ B2 模拟网元集成。

## 1. 缓存原语两态（CC-02 例外 / CC-08）

- [x] 1.1 先写 B1 用例 `internal/cache/ttl_lru_pending_test.go`：待同步不过期（远超 TTL 仍命中、`ClearExpired`/`Keys` 不剔除）、`MarkExpiring` 后自当下起算 TTL、写代守卫（旧 gen 标记失败）、再写入退回待同步且 gen 递增、既有 `Set` 语义不变、并发 race 用例（编译红 → 实现绿）
- [x] 1.2 实现：entry 增 `gen`/`pending`；`SetPending`/`MarkExpiring`/`Track`（后补 `DeleteIfGen`）；`Get/GetWithAge/Keys/ClearExpired` 对 pending 免过期，四处过期判定收口到 `expired()`
- [x] 1.3 跑 `go test -race ./internal/cache/` 全绿，既有用例不动

## 2. ConfigStore 接入可选接口（D2）

- [x] 2.1 先写用例 `pkg/yang-runtime/manager/configstore_sync_test.go`：`Set` 后 `Track.pending=true`；`MarkSynced(gen)` 成功/失败；`Abandon(gen)` 删条目且 gen 不匹配不删；`List/ListDevices` 含待同步条目；并发 race
- [x] 2.2 `reconcile` 包新增 `SyncTracker` 接口；`InMemoryConfigStore.Set` 改 `SetPending`，实现 `Track/MarkSynced/Abandon`
- [x] 2.3 编译期断言 `var _ reconcile.SyncTracker = (*InMemoryConfigStore)(nil)`

## 3. GenericReconciler 生命周期（YR-09）

- [x] 3.1 先写回归用例（红）`pkg/yang-runtime/manager/reconcile_pending_regression_test.go`：**用真 `InMemoryConfigStore` + 20ms TTL**——首轮 Set 失败 → sleep 60ms → 二轮 SHALL 仍读到 desired 并成功下发。**红灯证据**：把 `Set` 换回 `cache.Set` 旧语义实测，正好在「重试跨过 TTL 不得读空（假收敛根因）」断言处失败
- [x] 3.2 用例：零变更 → `MarkSynced` 被调用且 gen 匹配；`Changes>0` 首轮不标记；Get 与 MarkSynced 之间插入 Set → 标记失败、条目仍 pending
- [x] 3.3 用例：失败 + 待同步超放弃上限 → desired 删除、`Result.Terminal=true`、error 含「已放弃」（回读/diff/下发三分支）；未超限 → 普通 Requeue 且 desired 保留；desired nil → `Result.NoDesired=true`；gen 不匹配放弃失败 → 退回重投
- [x] 3.4 用例：ConfigStore 不实现 `SyncTracker`（既有 MockConfigStore）→ 行为与改前完全一致（既有 `reconcile_test.go` 全绿 + `TestPending_PlainStoreUnaffected`）
- [x] 3.5 实现：`Result` 增 `Terminal`/`NoDesired`；`Reconcile` 开头 `trackLifecycle`；零变更 `MarkSynced`；三处失败分支收口 `fail()` 做放弃判定；`AbandonAfter()`/`SetAbandonAfter()` 惰性解析 `USMP_DESIRED_ABANDON_AFTER`（非法/非正回退 + warning，用例覆盖）

## 4. Controller 结局处理（YR-02 / YR-04）

- [x] 4.1 先写用例 `pkg/yang-runtime/controller/terminal_nodesired_test.go`：`NoDesired` → recorder 未被调用、`Forget` 被调用、不重投；`Terminal` error → 记 `error`、`Forget`、不重投；非 Terminal error → 仍 `AddRateLimited`（防回归）
- [x] 4.2 实现 `process()` 两个新分支；既有 `status_record_test`/`drift_requeue_test` 全绿

## 5. B2 模拟网元集成

- [x] 5.1 `internal/controller/vlan/reconciler_pending_integration_test.go`（`testing.Short()` 跳过）：sim 启动 → 写 desired（40ms TTL 存储）→ **sim 停止** → 对账 error → sleep 3×TTL → sim 同地址同端口重启 → 再对账下发成功 → 复验收敛转已同步 → 回读 VLAN 410 在网元上 → TTL 释放后 NoDesired
- [x] 5.2 放弃路径集成：`SetAbandonAfter(150ms)` + sim 停止不恢复 → 未超限重投保留 desired、超限终态放弃且 desired 删除（`-race` 5.7s 全绿）

## 6. 收口

- [x] 6.1 `main.go` 启动打印生效的放弃上限（解析在 `reconcile` 包惰性完成，非法已回退+warning）
- [x] 6.2 全量 `go test -race ./...` 全绿（最终树 31 个测试包，分 6 块 `-p 1` 串行跑——本机整包 race+覆盖率插桩编译会触发低内存被杀，见备注）；CI 口径覆盖率 77.0% ≥ 基线 76.8，棘轮上调至 77.0
- [x] 6.3 `go-code-review-check`（独立代理只读评审）：🔴 0 / 🟡 2 / 🟢 5，两中危均采纳整改（快照先于读值堵 TOCTOU；下发成功但不收敛也套放弃上限）+ 四低危修复（零值 pendingSince 守卫、双 %w、时序用例余量 200ms、evictLRU 注释），整改为第 5 个 commit；`git-what-why-how-commit` 三段式，5 个 commit 均 ≤500 行
- [x] 6.4 记忆更新：`docs/memory/frontend-landing-risklog.md` desired 过期条目收口 + 新增 `docs/memory/desired-pending-lifecycle.md` + 索引（待单独 `docs:` commit，MEM04）
- [ ] 6.5 PR + CI 全绿后 `/opsx:sync` 合入主 spec、`/opsx:archive`

## 实施备注

- 提交被 commit-msg 500 行门禁拦过一次（reconcile+存储+controller 合计 746 行含测试），拆为两个原子提交后通过。
- 全量 `-race` 后台跑时 main 包报过一次 `build failed`：是与我同时编辑 `main.go`（先加 import 后加调用）撞上编译的瞬时现象，之后 `go build ./...` 通过，最终树分块跑已复验 main 包 ok。
- 本机（8GB，其他会话常驻 ~3.3GB）整包 `go test ./... -race -coverprofile` 连续三次被「低内存」杀掉（含 `-p 2`、去 `-race` 两种减负形态，均死在编译期未跑出一个包）；改为 `scripts` 外的临时脚本按 6 包一块、`-p 1` 串行跑并合并 profile，口径与 CI 的 tested-pkgs 过滤一致。跑 CI 口径覆盖率时若再遇低内存，直接用分块法。
