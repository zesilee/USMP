# actor-transaction — Actor/2PC/版本子系统架构设计（反向还原）

> **权威性**：⚠️ **与 R01 冲突（明文禁止 Actor 模型），标注为 `legacy`**——但它是当前代码量最大、且被生产控制器 `backend/controllers/*` 依赖的子系统。裁定 legacy ≠ 已退役，见 `system-architecture/design.md` §7。
> **还原基准**：`main@b1cfbae`，代码根 `backend/pkg/yang-runtime/actor/`。

## 1. 职责

以**消息驱动的单写者(actor)**模型管理每设备/每 YANG 模块的配置：串行化写入避免锁竞争，提供 candidate/commit 两阶段提交(2PC)、快照版本管理与回滚。驱动方**不是** yang-runtime `Manager`，而是 Kubernetes controller-runtime（`backend/controllers/*` + `cmd/controller/main.go`）。这构成 §3.2 的**数据流路径 A**。

## 2. 组件

### 2.1 `ModelActor[T YANGGoStruct]` — `actor/model_actor.go:70`
- 泛型于 ygot 结构类型的**邮箱 actor**：缓冲 `msgChan`（cap 100，`:138`），单 `runMessageLoop` goroutine **顺序处理**（`:225`），每消息一个 buffered promise。`Send` 有 5s 邮箱满超时（`:200`）。
- 状态 `desired/actual/state` 受 `sync.RWMutex`；`messageCount` 用 `atomic`。
- 消息类型（`actor/message.go`）：`TranslateCmd/ValidateCmd/PrepareCmd/CommitCmd/AbortCmd/RollbackCmd/ApplyCmd/StatusQueryCmd`。
- 处理器：`handleTranslate`（`:287`，Translator+ygot Validate+快照）、`handleApply`（`:333`，全量替换+commit）、`handlePrepare`（`:575`，2PC-1 candidate 写，`txActive` 守卫）、`handleCommit`（`:706`，checksum 守卫+commit+快照）、`handleAbort`（`:794`，DiscardCandidate）、`handleRollback`（`:847`，从 VersionManager 恢复）、`handleStatusQuery`（`:897`）。
- 设备 I/O `fetchActualFromDevice`（`:398`）：重反射 + XML 反序列化，含 Huawei 专有 `HuaweiVlan_Vlan_Vlans`/`HuaweiIfm_Ifm_Interfaces` 特判（`:456,495`）——**框架泄漏设备模型细节**。`parseDeviceID` 支持 `user:pass@ip:port`（`:955`）。

### 2.2 `DeviceActor` — `actor/device_actor.go:14`
- 协调一台设备下 `map[moduleName]Actor`（RWMutex），按模块路由消息。**路由是 stub**：`extractModuleFromPath` 恒返回 `"default"`（`:264`），回退到「首个模块」（`:159`）。
- 跨模块 **2PC 协调器**：`PrepareAll`/`CommitAll`/`AbortAll`/`PrepareAndCommitAll`（`:347,443,521,596`），`DeviceTransactionState`（`:333`）；prepare 失败尽力 abort。

### 2.3 `VersionManager[T]` — `actor/version.go:33`
快照历史（默认 cap 50），SHA256 校验和 + JSON `deepCopy`（`:190`），GetByNumber/Checksum、RollbackToVersion；RWMutex。

### 2.4 `Translator[T]` / `ReflectTranslator[T]` — `actor/translator.go`
反射 payload→ygot 映射，kebab→Camel，YANG-list-as-map 特判（`translateMapToListEntry` `:315`）。`ToPayload` 为未实现 stub（`:86`）。

### 2.5 `ModuleFactory` / `ActorReconciler` / `ActorManager`
- `ModuleFactory` `actor/module_factory.go:9`：硬编码创建 Huawei 的 `vlans`/`interfaces`/`system` actor。
- `ActorReconciler` `actor/reconciler.go:15`：把 `reconcile.Reconciler` 适配为 actor 消息（Translate 然后 Apply，`:35`）。
- `ActorManager` `actor/reconciler.go:125`：`map[deviceID]*DeviceActor`（RWMutex），`GetDeviceActor` 自动注册模块（`:145`）。

## 3. 数据流（路径 A，legacy）

```
K8s CRD 变更 (etcd Watch)
  → controllers/BusinessVlanReconciler.Reconcile
       translator.TranslateConfig(bizv1.Spec → huawei ygot)   # 翻译引擎
  → ActorManager.GetDeviceActor(ip)
       DeviceActor → ModelActor(mailbox)
         TranslateCmd → PrepareCmd(candidate) → CommitCmd(running)   # 2PC
         [失败] AbortCmd → DiscardCandidate
  → StatusQueryCmd 读回 → CR.Status.Phase=Synced，requeue 5min
```

## 4. 并发模型

actor-per-module 单 goroutine 邮箱 = 业务逻辑串行、无锁竞争；`DeviceActor`/`ActorManager` 用 RWMutex map；promise 为 buffered channel；2PC 逐模块顺序、NETCONF candidate + `DiscardCandidate` 实现。

## 5. 与 R01 的冲突及处置

- **R01**：「禁止回退 Actor 模型」。本子系统正是 Actor 模型，且承载 2PC/版本这类 Stack B `GenericReconciler` 目前不具备的能力。
- **处置建议**（不在本次决策范围）：这些能力（事务、版本回滚）需在退役前评估是否迁入 Stack B 的 Reconciler/plugin 钩子，否则直接删除会丢功能。属 `system-architecture/design.md` §8「确立单栈」的前置。

## 6. as-built 缺口

| 缺口 | 位置 |
|------|------|
| 模块路由 stub（恒 "default"→首模块） | `device_actor.go:264,159` |
| `ReflectTranslator.ToPayload` 未实现 | `translator.go:86` |
| 框架泄漏 Huawei 模型特判 | `model_actor.go:456,495` |
| `internal/controller/vlan/actor_reconciler.go` 的 `k8sConfigStore` 为 stub | `actor_reconciler.go:243` |

## 7. 关联
- `business-crd/design.md`（驱动方 CRD 控制器）、`translation-engine/design.md`（翻译层）、`yang-controller-runtime/design.md`（权威替代栈）。
