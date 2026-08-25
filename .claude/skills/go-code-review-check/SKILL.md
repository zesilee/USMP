---
name: go-code-review-check
description: 针对 USMP 项目技术栈的自动代码审查：yang-controller-runtime 架构合规、NETCONF 协议处理、TTL+LRU 缓存并发安全、Beego API 规范、Go 通用规范
---

# 技能详情

## 一、激活时机（何时自动触发）
1.  当用户需求包含「代码评审」「Review」「代码检查」等关键词时，自动激活。
2.  开发流程中，任何业务代码编写完成后，自动触发本技能，执行全维度代码评审。
3.  代码评审不通过时，禁止触发 Commit（T04），需整改后重新评审。

## 二、项目技术栈检查项（核心）

### 🎯 1. yang-controller-runtime 架构合规

| 检查项 | 检查内容 |
|--------|---------|
| **Reconciler 模式** | 是否正确嵌入 `*reconcile.GenericReconciler`<br>是否实现 `New(cs reconcile.ConfigStore, clientPool client.ClientPool)` 构造函数<br>是否正确实现 `DiffEngine` 适配层 |
| **Controller 模式** | 是否遵循 Manager → Controller → Reconciler 三层架构<br>是否正确注册到 Manager |
| **ConfigStore** | 是否使用内存缓存作为 ConfigStore 后端<br>禁止任何自管数据库依赖（MySQL/Redis/SQLite 等，R03）；持久元信息只走 K8s CRD |
| **ClientPool** | 是否正确使用连接池，连接是否复用<br>禁止业务代码自建连接/自写重连循环 |
| **Diff 逻辑** | 是否正确使用 `diff.DefaultDiffEngine`<br>desired/actual 状态比对逻辑是否正确 |

### 🎯 2. 生成代码与类型安全（R04）

| 检查项 | 检查内容 |
|--------|---------|
| **禁止手写 YANG 结构体** | YANG→Go 一律走自研 yanggen 生成管线（`make gen-yang` regen-and-diff），禁止手改 `backend/internal/generated/` |
| **禁止回引 ygot/openconfig** | 发布二进制与主 go.mod 零 openconfig 依赖，守护测试 `backend/ygot_retirement_guard_test.go` 拦截 |
| **空指针防护** | 访问生成结构体嵌套字段前逐层判 nil；禁止滥用 `interface{}` |

### 🎯 3. NETCONF 协议处理（唯一引擎 netconfcore，NC-01 禁 scrapligo）

| 检查项 | 检查内容 |
|--------|---------|
| **连接管理** | 是否经 ClientPool 获取连接；连接状态字段是否有锁保护 |
| **断线重连** | 重连由框架承担；重连间隔合理（避免风暴）、次数有限制 |
| **并发安全** | 客户端是否被多协程并发访问；写操作是否经 opMu 串行化 |
| **异常处理** | RPC 调用正确处理 error 与 `<rpc-error>` 响应；XML 解析失败优雅降级（R08） |
| **超时控制** | 所有网络操作有 context 超时 |
| **副作用** | `ExecuteRPC` 不重试、结果不入缓存 |
| **资源释放** | 连接关闭在 `defer` 中执行，无泄漏 |

### 🎯 4. TTL+LRU 内存缓存

| 检查项 | 检查内容 |
|--------|---------|
| **并发安全** | 所有缓存操作有 `sync.RWMutex` 保护；**RLock 临界区内禁止写 map/更新 LRU**（需要写就升级为 Lock） |
| **TTL 过期** | 定时清理协程有 `Stop()` 方法，过期逻辑正确 |
| **LRU 淘汰** | 队列更新与容量满淘汰策略正确 |
| **内存泄漏** | 协程退出通道正确关闭，map 有清理逻辑 |
| **主动失效** | 配置下发后主动失效对应 Key，失效范围合理 |

### 🎯 5. Beego API 规范（唯一框架 beego/v2 v2.3.0 钉死，禁止回引 gin）

| 检查项 | 检查内容 |
|--------|---------|
| **框架合规** | 禁止 gin 依赖回流（no_gin_guard 守护测试拦截）；勿升 beego v2.3.1+（连带抬 Go 版本，违反 Go1.22 钉死） |
| **路由规范** | RESTful 命名；每 Server 自持 ControllerRegister；`:splat` 无前导斜杠一律 wildcardPath |
| **参数校验** | `binding:"required"` 是 gin 专属**不生效**，必须显式校验（struct 标签留给 swag）；解析失败返回 400 |
| **错误响应** | 统一响应封装，错误码/信息规范；参考 docs/memory/gin-to-beego-migration.md |
| **Context 处理** | ctx 正确传递到下游，不忽略超时取消 |
| **并发安全** | Handler 中无共享状态竞态 |

### 🎯 6. Go 语言通用规范

| 分类 | 检查项 |
|------|--------|
| **并发安全** | 竞态检测（go test -race）<br>Channel 使用是否有死锁风险<br>WaitGroup 使用是否正确<br>goroutine 是否有退出机制 |
| **错误处理** | error 不能忽略（`_` 接收需有合理解释）<br>错误链是否使用 `%w` 包装<br>自定义错误是否有明确类型 |
| **资源管理** | `io.Closer` 是否在 `defer` 中关闭<br>文件/网络连接是否正确释放 |
| **代码规范** | 命名遵循 Go 规范（PascalCase 导出，camelCase 内部）<br>包名小写、无下划线<br>函数长度合理（一般不超过 80 行） |
| **测试覆盖** | 单元测试覆盖核心逻辑；边界与并发场景有测试；按 §5.6 选层无缺层 |

## 三、评审报告输出格式

```
📋 代码评审报告
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📁 评审对象
  backend/internal/controller/vlan/reconciler.go
  backend/pkg/yang-runtime/client/netconf.go

🎯 评审结论
  ❌ 评审不通过（存在 N 个严重问题，需修复后提交）
  OR
  ✅ 评审通过（仅 N 个建议优化项，可直接提交）

🔍 详细检查结果（按上述 6 维逐项 ✅/⚠️/❌，每个问题给出位置+风险+修改建议）

📊 问题汇总：严重 🔴 / 中危 🟡 / 低危 🟢 各 N 个

🛠️ 整改清单（按优先级排序，附修改前后代码片段）

📏 代码行数检查：本次新增 ≤500 行（§5.3）
```

## 四、评审严重等级定义

| 等级 | 颜色 | 说明 | 是否阻止提交 |
|------|------|------|-------------|
| 🔴 严重 | 红色 | 必然导致 panic、数据竞态、内存泄漏<br>或违反 §2 架构红线 | ✅ 必须修复 |
| 🟡 中危 | 黄色 | 极端情况下出问题<br>或影响可维护性 | ⚠️ 建议修复 |
| 🟢 低危 | 绿色 | 代码风格、可读性优化<br>不影响功能 | ❌ 不阻止 |

## 五、修复后重新评审

修复所有 **严重** 问题后，自动触发二次评审：
1.  验证严重问题是否全部修复
2.  检查修复是否引入新问题
3.  给出最终评审结论
