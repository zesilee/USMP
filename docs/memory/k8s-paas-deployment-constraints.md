---
name: k8s-paas-deployment-constraints
description: USMP 部署形态与持久化约束（2026-07-15 用户拍板）：K8s 内 PaaS 底座组件、必须多实例、禁止本地存储、持久化=K8s CRD——影响一切存储/HA 设计
metadata: 
  node_type: memory
  type: project
  originSessionId: 7cd790ff-8926-44da-99c5-7224d50e7e1b
---

USMP 部署到 K8s 里作为 PaaS 底座的一部分（管理 PaaS 基础设施的交换机）。用户 2026-07-15 拍板的硬约束：

1. **不可能是单实例**（可靠性要求）→ 任何"本地 JSON/本地文件"持久化方案都不成立（CLAUDE.md §8"元信息=本地 JSON"与 R03 措辞需随 spec 修订）。
2. **可用的持久化能力 = K8s CRD**（etcd via apiserver），无需也不引入数据库。
3. **CRD 定位精确切分**：Stack A 死于"CRD 当架构"（双重建模+Actor+前端消费 CRD）；允许的是"CRD 当持久化载体 + watch 事件源（C4 Source）"，Stack B 对账引擎不变。任务文件旧注"禁止复活 CRD 通道"按此重新解释。
4. business-network-config 拍板集：CR 是**对外公共契约**（kubectl/GitOps 直写受支持）→ 形态 B：**YANG→CRD OpenAPI codegen**（when/must 仍由约束引擎收敛侧兜底）；多设备下发=**跨设备 2PC**（candidate+confirmed-commit，事务成功才写 desired，稳态走声明式对账）；首能力=跨设备 VLAN 打通；归属=软归属（认领存 CR status+徽标+警告不拦截）。

**全局 HA 已交付（2026-07-16，change 归档 2026-07-16-global-ha-multi-instance，PR#175-#180）**：device store=Device CRD+凭据 Secret 引用+watch 镜像（`device.NewCRDStore`，无集群降级内存）、建连兜底收敛为 `device.ResolveConn` 单点（conninfo-debt 根治）、种子设备迁 `USMP_SEED_DEVICE`（集群模式忽略）、5 原生周期控制器统一选主（`pkg/yang-runtime/leader.Gate`，单 Lease `usmp-native-controllers`，开关 `USMP_NATIVE_LEADER_ELECTION`；intent 面复用同实现）、audit=AuditRecord CRD（每条一 CR、幂等清理，`USMP_AUDIT_FILE` 退役仅警告）。**部署验收已实测通过（2026-07-16 WSL kind，scripts/kind-deploy.sh 双副本五步演练全过）**。四项拍板：Secret 引用凭据/单全局 Lease/专用 Audit CRD/单 change 四波次。**follow-up 已全部清零（2026-07-17）**：retire-businessvlan-bridge（PR #186-#191，Stack A 载体归零）、ownership-hard-lock（PR #192-#193，认领路径手改缺省信封 409 拒绝 + force 逃生审计留痕 + 前端阻断确认流；BR-11 已重题「归属硬锁」）。

**交付状态（2026-07-16 收官）**：business-network-config **全部合入 main 并归档**——九段 PR #164-#172 串行合入 + #173 归档收官；change 在 openspec/changes/archive/2026-07-16-business-network-config；follow-up 三项已立独立任务（openspec/tasks/：global-ha-multi-instance、ownership-hard-lock、retire-businessvlan-bridge）。前端 functions 覆盖率阈值最终 73（CI 环境比本地低约 1 点，贴边即 flaky）。分段合入教训：pr-size 按「排除生成物后 插入+删除」计，切段前必须用同口径实测（P1 曾 1001/1000 翻车重切）。**E2E 不需要装 K8s**：envtest（setup-envtest 两个二进制）已接 compliance CI（PR#174）覆盖 A7 写入时拒绝+finalizer 真实生命周期；kind 级多副本验证归 global-ha-multi-instance。实现坑：驱动注册靠空白导入、ClientPool 按 IP 键控（双 sim 测试用 loopback 别名）、netconfsim 单-单 keyed merge 折叠已修（wellKnownListKeys）、fake client 普通 Update 会写 status:nil 导致 SetNestedField 静默失败。

**Why**: 存储/HA/契约设计若按单机本地文件思路走会整体返工；CRD 复用历史上有精确的坟场边界。

**How to apply**: 涉及持久化、多实例、意图层、CRD 的任何设计先对照本条；相关 [[dual-stack-migration]]、[[vlan-config-stackb]]、[[reconcile-conninfo-debt]]。
