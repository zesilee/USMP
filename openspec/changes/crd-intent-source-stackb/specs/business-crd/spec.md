## ADDED Requirements

### Requirement: 业务 CRD 意图经 Stack B 下发

业务 CRD（`api/biz/v1`：BusinessVlan/Interface/Route/Switch）声明的意图 SHALL 经 Stack B 的 CRD 意图源下发：`CRD → translator → desired ygot → ConfigStore → GenericReconciler → NETCONF`，替换 Stack A 的 Actor/2PC 下发链路。CRD 作为用户意图声明面契约保持不变。

#### Scenario: 业务 VLAN 意图下发
- **WHEN** 创建/更新一个 BusinessVlan CR
- **THEN** 其意图 SHALL 经 translator → ConfigStore → reconcile 下发到目标设备（`spec.deviceID`），最终 NETCONF edit-config+commit

#### Scenario: 下发链路等价（双路径验证）
- **WHEN** 同一 CRD Spec 分别经旧 Actor 路径与新 CRD-source 路径处理
- **THEN** 两者产出的 desired ygot SHALL 语义等价，落到设备的配置一致

### Requirement: CRD 树收敛到 api/biz/v1

business CRD 类型 SHALL 统一到 `api/biz/v1`+`api/core/v1`（唯一有生成 CRD YAML 的一套）；退役 `api/v1`，消除 `biz.usmp.io/v1` group 抢注与 schema 不兼容（迁移债 D1）。

#### Scenario: 单一 CRD 树注册
- **WHEN** 系统注册 business CRD 类型
- **THEN** SHALL 仅注册 `api/biz/v1`（+`api/core/v1`），`api/v1` 不再被引用

## REMOVED Requirements

### Requirement: Actor/2PC 下发子系统

**Reason**: Actor/2PC（`pkg/yang-runtime/actor`：mailbox + candidate/commit 两阶段 + 版本快照）违反 R01（明文禁 Actor 模型），是双栈半迁移的 legacy 部分（迁移债 D2）。其下发与事务价值由 Stack B 的 `GenericReconciler`（edit-config + commit）承接。

**Migration**: 每个 business CRD 先经 CRD 意图源并行接入 Stack B 并双路径验证 desired 等价，再删除该 CRD 对 Actor 的调用；全部迁移后删除 `pkg/yang-runtime/actor` 包与 `cmd/controller` 入口。跨设备事务性（若业务依赖）在 reconciler 层以批量 commit 承接或标注为能力差异。
