## REMOVED Requirements

### Requirement: Actor 事务下发（mailbox + 2PC + 版本快照）

**Reason**: `pkg/yang-runtime/actor`（DeviceActor/ModelActor mailbox、Translate/Prepare/Commit 两阶段提交、版本快照）违反 R01（明文禁止 Actor 模型），且是双栈半迁移的最大 legacy 子系统（迁移债 D2）。架构优化 P2 将场景②意图面收编到 R01 权威的 Stack B（`GenericReconciler`），Actor 无存在必要。

**Migration**: 见 `business-crd` 与 `yang-controller-runtime` delta——CRD 意图改经 Stack B CRD 意图源（translator → ConfigStore → GenericReconciler）下发。迁移遵 §5.3：每 CRD 并行接入 + netconfsim 双路径验证 desired 等价 → 切换生产入口 → 删除 Actor 包、`controllers/*` 的 Actor 调用与 `cmd/controller` 入口。两阶段提交语义由 reconciler 的 edit-config+commit 承接；跨设备事务如有业务依赖则在 reconciler 层批量 commit 或记为能力差异。
