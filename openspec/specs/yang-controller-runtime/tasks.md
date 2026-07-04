# yang-controller-runtime — 差异 / 补全清单（反向还原）

> as-built 与目标的差异 + 待办，非实施步骤。

## spec 与代码差异

- [ ] **plugin 从不被调用**：`process`/`GenericReconciler.Reconcile`/Actor 均不调 Validate/Mutate/Pre/PostReconcile（`plugin/*`）
- [ ] **schema 层运行时为空**：`main.go:22` 未设 `SchemeDir`，diff 传 `schema.Schema=nil`
- [ ] **ConfigStore.List/ListDevices = stub**：返回 nil,nil（`manager.go:55,61`）→ PeriodicSource 无法枚举设备
- [ ] **Source 接口不统一**：`gnmi_sub.go`/`file.go` 的 `Stop()` 不返回 error，不满足 `controller.Source`
- [ ] **`DoneWaitGroup` 非线程安全**：无锁 int 计数（`source/periodic.go:24`，R09 风险）
- [ ] **worker Get() goroutine 可能泄漏**：stop 竞态下阻塞 goroutine 存活（`controller.go:149`）
- [ ] **两套 Reconciler 形态并存**：GenericReconciler vs actor.ActorReconciler，DiffEngine 签名不一致需适配
- [ ] **`New` 死分支**：if/else 两支都调 NewSchema()（`manager.go:116`）

## 改进建议

- [ ] 接通 plugin 钩子（Validate/Mutate/Pre/PostReconcile）到 reconcile 流
- [ ] 加载 schema（设 `SchemeDir`）使 diff/路径校验 schema 感知
- [ ] 实现 ConfigStore.List/ListDevices 使周期源可枚举设备
- [ ] 统一 Source.Stop() 签名；用 `sync.WaitGroup` 替换 `DoneWaitGroup`
- [ ] 使 `queue.Get()` 支持 context，移除 worker 的 goroutine hack
