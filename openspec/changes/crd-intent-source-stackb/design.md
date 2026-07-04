# crd-intent-source-stackb — design（P2 场景② 意图面收编 Stack B）

> change：`crd-intent-source-stackb` | 依赖：`proposal.md`

## Context

USMP 双栈半迁移：Stack A（`cmd/controller` + Actor + etcd，legacy）跑场景②意图面；Stack B（`backend/main.go` + `GenericReconciler` + 内存 ConfigStore，R01 权威）跑场景①（P1 已完成）。Stack A 的 Actor/2PC 违反 R01 且是最大子系统（迁移债 D2）；`api/v1` 与 `api/biz/v1` 抢注同 group（D1）。P2 把意图面收编到 Stack B 单进程、退役 Actor。已定：统一进程（K8s CRD 作 Stack B EventSource）。总览见 `system-architecture/design.md`。

## Goals / Non-Goals

**Goals:**
- CRD 声明的意图经 Stack B 单进程下发：`CRD watch → translator → desired ygot → ConfigStore.Set → GenericReconciler → NETCONF`。
- 退役 `pkg/yang-runtime/actor`（合 R01）、`cmd/controller` 入口、`api/v1`（解 D1）。
- 全程不引入 DB（意图在 etcd/K8s、运行配置在内存 ConfigStore，R03）；desired 为 ygot（R04）；TDD（R06）。

**Non-Goals:**
- 场景①（P1 已完成，不改）。
- gNMI/plugin/多厂商翻译扩展（P3）。
- 不改 K8s CRD 用户接口（意图声明面保持）。

## Decisions

### D-1 KubernetesCRDSource：translate-and-project 源（C4）
- 实现 `controller.Source`（`Start(ctx, ctrl)`）：用 controller-runtime 的 cache/informer 或 client-go watch 监听某 CRD GVK；对每个 add/update 事件：
  1. `translator.TranslateConfig(vendor, configType, cr.Spec)` → 厂商 ygot desired。
  2. `ConfigStore.Set(deviceID, path, desired)`（deviceID/path 由参数化提取器从 CR 取，如 `spec.deviceID`）。
  3. `ctrl.Enqueue(reconcile.Request{deviceID, path})` → 既有 `GenericReconciler` diff+下发。
  4. delete 事件：`ConfigStore.Delete` + enqueue（reconcile 到空/删除语义）。
- 参数化：`(gvk, translatorVendor, configType, deviceIDFn, pathFn)`——一 CRD 一实例，复用同一源类型。
- 理由：把「意图→desired」收进 Stack B 的事件源层（C4），下发完全复用 Stack B 的 reconcile/diff/client，Actor 无存在必要。

### D-2 K8s 客户端复用现有依赖
- controller-runtime v0.19 / client-go v0.31 已是依赖（Stack A 在用）。用其 cache/manager 建 informer，避免新依赖（R10）。`backend/main.go` 启动 K8s manager 或裸 informer + Stack B manager 于同进程。

### D-3 §5.3 渐进迁移与双路径验证
- **并行**：先只加 CRD 源、注册 BusinessVlan，与 Stack A Actor 路径并存（Stack A 入口暂不删）。
- **双路径验证**：同一 CRD Spec，Actor 路径产出的 desired ygot 与 CRD-source 路径产出的 desired **对拍等价**；netconfsim 端到端：CRD-source → reconcile → netconfsim 落配，断言与 Actor 路径一致。
- **切换**：生产入口切到 `backend/main.go` 单进程跑全部 CRD 源。
- **删除**：删 `pkg/yang-runtime/actor`、`controllers/*` 的 Actor 调用、`cmd/controller`、`api/v1`。

### D-4 CRD 树收敛（解 D1）
- 统一到 `api/biz/v1`+`api/core/v1`（唯一有生成 CRD YAML）。退役 `api/v1`（旧 BusinessVlan 等）。消除 `biz.usmp.io/v1` group 抢注、schema 不兼容。

### D-5 翻译缺口
- Route 翻译返回裸 map（未完成）、System 翻译不支持（D8 相关）。迁移 Route/Switch CRD 时补齐 ygot 翻译或显式标注为受限（不阻塞 Vlan/Interface 主路径）。

## Risks / Trade-offs

- **退役 Actor 风险最高**：Actor 生产在用且代码量最大。以「并行 + 双路径 netconfsim 验证等价 → 切换 → 删」严格控制；每 CRD 独立验证后再删对应 Actor 用法，最后删 Actor 包。
- **2PC 语义**：Actor 提供 candidate/commit 两阶段 + 跨设备事务；Stack B 的 reconcile 是单设备 edit-config+commit。跨设备事务性若有业务依赖，需在 reconciler 层以批量 commit 承接或显式记为能力差异（多数场景单设备足够）。
- **K8s watch 于 Stack B 进程**：需在 `backend/main.go` 同时跑 K8s informer 与 Stack B manager，生命周期/优雅退出需协调；informer 断连重连沿用 controller-runtime。
- **CRD 树收敛破坏性**：退役 `api/v1` 若有外部 CR 实例引用旧 group/version，需迁移；本仓 `api/biz/v1` 为唯一生成 YAML，风险可控。
- **范围克制**：严格不碰场景①（P1）；Route/System 翻译缺口不阻塞主迁移。
