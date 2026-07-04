# 重构 NETCONF 模拟网元 — tasks

> change：`refactor-netconf-simulator` | 依赖：`proposal.md` `design.md`
> TDD（T01/R06）：每步先写测试再实现。渐进迁移（§5.3）：旧保留 → 新并行 → 对拍 → 切换 → 删除。
> 单次 commit ≤500 行；每步测试全绿方可进入下一步。

## T0 — 决策与基线（apply 首步）

- [x] **T0.1** test-server 去向：选 **(a) 显式内存 REST 桩**。理由：test-server 现仅服务前端 Playwright E2E 的 REST 接口，本就不经 NETCONF；抽成诚实命名的内存 fixture 即可删 netsim，(b) 全链路属过度工程，不在本次范围。
- [x] **T0.2** 基线：`go test ./simulator/... ./internal/controller/... ./pkg/yang-runtime/actor/...` 全绿。netconfsim 被 4 个 reconciler + actor + e2e 集成测试覆盖；netsim 仅 cmd/test-server。
- [x] **T0.3** scrapligo v1.4.0 **自动协商最高公共版本**，客户端（`netconf.go:65` 未强制版本）随服务端广告而定。模拟器只广告 `base:1.0` → 用 1.0 EOM framing（现状）。**结论：保持 1.0 framing，不实现 1.1 chunked（T4.3 移出范围）**，capability 补 `:candidate`/`:writable-running`。

## T1 — 测试脚手架 + 解耦（不改行为）✅

- [x] **T1.1** 新增 `netconfsim/testsupport` 子包，迁移 33 个 `Assert*`（scenarios.go 30 + simulator.go 3）为函数 `Assert*(t, sim, …)`
- [x] **T1.2** 6 个集成测试文件改调 `testsupport.Assert*`；core（scenarios.go/simulator.go）移除 `testing`/testify，`ScenarioConfig` 保留在 core
- [x] **T1.3** 验证：`go list -deps ./simulator/netconfsim` 与 `./cmd/netconf-simulator` 均不含 `testing`
- [x] **T1.4** 新增 `cmd/netconf-simulator/main.go`（flag -addr/-port）+ 核心 `SetListen`；二进制构建、绑定、优雅退出均通过

## T2 — 结构化 datastore（新旧并行）

> ⚠️ 方案修订：ygot 树经核实不可行（两独立根 + ygot 仅 JSON + 零 xml tag），改用**通用 XML 数据树**（Option C，见 design.md D1）。

- [x] **T2.1** 测试先行：`dataNode` parse/serialize 往返 + find/children + clone 隔离 + 解析错误（`datatree_test.go`）
- [x] **T2.2** 实现 `dataNode`（std encoding/xml 通用节点，含命名空间默认声明重建）+ `treeDatastore`（running/candidate 为 `*dataNode`），与旧 `Datastore` 并存
- [x] **T2.3** 对拍测试：同一 huawei/openconfig 配置经 legacy Datastore 与 treeDatastore 输出语义等价（`TestTreeVsLegacyDatastoreEquivalence`）
- [x] **T2.4** treeDatastore 的 SetCandidate/SetRunning/GetRunning/GetCandidate/Commit/Discard（整树替换 + 深拷贝隔离 + `-race` 并发测试）；merge/filter 留给 T3/T4

## T3 — edit-config 语义（TDD）

- [x] **T3.1** 先写测试：merge/create/delete/replace 各 operation 的候选库变更断言
- [x] **T3.2** 实现 `editconfig.go`：解析 operation，合并/删除进 candidate 树
- [x] **T3.3** 先写测试：get-config subtree filter 裁剪断言
- [x] **T3.4** 实现 `filter.go` subtree filter

## T4 — 协议保真（TDD）

- [x] **T4.1** 先写测试：hello capability 广告断言（base:1.0/1.1、:candidate、:writable-running）
- [x] **T4.2** 实现 capability 广告 + `encoding/xml` RPC 分发（替换 strings.Contains）
- [x] **T4.3** ~~1.1 chunked framing~~ **移出范围**（T0.3 结论：模拟器只广告 1.0，scrapligo 随之用 1.0 framing，无需 1.1）

## T5 — 双路径验证与切换

- [ ] **T5.1** 现有 `*_integration_test.go` 对新树 datastore 各跑一遍，断言与旧实现等价
- [ ] **T5.2** 集成测试切到新实现（切换入口）
- [ ] **T5.3** test-server 按 T0.1 选型改接口，前端 E2E 绿

## T6 — 删除旧代码

- [ ] **T6.1** 删除旧 XML datastore、`Extract*`、`*TestData`、旧 server 字符串解析分支
- [ ] **T6.2** 删除 `backend/simulator/netsim/`，清理引用
- [ ] **T6.3** 更新 `openspec/specs/netconf-simulator/{spec,design,tasks}.md`，D10 迁移债勾除
- [ ] **T6.4** 全量 `go test ./...` 绿；`go build ./cmd/netconf-simulator` 绿

## 验收（对应 design §5）

- [ ] 新增测试覆盖 edit-config merge/delete、get-config filter、capability 广告
- [ ] netconfsim core 无 testing 依赖；独立二进制可构建
- [ ] netsim 删除无残留；datastore string-parsing 消除
- [ ] 满足 R02/R04/R06/§5.3
