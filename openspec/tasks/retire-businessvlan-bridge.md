---
id: retire-businessvlan-bridge
title: 旧 BusinessVlan/BusinessInterface CRD 桥接退役（渐进替换收尾）
status: pending
priority: low
branch: (未开始)
worktree: (未创建)
change: (启动时 /opsx:propose 立项)
updated: 2026-07-16
origin: business-network-config 收官 follow-up（存量改造军规：旧代码并行→切换→删除 的最后一步）
---

## 目标

业务意图控制器（internal/intent，BusinessVlanService CR）已接管意图面，与旧桥接并行运行中。收尾删除：

1. `internal/crdsource` 的 BusinessVlan/BusinessInterface 桥接（main.go RegisterIntentSources 调用点）。
2. `backend/api/biz/v1` 旧 CRD 类型与 `pkg/translator` 中被替代的映射（保留仍被引用的部分需先审计）。
3. `pkg/yang-runtime/actor` 若仅剩旧桥接引用则一并评估（对照 arch-optimization-roadmap 的物理删除机械债）。

## 前置

确认无环境仍在用 biz.usmp.io/BusinessVlan 旧 CR（新 Kind 是 BusinessVlanService，不冲突可共存观察一段）。
