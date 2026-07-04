# yang-controller-runtime — 行为契约（反向还原）

> 反向还原自 `backend/pkg/yang-runtime/`，忠实 as-built。详细架构见 `design.md`。权威栈（R01）。

## 能力概述

声明式配置对齐框架（C1–C5）：把设备 actual 配置向 desired 对齐。用户实现 C3 `Reconciler`，框架承担连接/排队/限频/反射 diff/协议编解码。

## 行为契约

### YR-01 期望态触发对齐
- **Given** 已 `ConfigStore.Set(deviceID:path, desired)`
- **When** 调用 `Manager.TriggerReconcile(deviceID, path)` 或 `PeriodicSource` 到期
- **Then** 事件经 predicate 过滤后入 `RateLimitingQueue`，worker 调用对应 `Reconciler.Reconcile`

### YR-02 diff-then-push
- **Given** desired 与 actual 存在差异
- **When** `GenericReconciler.Reconcile` 执行
- **Then** 反射 diff 产出 `[]Change`，经 `DeviceClient.Set` 下发（edit-config + commit）；desired 为 nil 时视为 no-op（**不删除**）

### YR-03 失败重投带退避
- **Given** Reconcile 返回 error 或 Requeue
- **When** `process` 处理 Result
- **Then** `AddRateLimited`（指数退避 1s–30s + 令牌桶 10qps）或 `AddAfter(RequeueAfter)`；成功则 `Forget`

### YR-04 每模块一控制器
- **Given** 一个 YANG 模块（vlan/ifm/system）
- **When** 经 `ControllerManagedBy(name).WithReconciler(...).WithSource(...).Build()` 注册
- **Then** 独立事件队列 + worker 池处理该模块，模块间隔离

### YR-05 事件源多样性
- **Given** 需要产生 reconcile 事件
- **When** 配置 Source
- **Then** 支持 PeriodicSource(轮询)/GNMISubSource(订阅)/FileSource(文件变更)

## 契约缺口（详见 design.md §5）

- plugin 钩子声明存在但**从不被执行**；schema 层运行时为空；`ConfigStore.List/ListDevices` 为 stub。

## 关联
- `design.md`、`device-protocol/spec.md`（C5）、`config-cache/spec.md`（ConfigStore）、根 `yang-controller-runtime.md`。
