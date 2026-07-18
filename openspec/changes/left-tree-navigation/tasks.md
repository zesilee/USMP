# Tasks — left-tree-navigation

> spec 已先行（R17）。worktree 隔离。单 PR 目标，超 1000 行手写面则拆后端/前端两 PR。

## 1. lefttreegen（LT-01）

- [ ] 1.1 B1 红灯：testdata（迷你 left-tree.json + demo yang）——层级/双语/rootContainers 映射、解析失败叶容器为空、畸形 JSON 明确报错
- [ ] 1.2 `tools/lefttreegen` 实现 + `internal/yangschema` go:generate 接线，生成 `lefttree.gen.go`（对真实 left-tree：14 组/65 叶）；重复生成零漂移
- [ ] 1.3 门禁：`go test ./tools/lefttreegen/ ./internal/yangschema/ -race` 全绿

## 2. left-tree API（LT-02）

- [ ] 2.1 B3 红灯：available/module 标注（vlan 已加载→true+"vlan"；未生成模块→false 仍在树）、`?device=` supported 叠加与省略语义（复用②期 sim 装配）、未注册 404
- [ ] 2.2 `internal/api` LeftTree handler + main.go 路由注册 + swagger 注解 → 转绿
- [ ] 2.3 `make gen-contract` 契约同步；`go test ./internal/api/... -race` 全绿

## 3. 前端左树（LT-03）

- [ ] 3.1 F1 红灯：menu store `loadLeftTree`（装配/失败降级标志）；F2 红灯：Sidebar 树渲染（14 组、可点叶路由、禁用叶提示、data-test 锚点、降级回 category 分组）
- [ ] 3.2 实现：api client + store + Sidebar 递归树渲染 → 转绿；`npm test` 全绿
- [ ] 3.3 staging-smoke.spec.ts 选择器适配（data-test 锚点）；本地 `make e2e-local` 全绿（§6.2 门禁）

## 4. 收官

- [ ] 4.1 全量 `go test ./... -race` + 前端全绿 + 覆盖率棘轮（现 70.0，留 0.1 余量）
- [ ] 4.2 `go-code-review-check` → 原子提交（≤500 行/次）→ PR → CI 绿直接合（已授权）
- [ ] 4.3 `/opsx:sync` + `/opsx:archive`；更新 [[snd-integration-program]] ③期状态、④期入口
