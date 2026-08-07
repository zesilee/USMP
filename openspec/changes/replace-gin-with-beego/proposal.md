# replace-gin-with-beego

## Why

开源选型评估拍板：后端 HTTP 框架统一切换为 beego（github.com/beego/beego/v2 v2.3.0），彻底移除 gin-gonic/gin 与 gin-contrib/cors。可行性已验证：beego v2.3.0 要求 go1.20（兼容交付钉死的 Go 1.22），Go 1.22 语言级别冒烟编译通过；实测编译产物仅链接 16 个模块、不含 beego 声明的任何数据库驱动（审计口径已确认按产物评估）。

## What Changes

- `internal/api/` 全部 11 个 handler + `server.go` + `response.go`：`gin.Engine`/`gin.Context` → beego `web.ControllerRegister` + `context.Context`（函数式路由，非 MVC Controller）
- CORS：`gin-contrib/cors` → beego 自带 `server/web/filter/cors`
- `cmd/test-server/main.go`（前端 E2E 内存 REST 桩）同步切换
- 约 20 个直接引用 gin 的测试文件同步改写（测试仍走 `httptest` + `http.Handler`，行为断言不变）
- `go.mod`：移除 `github.com/gin-gonic/gin`、`github.com/gin-contrib/cors`，新增 `github.com/beego/beego/v2 v2.3.0`
- **对外 API 不变**：全部路由路径、请求/响应 JSON 格式（统一 Response 信封）、错误码语义、CORS 行为逐一保持，前端零感知；swagger 注释原样保留，前端契约生成管线不动

## Capabilities

### New Capabilities

（无——本变更为实现层框架替换，不引入新的对外能力）

### Modified Capabilities

- `netconf-simulator`: NS-05「前端 E2E 后端（内存 REST 桩）」需求文本中「经 Gin REST 直供」的框架措辞改为框架中立表述（内存 REST 桩），行为约束（不经 NETCONF、命名诚实）不变。这是唯一把 Gin 写进需求级文本的 spec；其余 API spec（config-api/devices-api/yang-api 等）均为行为级契约，不受实现框架影响，无 delta。

## Impact

- **代码**：`backend/internal/api/`（12 个非测试文件 + 约 20 个测试文件）、`backend/cmd/test-server/main.go`
- **依赖**：go.mod 直接依赖 -2（gin、gin-contrib/cors）+1（beego/v2）；go.sum 新增约 40 个传递声明条目（产物不链接，已确认按产物口径评估）；产物新增运行时链接 prometheus client（beego web 内置指标）
- **文档**：`CLAUDE.md` §3 技术栈表 Gin → Beego；本次开源依赖分析结论同步更新
- **风险点**（design.md 展开）：gin `/*path` 通配参数含前导斜杠 vs beego `:splat` 不含；`/config/changeset/*` 静态段与 `/:ip/*path` 参数段的路由优先级；beego 函数式路由读请求体需 `CopyRequestBody`；`go.sum` 增量可能触发 pr-size 门禁需拆 PR
- **测试兜底**：现有 B3 API 契约测试套件全量回归 = 双路径验证（§5.3 存量改造要求的等价保障）
