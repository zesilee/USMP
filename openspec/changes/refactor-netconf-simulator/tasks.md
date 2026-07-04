# 重构 NETCONF 模拟网元 — tasks

> change：`refactor-netconf-simulator` | 依赖：`proposal.md` `design.md`
> TDD（T01/R06）：每步先写测试再实现。渐进迁移（§5.3）：旧保留 → 新并行 → 对拍 → 切换 → 删除。
> 单次 commit ≤500 行；每步测试全绿方可进入下一步。

## T0 — 决策与基线（apply 首步）

- [ ] **T0.1** 确认 test-server 去向：(a) 显式内存 REST 桩 / (b) 复用 netconfsim 全链路（design §D6）→ 记录选型
- [ ] **T0.2** 跑基线 `go test ./...` 全绿，记录当前 netconfsim/netsim 覆盖的测试清单
- [ ] **T0.3** 实测 scrapligo 客户端 hello 是否协商 1.1 chunked framing（决定 D4 深度）

## T1 — 测试脚手架 + 解耦（不改行为）

- [ ] **T1.1** 新增 `netconfsim/testsupport` 子包，迁移 `Assert*`（`simulator.go:185-236`）到此
- [ ] **T1.2** 集成测试改调 `testsupport.Assert*`，`netconfsim` core 移除 `import "testing"`/testify
- [ ] **T1.3** 验证：`go list -deps ./backend/simulator/netconfsim` 不含 `testing`
- [ ] **T1.4** 新增 `cmd/netconf-simulator/main.go`（flag 端口/初始配置），`go build` 通过

## T2 — 结构化 datastore（新旧并行）

- [ ] **T2.1** 先写测试：ygot 树 datastore 的 Set/Get/Commit/Discard 单测（正常/异常/并发）
- [ ] **T2.2** 实现 `treeDatastore`（running/candidate 为 ygot `*Device`），与旧 XML datastore 并存
- [ ] **T2.3** 先写测试：客户端产出 XML → `ygot.Unmarshal` → 树的往返等价（对拍旧 Extract*）
- [ ] **T2.4** 实现 get-config：树 + filter → Marshal（先无 filter，全量往返绿）

## T3 — edit-config 语义（TDD）

- [ ] **T3.1** 先写测试：merge/create/delete/replace 各 operation 的候选库变更断言
- [ ] **T3.2** 实现 `editconfig.go`：解析 operation，合并/删除进 candidate 树
- [ ] **T3.3** 先写测试：get-config subtree filter 裁剪断言
- [ ] **T3.4** 实现 `filter.go` subtree filter

## T4 — 协议保真（TDD）

- [ ] **T4.1** 先写测试：hello capability 广告断言（base:1.0/1.1、:candidate、:writable-running）
- [ ] **T4.2** 实现 capability 广告 + `encoding/xml` RPC 分发（替换 strings.Contains）
- [ ] **T4.3**（条件，依 T0.3）实现 1.1 chunked framing；否则仅广告不实现并记录

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
