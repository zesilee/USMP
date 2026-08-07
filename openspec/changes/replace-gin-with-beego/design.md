# replace-gin-with-beego — 设计

## Context

后端 HTTP 层现状：`internal/api/Server` 持有 `*gin.Engine`，`NewServer` 注册 CORS + 全部路由；11 个 handler 以 `func(c *gin.Context)` 形态存在，对 gin 的使用面很浅（`Param`/`Query`/`ShouldBindJSON`/`c.JSON`，每文件 1–8 处）；统一响应信封集中在 `response.go` 4 个函数（**全部以 HTTP 200 携带信封内 code**，含错误）。测试两种形态：① `httptest` 打路由（经 Server）；② `gin.CreateTestContext` + 手工 `c.Params` 直调 handler。`cmd/test-server` 是前端 E2E 的内存 REST 桩，独立复刻了一小套 gin 路由。

约束：Go 1.22 钉死（beego v2.3.0 要求 go1.20，已冒烟验证）；对外 API 契约零变化；swagger 注释保留（前端契约生成管线依赖）；pr-size 门禁排除 go.sum（手写代码口径），beego 带来的 go.sum 膨胀不计体积。

## Goals / Non-Goals

**Goals:**
- go.mod 彻底移除 gin / gin-contrib-cors，唯一 Web 框架为 beego/v2 v2.3.0
- 路由路径、请求/响应 JSON、错误码语义、CORS 行为逐一等值，前端与 Playwright 套件零感知
- 测试套件全量迁移且断言语义不变，覆盖率不下降（T08 棘轮）

**Non-Goals:**
- 不采用 beego MVC Controller/ORM/session/日志等子系统（只用 router+context+cors filter）
- 不做两框架并行运行（单 router 无法并行挂载；等价性由契约测试红绿对照保障，见「迁移与验证」）
- 不改任何 API 行为、不借机重构 handler 业务逻辑

## Decisions

### D1 函数式路由，不用 MVC Controller
用 `web.ControllerRegister` 的 `Get/Post/Delete(pattern, func(ctx *context.Context))` 直接挂函数，handler 保持现有「方法 + 依赖注入构造器」形态，逐函数机械替换签名。备选 beego MVC（每资源一个 Controller struct）被否：diff 面大、丢失现有构造器注入模式、与「零行为变化」目标冲突。

### D2 每个 Server 自持 `web.NewControllerRegister()`，不碰全局 `web.BeeApp`
`ControllerRegister` 实现 `http.Handler`：测试沿用 `httptest` 直打；多测试并行各自建 Server 无全局状态污染（对称现状 gin.Engine 语义）。`Run(addr)` 用 `http.ListenAndServe(addr, register)`（gin 的 `router.Run` 本质相同）。备选全局 `web.Run()` 被否：全局单例毁掉测试隔离，且引入 beego 全家桶配置面。

### D3 请求体绑定：自建 `bindJSON` 助手直读 `ctx.Request.Body`，不开 `CopyRequestBody`
beego 函数式路由下 `ctx.Input.RequestBody` 依赖全局 `web.BConfig.CopyRequestBody=true`。改动全局配置对测试并行不友好，且多一次内存拷贝。`bindJSON(ctx, &v)` 用 `json.NewDecoder(ctx.Request.Body)`，语义对齐 `ShouldBindJSON`。

### D4 通配路径参数适配：`wildcardPath(ctx)` 统一补前导斜杠
gin `/*path` 的 `c.Param("path")` 含前导 `/`（如 `/ifm/interfaces`）；beego `*` 的 `:splat` 不含。所有 `/config/:ip/*path` 类路由经 `wildcardPath` 取参，保证下游 YANG 路径解析零改动。

### D5 响应信封原样平移
`response.go` 4 个函数签名改为 `*context.Context`，继续**一律 HTTP 200 + 信封内 code**（含 Error/DeviceOfflineError），JSON 序列化字段与顺序不变。

### D6 CORS 用 beego 自带 filter
`register.InsertFilter("*", web.BeforeRouter, cors.Allow(&cors.Options{...}))`，Options 逐项对齐现配置（AllowAllOrigins/Methods/Headers）。

### D7 测试上下文助手 `newTestContext(method, target, body, params...)`
一处封装 beego `context.NewContext()+Reset(w,req)+Input.SetParam(":k",v)`，机械替换全部 `gin.CreateTestContext + c.Params` 用法，压缩 diff、statically 保证两形态测试都迁移。

### D8 路由注册顺序与优先级
静态段 `changeset/preview|commit` 先于 `/:ip/*path` 注册（对齐现注释意图）；beego 路由树静态优先，行为由现有 changeset B3 测试锁死。

### D9 单 PR 交付（预算内），docs/memory 单独 commit
手写 diff 估算 400–700 行（非测试 ~45 处 + 测试 ~92 处引用 + 助手），go.sum 免计体积 → 单 PR ≤1000 行可行。若实做超限，按「internal/api + 测试」/「cmd/test-server + 文档」两 PR 拆，前者合入时 go.mod 仍暂留 gin（test-server 尚未切）、后者收尾移除。

## Risks / Trade-offs

- [beego 路由对含 `.` 的段/通配尾段有扩展名拆分特性（`:path`/`:ext`），`:ip` 为点分 IP、`*path` 尾段可能含点] → apply 首任务写「路由等价性冒烟测试」（点分 IP、含点通配尾、静态/参数共存三类），红灯先行确认 beego 实际行为，必要时在 D4 助手内拼回
- [beego 函数式 API 的确切方法签名（`ControllerRegister.Get` 等）凭记忆有出入风险] → 同一冒烟测试第一时间验证，偏差只影响 D1 的调用拼写不影响架构
- [产物新增链接 prometheus client（beego web 内置）] → 不注册 metrics 路由即无端点暴露；依赖分析文档如实记录
- [test-server 是 Playwright 套件的后端，后端-only 改动不触发 pre-push e2e 门禁] → 主动跑 `make e2e-local` 全绿再合入，不留盲区
- [swag 对 beego 注释解析差异] → swag 解析的是注释不是框架代码，合入前跑 `make gen-contract` 验证零漂移

## 迁移与验证（替代双路径并行）

§5.3「旧代码保留+新代码并行」在单 router 场景不可行（两框架无法共挂一个端口/路由树）。等价保障改为**契约测试红绿对照**：切换前全量 B3 套件绿（基线）→ 切换后同一套件不改断言跑绿（等价证明）→ `make e2e-local` 前端冒烟绿（端到端证明）。回滚策略：单 PR revert 即回到 gin，无数据/接口迁移残留。

## Open Questions

（无——审计口径、切换范围、Go 版本兼容均已拍板/验证）
