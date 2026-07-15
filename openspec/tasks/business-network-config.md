---
id: business-network-config
title: 业务网络配置——业务侧 YANG 模型定义网络自动化能力，USMP 编排为原生配置下发
status: in_progress
priority: medium
branch: worktree-business-network-config
worktree: .claude/worktrees/business-network-config
change: (explore 进行中 → /opsx:propose 立项)
updated: 2026-07-15
origin: 用户拍板 2026-07-13；概念更名与前端死路退役见 change native-config-reposition
---

## 目标（概念定义，用户拍板原文语义）

- **原生配置**（已交付，本任务的地基）：直接基于 YANG 模型对交换机进行配置管理 = 模块控制台 `/module/:name` + config-api + driver registry + xmlcodec + gen-yang 全链路。
- **业务网络配置**（本任务）：在业务侧基于 YANG 模型定义网络自动化配置能力（意图模型）；USMP 将业务自动化配置模型**编排**为原生配置下发。

```
业务网络配置（意图层）: 业务 YANG 模型（如「业务VLAN打通」「园区接入」）
        │ USMP 编排引擎：意图 → 多设备/多模块原生配置展开
        ▼
原生配置（设备层）: 华为 YANG 模型 → NETCONF 下发（现有全链路，不动）
```

## 架构落位思路（explore 输入，非结论）

- **意图模型也是 YANG**：业务自动化能力用自定义 YANG 模块定义 → 复用 gen-yang manifest 管线（加一条 gen.conf）+ ygot 生成 + R05 自动渲染（业务模型零前端代码得到表单/控制台）+ 前端「业务网络配置」菜单组（与「原生配置」并列，代码标识符 business* 已清场）。
- **意图也走 Reconciler**：业务模型实例作为 desired，Reconciler 的「对齐」= 展开为原生配置并经现有原生链路下发/收敛（C3 用户只写编排逻辑，框架管 diff/事件/重试）。
- **编排 = 模型间映射**：意图模型 → N×(设备, 原生模块, 配置片段)；与 translator（厂商翻译）不同层——编排在意图→原生模型层，翻译在原生模型→设备方言层（已由 driver registry 承担）。
- **历史教训**（[[dual-stack-migration]]、business-crd spec）：Stack A 的 BusinessVlan/BusinessRoute CRD 意图面思想同源但死于 K8s CRD 载体——本任务**必须**长在 Stack B（yang-controller-runtime），禁止复活 CRD 通道（R01/R03）。
- **待 explore 的关键问题**：意图模型的存储与生命周期（R03 无数据库——意图实例存本地 JSON 元信息？）；多设备事务语义（部分失败怎么回滚/呈现）；意图与原生配置的归属标记（原生侧被意图管理的字段是否锁定）；首个业务能力选型（建议从「跨设备 VLAN 打通」起步——VLAN+IFM 原生链路最成熟）。

## 上下文恢复提示

- 地基（全部已交付，别重建）：通用模块控制台（[[generic-module-console]]）、driver registry + xmlcodec + gen-yang（[[snd-driver-registry]]）、删除命令通道（[[config-delete-semantics]]）、约束引擎（[[yang-constraint-engine]]）。
- 前端命名已清场：菜单「原生配置」= 模块控制台；`business*` 标识符已释放给本任务（change native-config-reposition）。
- 相关 spec：frontend（FE-03/FE-10/FE-13）、config-api、yang-controller-runtime、device-driver-registry、yang-codegen-pipeline；business-crd 为 legacy 历史契约仅供思想参考。

## 恢复指令

1. 新会话：`/task resume business-network-config`。
2. 启动时 `EnterWorktree` → `/opsx:explore`（先解决「待 explore 的关键问题」四项，尤其意图实例存储与 R03 的关系、首个业务能力选型）→ `/opsx:propose`。
3. 每阶段完成照例 sync + 归档 + 回写本文件。
