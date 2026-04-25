---
name: go-code-review-check
description: 针对 USMP 项目技术栈的自动代码审查：yang-controller-runtime 架构合规、ygot 类型安全、NETCONF 协议处理、TTL+LRU 缓存并发安全、Actor 模型、Gin API 规范
---

# 技能详情

## 一、激活时机（何时自动触发）
1.  当用户需求包含「代码评审」「Review」「代码检查」等关键词时，自动激活。
2.  开发流程中，任何业务代码编写完成后，自动触发本技能，执行全维度代码评审。
3.  代码评审不通过时，禁止触发 Commit 技能，需整改后重新评审。
4.  Hook 中自动触发：Stop（提交前）、PostToolUse（go test 完成后）。

## 二、项目技术栈检查项（核心）

### 🎯 1. yang-controller-runtime 架构合规

| 检查项 | 检查内容 |
|--------|---------|
| **Reconciler 模式** | 是否正确嵌入 `*reconcile.GenericReconciler`<br>是否实现 `New(cs reconcile.ConfigStore, clientPool client.ClientPool)` 构造函数<br>是否正确实现 `DiffEngine` 适配层 |
| **Controller 模式** | 是否遵循 Manager → Controller → Reconciler 三层架构<br>是否正确注册到 Manager |
| **ConfigStore** | 是否使用内存缓存作为 ConfigStore 后端<br>禁止任何数据库依赖（MySQL/Redis/SQLite 等） |
| **ClientPool** | 是否正确使用连接池<br>连接是否正确复用，避免重复创建 |
| **Diff 逻辑** | 是否正确使用 `diff.DefaultDiffEngine`<br>desired/actual 状态比对逻辑是否正确 |

### 🎯 2. ygot 类型安全

| 检查项 | 检查内容 |
|--------|---------|
| **结构体生成** | 是否使用 `ygot` 自动生成的结构体<br>禁止手写 YANG 对应结构体 |
| **类型转换** | `ygot.Get()` 调用是否检查 `ok` 返回值<br>`Enum()` 调用是否存在 nil 风险<br>类型断言是否有 ok 检查 |
| **空指针防护** | 访问嵌套字段前是否检查父节点非 nil<br>例如 `vlan.GetConfig().GetVlanId()` 需逐层检查 |
| **JSON/XML 序列化** | 是否正确处理 ygot 结构体的序列化<br>是否使用 `ygot.EmitJSON` 等标准方法 |

### 🎯 3. NETCONF 协议处理

| 检查项 | 检查内容 |
|--------|---------|
| **连接管理** | 是否正确实现 `Connect()` / `Disconnect()`<br>连接状态 `connected` 字段是否有锁保护<br>是否正确处理连接失败重试 |
| **断线重连** | 是否有重连机制<br>重连间隔是否合理（避免风暴）<br>重连次数是否有限制 |
| **并发安全** | `*netconf.Driver` 是否被多协程并发访问<br>读写操作是否有互斥锁 |
| **异常处理** | NETCONF RPC 调用是否正确处理 error<br>是否正确处理 `<rpc-error>` 响应<br>XML 解析失败是否优雅降级 |
| **超时控制** | 所有网络操作是否有 context 超时<br>是否使用 `options.WithTimeoutSocket` 等 |
| **资源释放** | 连接关闭是否在 `defer` 中执行<br>是否有资源泄漏风险 |

### 🎯 4. TTL+LRU 内存缓存

| 检查项 | 检查内容 |
|--------|---------|
| **并发安全** | 所有缓存操作是否有 `sync.RWMutex` 保护<br>读写锁使用是否合理（读用 RLock，写用 Lock） |
| **TTL 过期** | 定时清理协程是否有 `Stop()` 方法<br>过期清理逻辑是否正确 |
| **LRU 淘汰** | LRU 队列更新逻辑是否正确<br>容量满时淘汰策略是否正确 |
| **内存泄漏** | 协程退出通道是否正确关闭<br>map 是否有清理逻辑 |
| **主动失效** | 配置下发后是否主动失效对应缓存 Key<br>失效范围是否合理（避免过度清理） |

### 🎯 5. Proto.Actor 模型

| 检查项 | Actor 实现检查 |
|--------|-------------|
| **Actor 生命周期** | `Receive()` 方法是否正确处理系统消息<br>`Init()` / `PostStop()` 是否正确实现 |
| **消息处理** | 消息处理是否非阻塞<br>禁止在 Actor 中执行长时间同步操作 |
| **状态管理** | Actor 内部状态是否仅在 Receive 协程中修改<br>禁止外部直接访问 Actor 内部状态 |
| **错误处理** | 消息处理 panic 是否有 `recover()` 防护<br>错误是否通过消息反馈，不直接 crash Actor |

### 🎯 6. Gin API 规范

| 检查项 | 检查内容 |
|--------|---------|
| **路由规范** | RESTful 命名规范<br>HTTP Method 使用正确（GET/POST/PUT/DELETE） |
| **参数校验** | 路径参数/Query/Body 是否有校验<br>参数解析失败是否返回 400 |
| **错误响应** | 是否统一使用 `response.Error()` / `response.Success()`<br>错误码/错误信息是否规范 |
| **Context 处理** | `ctx` 是否正确传递到下游<br>禁止忽略 ctx 超时取消 |
| **并发安全** | Handler 中是否有共享状态竞态<br>避免在 Handler 中直接修改全局变量 |

### 🎯 7. Go 语言通用规范

| 分类 | 检查项 |
|------|--------|
| **并发安全** | 竞态检测（go test -race）<br>Channel 使用是否有死锁风险<br>WaitGroup 使用是否正确<br>goroutine 是否有退出机制 |
| **错误处理** | error 不能忽略（`_` 接收需有合理解释）<br>错误链是否使用 `%w` 包装<br>自定义错误是否有明确类型 |
| **资源管理** | `io.Closer` 是否在 `defer` 中关闭<br>文件/网络连接是否正确释放 |
| **代码规范** | 命名遵循 Go 规范（PascalCase 导出，camelCase 内部）<br>包名小写、无下划线<br>函数长度合理（一般不超过 80 行） |
| **测试覆盖** | 单元测试是否覆盖核心逻辑<br>边界条件是否测试<br>并发场景是否测试 |

## 三、评审报告输出格式

```
📋 代码评审报告
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📁 评审对象
  internal/controller/vlan/reconciler.go
  pkg/yang-runtime/client/netconf.go

🎯 评审结论
  ❌ 评审不通过（存在 2 个严重问题，需修复后提交）
  OR
  ✅ 评审通过（仅 1 个建议优化项，可直接提交）

🔍 详细检查结果
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🏗️  yang-controller-runtime 架构
  ✅ Reconciler 模式正确嵌入 GenericReconciler
  ✅ 无数据库依赖，符合无 DB 约束
  ⚠️  DiffEngine 适配层缺少 error 包装（中危）
     └─ 建议：return nil, fmt.Errorf("diff failed: %w", err)

📦 ygot 类型安全
  ❌  vlan.GetConfig().GetVlanId() 缺少空指针检查（严重）
     └─ 位置：reconciler.go:142
     └─ 风险：device 返回空配置时直接 panic
  ✅ ygot.Get() 都有 ok 检查

🔌 NETCONF 协议
  ✅ 连接有超时控制
  ❌  connected 字段无锁保护，多协程访问有竞态（严重）
     └─ 位置：netconf.go:26
     └─ 建议：添加 sync.Mutex，所有访问加锁
  ✅ 断线重连机制完整

💾 TTL+LRU 缓存
  ✅ 所有操作有 RWMutex 保护
  ✅ 定时清理有 Stop 机制

🎭 Actor 模型（如涉及）
  ✅ 消息处理非阻塞

🌐 Gin API（如涉及）
  ✅ 响应格式统一

🐛 Go 通用规范
  ✅ 错误处理使用 %w 包装
  ⚠️  第 256 行函数过长（120 行），建议拆分（低危）

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 问题汇总

  严重问题：2 个 🔴
  中危问题：1 个 🟡
  低危建议：1 个 🟢

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🛠️  整改清单（按优先级排序）

1. [严重] 添加 VLAN 配置空指针检查
   ```go
   // 修改前
   vlanID := vlan.GetConfig().GetVlanId()

   // 修改后
   if vlan == nil || vlan.GetConfig() == nil {
       return nil, fmt.Errorf("vlan config is nil")
   }
   vlanID := vlan.GetConfig().GetVlanId()
   ```

2. [严重] NETCONFClient.connected 添加锁保护
   ```go
   type NETCONFClient struct {
       mu        sync.RWMutex
       info      DeviceConnectionInfo
       driver    *netconf.Driver
       connected bool
   }

   // 访问时：
   c.mu.RLock()
   defer c.mu.RUnlock()
   return c.connected
   ```

3. [中危] DiffEngine error 包装优化
4. [低危] 拆分过长函数

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📏 代码行数检查
  本次新增代码 238 行，≤ 500 行 ✅

📝 备注
  修复 1、2 项严重问题后，重新评审即可通过
```

## 四、评审严重等级定义

| 等级 | 颜色 | 说明 | 是否阻止提交 |
|------|------|------|-------------|
| 🔴 严重 | 红色 | 必然导致 panic、数据竞态、内存泄漏<br>会引发线上事故 | ✅ 必须修复 |
| 🟡 中危 | 黄色 | 极端情况下出问题<br>或影响可维护性 | ⚠️ 建议修复 |
| 🟢 低危 | 绿色 | 代码风格、可读性优化<br>不影响功能 | ❌ 不阻止 |

## 五、修复后重新评审

修复所有 **严重** 问题后，自动触发二次评审：
1.  验证严重问题是否全部修复
2.  检查修复是否引入新问题
3.  给出最终评审结论
