# actor-transaction — 行为契约（反向还原）

> 反向还原自 `backend/pkg/yang-runtime/actor/`，忠实 as-built。详见 `design.md`。**⚠️ 与 R01 冲突，legacy，但生产在用。**

## 能力概述

消息驱动的单写者(actor)配置管理：每设备/每模块串行写入，提供 candidate/commit 2PC、快照版本、回滚。由 K8s controller-runtime 驱动（数据流路径 A）。

## 行为契约

### AT-01 串行邮箱
- **Given** 并发配置消息到达同一模块
- **When** 发往 `ModelActor`
- **Then** 单 goroutine 顺序处理（mailbox cap 100），写入无锁竞争；邮箱满 5s 超时

### AT-02 两阶段提交
- **Given** 需下发配置
- **When** 依次 Prepare→Commit
- **Then** Prepare 写 candidate（`txActive` 守卫）；Commit 经 checksum 守卫后 commit(running) 并快照；失败 Abort→DiscardCandidate

### AT-03 跨模块事务
- **Given** 一台设备多个模块需原子变更
- **When** `DeviceActor.PrepareAndCommitAll`
- **Then** 逐模块 Prepare，全成功则逐模块 Commit；任一 Prepare 失败尽力 AbortAll

### AT-04 版本回滚
- **Given** 已提交若干版本快照
- **When** `RollbackToVersion(n)`
- **Then** 从 `VersionManager`（SHA256 校验和 + JSON 深拷贝，cap 50）恢复该版本并下发

### AT-05 状态读回
- **Given** 下发完成
- **When** `StatusQueryCmd`
- **Then** 从设备读回 actual 供 CR.Status 写回（Phase=Synced）

## 契约缺口（详见 design.md §6）

- 模块路由 stub（`extractModuleFromPath` 恒 "default"）；`ReflectTranslator.ToPayload` 未实现；框架泄漏 Huawei 模型特判。

## 关联
- `design.md`、`business-crd/spec.md`（驱动方）、`translation-engine/spec.md`、`yang-controller-runtime/spec.md`（权威替代栈）。
