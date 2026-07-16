---
id: global-ha-multi-instance
title: 全局 HA——device store 共享化、全控制器 leader election、audit 迁出本地文件
status: in_progress
priority: medium
branch: (未开始，apply 时按波次开 worktree)
worktree: (未创建)
change: openspec/changes/global-ha-multi-instance（proposal/design/specs/tasks 四件齐，2026-07-16 立项）
updated: 2026-07-16
origin: business-network-config 收官 follow-up（SC-06 遗留）；部署约束见记忆 k8s-paas-deployment-constraints
---

## 目标

USMP 多实例（≥2 副本）真 HA：意图层已就绪（CR 持久化 + leader election 接缝 USMP_INTENT_LEADER_ELECTION），但存量仍单实例假设——

1. **device store 共享化**：`pkg/yang-runtime/device` 内存注册表 → CRD/共享存储（reconcile-conninfo-debt 的根治面）。
2. **全控制器 leader election**：原生周期控制器（vlan/ifm/system/bgp/ni/…）N 副本会对同一设备双份下发 NETCONF，需统一选主（可复用 intent/leader.go 的 gateSources 模式）。
3. **audit 迁出本地文件**：manager.WithAuditFile 本地 JSON 违反 SC-06（多实例禁本地持久），迁 CR events 或共享后端。

## 上下文

- SC-02/SC-06（openspec/specs/system-architecture）已把约束写死；deploy/README 标注了本任务边界。
- intent 包的 gateSources/leaderGatedSource 是现成选主接缝范式。
