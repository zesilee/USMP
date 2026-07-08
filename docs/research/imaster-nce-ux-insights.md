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
