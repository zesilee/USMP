---
id: business-network-config
title: 业务网络配置——业务侧 YANG 模型定义网络自动化能力，USMP 编排为原生配置下发
status: completed
priority: medium
branch: (已合入 main，分支可清理)
worktree: .claude/worktrees/business-network-config
change: archive/2026-07-16-business-network-config（九段 PR #164-#172 全部合入 main，2026-07-16 收官）
updated: 2026-07-16
origin: 用户拍板 2026-07-13；2026-07-15 拍板部署约束（K8s 多实例、CRD 公共契约、跨设备 2PC）见记忆 k8s-paas-deployment-constraints
---

## 交付状态（2026-07-15，实现全部完成）

- **意图管线**：usmp-business-vlan.yang（唯一 schema 源）→ ygot（generated/business）+ CRD OpenAPI（tools/crdgen，deploy/crds 生成物 + regen-and-diff 门禁）+ /yang/schema 表单渲染 + task-name=business-network 菜单 category 四路派生。
- **编排**：internal/intent——expand 纯函数（BIO-02）、TxCoordinator 跨设备 2PC（prepare→discard-all / confirmed-commit(60s)→confirming；能力缺失降级普通 commit 标非事务）、事务成功才写 desired（合并防抹除）+ TriggerReconcile、周期 resync 对冲 desired TTL、finalizer 删除 + 收缩差集（差集只依赖 CR status）、软归属索引。
- **协议层**：netconfsim confirmed-commit 仿真（NS-07）+ 客户端 CommitConfirmed/ConfirmCommit（DP-08）；顺手修 sim 单条目 keyed list merge 折叠 bug（wellKnownListKeys）。
- **API**：/business/vlan-services CRUD 代理（写前约束校验，无集群 503 降级）、/ownership/:device、SetConfig/Delete 附 ownershipWarning。
- **前端**：业务菜单组（category 分桶零硬编码）、/business/:module 平台作用域控制台（列表/表单/详情三态，复用 useConfigForm+FieldRenderer）、原生控制台归属徽标 + 手改警告；F1/F2/F3/F4 全层绿（make e2e-local 13/13）。
- **部署**：deploy/crds + deploy/rbac + README（CRD→RBAC→应用滚动）；leader election 接缝（USMP_INTENT_LEADER_ELECTION，默认关）。
- **spec**：8 个 delta 已 sync 入主 spec（validate 27/27）；CLAUDE.md R03/§8 措辞已修订；覆盖率棘轮 后端 65.1 / 前端 80/75/74/80。

## 收官记录（2026-07-16）

九段 PR（#164-#172）串行全部合入 main；change 已归档；follow-up 三项已立独立任务（见 openspec/tasks/）。

## 原剩余步骤（已完成，留档）

1. **PR 分段合入**（TM04，串行，前段 merge 后开下一段——#129 死分支坑）：
   - s1 = **PR #164**（立项制品 + netconfsim confirmed-commit）——已开，等 CI + merge。
   - s2..s6 建议切法：s2 客户端 2PC 原语+crdgen 实现；s3 crdgen 测试+意图模型管线；s4 展开引擎+意图控制器；s5 2PC 下发+生命周期+软归属；s6 API 代理+前端+部署+spec 同步。每段 ≤1000 行（pr-size 排除 generated/）。
   - 分段方式：`git branch business-network-config-sN <worktree 分支上的对应提交>` + push + PR（复用 s1 正文格式）。
2. 全部 merge 后：`/opsx:archive business-network-config` 归档 change；`/task archive business-network-config`。
3. Follow-up 任务（已在本文件登记，立项时另开 openspec/tasks）：
   - **全局 HA**：device store 上共享存储、全控制器 leader election、audit 迁出本地文件（SC-06 遗留）。
   - **归属硬锁二期**：config-api 命中认领路径 409 拒绝（当前软归属仅警告）。
   - **旧 BusinessVlan 桥接退役**：internal/crdsource 的 BusinessVlan/BusinessInterface 与 pkg/translator 被意图控制器替代后删除（渐进替换收尾）。

## 上下文恢复提示

- 拍板与坟场边界：记忆 [[k8s-paas-deployment-constraints]]（CRD 当载体不当架构、2PC、多实例约束）。
- 制品：openspec/changes/business-network-config/（proposal/design/specs/tasks/test-matrix 全齐）。
- 关键实现坑已记录在提交信息：驱动注册空白导入、池按 IP 键控（测试用 loopback 别名）、sim 单-单 merge 折叠、fake client status:nil、confirming 部分失败残余窗口。

## 恢复指令

1. `/task resume business-network-config` → `EnterWorktree`（path=.claude/worktrees/business-network-config）。
2. 查 `gh pr list` 分段进度，按「剩余步骤 1」继续切段/开 PR；CI 绿后自助 merge（落地工作方式记忆）。
3. 全段合入后走归档（剩余步骤 2）并立 follow-up 任务（步骤 3）。
