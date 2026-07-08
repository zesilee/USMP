# iMaster NCE 洞察 — USMP 前端/架构可迁移参考

> 状态:研究结论(尚未立项)。用途:后续优化 USMP 前端(R05 YANG 自动渲染)与南向驱动架构的**依据**。
> 立项时以此为输入走 `/opsx:explore → /opsx:propose`。
> 调研日期:2026-07-08。证据分级见文末「查证边界」。

## 0. 一句话本质

华为把 USMP 的 **R05(前端由 YANG 模型自动渲染)** 做成了框架级产品 **AOC(开放可编程)**:
**一份业务 YANG 模型 → 框架自动生成「北向 API + 配置界面 + 数据表 + 南向协议报文」四件套**。
前端界面不是被"设计"出来的,是被模型"生成"出来的 —— 这正是 USMP 想做的事。

对照:**华为 SND(设备特定驱动包)≈ USMP 的 C3 Reconciler + huawei translator**;
**AOC 自动生成 UI ≈ USMP 的 `frontend-yang-dynamic-form` + `/yang/schema`**。

---

## 1. 架构层洞察(南向开放 / 网元管理 / 设备特定编程)

| 华为术语 | 实质 | USMP 对应物 |
|---------|------|-----------|
| **AOC** 开放可编程 | 模型驱动框架,南向 NETCONF/SNMP/OpenFlow,低/无代码接入设备 | yang-controller-runtime |
| **SND** 设备特定驱动包 | 每款设备一个包,Python 类继承 `NetconfSND`(如 `NE8000M8_SND`/`S6720-SND`),声明式元数据 `snd-id`/`python-class-name` | Reconciler + translator |
| 网元管理 + 自动生成配置界面 | 装入设备 YANG → 自动渲染该网元增删改查 UI | DeviceConfigPage + 动态表单 |

**开发者只写三样**:业务 YANG 模型、南向 Jinja2 模板(YANG 节点→设备报文)、Mapping/SND 驱动类。
**框架自动生成**:北向 REST API、**配置界面(前端 UI)**、数据表结构、南向协议报文。
→ 洞察:**前端 UI 与后端契约同源、零手写表单**;改一个 YANG 节点,前端表单自动跟着变。

### 1.1 关键机制:把"渲染意图"编码进 YANG 扩展语句(《开放可编程开发指南》第 14 章)

华为用一套 **40+ 条 render 扩展语句**标注"该节点怎么渲染",如:
`render-view`(是否单独成视图/页面)、`render-name`(是否显示标签)、
`render-single-line`/`render-multi-line`(平铺 vs 展开)、`render-flat-container`(打平嵌套)、
`render-order-by-user`(list 用户可拖拽排序)、`mandatory-in-cli`(必填)、
`insert-before/after`(字段排序)、`delete-siblings-on-delete`(级联删语义)…

**洞察(最重要):华为把"渲染决策"从前端代码里抽走,变成模型的元数据。**
前端渲染器是纯哑终端,只执行注解;新设备/新字段的界面调整**不改前端不改后端,只改 YANG 注解**。
"怎么渲染"成为契约的一部分,与数据结构一起版本化、评审、下发 —— 完美契合 USMP 的 **R17 spec-first**。
> 边界:第 14 章 render-* 主要服务 CLI 生成,兼管视图结构;Web 控件细粒度样式在框架内置渲染器里,公开文档不展开。

---

## 2. 前端 UI 洞察(基于真实截图,非推断)

证据来源:Fortinet《FortiNAC × Huawei iMaster 集成指南》p7–9 内含未打码的 **iMaster NCE-Campus 后台真实截图**,已归档:
- `assets/nce-user-list.png` — 用户管理列表页
- `assets/nce-create-user-wizard.png` — Create User 多步向导
- `assets/nce-create-radius-form.png` — Create RADIUS Server 配置表单

### 2.1 NCE 设计系统(逐条有截图佐证)

| 维度 | NCE 实际做法 |
|------|-------------|
| 主题 | 纯**浅色**:白色面板 + 极浅灰页底,无暗色 |
| 强调色 | 一种**华为蓝**(≈`#307FE2`,肉眼估读)专用于主按钮(Create/OK)与选中的分段控件 |
| 状态色 | **绿色**=启用开关 + Stepper 成功勾;**红星 `*`**=必填 |
| 信息密度 | **高密度运维台账风**,列多、留白克制;**零渐变、零圆角大卡片**(撞 USMP R11 反 AI 陈词滥调) |
| 导航 | 面包屑 → 左树 → 右内容;页内横向 Tab;多步操作用横向 Stepper |
| 表单 | **左标签右控件** + 分组 + 必填红星 + `?` 帮助图标 + 控件下**灰色行内说明** |
| 复杂度收纳 | `Advanced ⌄` 折叠、浅蓝信息条给建议、`Selected/Max: 1/20` 配额计数 |

### 2.2 YANG 类型 → 控件 → Element Plus 映射(核心可迁移资产)

截图里的每个控件都能对到 YANG 类型和 USMP 现有 Element Plus 栈 —— 三者一一对应,可近像素复刻:

| YANG 构造 | NCE 控件 | Element Plus |
|-----------|---------|-------------|
| `leaf` string | 文本输入 | `el-input` |
| `leaf` boolean | 绿色开关 | `el-switch` |
| `leaf` enum(≤3 选项) | **分段控件**(Local/Third-party/Remote) | `el-segmented` |
| `leaf` enum(>3 选项) | 下拉 | `el-select` |
| `mandatory true` | 红星 `*` | `el-form` `required` |
| `description` | 控件下灰色行内说明 | form-item help slot |
| `list` | **内嵌子表格 + Create/Delete + 空态 "No records found."** | `el-table` + 工具栏 |
| `leafref`/引用 | **双栏穿梭框**(角色选择) | `el-transfer` |
| `container` | 表单分组 / `Advanced ⌄` 折叠 | `el-collapse` |
| `max-elements` | `Selected/Max: 1/20` 计数 | 自定义计数 |
| 多步下发 | 横向 **Stepper** | `el-steps` |
| 层级导航 | 面包屑 + 左树 | `el-breadcrumb` + `el-tree` |

> AOC 从 YANG 自动生成的"配置界面",长的正是 `nce-create-user-wizard.png` / `nce-create-radius-form.png` 这个样子:
> 左标签表单 + 分段枚举 + 开关布尔 + list 变内嵌子表 + 空态。USMP R05 自动渲染的**终态视觉参照物**。

---

## 3. 对 USMP 可直接迁移的洞察(actionable)

| # | 华为做法 | USMP 可借鉴的动作 |
|---|---------|-----------------|
| A | **render-* 扩展语句**把渲染意图编码进 YANG | 给 `/yang/schema` + `frontend-yang-dynamic-form` 引入一套**轻量 UI 注解**(YANG extension 或 sidecar JSON):`ui:widget`/`ui:view`/`ui:order`/`ui:hidden`。让"怎么渲染"进契约、受 R17 spec-first 管辖,而非散落前端 |
| B | **一模型四件套同源生成** | 已有前端契约生成(见 [[frontend-contract-gen]])。把"UI 注解 → 前端渲染"也纳入生成 + 漂移门禁,彻底消灭手写表单残留 |
| C | **SND = 每设备一驱动包**,声明式元数据注册 | huawei translator/Reconciler 参考:**每设备型号一个声明式驱动描述**,加设备只加一个包不碰核心 |
| D | **视图切分**(`render-view`)避免巨型表单 | DeviceConfigPage 泛化流里,大模型按 container 自动切 Tab/Stepper,而非一屏平铺 |
| E | **删除语义进模型**(`delete-siblings-on-delete` 等) | 测试军规 add/edit/remove 分支(F2/F3)的级联删/幂等语义也可模型化,而非硬编码在组件里 |
| F | **锁定设计 token** | 浅色 + 华为蓝主色(`#307FE2`)+ 绿色状态色 + 高密度表格,写进设计系统(与已合入的"浅色 iMaster NCE 气质"redesign 对齐、并量化上表) |
| G | **补全类型→控件映射** | 渲染器按 §2.2 补 `enum→segmented(≤3)/select(>3)`、`list→内嵌子表+空态`、`leafref→transfer`,而不只是 boolean→开关/enum→下拉的粗映射 |
| H | **多步下发用 Stepper** | 设备"编辑→校验→下发→回读收敛"可视化成 4 步 Stepper,接上"收敛台账/新鲜度环" |

**最大一条**:华为证明 R05 的终局不是"前端会读 YANG",而是
**"渲染意图本身成为受版本管理的契约"**。USMP 现在的 render 逻辑还有一部分活在前端组件判断里(boolean→开关是硬映射);
把这层外化进模型注解,是 model-driven 前端的下一步演进方向。

---

## 3.5 一致性审计 —— USMP 当前实现 vs 华为"哑终端"理想（2026-07-08，代码核对 @origin/main）

> 命题:华为把"渲染决策"从前端代码抽走、变成模型元数据,前端渲染器纯执行 `render-*` 注解 → ①新字段不改前端 ②"怎么渲染"进契约。
> **审计结论:USMP 约 2/3 一致——数据+抽象类型+领域门禁已模型驱动(哑终端成立);但纯呈现决策仍是前端启发式,未进模型/契约。命中后果①,半中后果②。**

### ✅ 已是哑终端(渲染器只执行,不判断)
| 决策 | 驱动来源 | 证据 |
|------|---------|------|
| 控件类型 input/switch/select/number | 后端从 YANG leaf 类型派生 | `backend/internal/api/field_gen.go:162` `fieldType(leaf.LeafType())`；前端 `FieldRenderer.vue:5,26,34` `v-if field.type===x` 仅执行 |
| 必填/isKey/create-only 禁用 | YANG `isKey`/`operationExclude` | `field_gen.go:132,69` → `utils/moduleConsole.ts:63,76` |
| presence 开关 / when 条件可见 | YANG `presence`/`when` | `field_gen.go:78`；`FieldRenderer.vue:52`、`moduleConsole.ts:119` |
| 高级搜索字段 | 厂商 `ext:support-filter` | `moduleConsole.ts:95` `filterableFields` |

→ 加新模块/字段零前端代码（`/yang/schema`→FieldDef→渲染器）。**命中华为后果①。**

### ⚠️ 仍是前端业务判断(华为会放进模型注解)
| 前端自拍的渲染决策 | 证据 | 华为对应注解 |
|-----------------|------|-------------|
| 控件变体不可覆盖(enum 恒 select、string 恒 input;无法从模型指定 segmented/password/textarea) | `FieldRenderer.vue:33,4`;FieldDef **无 `Widget` 字段** | widget 注解 |
| 列选择+优先级+封顶9 是前端算法 | `moduleConsole.ts:72-92` `deriveColumns`(key→identity→when→enum→其余,cap=9) | `render-view`/`render-in-view`/`insert-before` |
| Tab/视图切分是前端启发式 | `moduleConsole.ts:30-60` `deriveTabs` | `render-view` |
| choice 用 radio 还是 tabs 是前端阈值 | `FieldRenderer.vue:231` `choiceUsesRadio` | case 呈现注解 |
| 字段顺序=schema 顺序,不可从模型调 | 无 `Order` 字段 | `insert-before/after` |
| 状态色 up→绿/down→红 硬编码 | `moduleConsole.ts:135` `statusTone` | 值映射注解 |

→ 这些改动要动前端代码,且**不在契约里**（`make gen-contract` 漂移门禁管不到）。**华为后果② 仅半中**：领域门禁(`support-filter`/`operation-exclude`/`presence`/`isKey`)进了契约,纯呈现决策没进。

### 差距本质与闭合路径（= 路线图 P3'+P1' 的立项依据）
USMP 把**数据契约**做成模型驱动,但**呈现契约**只做了一半:领域语义进模型,纯呈现语义(长什么样/排什么序/切几个视图)还在前端。当前是**"类型驱动"**而非华为的**"注解驱动"**。闭合:
1. 后端 FieldDef 加 `Widget`/`Order`/`View` 字段,从 `Entry.Exts` 读 `ext:ui-widget`/`ext:ui-order`/`ext:ui-view`（复用已验证的 `ext:*` 消费管线）
2. 前端 `deriveColumns`/`deriveTabs`/`choiceUsesRadio` 从"算法决定"改为"注解优先、算法兜底"
3. 新注解纳入 `make gen-contract` 漂移门禁 → 呈现决策才真正进契约

---

## 3.6 SND 在 Go 栈的可行性 —— 异构多设备（洞察 C 展开，2026-07-08 代码核对 @origin/main）

> 疑问:华为 SND 用 Python 热插拔驱动;USMP 后端是 Go,能实现 SND 吗?
> **结论:能,且骨架已在;后端是 Go 不阻塞 SND。SND 是模式非 Python 专利。**

### SND 等价物已在 Go 后端（不是绿地）
| 华为 SND 概念 | USMP 现有 Go 实现 | 证据 |
|-------------|-----------------|------|
| 设备驱动接口 | `Translator` 接口(Vendor/TranslateVlan/Interface/Route/System/Validate) | `backend/pkg/translator/translator.go:31` |
| 驱动注册表 | `RegisterTranslator`+`GetTranslator`+`map[VendorType]Translator` | `factory.go:9,14,19` |
| 多厂商抽象 | 枚举**已内置** Huawei/Cisco/H3C/Juniper（仅实现 Huawei） | `translator.go:13-16` |
| 驱动钩子 | plugin 包 Validation/Mutation/Notification/ReconciliationHook 接口 | `plugin/plugin.go` |

→ 要做的是**泛化 + 声明式化**（把手写 `TranslateXxx` 胶水变数据驱动），非从零建。

### 唯一实质差异 + 三条 Go 地道路径
华为 SND 是**运行时热插拔的 Python**（第三方丢 .py+YANG+Jinja2+JSON 不重编译）；Go 编译型，当前注册要求编译进二进制。但对 USMP（一组已知交换机厂商，非开放插件市场）几乎不是问题：
| 方案 | 做法 | 适配 |
|------|------|------|
| ① 声明式数据驱动（推荐终态） | 设备特定=数据非代码（ygot 厂商结构 R04 + 路径/模板描述符 + sysoid/协议/能力元数据），加设备=加数据+重跑 ygot | ⭐ 最契合 R04/R05 |
| ③ 编译期注册表（现状，起步） | 保留 RegisterTranslator，每厂商一 Go 包，重编译 | 够用于可控厂商集 |
| ② 进程外 gRPC 插件（逃生舱） | 独立二进制走 gRPC（Terraform provider/HashiCorp go-plugin），**驱动可为 Python**，真热插拔 | 最重，仅第三方/热加载才需 |

**建议 ③ 起步 → ① 成熟；不需要 Python、不需要 Go .so 插件。** 万一将来要托管 Python 驱动，② gRPC 是逃生舱——「Go 后端」在任何情况下都不锁死 SND。

### ⚠️ 待厘清（explore 阶段）
- 现有 `Translator` 吃 CRD Spec(`bizv1.BusinessInterfaceSpec`)=**Stack A 遗留**；SND 必须瞄准 **Stack B（yang-controller-runtime Reconciler + 通用引擎）**，别复活 CRD translator（[[dual-stack-migration]]）。
- **待用户拍板**：驱动是否需"运行时可插拔/第三方热加载"？否→③/①（纯 Go）；是→②（gRPC 进程外）。
- 对应路线图 **P5（异构多设备 SND，从 P4 拆出的战略项，独立立项）**。

---

## 4. 查证边界

- ✅ **确凿(一手原文/截图)**:AOC/SND 架构、四件套自动生成、render-* 扩展语句、Jinja2 南向模板、NE8000/S6720 交换机 SND 示例(引自《开放可编程 V100R023C00 开发指南》643 页);§2 三张 UI 截图为**实际渲染看过**的真实 iMaster NCE-Campus 后台(Campus 版,与 Fabric 版共用 NCE 设计系统)。
- ⚠️ **估读/推断**:配色 hex(`#307FE2`)为肉眼估读;NCE-Fabric Web 前端像素级视觉在华为 support 登录墙后,§2.1 视觉取向从截图 + 产品文档反推。
- 🔗 **未亲验**:官方 GUI 演示视频(无法观看,列于下方供人工核对)。

### 参考来源
- [DevZone 数通网络开放可编程(AOC)](https://devzone.huawei.com/cn/network/aoc.html)
- [《开放可编程 V100R023C00 开发指南》PDF(643 页)](https://devzone.huawei.com/cn/enterprise/aoc/V100R023C00/pdf/%E5%BC%80%E6%94%BE%E5%8F%AF%E7%BC%96%E7%A8%8B%E5%BC%80%E5%8F%91%E6%8C%87%E5%8D%97.pdf)
- [iMaster NCE-Fabric 产品页](https://e.huawei.com/en/products/network-analysis/imaster-nce-fabric)
- [FortiNAC × iMaster 集成指南 PDF(§2 截图来源,p7–9)](https://fortinetweb.s3.amazonaws.com/docs.fortinet.com/v2/attachments/db9bc2ca-4f80-11f0-a9d0-d2b0d2e22f7d/FortiNAC-F_7.x-Huawei_iMaster_Integration_Guide.pdf)
- [Huawei NCE 通过 EANTC NETCONF/YANG 互操作测试](https://www.huawei.com/en/press-events/news/2019/4/huawei-nce-eantc-pcep-netconf-yang-tests)
- 官方 GUI 演示视频:[Application Launch](https://www.youtube.com/watch?v=o1ZxA6uoWjI) · [Three-Level Rollback](https://www.youtube.com/watch?v=B9t9OQEm6-U) · [Network Change Simulation](https://www.youtube.com/watch?v=j41L8jey78Y) · [DC Construction](https://www.youtube.com/watch?v=r3H1DCmTQHE) · [播放列表](https://www.youtube.com/playlist?list=PLbWkJFbZ4h5fIt3dQz6aGUvyQQO5OcD6m)
