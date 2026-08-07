---
name: gin-to-beego-migration
description: 后端Web框架已从gin全量切换beego/v2 v2.3.0——版本钉死陷阱/binding校验缺口/路由语义差异/迁移助手，碰HTTP层或加handler前必读
metadata:
  type: project
---

后端 HTTP 层唯一框架 = **beego/v2 v2.3.0**（2026-08-07 开源选型拍板，change: replace-gin-with-beego）。gin/gin-contrib 已从 go.mod 清除，`internal/api/no_gin_guard_test.go` 守护禁止回流（对齐 [[netconf-selfdev]] 的 NC-01 模式）。

**Why:** 开源评估要求统一框架为 beego；审计口径确认按产物评估（beego 声明的 40 个传递依赖含 MySQL/Redis/etcd 驱动，产物只链接 16 模块、无数据库驱动，新增运行时链接仅 prometheus client）。

**How to apply:**
- **版本钉死陷阱**：beego 只能 v2.3.0（要求 go1.20，兼容 [[go-122-pin]]）。**v2.3.1+ 的 go.mod 是 go 1.24.x**，`go mod tidy` 在无 require 行时会解析到最新版并连带把 go 指令抬到 1.24.2、x/crypto 抬到 v0.38——先 `go mod edit -require=...@v2.3.0` 再写 import 再 tidy；若已被抬高，从 main 基线重建 go.mod 再 pin（tidy 不会自动降级）。
- **binding:"required" 是 gin 专属语义**：beego 下 `bindJSON`（纯 encoding/json）不执行校验，必填校验必须在 handler 显式写（device_handler/business_handler 已补）。**标签本身保留勿删**——swag 靠它生成 OpenAPI required 标记，删了契约漂移。新 handler 加必填字段时两件事都要做：标签（给 swag）+ 显式校验（给运行时）。
- **路由语义差异**：gin `/*path` 参数含前导斜杠，beego `*` 的 `:splat` 不含 → 一律经 `wildcardPath(c)`（internal/api/beego_helpers.go）取通配参数。`/:ip/*` 组合被 beego 转成 regexp leaf `([^/]+)/(.+)`，点分 IP 与含点尾段安全；纯 `*` 不做 .json/.html 扩展名拆分。静态段（changeset）优先于参数段，行为由 beego_router_equiv_test.go 锁死。
- **测试写法**：直调 handler 用 `newTestContext(method, target, body, "k", "v", ...)`（beego_helpers_test.go，替代 gin.CreateTestContext；param key 不带冒号）；打路由用 `web.NewControllerRegister()`（实现 http.Handler，httptest 直打）。每 Server 自持 register，禁碰全局 web.BeeApp/web.Run。
- **请求体**：函数式路由下 `ctx.Input.RequestBody` 恒空（除非开全局 CopyRequestBody）——一律 `bindJSON(c, &v)` 直读 Request.Body。
- **响应信封**：`response.go` 一律 HTTP 200 + 信封内 code（含错误），`Output.JSON(obj, false, false)`；非 200 物理状态码（仅 test-server 用）先 `Output.SetStatus`。
- **启动噪音**：beego import 时打一行 `init global config instance failed... conf/app.conf` 到 stderr，无害（不用其配置系统），勿当故障排查。
