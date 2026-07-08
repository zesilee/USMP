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

把华为 iMaster NCE 调研的可迁移洞察（`docs/research/imaster-nce-ux-insights.md` 洞察 A–H）落地，
优化 USMP 的 R05（YANG 自动渲染）与南向驱动架构。**不 big-bang**：每阶段独立 worktree + OpenSpec change + ≤1000 行 PR，spec-first（R17）。

本任务是**跨会话总跟踪器**（防止长周期迭代中漏做后续阶段）；每阶段真正开工时再各自 `/opsx:propose` 建 change。

## ⚠️ 对账结论（2026-07-08，与 [[generic-module-console]] 存量核对后校准）

**原路线图是盲写的，与已交付/在途的 `generic-module-console` 分支重叠严重，剩余工作量被大幅高估。经代码核对（非仅记忆）校准：**

- **前置：`generic-module-console` 分支（未合 main，基于 52a5a36=#119）先合入。** 它交付了模型驱动通用控制台 `/module/:module`（左导航 /yang/modules 驱动 + 面包屑+一级 Tab，零 per-module 前端代码），并**退役** DeviceConfigPage/frontend-yang-dynamic-form。本路线图所有阶段都应叠在它之上，**不要**再基于已退役组件。
- **P1 渲染器 ~80% 已交付**：`FieldRenderer.vue` 已映射 string→input / number→input-number / boolean→switch / enum→select / group / **presence→switch** / leaf-list / **choice→radio** / list。剩余仅**保真微调**（enum≤3→segmented、leafref→transfer、设计 token），不是一个完整 Phase。
- **P2 视图切分已交付**：模块根派生 Tab + 左导航已在 module console 里。仅「下发 Stepper」可能是新增，且当前是单个「下发」按钮 → 低优先。
- **P3 注解机制已在跑**：华为 `ext:support-filter`/`ext:operation-exclude` 已从 goyang `Entry.Exts` 消费，FieldDef 已带 supportFilter/operationExclude/presence/isKey，`make gen-contract` 漂移门禁已存在。**「注解存 YANG extension vs sidecar JSON」的分叉已被现实解答：走 YANG extension（Entry.Exts/Extra）。** P3 从「绿地战略赌注」降级为「扩展已验证的注解词汇表」，风险大减。
- **P4 删除语义债已定位**：删除按钮按门禁渲染但**禁用**（后端 /config 无 DELETE、POST 合并语义）→ 精确已知起点。声明式 SND 仍是绿地后端。

**校准后的真实路线（见下方 [x]/[ ] 已更新）：先合 module-console，再做 P1' 保真微调（可选低优先），P3' 扩展注解词汇（去风险后的主线），P4 删除语义+SND（最后）。P2 基本已完成。**

## 路线图与进度（已按对账校准）

### [ ] P0 — 前置：合入 `generic-module-console` 分支｜真正的下一步
- 未合 main，基于 52a5a36（=#119），PR 前须 `rebase --onto origin/main 52a5a36`（[[generic-module-console]] 记）
- 它交付模型驱动通用控制台 + FieldRenderer + 注解消费管线，是后续所有阶段的地基
- **本任务后续阶段全部叠在它合入之后**；未合入前不要另起 P1'/P3'

### [~] P1' — NCE 保真微调（洞察 G + F）｜大部分已由 FieldRenderer 交付，仅剩微调，低优先
- [x] 类型→控件主映射已在 `FieldRenderer.vue`（string/number/boolean/enum/group/presence/leaf-list/choice/list）
- [ ] enum≤3 → `el-segmented`（当前恒 el-select）— UX 保真，非缺能力
- [ ] leafref → `el-transfer`（双栏穿梭，多对多引用场景才需，先核实是否有该场景）
- [ ] 设计 token：华为蓝 `#307FE2` 主色 + 绿色状态色 + 高密度表格（与 [[frontend-redesign]] 对齐、量化 evidence §2.1）
- spec-first：刷 `frontend` spec 对应能力；测试层 F2 +（transfer/segmented 若加）F3 真浏览器

### [x] P2 — 视图切分：基本已由 module console 交付
- [x] 大模型按模块根/container 派生 Tab + 左导航（module console 已实现）
- [ ] （可选低优先）「编辑→校验→下发→回读收敛」下发 `el-steps` Stepper（当前是单「下发」按钮）；接收敛台账/新鲜度环

### [ ] P3' — 扩展注解词汇表（洞察 A + B）｜去风险后的主线
- 机制已在跑：`ext:support-filter`/`ext:operation-exclude` 从 `Entry.Exts` 消费 → FieldDef → `make gen-contract` 漂移门禁
- **分叉已解答**：走 YANG extension（Entry.Exts/Extra），不再纠结 sidecar JSON
- 剩余：按需扩展新注解关键字（如 `ext:ui-widget`/`ext:ui-order`/`ext:ui-view` 控制 widget/排序/视图切分），复用现有消费+门禁管线
- ⚠️ choice 元数据被 ygot 拍平（[[yang-constraint-engine]]），涉 choice 的注解需构建期 codegen 兜底
- 测试层 B1（扩展解析）+ B3（schema API 契约）+ F1/F2（渲染器消费）；存量并行→切换（§5.3）

### [ ] P4 — 删除语义模型化 + 声明式 SND（洞察 E + C）｜最后
- [ ] 删除语义：已知精确起点——删除按钮按门禁渲染但**禁用**（后端 /config 无 DELETE、POST 合并语义）。补后端 DELETE/删除下发契约 + 前端启用行删除（[[generic-module-console]] follow-up 债）
- [ ] 声明式 SND 驱动描述（绿地）：每设备型号一个声明式描述，加设备只加一个包不碰核心（参照华为 SND）
- 动 huawei translator/Reconciler 核心，爆炸半径最大；VLAN/IFM 链路已跑通（[[vlan-config-stackb]]）不急
- 测试层 B1+**B2 集成**；涉新模型接入触发 `yang-config-test-design` 完备矩阵（T02b）

## 上下文恢复提示

- **地基（别重建）**：模型驱动通用控制台 `/module/:module` + `FieldRenderer.vue`（[[generic-module-console]] 分支，未合 main）、
  `/yang/schema` 接口、前端契约生成+`make gen-contract` 漂移门禁、YANG 约束引擎（PR#116）、
  华为 `ext:support-filter`/`ext:operation-exclude` 注解消费管线。**DeviceConfigPage/frontend-yang-dynamic-form 已在 module console 里退役——别再基于它们。** 本路线是增量 delta。
- **渲染器现状（已核对代码）**：`FieldRenderer.vue` 已映射 string/number/boolean→switch/enum→select/group/presence→switch/leaf-list/choice→radio/list。剩余仅 enum≤3→segmented、leafref→transfer、设计 token 等保真微调（P1'）。
- **视觉基线**：已合入的前端 redesign（[[frontend-redesign]]）是「浅色 iMaster NCE 气质」，P1' 把它从视觉落到控件级保真。
- **最大战略洞察**：华为把「渲染意图」编码进模型元数据，前端渲染器退化成哑终端；R05 终局是「渲染意图成为受版本管理的契约」= Phase 3。
- 相关记忆：`[[imaster-nce-ux-insights]]`、`[[frontend-contract-gen]]`、`[[frontend-redesign]]`、`[[yang-constraint-engine]]`、`[[vlan-config-stackb]]`。

## 恢复指令

1. 新 session：`/task resume optimize-frontend-nce-insights`，读 `docs/research/imaster-nce-ux-insights.md`（洞察证据）+ [[generic-module-console]] 记忆（存量地基）补齐上下文。
2. **先看 P0**：`generic-module-console` 分支是否已合 main？未合则先推进它合入（它是所有后续阶段的地基）。
3. P0 合入后，找第一个未勾选 `[ ]` 的阶段，`EnterWorktree` → `/opsx:explore`（审计存量、确认在 module console/FieldRenderer 上做增量而非新建）→ `/opsx:propose`。**主线是 P3'（扩展注解词汇表）；P1' 保真微调低优先、按需做。**
4. 每阶段完成：`/opsx:sync` + 合 PR + 回本文件把该阶段勾成 `[x]` 并 `/task sync`。
5. 全部阶段 `[x]` 后 `/task archive optimize-frontend-nce-insights`。
6. ~~注解存哪的分叉~~ 已解答：走 YANG extension（Entry.Exts/Extra），复用现有 support-filter/operation-exclude 消费管线。
