---
name: imaster-nce-ux-insights
description: 优化 USMP 前端/YANG 自动渲染前必读：华为 iMaster NCE 调研的可迁移洞察，落在 docs/research/
metadata: 
  node_type: memory
  type: project
  originSessionId: d10c6dad-b098-4d16-8f18-ac7efad12905
---

华为 iMaster NCE(AOC 开放可编程 + SND 设备驱动)调研结论,作为**后续优化 USMP 前端(R05 YANG 自动渲染)与南向驱动架构的依据**。全文 + 3 张真实 NCE 后台截图落在仓库:`docs/research/imaster-nce-ux-insights.md`(+ `assets/`)。调研于 2026-07-08。

**核心可迁移点**：
- 华为把 R05 做成了框架产品 AOC：一份 YANG → 自动生成「北向 API + 配置界面 + 数据表 + 南向报文」四件套；SND≈USMP 的 Reconciler+translator。
- 最大洞察：华为用 40+ 条 `render-*` YANG 扩展语句**把"渲染意图"编码进模型**，前端渲染器纯哑终端 → 改界面只改 YANG 注解。契合 R17 spec-first。建议 USMP 给 `/yang/schema`+`frontend-yang-dynamic-form` 引入 `ui:widget/ui:view/ui:order` 注解层，纳入契约生成+漂移门禁([[frontend-contract-gen]])。
- 有**YANG 类型→控件→Element Plus 映射表**(enum≤3→segmented、list→内嵌子表+空态、leafref→transfer…)，可近像素复刻 NCE，补全现有粗映射(只有 boolean→开关/enum→下拉)。
- NCE 设计系统：浅色+华为蓝 #307FE2+绿色状态色+高密度台账风，零渐变零圆角大卡片(撞 R11)。与已合入的[[frontend-redesign]]"浅色 iMaster NCE 气质"对齐。

**How to apply**：真要动前端渲染/加 YANG UI 能力时，先读 docs/research 那篇当输入，走 /opsx:explore→propose 立项，别从零重推。

**落地路线图**已存为跨会话持久任务 `openspec/tasks/optimize-frontend-nce-insights.md`(status: in_progress，SR01 会在新会话提示 `/task resume`)。恢复用 `/task resume optimize-frontend-nce-insights`。

**⚠️ 已与 [[generic-module-console]] 对账校准(2026-07-08，核对代码非仅记忆)**：原 4 阶段盲写、剩余工作量大幅高估——
- 前置 P0：先合 `generic-module-console` 分支(未合main，模型驱动通用控制台+FieldRenderer，退役DeviceConfigPage/frontend-yang-dynamic-form，后续都叠其上)。
- P1 渲染器 ~80% 已交付(FieldRenderer 已映射 string/number/boolean/enum/group/presence/leaf-list/choice/list)，仅剩 enum≤3→segmented、leafref→transfer、设计token 保真微调，低优先。
- P2 视图切分基本已交付(module console 派生 Tab+左导航)。
- **P3「注解存YANG-extension vs sidecar-JSON」分叉已被现实解答=走YANG extension**：华为 ext:support-filter/operation-exclude 已从 goyang Entry.Exts 消费、FieldDef 已带、make gen-contract 漂移门禁已存在；P3 降级为「扩展注解词汇表(ext:ui-widget等)」，主线、去风险。
- P4 删除语义债已定位(删除按钮禁用/后端无DELETE/POST合并)。
- **P5 异构多设备SND(2026-07-08从P4拆出的战略项)**：疑问「Go后端能否实现华为Python的SND」→**能且骨架已在**(pkg/translator 有 Translator接口+RegisterTranslator注册表+Huawei/Cisco/H3C/Juniper枚举+plugin钩子，仅实现Huawei)，非绿地，要做的是泛化+声明式化。三条Go路径:①声明式数据驱动(推荐,ygot结构+描述符,契合R04/R05)/③编译期注册表(现状起步)/②进程外gRPC插件(逃生舱,可跑Python驱动)。**Go不阻塞SND，不需Python/不需.so插件**。⚠️现有Translator吃CRD Spec=StackA遗留，SND须瞄准StackB([[dual-stack-migration]])；待用户拍板是否要运行时热插拔。详见 docs/research §3.6。
