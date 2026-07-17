---
id: retire-businessvlan-bridge
title: 旧 BusinessVlan/BusinessInterface CRD 桥接退役（渐进替换收尾）
status: in_progress
priority: low
branch: retire-businessvlan-bridge
worktree: .claude/worktrees/retire-businessvlan-bridge
change: retire-businessvlan-bridge
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

> ✅ 2026-07-16 已确认（仓库证据）：现行部署链只装 deploy/crds/（无旧 CRD），旧 CRD 清单仅存于退役 Stack A 目录；crdsource envtest 亦以「旧 CRD 未安装」为前提。本机无 kubectl 未查活集群，风险已在 design.md 评估（惰性数据无故障面）。

## 上下文恢复提示

- change 四件制品齐全：`openspec/changes/retire-businessvlan-bridge/`（proposal/design/specs/tasks），`openspec validate` 通过。
- 三波 PR 交付（design.md D6）：W1 桥接摘除+translator 退役 → W2 biz/v1+Stack A 载体 → W3 actor 整包删除。进度看 change 的 tasks.md 勾选。
- 关键决策：D2 vendor 门禁切 driver 注册表（新增 DR-04 VendorSupported）；D5 pr-size 纯删除豁免（insertions≤50 上限 6000，TM04 契约变更需用户 PR review 确认）。
- 恢复指令：EnterWorktree path=.claude/worktrees/retire-businessvlan-bridge → `/opsx:apply retire-businessvlan-bridge` 按 tasks.md 续做。
