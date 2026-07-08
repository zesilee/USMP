---
id: optimize-frontend-nce-insights
title: 基于 iMaster NCE 洞察优化 USMP 前端渲染 + YANG UI 契约（4 阶段路线图）
status: in_progress
priority: medium
branch: (未开始 — 每阶段独立 worktree)
worktree: (未创建)
change: (每阶段独立 OpenSpec change)
updated: 2026-07-08
evidence: docs/research/imaster-nce-ux-insights.md
---

## 目标

把华为 iMaster NCE 调研的可迁移洞察（`docs/research/imaster-nce-ux-insights.md` 洞察 A–H）分 4 个递进阶段落地，
优化 USMP 的 R05（YANG 自动渲染）与南向驱动架构。**排序原则：价值高 × 爆炸半径小者先行。**
**不 big-bang**：每阶段一个独立 worktree + 一个 OpenSpec change + 一个 ≤1000 行 PR，spec-first（R17）。

本任务是**跨会话总跟踪器**（防止长周期迭代中漏做后续阶段）；每阶段真正开工时再各自 `/opsx:propose` 建 change。

## 4 阶段路线图与进度

### [ ] Phase 1 — 渲染器补全 + 设计 token（洞察 G + F）｜先做，风险最低
- 渲染器补全类型→控件映射（对照 evidence §2.2）：`enum≤3→el-segmented`、`enum>3→el-select`、
  `list→内嵌子表格+Create/Delete+空态`、`leafref→el-transfer`、`container→el-collapse/Advanced`
- 抽取设计 token：浅色 + 华为蓝 `#307FE2` 主色 + 绿色状态色 + 高密度表格
- spec-first：刷 `frontend` spec「YANG 类型映射」能力（新增 segmented/transfer/内嵌 list 分支）
- 测试层（§5.6）：F2 组件单测（每控件 add/edit/remove/校验态）+ F3 真浏览器（transfer 弹层、嵌套 list 增删改）
- 纯前端、零契约变更、爆炸半径最小

### [ ] Phase 2 — IA 收敛：视图切分 + 下发 Stepper（洞察 D + H）
- 大 YANG 模型按 `container` 自动切 Tab/视图，不再一屏平铺
- 「编辑→校验→下发→回读收敛」可视化成 4 步 `el-steps`，接上收敛台账/新鲜度环
- 依赖 Phase 1 控件就位；测试层 F2 + F4 staging-smoke

### [ ] Phase 3 — UI 注解契约层（洞察 A + B）｜战略核心
- `/yang/schema` 输出加轻量 UI 注解：`ui:widget`/`ui:view`/`ui:order`/`ui:hidden`
- 注解纳入前端契约生成 + 漂移门禁（[[frontend-contract-gen]]），消灭手写表单残留
- **⚠️ 需用户在 explore 阶段拍板的架构分叉：注解存哪？**
  - YANG extension（走约束引擎同路 goyang `Entry.Extra`，但 choice 会被 ygot 拍平 — 见 [[yang-constraint-engine]]）
  - Sidecar JSON（每模块 `*.ui.json`，简单无 codegen 依赖，运行镜像可带）
  - **建议先 sidecar JSON 起步**验证价值，再评估升级 YANG extension
- 依赖 Phase 1 定型（先知道注解成哪些 widget）；测试层 B1+B3+F1/F2
- 存量改造军规：旧硬映射保留→双跑→切换（§5.3）

### [ ] Phase 4 — 声明式 SND 驱动描述 + 删除语义模型化（洞察 C + E）｜可延后
- 每设备型号一个声明式驱动描述，加设备只加一个包不碰核心（参照华为 SND）
- list 级联删/幂等语义模型化（参照 `delete-siblings-on-delete`），而非硬编码组件
- 动 huawei translator/Reconciler 核心，爆炸半径最大；当前 VLAN/IFM 链路已跑通（[[vlan-config-stackb]]）不急
- 测试层 B1+**B2 集成**；涉新模型接入触发 `yang-config-test-design` 完备矩阵（T02b）

## 上下文恢复提示

- **地基（别重建）**：已有 `frontend-yang-dynamic-form` 渲染器、`/yang/schema` 接口、前端契约生成+漂移门禁、
  DeviceConfigPage 泛化流、YANG 约束引擎（when/must/pattern/range 数据驱动已交付 PR#116）。本路线是增量 delta。
- **现有粗映射**：当前只有 boolean→开关、enum→下拉；Phase 1 就是补全到 evidence §2.2 全表。
- **视觉基线**：已合入的前端 redesign（[[frontend-redesign]]）是「浅色 iMaster NCE 气质」，Phase 1 把它从视觉落到控件级保真。
- **最大战略洞察**：华为把「渲染意图」编码进模型元数据，前端渲染器退化成哑终端；R05 终局是「渲染意图成为受版本管理的契约」= Phase 3。
- 相关记忆：`[[imaster-nce-ux-insights]]`、`[[frontend-contract-gen]]`、`[[frontend-redesign]]`、`[[yang-constraint-engine]]`、`[[vlan-config-stackb]]`。

## 恢复指令

1. 新 session：`/task resume optimize-frontend-nce-insights`，读 `docs/research/imaster-nce-ux-insights.md` 补齐上下文。
2. 找到第一个未勾选 `[ ]` 的 Phase，`EnterWorktree` 建隔离环境 → `/opsx:explore` 该阶段（审计存量、标 legacy/新架构边界）→ `/opsx:propose`。
3. 每阶段完成：`/opsx:sync` + 合 PR + 回本文件把该 Phase 勾成 `[x]` 并 `/task sync`。
4. 全部 4 阶段 `[x]` 后 `/task archive optimize-frontend-nce-insights`。
5. Phase 3 开工前必须先让用户拍板「注解存哪」（YANG extension vs sidecar JSON）。
