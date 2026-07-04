## Why

场景②「业务/网络自动化配置」经 K8s CRD 声明意图、后端翻译为设备配置——目前这条链路是 **Stack A（legacy）**：`cmd/controller`（controller-runtime 独立进程）+ `pkg/translator` + **Actor/2PC**（`pkg/yang-runtime/actor`）+ etcd。它违反 R01（明文禁 Actor，且 Actor 是代码量最大的子系统、生产在用，迁移债 D2）。架构优化 P2 把意图面收编到 R01 权威的 **Stack B 单进程**，退役 Actor，使两个配置面（P1 已完成的设备原生面 + 本次意图面）汇入同一 `ConfigStore → GenericReconciler → NETCONF` 核心。

## What Changes

- **新增 K8s CRD EventSource（C4）**：`backend/main.go`（Stack B）用已有依赖 controller-runtime/client-go watch 各 business CRD；CRD 变更时 `translator.TranslateConfig(Spec → 厂商 ygot desired)` → `ConfigStore.Set(deviceID, path, desired)` → `Enqueue` → 既有 `GenericReconciler` 统一 diff + 下发（意图 desired **投影进内存 ConfigStore**，etcd 仅存意图声明，符 R03）。
- **迁移 4 个 business CRD**（`api/biz/v1`：BusinessVlan/Interface/Route/Switch）从 Actor 路径到 CRD-source 路径；Route/System 翻译 stub 补齐或显式标注。
- **BREAKING（内部）退役 Actor**：删除 `pkg/yang-runtime/actor`（2PC/mailbox/版本快照）与其在 `controllers/*` 的调用（迁移债 D2，合 R01）。
- **切换生产入口**：`backend/main.go` 单进程跑全部 CRD 源 + 设备原生面 + 北向 API；退役 `cmd/controller` 入口。
- **收敛 CRD 树**：退役 `api/v1`（旧），统一到 `api/biz/v1`+`api/core/v1`（唯一有生成 YAML 的一套），解 `biz.usmp.io/v1` group 抢注冲突（迁移债 D1）。
- **迁移策略（§5.3）**：新 CRD-source 路径与旧 Actor 路径**并行 → netconfsim 端到端双路径验证 desired 等价 → 切换入口 → 删除 Actor/Stack A 入口/旧 CRD 树**。不碰场景①（P1 已完成）。

## Capabilities

### New Capabilities
<!-- 无新能力域：K8s CRD 源归入 yang-controller-runtime 的 C4 EventSource 能力。 -->

### Modified Capabilities
- `yang-controller-runtime`: 新增 K8s CRD EventSource（C4）——CRD 变更经 translator 投影 desired 进 ConfigStore 并触发 reconcile；生产入口收敛为单进程（backend/main.go 跑 CRD 源 + 原生面）。
- `business-crd`: CRD 意图**下发链路**契约从「controller-runtime + Actor/2PC（Stack A）」变为「Stack B CRD-source → ConfigStore → GenericReconciler」；CRD 收敛到 `api/biz/v1`+`api/core/v1`，退役 `api/v1`。
- `actor-transaction`: **移除**——Actor/2PC 子系统整体退役（合 R01），其事务价值由 reconciler 层承接。
- `translation-engine`: 翻译入口不变（`TranslateConfig`），消费方从 Actor 改为 CRD-source；Route/System 翻译从 stub 补齐或标注。

## Impact

- **后端**：`pkg/yang-runtime/source`（新增 KubernetesCRDSource）、`backend/main.go`（注册 CRD 源 + 单入口）、`controllers/*`（去 Actor，逻辑迁入 CRD 源/reconciler）、`pkg/yang-runtime/actor/*`（删除）、`cmd/controller/main.go`（删除）、`api/v1`（删除）、`pkg/translator`（补 Route/System 或标注）。
- **测试**：CRD 源单测（translate→ConfigStore→enqueue）；netconfsim 端到端双路径 desired 等价（Actor vs CRD-source）；退役后全量 `go test ./...` 绿。
- **红线**：R01（收编到 Stack B、删 Actor）、R03（意图投影内存 ConfigStore、不为运行配置建库）、R04（ygot desired）、R06（TDD）。
- **不在范围**：场景①（P1 已完成）、gNMI/plugin/多厂商翻译扩展（P3）。
- **对外契约**：K8s CRD API（用户声明意图的接口）保持；内部下发机制替换；迁移期新旧并行，切换后删旧。
