---
id: refactor-netconf-simulator
title: 重构 NETCONF 模拟网元
status: in_progress
priority: high
branch: 各阶段独立分支（refactor/sim-tN-*）
change: refactor-netconf-simulator
updated: 2026-07-04
---

## 目标

把 `backend/simulator` 下两个职责重叠的模拟器（`netconfsim` 真 SSH+2PC / `netsim` 裸 TCP）统一重构为**单一、结构化（通用 XML 数据树）、协议保真、测试/部署解耦**的 NETCONF 模拟网元。
完整规划见 `openspec/changes/refactor-netconf-simulator/{proposal,design,tasks}.md`。

## 进度（截至 2026-07-04）

- [x] **T0** 决策 + **T1** 测试解耦 + 独立可部署二进制 — PR #11 / #12 已合入
  - core 去 `testing`/testify 依赖；`Assert*` 迁入 `netconfsim/testsupport`；新增 `cmd/netconf-simulator` + `SetListen`
- [x] **T2** 结构化 datastore（通用 XML 数据树）— PR #14 已合入（main `657a64b`）
  - `datatree.go`（`dataNode`）+ `tree_datastore.go`（`treeDatastore`），与旧 blob `Datastore` **并存未切换**；TDD 全绿含 -race + 与 legacy 对拍等价
- [ ] **T3** edit-config `operation=merge/replace/create/delete/remove`（在 `treeDatastore` 树上做增量语义，元素路径 + list 键启发式匹配）← **下一步**
- [ ] **T4** get-config subtree `<filter>`（对 running 树裁剪后序列化）
- [ ] **T5** 切换 `server.go` 到 `treeDatastore`（双路径验证后），`testsupport` 断言改查通用树
- [ ] **T6** 删 `netsim` + 旧 blob `Datastore`/`Extract*`/`*TestData`；test-server 改内存 REST 桩；勾除迁移债 D10

## 上下文恢复提示

- **权威 change**：`openspec/changes/refactor-netconf-simulator/`。`design.md` D1 已记录「ygot `*Device` 树不可行 → 改用通用 XML 树（Option C）」的关键修订及原因（两独立 ygot 根 + ygot 仅解析 JSON + 生成体零 xml tag）。
- **T2 落地代码**：`backend/simulator/netconfsim/datatree.go`（`dataNode`：parse/serialize/find/clone，命名空间以默认声明重建）、`tree_datastore.go`（`treeDatastore`，方法面镜像旧 `Datastore` 便于 T5 平滑替换）。
- **待改造的旧代码**：`datastore.go`（1068 行 blob + `Extract*` + `*TestData`，T6 删）；`server.go`（`strings.Contains` RPC 分发 + 整树覆盖式 edit-config，T3/T5 改）；`scenarios.go`（`ScenarioConfig` 故障注入，保留）；`testsupport/`（T1 断言，T5 改查树）。
- **T0 既定决策**：test-server 走内存 REST 桩（方案 a）；模拟器只广告 `base:1.0` → 保持 1.0 EOM framing，1.1 chunked **移出范围**。
- **环境注意**：pre-commit hook 已修（PR #13），Go 提交**无需 `--no-verify`**；但本机 go1.26 与仓库 go1.21 gofmt 有分歧——**只 `gofmt -w` 自己改动的文件**，勿动 pre-existing。集成测试默认（非 `-short`）会真跑 SSH 模拟器，用 `go test`（不加 -short）验证。
- **每阶段流程**：`EnterWorktree` → TDD 测试先行 → **≤500 行/commit**（commit-msg hook 硬限）→ 独立分支 → PR（≤800 行/≤20 文件，否则 3000）→ CI 全绿 → squash 合入 → 删远端分支（gh 常跳过需手删）→ `ExitWorktree remove` 清理。

## 恢复指令

1. 新 session：`git pull origin main`（确保含 `657a64b` 及后续合入）。
2. 恢复本任务：`/task resume refactor-netconf-simulator`。
3. 继续实现：`/opsx:apply refactor-netconf-simulator`，从 **T3** 起。
4. 相关记忆：`[[dual-stack-migration]]`（双栈架构背景）、`[[openspec-cli]]`（CLI 安装/使用注意）。

## 遗留旁项（非本迭代阻塞）

- `frontend/package-lock.json` 有一处历史遗留未提交改动，全程未纳入任何 PR，保持原样待确认。
- pre-commit hook 可进一步增强（如按仓库 go 版本对齐 gofmt），非必须。
