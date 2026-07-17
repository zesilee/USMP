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

## 上下文恢复提示（2026-07-16 W1/W2 已出 PR）

- change 四件制品齐全且已提交：`openspec/changes/retire-businessvlan-bridge/`。
- **W1 = PR #186**（分支 retire-businessvlan-bridge，CI 全绿）：桥接摘除 + translator 退役 + DR-04。**W2 = PR #187**（堆叠分支 retire-businessvlan-bridge-w2，base=#186）：biz/v1 + Stack A 载体清理。**两个 PR 的合入均被 auto-mode 拦截，需用户 review 后自行 merge**（先 #186 后 #187，#187 合前 rebase/重定向 main）。
- **W3 被 tasks 3.0 阻塞**：actor 4718 行整包删除需 pr-size（CI）+ commit-msg（本地钩子）纯删除豁免（insertions≤50 上限 6000，design D5）；该 TM04 契约变更的提交被权限分类器拦截，须用户显式批准/自行提交后 W3 才能动。
- 关键决策：D2 vendor 门禁切 driver 注册表；D3 载体一并删；D4 actor 整包一次删；覆盖率棘轮 65.1→67.4。
- 恢复指令：EnterWorktree path=.claude/worktrees/retire-businessvlan-bridge → 确认 #186/#187 合入状态 → 若 3.0 已批准则开 W3（新分支基于 main），否则先向用户要决策。
