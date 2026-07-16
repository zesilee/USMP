---
id: global-ha-multi-instance
title: 全局 HA——device store 共享化、全控制器 leader election、audit 迁出本地文件
status: completed
priority: medium
branch: global-ha-w1/w2/w2a/w3/w4/final（全部已合入）
worktree: .claude/worktrees/global-ha-w1
change: 已归档（2026-07-16 交付：PR#175 W1 兜底收敛+种子迁env、#177+#176 W2 Device CRD store、#178 W3 全控制器选主、#179 W4 audit 迁 CRD；spec 已 sync）
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

## 交付记录（2026-07-16）

- 四波次全部合入 main（#175/#177/#176/#178/#179），主 spec 已 sync（device-store/devices-api/yang-controller-runtime/system-architecture + 新能力 operation-audit）。
- **部署时验收项（✅ 2026-07-16 WSL kind 实测通过：双副本+双选主开关，单 leader/接管/跨副本可见/审计跨重启/无本地持久全过；环境=scripts/kind-deploy.sh，踩坑记录见记忆 kind-deploy-gotchas 与 PR#182-#184）**：真实集群两副本 + `USMP_INTENT_LEADER_ELECTION=1` + `USMP_NATIVE_LEADER_ELECTION=1` → 验证单 leader 日志、设备/审计跨副本可见、杀 leader 接管。多副本性质已由 envtest 双实例矩阵覆盖（双 store watch、双 gate 接管、并发清理收敛）。
