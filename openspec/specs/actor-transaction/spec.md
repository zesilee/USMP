# actor-transaction — Actor 单写者配置事务（candidate/commit 2PC + 版本回滚）

## Purpose

> **LEGACY / 已退出生产。** 本能力属于 **Stack A（K8s controller-runtime + Actor 模型）**，该栈**已退出生产**：生产入口 `cmd/controller` 已删除，`backend/main.go`（Stack B）是唯一入口。`backend/pkg/yang-runtime/actor/` 包代码已于 2026-07-17 **物理删除**（change retire-businessvlan-bridge，PR #190）。本 spec 作为**历史契约保留**，忠实还原 actor 包当年的代码行为；权威配置管理栈见 [[yang-controller-runtime]]。

actor-transaction 描述消息驱动的单写者（actor）配置管理契约：每设备/每模块**串行写入**，提供 candidate/commit 两阶段提交（2PC）、跨模块原子事务、SHA256 校验的快照版本与回滚、以及下发后状态读回。历史上由 K8s controller-runtime 驱动（数据流路径 A）。

**已知契约缺口**（历史遗留，见 `design.md §6，本能力已 legacy 不再修复）：模块路由为 stub（`extractModuleFromPath` 恒返回 `"default"`）；`ReflectTranslator.ToPayload` 未实现；框架层泄漏 Huawei 模型特判。

## Requirements

### Requirement: AT-01 串行邮箱

`ModelActor` SHALL 以单 goroutine 顺序处理发往同一设备/模块的配置消息（mailbox 容量 100），保证写入无锁竞争；邮箱满时投递 SHALL 在 5s 后超时并返回错误，SHALL NOT 阻塞调用方。

#### Scenario: 并发消息串行化
- **WHEN** 并发配置消息到达同一模块并发往 `ModelActor`
- **THEN** 单 goroutine SHALL 顺序处理，写入无数据竞争

#### Scenario: 邮箱满超时
- **WHEN** mailbox 已满且新消息投递
- **THEN** SHALL 在 5s 后超时返回错误，不阻塞调用方

### Requirement: AT-02 两阶段提交

单模块下发 SHALL 依次执行 Prepare→Commit：Prepare SHALL 在 `txActive` 守卫下写入 candidate；Commit SHALL 经 checksum 守卫后 commit 到 running 并生成快照；任一阶段失败 SHALL 走 Abort→DiscardCandidate 回退，SHALL NOT 提交部分配置。

#### Scenario: 正常提交
- **WHEN** 依次 Prepare 与 Commit 且均成功
- **THEN** SHALL 写 candidate 后 commit(running) 并快照该版本

#### Scenario: 提交失败回退
- **WHEN** Prepare 或 Commit 阶段失败
- **THEN** SHALL 执行 Abort→DiscardCandidate，running 保持原配置

### Requirement: AT-03 跨模块事务

`DeviceActor.PrepareAndCommitAll` SHALL 对一台设备的多个模块提供原子变更：逐模块 Prepare，全部成功后逐模块 Commit；任一模块 Prepare 失败 SHALL 尽力对已 Prepare 的模块执行 AbortAll，SHALL NOT 只提交部分模块。

#### Scenario: 全部模块成功
- **WHEN** 多模块依次 Prepare 全部成功
- **THEN** SHALL 逐模块 Commit，设备整体原子生效

#### Scenario: 某模块 Prepare 失败
- **WHEN** 任一模块 Prepare 失败
- **THEN** SHALL 尽力 AbortAll 已 Prepare 的模块，设备不发生部分提交

### Requirement: AT-04 版本回滚

`RollbackToVersion(n)` SHALL 从 `VersionManager`（SHA256 校验和 + JSON 深拷贝快照，容量 50）恢复第 n 版配置并重新下发；请求不存在的版本 SHALL 返回错误而非崩溃。

#### Scenario: 回滚到历史版本
- **WHEN** 已提交若干版本快照后调用 `RollbackToVersion(n)`
- **THEN** SHALL 从 `VersionManager` 恢复该版本配置并下发

#### Scenario: 回滚不存在版本
- **WHEN** 请求的版本号超出快照范围
- **THEN** SHALL 返回错误，SHALL NOT panic（R08）

### Requirement: AT-05 状态读回

下发完成后收到 `StatusQueryCmd` 时，actor SHALL 从设备读回 actual 状态供 CR.Status 写回（`Phase=Synced`）；读回失败 SHALL 降级返回错误而非崩溃。

#### Scenario: 下发后读回
- **WHEN** 下发完成后收到 `StatusQueryCmd`
- **THEN** SHALL 从设备读回 actual，供 CR.Status 写回 `Phase=Synced`
