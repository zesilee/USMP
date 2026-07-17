---
id: retire-businessvlan-bridge
title: 旧 BusinessVlan/BusinessInterface CRD 桥接退役（渐进替换收尾）
status: completed
priority: low
branch: retire-businessvlan-bridge
worktree: .claude/worktrees/retire-businessvlan-bridge
change: retire-businessvlan-bridge
updated: 2026-07-17
origin: business-network-config 收官 follow-up（存量改造军规：旧代码并行→切换→删除 的最后一步）
---

## 目标

业务意图控制器（internal/intent，BusinessVlanService CR）已接管意图面，与旧桥接并行运行中。收尾删除：

1. `internal/crdsource` 的 BusinessVlan/BusinessInterface 桥接（main.go RegisterIntentSources 调用点）。
2. `backend/api/biz/v1` 旧 CRD 类型与 `pkg/translator` 中被替代的映射（保留仍被引用的部分需先审计）。
3. `pkg/yang-runtime/actor` 若仅剩旧桥接引用则一并评估（对照 arch-optimization-roadmap 的物理删除机械债）。

## 前置

确认无环境仍在用 biz.usmp.io/BusinessVlan 旧 CR（新 Kind 是 BusinessVlanService，不冲突可共存观察一段）。

> ✅ 2026-07-16 已确认（仓库证据）：现行部署链只装 deploy/crds/（无旧 CRD），旧 CRD 清单仅存于退役 Stack A 目录；crdsource envtest 亦以「旧 CRD 未安装」为前提。本机无 kubectl 未查活集群，风险已在 design.md 评估（惰性数据无故障面）。

## 交付记录（2026-07-17 完成）

- **PR #186**（W1）：桥接摘除 + translator 退役 + DR-04 门禁切换；**PR #187**（W2）：api/biz/v1 + Stack A 载体清理；**PR #188/#189**：旧 CRD 文档退役（因 TM04 体积从 #187 拆出）；**PR #190**（W3）：actor 整包物理删除 + 纯删除门禁豁免（insertions≤50 上限 6000，用户 2026-07-17 批准）。
- spec 已 sync（DR-04 新增 / translation-engine 全 REMOVED 留 TE-00 墓碑 / SC-01 修订），change 归档于 openspec/changes/archive/2026-07-17-retire-businessvlan-bridge/。
- 覆盖率棘轮 65.1→67.4。遗留 follow-up：e2e 集群脚手架与 backend/deploy 目录级退役、NativeDeviceConfig 清单收敛（roadmap D1）。
