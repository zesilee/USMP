## ADDED Requirements

### Requirement: K8s CRD EventSource（意图投影进 ConfigStore）

框架 SHALL 提供一个 K8s CRD 事件源（C4），使 Stack B 单进程能消费 K8s CRD 声明的意图：监听某 CRD GVK，在 CRD 增改时经 translator 翻译为 desired ygot、写入内存 ConfigStore、并触发 reconcile，从而 CRD 意图与设备原生配置汇入同一 `ConfigStore → GenericReconciler → NETCONF` 核心。SHALL NOT 依赖 Actor/2PC 子系统，SHALL NOT 为运行配置引入数据库（R03：意图在 K8s、运行配置在内存）。

#### Scenario: CRD 增改触发翻译投影与 reconcile
- **WHEN** 被监听的 CRD 发生 add/update
- **THEN** 源 SHALL 用 translator 将其 Spec 翻译为厂商 ygot desired、`ConfigStore.Set(deviceID, path, desired)`、并 Enqueue 该 deviceID/path 的 reconcile

#### Scenario: CRD 删除触发清除与 reconcile
- **WHEN** 被监听的 CRD 被删除
- **THEN** 源 SHALL `ConfigStore.Delete(deviceID, path)` 并 Enqueue reconcile（对齐删除语义）

#### Scenario: 参数化复用
- **WHEN** 为不同 CRD（GVK/厂商/configType/deviceID·path 提取）注册源
- **THEN** SHALL 复用同一源类型，一 CRD 一实例

### Requirement: 生产入口单进程收敛

系统 SHALL 收敛为单一生产入口（`backend/main.go`）同时运行 CRD 意图源、设备原生面与北向 API；退役独立的 controller-runtime 入口（`cmd/controller`）与 Actor 子系统。

#### Scenario: 单进程跑意图面与原生面
- **WHEN** 启动 `backend/main.go`
- **THEN** SHALL 注册全部 business CRD 意图源 + 设备原生 reconciler，经同一 Manager 生命周期管理
