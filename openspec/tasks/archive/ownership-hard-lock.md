---
id: ownership-hard-lock
title: 归属硬锁二期——config-api 命中业务意图认领路径时 409 拒绝手改
status: completed
priority: low
branch: worktree-ownership-hard-lock
worktree: .claude/worktrees/ownership-hard-lock
change: ownership-hard-lock
updated: 2026-07-17
origin: business-network-config 收官 follow-up（BIO-07 软归属一期拍板：硬锁留二期）
---

## 目标

一期软归属=认领记录（CR status/OwnershipIndex）+ 前端徽标 + ownershipWarning 警告不拦截。二期把「意图收敛覆盖手改」的静默拉锯升级为写入时硬拒绝：

1. config-api SetConfig/DeleteConfig 命中认领路径 → 409（信封码）+ 指引先删/改意图。
2. 需要逃生通道设计（如 force 参数 + 审计），避免运维死锁。
3. 前端把警告提示升级为阻断确认流。

## 上下文

- 数据面已就绪：intent.DefaultOwnership.Owners（双向前缀匹配）、GET /ownership/:device。
- BR-11（openspec/specs/config-api）明示「硬锁不在一期范围」——二期需刷 delta spec（R17）。

## 交付记录（2026-07-17 完成）

- **PR #192**：后端 409 硬锁 + force 逃生（审计 Forced/ForcedOwners 留痕、CRD 可选字段）+ 前端 ownershipGate 阻断确认流（三调用点）+ 契约再生；CI 全绿合入（用户授权 merge-on-green）。
- spec 已 sync：config-api BR-11 重题「归属硬锁」、operation-audit OA-01、frontend FE-18；change 归档 openspec/changes/archive/2026-07-17-ownership-hard-lock/。
- business-network-config 三项 follow-up 至此全部清零。
