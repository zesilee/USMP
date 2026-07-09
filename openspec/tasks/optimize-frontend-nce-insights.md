---
id: optimize-frontend-nce-insights
title: 基于 iMaster NCE 洞察优化 USMP 前端渲染 + YANG UI 契约 + 异构多设备 SND（分阶段路线图）
status: in_progress
priority: medium
branch: (未开始 — 每阶段独立 worktree)
worktree: (未创建)
change: (每阶段独立 OpenSpec change)
updated: 2026-07-08
evidence: docs/research/imaster-nce-ux-insights.md
p0_done: generic-module-console 已合入 main (#121/#123/#124, 2026-07-08)
p3_done: ext-ui-annotations 已合入 main (#126) + sync + 归档 (2026-07-08)
p4_done: config-delete-semantics 已合入 main (#128) + sync + 归档 (2026-07-09)
next: 剩余 P1' 保真微调（低优先、按需）或 P5 异构多设备 SND（战略项、独立 explore/propose）
---

## 目标

把华为 iMaster NCE 调研的可迁移洞察（`docs/research/imaster-nce-ux-insights.md` 洞察 A–H）落地，
优化 USMP 的 R05（YANG 自动渲染）与南向驱动架构。**不 big-bang**：每阶段独立 worktree + OpenSpec change + ≤1000 行 PR，spec-first（R17）。

本任务是**跨会话总跟踪器**（防止长周期迭代中漏做后续阶段）；每阶段真正开工时再各自 `/opsx:propose` 建 change。

## ⚠️ 对账结论（2026-07-08，与 [[generic-module-console]] 存量核对后校准；P0 已合入后二次刷新）

**原路线图是盲写的，与 `generic-module-console` 重叠严重、剩余工作量被大幅高估。经代码核对（非仅记忆）校准：**

- **✅ P0 已完成：`generic-module-console` 已合入 main（PR #121 呈现扩展透出 → #123 通用控制台 → #124 sync+归档）。** 它交付了模型驱动通用控制台 `/module/:module`（左导航 /yang/modules 驱动 + 面包屑+一级 Tab，零 per-module 前端代码），并**退役** DeviceConfigPage/frontend-yang-dynamic-form（`/config/interface`·`/config/vlan` 已 redirect 到 `/module/*`）。本路线图所有剩余阶段都叠在它之上，**不要**再基于已退役组件。
- **P1 渲染器 ~80% 已交付**：`FieldRenderer.vue` 已映射 string→input / number→input-number / boolean→switch / enum→select / group / **presence→switch** / leaf-list / **choice→radio** / list。剩余仅**保真微调**（enum≤3→segmented、leafref→transfer、设计 token），不是一个完整 Phase。
- **P2 视图切分已交付**：模块根派生 Tab + 左导航已在 module console 里。仅「下发 Stepper」可能是新增，且当前是单个「下发」按钮 → 低优先。
- **P3 注解机制已在跑**：华为 `ext:support-filter`/`ext:operation-exclude` 已从 goyang `Entry.Exts` 消费，FieldDef 已带 supportFilter/operationExclude/presence/isKey，`make gen-contract` 漂移门禁已存在。**「注解存 YANG extension vs sidecar JSON」的分叉已被现实解答：走 YANG extension（Entry.Exts/Extra）。** P3 从「绿地战略赌注」降级为「扩展已验证的注解词汇表」，风险大减。
- **P4 删除语义债已定位**：删除按钮按门禁渲染但**禁用**（后端 /config 无 DELETE、POST 合并语义）→ 精确已知起点。声明式 SND 仍是绿地后端。

**校准后的真实路线（P0 已合入，均已代码核验 @origin/main）：主线 = P3' 扩展注解词汇（机制已在跑、去风险）；P1' 保真微调（segmented/transfer/token 均未做，低优先）；P2 视图切分已完成、仅下发 Stepper 可选；P4 删除语义（删除按钮仍禁用、后端 /config 无 DELETE 已确认）；P5 异构多设备 SND（从 P4 拆出的战略项——骨架已在、要泛化+声明式化，Go 后端不阻塞，独立立项）。**

## 路线图与进度（已按对账校准）

### [x] P0 — 前置：合入 `generic-module-console`｜已完成
- ✅ 已合入 main：PR #121（呈现扩展透出 support-filter/operation-exclude/presence/isKey）→ #123（通用控制台 FE-10~13）→ #124（sync 主 spec + 归档）
- 交付模型驱动通用控制台 `/module/:module` + `FieldRenderer.vue` + `ext:*` 注解消费管线，是后续所有阶段的地基
- 覆盖率棘轮已上调：后端 57、前端 73/70/65/73（改这块前对齐，见 [[test-governance-military-rules]]）

### [~] P1' — NCE 保真微调（洞察 G + F）｜大部分已由 FieldRenderer 交付，仅剩微调，低优先
- [x] 类型→控件主映射已在 `FieldRenderer.vue`（string/number/boolean/enum/group/presence/leaf-list/choice/list）
- [ ] enum≤3 → `el-segmented`（当前恒 el-select）— UX 保真，非缺能力
- [ ] leafref → `el-transfer`（双栏穿梭，多对多引用场景才需，先核实是否有该场景）
- [ ] 设计 token：华为蓝 `#307FE2` 主色 + 绿色状态色 + 高密度表格（与 [[frontend-redesign]] 对齐、量化 evidence §2.1）
- spec-first：刷 `frontend` spec 对应能力；测试层 F2 +（transfer/segmented 若加）F3 真浏览器

### [x] P2 — 视图切分：基本已由 module console 交付
- [x] 大模型按模块根/container 派生 Tab + 左导航（module console 已实现）
- [ ] （可选低优先）「编辑→校验→下发→回读收敛」下发 `el-steps` Stepper（当前是单「下发」按钮）；接收敛台账/新鲜度环

### [x] P3' — 扩展注解词汇表（洞察 A + B）｜已完成（PR #126 合入 + sync + 归档，2026-07-08）
- 机制已在跑：`ext:support-filter`/`ext:operation-exclude` 从 `Entry.Exts` 消费 → FieldDef → `make gen-contract` 漂移门禁
- **分叉已解答**：走 YANG extension（Entry.Exts/Extra），不再纠结 sidecar JSON
- ✅ 2026-07-08 apply 完成（change `ext-ui-annotations`，PR #126）：经审计重划范围为**收割存量**（用户拍板）——S1 config-false→readonly+前端只读降级（还债#3）/ S2 dynamic-default 占位 / S3 units 后缀 / S4 task-name 构建期 codegen→category→左导航分组。**零自造 ui-* 词汇**（无消费场景+fork 维护债，决策在 design.md）。关键实证：模块级扩展不存活于运行期 schema（全树扫描=0）→ S4 走 tasknamegen 构建期生成
- ✅ 收尾完成：delta 已 sync 主 spec（yang-api BR-09/BR-10+BR-01、frontend FE-14/FE-15+FE-13）、change 归档 archive/2026-07-08-ext-ui-annotations；覆盖率棘轮升至 后端 57.8 / 前端 74/71/67/74
- ~~choice 拍平顾虑~~ 已被 P3 实测推翻（choice/case 完整保留），本期未涉 choice 注解
- 测试层 B1（扩展解析）+ B3（schema API 契约）+ F1/F2（渲染器消费）；存量并行→切换（§5.3）

### [x] P4 — 删除语义模型化（洞察 E）｜已完成（PR #128 合入 + sync + 归档，2026-07-09）
- ✅ 2026-07-09 apply 完成（change `config-delete-semantics`，PR #128）：DELETE 显式命令通道（BR-09/BR-10）+ NETCONF 键式删除编码（DP-07）+ 前端行删除（FE-16）。关键实证：声明式通道被 walkMap merge/subset 语义刻意封死→删除必须走命令通道；顺带交付 netconfsim RFC edit-config 接线 + NETCONF 客户端 opMu 写事务串行化（R09）
- ✅ 收尾完成：delta 已 sync 主 spec（config-api BR-09/BR-10、device-protocol DP-07、frontend FE-16）、change 归档 archive/2026-07-09-config-delete-semantics；覆盖率棘轮升至 后端 58.3

### [ ] P5 — 异构多设备 SND 驱动：泛化 + 声明式化（洞察 C）｜战略项，独立立项
> 从 P4 拆出（原被低估地捆在删除债里）。USMP 后续对接异构多厂商设备的**核心能力**，值得独立 explore/propose，非"最后随手做"。

- **纠正认知：不是绿地。SND 等价骨架已在 Go 后端**（2026-07-08 核对代码）：
  - `Translator` 接口（`pkg/translator/translator.go:31`：Vendor/TranslateVlan/Interface/Route/System/Validate）
  - 驱动注册表 `RegisterTranslator`+`GetTranslator`+`map[VendorType]Translator`（`factory.go:9,14,19`）
  - 厂商枚举**已内置** Huawei/Cisco/H3C/Juniper（`translator.go:13-16`），只实现了 Huawei
  - plugin 钩子接口 Validation/Mutation/Notification/ReconciliationHook（`plugin/plugin.go`）
  - → 要做的是**泛化 + 声明式化**（把手写 `TranslateXxx` 胶水变数据驱动），不是从零建
- **Go 后端可行性已定（后端是 Go 不阻塞 SND）**，三条地道路径按契合度：
  - ① **声明式数据驱动**（推荐终态）：设备特定部分=数据非代码（ygot 生成厂商 YANG 结构 R04 + 路径/模板描述符 + sysoid/协议/能力元数据），加设备=加数据+重跑 ygot，核心几乎不动。最契合 R04/R05
  - ③ **编译期注册表**（现状，起步）：保留 RegisterTranslator，每厂商一个 Go 包，重编译。够用于可控厂商集合
  - ② **进程外 gRPC 插件**（逃生舱，Terraform provider/HashiCorp go-plugin 模式）：语言无关，**驱动甚至可以是 Python**，真热插拔但最重；仅"托管第三方/热加载"才需
  - **建议 ③ 起步 → ① 成熟；不需要 Python，也不需要 Go .so 插件**
- **⚠️ 待厘清（explore 阶段）**：现有 `Translator` 吃 CRD Spec（`bizv1.BusinessInterfaceSpec`）= **Stack A 遗留**；SND 设计必须瞄准 **Stack B（yang-controller-runtime 的 Reconciler + 通用引擎）**，别复活 CRD translator（[[dual-stack-migration]]）
- **⚠️ 待用户拍板的设计决策**：驱动是否需要"运行时可插拔/第三方热加载"？否→③/①（纯 Go 编译期）；是→②（gRPC 进程外）
- 动 translator/Reconciler 核心，爆炸半径最大；VLAN/IFM 链路已跑通（[[vlan-config-stackb]]）；测试层 B1+**B2 集成**；新模型接入触发 `yang-config-test-design` 完备矩阵（T02b）

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
