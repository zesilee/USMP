# replace-gin-with-beego — 任务清单

## 1. Worktree 与基线

- [x] 1.1 EnterWorktree 创建 `replace-gin-with-beego` 隔离环境，跑 `go test ./...` 确认基线全绿（红绿对照的「绿基线」）
- [x] 1.2 go.mod 引入 `github.com/beego/beego/v2 v2.3.0`（此阶段 gin 暂留，包不可编译不提交）

## 2. 路由等价性冒烟（TDD 红灯先行，D4/D8 风险验证）

- [x] 2.1 新建 `internal/api/beego_router_equiv_test.go`：验证 beego 函数式路由三类行为——①点分 IP 段 `/:ip/status`；②通配尾段 `:splat`（含前导斜杠适配、含 `.` 的尾段不被扩展名拆分）；③静态段 `changeset/preview` 与 `/:ip/*path` 共存优先级。先红后绿，确认 `ControllerRegister` 实际 API 拼写
- [x] 2.2 据 2.1 结果实现 `internal/api/beego_helpers.go`：`wildcardPath` / `bindJSON` / `newTestContext` 三助手（D3/D4/D7），表格驱动单测（B1，含并发 race）

## 3. API 层切换（对外行为零变化）

- [x] 3.1 `response.go`：4 个信封函数签名换 `*context.Context`，HTTP 200+信封 code 语义逐字保留；同文件测试同步
- [x] 3.2 `server.go`：`web.NewControllerRegister()` + `InsertFilter` CORS（D2/D6），全部路由原路径重注册（静态段先注册，D8）
- [x] 3.3 11 个 handler 逐文件机械替换签名与取参（`Param`→`Input.Param(":k")`、`Query`→`Input.Query`、`ShouldBindJSON`→`bindJSON`、通配→`wildcardPath`），每文件改完即跑该文件对应测试
- [x] 3.4 全部 `*_test.go` 迁移：`gin.CreateTestContext+Params` → `newTestContext`；httptest 直打路由的测试仅改 import；**断言一行不改**
- [x] 3.5 `go test ./internal/api/... -race` 全绿（红绿对照的「等价证明」）

## 4. test-server 与 gin 退场

- [x] 4.1 `cmd/test-server/main.go` 同法切换（NS-05 delta 对应实现）
- [x] 4.2 go.mod 移除 gin / gin-contrib-cors，`go mod tidy`，全仓 `go test ./...` -race 全绿
- [x] 4.3 守护测试：新增禁止重引 gin 的依赖黑名单测试（对齐 scrapligo NC-01 模式），防回流

## 5. 门禁与端到端验证

- [x] 5.1 `make gen-contract` 验证 swagger→前端契约零漂移
- [x] 5.2 `make e2e-local` Playwright 冒烟全绿（test-server 是 E2E 后端，主动补盲区）
- [x] 5.3 覆盖率对比基线不下降（T08）；`go-code-review-check` 通过
- [x] 5.4 CLAUDE.md §3 技术栈表 Gin→Beego（含 cors 行）；开源依赖分析口径同步（prometheus client 新增链接如实记录）

## 6. 合入与收尾

- [x] 6.1 What/Why/How 三段式 commit（功能与 docs/memory 分 commit，MEM04）
- [x] 6.2 finishing-a-development-branch：push + PR，CI 全绿自助 merge（体积超 1000 行则按 design D9 两 PR 拆分）
- [x] 6.3 `/opsx:sync` 合并 netconf-simulator delta 到主 spec，`/opsx:archive` 归档
