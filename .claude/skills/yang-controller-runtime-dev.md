# yang-controller-runtime 开发技能

## 技能名称
`yang-controller-runtime-dev`

## 核心作用
基于 `yang-controller-runtime` 框架（Kubernetes controller-runtime 架构风格）开发 YANG 模型控制器，实现网络交换机配置的声明式 reconcilliation 循环。框架处理所有 boilerplate，开发者只需编写业务逻辑。

## 架构理解

### 核心组件关系
```
Manager (全局生命周期)
  ├─ Schema (加载的YANG模型)
  ├─ ClientPool (设备连接池)
  ├─ PluginManager (插件扩展点)
  └─ Controllers (每个YANG模块一个Controller)
        ├─ EventSource (事件源: 周期轮询/gNMI订阅/文件变更)
        ├─ WorkQueue (带指数退避的工作队列)
        ├─ Predicate (事件过滤)
        └─ Reconciler (差异比对+配置对齐 - 用户实现)
```

### 角色职责

| 组件 | 职责 | 用户需要做 |
|------|------|-----------|
| **Manager** | 全局启动/停止，依赖注入 | 框架已实现，`main.go` 中创建启动 |
| **Schema** | YANG模型元数据缓存 | 框架从文件加载，用户不需要改 |
| **ClientPool** | 设备连接池，自动重连 | 框架已实现，通过 `manager.GetClientPool()` 获取 |
| **Controller** | 事件出队 → 过滤 → 调用 Reconciler | 框架已实现，通过 Builder 配置 |
| **EventSource** | 产生 reconcile 事件 | 使用内置: `source.NewPeriodicSource`, `source.NewGNMISubSource` |
| **Predicate** | 过滤不需要 reconcile 的事件 | 使用内置: `predicate.PathPrefix`, `predicate.Always` 等 |
| **Reconciler** | 比对 desired ↔ actual，推送变更 | **用户需要实现** |
| **Plugin** | 验证/变更/通知钩子 | 可选，用户可实现扩展 |

## 开发流程（严格遵循）

### 1. 创建新 Controller

```go
import (
  "github.com/leezesi/usmp/pkg/yang-runtime/controller"
  "github.com/leezesi/usmp/pkg/yang-runtime/source"
  "github.com/leezesi/usmp/pkg/yang-runtime/predicate"
  "github.com/leezesi/usmp/pkg/yang-runtime/reconcile"
  "github.com/leezesi/usmp/pkg/yang-runtime/manager"
)

// 1. 在 main.go 或入口处创建 Controller 并添加到 Manager
ctrl := controller.ControllerManagedBy("openconfig-interfaces").
  WithReconciler(NewInterfacesReconciler()).  // 用户实现 Reconciler
  WithSource(source.NewPeriodicSource(5*time.Minute, deviceIDs, "/interfaces")).
  WithPredicate(predicate.PathPrefix("/interfaces")).
  WithMaxWorkers(2).
  Build()

mgr.AddController(ctrl)
```

### 2. 实现 Reconciler 接口

```go
// Reconciler 接口定义
type Reconciler interface {
  Reconcile(ctx context.Context, req Request) Result
}

// Request 包含 reconcile 请求信息
type Request struct {
  DeviceID string  // 设备IP
  Path     string  // 触发 reconcile 的路径
}

// Result 返回 reconcile 结果
type Result struct {
  Requeue      bool          // 是否重新入队
  RequeueAfter time.Duration // 延迟多久重新入队
}

// 用户实现示例
type InterfacesReconciler struct {
  // 可以持有依赖: store, client, etc.
}

func NewInterfacesReconciler() *InterfacesReconciler {
  return &InterfacesReconciler{}
}

func (r *InterfacesReconciler) Reconcile(ctx context.Context, req reconcile.Request) reconcile.Result {
  // 1. 从 ConfigStore 获取 desired 配置（用户期望的配置）
  // 2. 从 设备 获取 actual 配置（设备当前运行的配置）
  // 3. diff 计算差异
  // 4. 推送变更到设备
  // 5. 返回 Result{Requeue: false} 成功
  
  // GenericReconciler 已经实现了这个流程，可以直接继承使用
}
```

### 3. 使用 GenericReconciler（推荐）

对于标准使用场景，直接继承 `GenericReconciler`，它会自动处理：
- 获取 desired 配置
- 获取 actual 配置
- 计算 diff
- 应用变更到设备
- 错误处理和重试

```go
type MyReconciler struct {
  *reconcile.GenericReconciler
}

// 只需要定制你需要的部分，默认行为已经够用
```

## 编码规范

### 接口遵循
- 严格使用框架定义的接口，不要绕过框架直接调用 client
- 所有设备通信通过 `client.Client` 接口，支持 NETCONF/gNMI 切换
- 所有事件通过 `Source` 产生，不要在 controller 自己轮询

### 错误处理
- 可重试错误（网络断开）返回 `reconcile.Result{Requeue: true}`，框架会自动指数退避
- 不可重试错误（配置非法）直接返回 `reconcile.Result{Requeue: false}`，记录错误
- 使用 `reconcile.NewReconcileError(err)` 包装错误

### 并发安全
- Reconciler 被多个 worker 并发调用，保持无状态
- 如果需要缓存，使用 `sync.RWMutex` 保护
- 不要在 Reconciler 中存储可变状态

### 依赖注入
- Manager 会注入 Schema, ClientPool，从 Manager 获取不要自己创建
- 自定义依赖通过结构体字段持有，在创建 Reconciler 时注入

## 常用 API 速查

### Client 操作
```go
// Get 配置
result, err := client.Get(ctx, path, client.WithDatastore("running"))

// Set 配置
changes := []client.Change{
  {Type: client.ModifyChange, Path: path, NewValue: data},
}
result, err := client.Set(ctx, changes, client.WithCommit(true))
```

### Diff 操作
```go
engine := diff.NewDefaultDiffEngine()
changes := engine.Diff(desired, actual)
```

### Predicate 组合
```go
// 组合多个条件
pred := predicate.And(
  predicate.PathPrefix("/interfaces"),
  predicate.ByType(predicate.UpdateEvent),
)
```

## 测试规范

### 单元测试
- 对 Reconciler 写单元测试，使用 mock Client
- 使用 `github.com/stretchr/testify/mock` 做 mock
- 覆盖正常路径、错误路径、边界情况

### 集成测试
- 端到端测试用真实设备连接（可选，标记为长测试）
- 验证 diff 算法正确性
- 验证配置下发真的生效

## 故障排查

### 常见问题
1. **Reconcile 不触发** → 检查 Predicate 是否过滤掉了事件，检查 EventSource 是否正确启动
2. **连接不上设备** → 检查 ClientPool 中设备信息是否正确，检查协议端口（NETCONF 830, gNMI 9339）
3. **diff 出了错误的变更** → 检查列表键提取是否正确，检查节点类型是否匹配
4. **队列卡住** → 检查 Reconciler 是否panic，检查是否有死锁，查看日志

### 调试
- 使用 `manager.WithDebug()` 开启调试日志
- 检查 controller 工作队列长度：`controller.Queue().Len()`
- 检查 client 连接状态：`client.IsConnected()`

## 与原有架构对比（迁移参考）

| 原有 Actor 架构 | 新 controller-runtime 架构 |
|----------------|---------------------------|
| 每个设备一个 DeviceActor | 连接池复用设备连接，无 Actor |
| 每个 YANG 对象一个 Actor | 一个 Controller 处理所有设备的同类型对象 |
| 异步消息通信 | 同步 reconcile 循环，框架处理排队 |
| 用户处理所有连接/重连 | 框架连接池自动处理 |

## 记住一句话
**框架处理所有 boilerplate，你只需要写 Reconcile 逻辑**
