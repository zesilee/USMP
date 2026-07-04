# yang-controller-runtime — 框架架构设计（反向还原）

> **权威性**：✅ **权威目标架构**（CLAUDE.md R01）。本能力是 §4 分层 C1–C5 的实现。
> **还原基准**：`main@b1cfbae`，`file:line` 引用以 `backend/pkg/yang-runtime/` 为根。
> **上层导航**：`openspec/specs/system-architecture/design.md`。

## 1. 职责

一个 Go 语言实现、刻意仿照 Kubernetes controller-runtime 的**声明式配置对齐框架**：把「设备当前配置(actual)」向「期望配置(desired)」对齐。框架承担全部 boilerplate（连接管理、事件排队、限频重试、反射 diff、协议编解码），用户理论上只需实现 C3 `Reconciler`。约 8,200 LoC（不含测试）。

## 2. 组件（C1–C5 + 支撑包）

### C1 `manager` — 全局生命周期
- `Manager` 接口 `manager/manager.go:73`：Start/Stop/AddController/GetSchema/GetClientPool/GetConfigStore/GetPluginManager/AddPlugin/**TriggerReconcile**。
- `DefaultManager` `manager.go:96`；`New(...Option)` `manager.go:109`；函数式选项 `manager/options.go`。
- `InMemoryConfigStore` `manager.go:19`：实现 `reconcile.ConfigStore`，底层为 `internal/cache.TTLLRUCache`（1000 条 / 1min 清理 / **5min** TTL，`manager.go:125`），key = `deviceID:path`（`manager.go:32`）。详见 `config-cache/design.md`。
- `TriggerReconcile(deviceID, path)` `manager.go:236`：按 **controller 名对硬编码路径子串匹配**（`"system:"` / `"vlan:"|"vlans"` / `"ifm:"|"interfaces"`）路由到 controller 后 `Enqueue`。
- `Start` `manager.go:141`：若设 `SchemeDir` 则加载 schema，然后逐个 `ctrl.Start(ctx)`；`Stop` `manager.go:170`：停 controllers → `clientPool.CloseAll()` → cancel。

### C2 `controller` — 事件循环 + worker 池
- `Controller` 接口 `controller/controller.go:24`（Start/Stop/Enqueue/Name）；`Source` 接口在此声明 `controller.go:14`。
- `DefaultController` `controller.go:36`；流式 `Builder`：`ControllerManagedBy(name).WithReconciler/WithSource/WithPredicate(s)/WithWorkerCount/WithQueue/Build`（`controller/builder.go`）。默认队列 `queue.NewRateLimitingQueue(DefaultRateLimiter())`。
- `Enqueue` `controller.go:125`：按事件类型跑 predicates，通过则 `Event.ForRequest()` → `queue.Add`。
- `worker` `controller.go:149`：**每次迭代额外起一个 goroutine** 调用阻塞的 `queue.Get()`，以便 `select` 于 `stopChan`——因为队列 `Get()` 不支持 context。
- `process` `controller.go:181`：`RequeueAfter>0 → AddAfter`；`Requeue → AddRateLimited`；`Error → AddRateLimited`；否则 `Forget`。

### C3 `reconcile` — 用户契约 + 通用引擎
- **`Reconciler` 接口 `reconcile/reconcile.go:10` — 主用户扩展点**（`Reconcile(ctx, Request) Result`）；`ReconcilerFunc` 适配器。
- 支撑接口：`ConfigStore`（desired，`reconcile.go:27`）、`DeviceClient`（actual，`reconcile.go:42`）、`DiffEngine`（`reconcile.go:58`）。
- `GenericReconciler` `reconcile.go:51`；`Reconcile` `reconcile.go:77`：get desired → nil 视为 no-op（**不删除**）→ get actual → diff → 有变更则 `dc.Set`；任何 error 包装为 `*ReconcileError` 并 `Requeue:true`。
- `Request{DeviceID,Path}` / `Result{Requeue,RequeueAfter,Error}` / `Change` / `ReconcileError`：`reconcile/request.go`。

### C4 `source` — 事件源
- `PeriodicSource` `source/periodic.go`：`time.Ticker` 逐 deviceID 发 `GenericEvent`。
- `GNMISubSource` `source/gnmi_sub.go`：包 `client.Subscribe`，通知→`UpdateEvent`。
- `FileSource` `source/file.go`：`fsnotify` + 100ms 去抖。

### C5 `client` — 连接池 + 协议客户端
详见 `openspec/specs/device-protocol/design.md`。要点：`ClientPool` 每设备 IP 一个持久 `Client`；`DefaultClientFactory` 按 Protocol/端口派发 NETCONF/gNMI。

### 支撑包
- `queue`：K8s 式工作队列——阻塞基队列 + 延迟重投（min-heap）+ 每项指数退避 + 令牌桶。`DefaultRateLimiter` = Max(exp 1s–30s, bucket 10qps/100)（`queue/rate_limit.go:175`）。
- `predicate`：Create/Update/Delete/Generic 过滤 + And/Or/Not 组合 + Prefix/Exact/Contains 路径谓词。
- `diff`：**基于反射**的 ygot 结构树 diff，list 按 `*Key` 字段匹配，`pruneChanges` 剔除已增删父节点的后代（`diff/diff.go`）。注意 `Diff(...schema.Schema)` 的 schema 参数实际未用。
- `schema`：YANG schema 树 + 路径缓存 + Loader。
- `plugin`：Validation/Mutation/Notification/ReconciliationHook 四类注册表。

## 3. 装配与数据流（Stack B，`backend/main.go`）

```
manager.New → InMemoryConfigStore(TTL-LRU) + DefaultClientPool + plugin.Manager
  → ControllerManagedBy("vlan").WithReconciler(vlan.New(cs,pool))
       .WithSource(PeriodicSource(5m, nil, "/vlan:vlan/vlan:vlans"))
       .WithPredicate(predicate.Prefix(...)).WithWorkerCount(2).Build()
  → mgr.AddController → mgr.Start
Ticker → Source.EnqueueEvent → Controller.Enqueue(predicates)
  → RateLimitingQueue.Add → worker.Get
  → GenericReconciler.Reconcile
       ConfigStore.Get(desired) + DeviceClient.Get(actual, NETCONF)
       → DiffEngine.Diff → DeviceClient.Set(edit-config + commit)
  → Result: requeue / rate-limit / forget
```
`internal/controller/{vlan,ifm,system}` 各自 `New(cs, pool)` 返回内嵌 `GenericReconciler` 的实例，提供 `deviceClient`（NETCONF Get/Set）+ `diffEngineAdapter`（因 `reconcile.DiffEngine(...path string)` 与 `diff.DiffEngine(...schema.Schema)` 签名不同需适配，`internal/controller/vlan/reconciler.go:21,27`）。

## 4. 并发模型

- 每 controller `workerCount` 个 worker goroutine（默认 1；main 用 2）+ 每次 `Get()` 一个临时 goroutine。
- 队列：`sync.Mutex` + `atomic.Bool` shutdown；延迟层 min-heap 后台轮询 ≤100ms。
- 限频器：`map` + `sync.Mutex`，指数退避带 20% 抖动。
- schema/plugin 注册表：`sync.RWMutex`。

## 5. as-built 缺口 / 空转件（诚实标注）

| 缺口 | 位置 | 说明 |
|------|------|------|
| **plugin 从不被调用** | `plugin/*` | 四类插件可注册，但 `process`/`GenericReconciler.Reconcile`/Actor 均不调用 Validate/Mutate/Pre/PostReconcile。扩展点空转。 |
| **schema 层运行时为空** | `main.go:22`（`SchemeDir` 未设） | diff 纯靠反射；diff 适配器传 `schema.Schema = nil`（`vlan/reconciler.go:27`）。 |
| **ConfigStore.List/ListDevices = stub** | `manager.go:55,61` 返回 `nil,nil` | `PeriodicSource(deviceIDs=nil)` 无法枚举设备。 |
| **Source 接口不统一** | `gnmi_sub.go`/`file.go` 的 `Stop()` 不返回 error | 不满足 `controller.Source`，无法直接传给 Builder。 |
| **`DoneWaitGroup` 非线程安全** | `source/periodic.go:24` | 无锁 int 计数，潜在竞态（R09 风险）。 |
| **worker 的 Get() goroutine 可能泄漏** | `controller.go:149` | stop 竞态下阻塞在 `Get()` 的 goroutine 可能存活。 |
| **两套 Reconciler 形态并存** | `GenericReconciler` vs `actor.ActorReconciler` | 见 `actor-transaction/design.md`；后者属 legacy 栈。 |
| **`New` 死分支** | `manager.go:116` | if/else 两支都调 `NewSchema()`。 |

## 6. 扩展点

用户**应当**只实现 C3 `Reconciler`（`reconcile.go:10`）。现实中 `internal/controller/*` 的「用户 reconciler」只是内嵌 `GenericReconciler` 并提供 `DeviceClient` 适配器——框架做了 diff/push 主体，故 CLAUDE.md「用户只需实现 C3」在当前代码里**被高估**：schema/plugin 基本休眠，而最庞大的 Actor/2PC 子系统根本不在 C1–C5 叙事内（见 `actor-transaction/design.md`）。

## 7. 关联
- 根 `yang-controller-runtime.md`（设计宣言，忠实基线）、`spec/openconfig-vlan-controller.md`（VLAN 控制器范例）。
- `device-protocol/design.md`（C5）、`config-cache/design.md`（ConfigStore 后端）、`actor-transaction/design.md`（legacy 并行栈）。
