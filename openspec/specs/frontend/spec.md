# frontend — YANG/CRD 模型驱动的设备配置前端

## Purpose

frontend 是 Vue3 + Element Plus 平台前端：由后端 YANG schema **自动渲染**表单/表格/分组（R05，禁止手写固定表单），编辑→校验→提交→联动后端下发，并展示设备/缓存/对账状态。下发链路唯一：**Stack B 直连**（`POST /api/v1/config/:ip/*path` + 轮询对账），动态表单由 `FieldRenderer` 直渲；legacy K8s CRD 链路（ConfigPage/useK8sCRD/DynamicForm）已随 native-config-reposition 退役删除。概念分层：**原生配置** = 直接基于 YANG 模型的设备配置管理（模块控制台 `/module/:name`，本 spec 的全部范围）；**业务网络配置**为未来扩展层（业务侧 YANG 模型定义网络自动化能力，USMP 编排为原生配置下发，方向见 openspec/tasks/business-network-config.md）。

## Requirements

### Requirement: FE-01 schema 驱动渲染

前端 SHALL 将后端 YANG nested schema 经 `crdSchemaParser` 逐属性映射为 `Field[]`，类型映射为 boolean→switch、number→input-number、object→group；enum SHALL 按选项数与必填性细分：**必填且选项 ≤3 → segmented 分段控件，其余（可选或 >3 选项）→ select 下拉**（可选枚举 SHALL 保留清空能力，清空即该键不入 payload）。映射经 `FieldRenderer` 渲染为 Element Plus 控件（R05）。SHALL NOT 手写固定表单。

#### Scenario: 类型到控件的自动映射
- **WHEN** `getYangSchema(module, 'nested')` 返回带类型的属性
- **THEN** SHALL 生成对应 `Field[]`，并按类型渲染对应控件（boolean→switch、number→input-number、object→分组）

#### Scenario: 必填短枚举分段控件
- **WHEN** enum 字段 `required=true` 且选项数 ≤3
- **THEN** SHALL 渲染分段控件展示全部选项，选中 SHALL 触发值更新；readonly/禁用态 SHALL 透传为控件禁用

#### Scenario: 可选或长枚举保持下拉（边界）
- **WHEN** enum 字段可选（`required=false`）或选项数 >3
- **THEN** SHALL 渲染 select 下拉；可选枚举 SHALL 可清空，清空后该键 SHALL NOT 进入下发 payload

#### Scenario: 无有效 schema
- **WHEN** schema 拉取失败或为空
- **THEN** SHALL NOT 崩溃（R08），页面继续可用，仅不渲染该模块字段

### Requirement: FE-02 分组与校验

Field 带 group/pattern/min/max/required 时，前端渲染 SHALL 按分组组织（>1 分组时用 `el-collapse` 折叠），并由约束生成校验 rules；校验失败 SHALL NOT 提交，且 SHALL 行内提示 YANG 约束（§9、R08）。

#### Scenario: 多分组折叠
- **WHEN** 字段分布在 >1 个 group
- **THEN** SHALL 用 `el-collapse` 折叠分组渲染

#### Scenario: 校验失败不提交
- **WHEN** 存在缺失必填或数值越界（超出 min/max）
- **THEN** SHALL 阻止提交并在行内展示约束提示

### Requirement: FE-03 配置下发主链路（Stack B 直连）

原生配置 SHALL 走 Stack B 直连主链路：通用模块控制台（ModuleListTab/ModuleFormTab）以 YANG schema 渲染模型驱动表单，编辑→校验通过→提交时 SHALL 经 `useConfigSubmit`（或表单直调）`POST /api/v1/config/:ip/*path`，请求体为以 path 为根的 RFC7951 子树（YANG 真名、枚举名字符串；list 流的包裹键跟随回读的 RFC7951 键），随后 SHALL 以 `force_refresh` 强制回读实际态、轮询单设备 reconcile 结局，驱动 pushing→reading→converged/drifted/error/timeout 进度。下发失败 SHALL 降级、不误报成功（R08）。历史专用页 `DeviceConfigPage.vue` 已物理删除（通用模块控制台 FE-10~FE-16 取代）。

#### Scenario: 编辑并下发触发对账
- **WHEN** 用户在模块控制台提交一条合法（校验通过）配置
- **THEN** SHALL `POST /config` 下发 → `force_refresh` 回读 → 轮询 `getDeviceReconcile`，直到出现推进过 baseline 的终态（收敛/漂移/失败）或超时

#### Scenario: 下发失败降级
- **WHEN** `setConfig` 报错或返回失败信封
- **THEN** SHALL 置 error 相位、SHALL NOT 重读列表、保留原表单，不崩溃（R08）

#### Scenario: 对账超时
- **WHEN** 轮询达到上限仍无终态
- **THEN** SHALL 标注 `timedOut` 停在 reading 相位，SHALL NOT 误报成功

### Requirement: FE-04 原生/预建模块 schema

需要预建 fields 的模块 SHALL 经后端 `GET /api/v1/yang/schema/${module}` 取回预建 fields 后渲染，而非在前端硬编码表单结构（R05）。

#### Scenario: 拉取预建 schema
- **WHEN** 调用 `getSchema(module)`
- **THEN** SHALL 从后端 `GET /api/v1/yang/schema/${module}` 取 fields 并据此渲染

### Requirement: FE-07 约束引擎（when 显隐 / must 校验）

前端 SHALL 提供**通用**约束引擎（`utils/xpathEval` + `composables/useConstraintEngine`），把 schema 中的 `when`/`must` XPath 表达式求值为运行时行为，SHALL NOT 硬编码任何厂商/模型/字段名。求值器 SHALL 为自研 YANG XPath 子集解析器（相对路径 `../leaf`、`= != > < >= <=`、`and`/`or`/`not()`、`mod`、字面量），SHALL NOT 引入 `eval`/`safe-eval` 等依赖（R10）。表达式解析失败 SHALL 降级（when 失败=字段可见、must 失败=不阻断）并记录告警，SHALL NOT 崩溃（R08）。

#### Scenario: when 驱动显隐
- **WHEN** 字段带 `when`（如 `../class='sub-interface'`），用户改动被引用字段的值
- **THEN** 引擎 SHALL 实时重算该字段 `visible`；`visible=false` 的字段 SHALL 隐藏且 SHALL NOT 参与提交与校验

#### Scenario: must 阻断非法提交
- **WHEN** 字段带 `must`（如 `(../suppress>../reuse)` 或 `(../interval) mod 10 = 0`）且当前表单违反该约束
- **THEN** 引擎 SHALL 返回违例，前端 SHALL 阻止提交并行内提示（message 取 YANG `description` 或生成的通用提示）

#### Scenario: 表达式语法错误降级
- **WHEN** `when`/`must` 表达式无法被求值器解析
- **THEN** SHALL 降级（可见 / 不阻断）并记录告警，页面 SHALL NOT 崩溃（R08）

### Requirement: FE-08 choice/case 渲染

`FieldRenderer` SHALL 将 `type:"choice"` 的字段渲染为互斥切换控件（任一 case 含多字段→`el-tabs`，所有 case 均为单叶→`el-radio-group`），分支内子字段按 `cases[].fields` 递归渲染。切换到某 case 时 SHALL 清空其它非激活 case 的数据（YANG choice 互斥语义），提交 payload SHALL 只含激活 case 的字段且保持其扁平 path。

#### Scenario: choice 渲染为切换控件
- **WHEN** schema 含 `type:"choice"` 节点（如 IFM `bandwidth-type` 的 mbps/kbps 两 case）
- **THEN** SHALL 渲染为 Tabs/RadioGroup，可切换不同 case 的配置块

#### Scenario: 切换 case 清空非激活分支
- **WHEN** 用户从 case A 切到 case B
- **THEN** SHALL 清空 case A 字段值，提交时 SHALL 只携带 case B 字段（扁平 path）

### Requirement: FE-09 leaf-list 与 pattern 校验

`FieldRenderer` SHALL 支持 `type:"leaf-list"`（可增删的多值输入行，成员复用叶渲染），并 SHALL 对带 `pattern` 的 string 字段绑定正则校验；非法正则 SHALL 降级为不校验并告警（R08），SHALL NOT 崩溃。

#### Scenario: leaf-list 增删多值
- **WHEN** 字段为 `type:"leaf-list"`
- **THEN** SHALL 渲染可增删的多值输入，提交为数组

#### Scenario: pattern 校验
- **WHEN** string 字段带 `pattern`（如 IFM `number` 的接口编号正则）
- **THEN** SHALL 以该正则校验输入，不匹配时行内报错、阻止提交

### Requirement: FE-10 通用模块控制台（Tab 由模块根派生）

前端 SHALL 提供通用模块控制台页（路由 `/module/:module`，零 per-module props）：
右侧内容区 SHALL 渲染面包屑（配置/厂商/模块/激活 Tab）与一级 Tab；Tab 集合 SHALL 由
nested schema 模块根的顶层子节点自动派生——list→列表 Tab、group/choice→表单 Tab、
散落根叶子聚合为「基本属性」表单 Tab。SHALL NOT 针对任一具体 YANG 模块硬编码
Tab/列/字段。Tab 切换 SHALL 保留各 Tab 的表单与搜索状态。

#### Scenario: huawei-ifm 派生

- **WHEN** 打开 `/module/ifm`
- **THEN** Tab 集合 SHALL 含 `global`（表单）、`damp`（表单）、`auto-recovery-times`（列表或表单）、
  `interfaces`（列表）等根子节点，无任何硬编码模块名

#### Scenario: schema 加载失败降级

- **WHEN** schema API 失败
- **THEN** 页面 SHALL 展示错误提示且不崩（R08），设备选择仍可用

### Requirement: FE-11 模型驱动列表 Tab（列派生/高级搜索/分页/操作门禁）

列表 Tab SHALL：
- 按分层启发式（key→operationExclude∋update 的 identity 叶→带 when 的条件叶→enum→其余标量）
  从 list 子叶派生表格列并封顶，enum 列渲染 Tag、值 up/down 类渲染状态点（值驱动）；
- 对带 `when` 的列按行数据求值：不满足 SHALL 显示 `-`，求值失败 SHALL 降级正常渲染（R08）；
- 工具栏 SHALL 含新增按钮与「高级搜索」折叠面板，搜索字段集 SHALL 仅取 `supportFilter=true`
  的叶（enum→下拉、其余→文本），支持查询/重置，客户端过滤；
- 表格底部 SHALL 分页（总数/页码/每页条数）；
- 操作列 SHALL 受 `operationExclude` 门禁：list 级含 update/delete 时隐藏对应按钮；
  编辑抽屉中叶级含 update 的字段 SHALL 禁用（新增态可填）。

#### Scenario: 高级搜索过滤

- **WHEN** 数据含 3 条 main-interface 与 2 条 sub-interface，按 class=sub-interface 查询
- **THEN** 表格 SHALL 仅显 2 行；重置后 SHALL 还原 5 行

#### Scenario: 行级 when 单元格

- **WHEN** 行 class=main-interface 且 parent-name 列的 when 为 `../class='sub-interface'`
- **THEN** 该行 parent-name 单元格 SHALL 显示 `-`；sub-interface 行 SHALL 显示其父接口名

#### Scenario: 编辑态 identity 字段禁用

- **WHEN** 编辑一条既有记录且某叶 `operationExclude` 含 `update`
- **THEN** 该字段 SHALL 禁用；新增抽屉中同字段 SHALL 可编辑

### Requirement: FE-12 presence 容器渲染与门禁

`presence=true` 的 group SHALL 渲染为开关：关闭时对应键 SHALL NOT 进入 payload
（YANG presence 语义）；容器 `must` 依赖不满足时开关 SHALL 禁用并强制关闭，
must 求值失败 SHALL 降级为可用（R08）。

#### Scenario: 条件互斥开关

- **WHEN** `ipv4-ignore-primary-sub=true`
- **THEN** `ipv4-conflict-enable` 开关 SHALL 禁用且为关；置 false 后 SHALL 恢复可用

### Requirement: FE-13 模型驱动原生配置导航与路由迁移

左侧**原生配置**菜单 SHALL 由 `/yang/modules` 返回的模块列表驱动生成（指向 `/module/:name`），
加载失败 SHALL 回退既有硬编码项（R08）；旧路由 `/config/interface`、`/config/vlan`
SHALL 重定向到对应 `/module/:module`。模块项携带 `category` 时菜单 SHALL 按 category
分组展示；无 `category` 的模块 SHALL 归入默认分组，分组渲染 SHALL NOT 因缺失 category 失败（R08）。
legacy 路由 `/native/:module` 与 `/config/route` SHALL 不存在（Stack A CRD 死路已退役，生产中从未可用，无重定向义务）。

#### Scenario: 菜单生成与回退

- **WHEN** `/yang/modules` 返回含 `ifm`（模块根名）的列表
- **THEN** 菜单 SHALL 含指向 `/module/ifm` 的项
- **WHEN** 该 API 失败
- **THEN** 菜单 SHALL 显示回退项且不崩

#### Scenario: 任务域分组

- **WHEN** 模块列表含带 `category` 与不带 `category` 的模块
- **THEN** 菜单 SHALL 按 category 分组展示带值模块，无值模块 SHALL 归入默认分组且渲染不失败

#### Scenario: 菜单命名与概念对齐

- **WHEN** 渲染左侧导航
- **THEN** 模块控制台菜单组标题 SHALL 为「原生配置」，SHALL NOT 存在指向 `/native/*` 的菜单项

### Requirement: FE-14 state 子树只读降级

通用模块控制台 SHALL 将 `readonly=true` 的字段降级为只读呈现而非可编辑控件：
整棵 readonly 子树派生的 Tab SHALL 渲染只读视图（可查看、不可编辑）；混合容器内的
readonly 叶 SHALL 呈现禁用态。readonly 字段 SHALL NOT 进入 diff/下发 payload/校验门禁。

#### Scenario: 只读 Tab 降级

- **WHEN** 模块根下某容器整棵为 readonly（如 ifm `remote-interfaces`）
- **THEN** 其 Tab SHALL 以只读视图呈现且 SHALL NOT 提供编辑/下发入口

#### Scenario: 混合容器内只读叶

- **WHEN** 可编辑容器内存在 readonly 叶
- **THEN** 该叶 SHALL 渲染禁用态且 SHALL NOT 参与 payload 与校验

#### Scenario: 只读 list 呈现（边界）

- **WHEN** readonly 子树含 list 节点
- **THEN** SHALL 以只读表格呈现行数据，SHALL NOT 渲染增删改操作列

### Requirement: FE-15 动态缺省占位与单位后缀

字段渲染器对 `dynamicDefault=true` 的字段 SHALL 呈现「系统自动分配」占位语义：
空值 SHALL NOT 触发必填校验、SHALL NOT 视为待下发变更；对携带 `units` 的字段
SHALL 在输入控件展示单位后缀。

#### Scenario: 动态缺省占位

- **WHEN** 字段 `dynamicDefault=true` 且用户未填写
- **THEN** 输入框 SHALL 展示系统自动分配占位提示
- **AND** 空值 SHALL NOT 计入 diff/payload，SHALL NOT 报必填错误

#### Scenario: 用户显式覆写动态缺省（边界）

- **WHEN** 用户对 `dynamicDefault` 字段输入了显式值
- **THEN** 该值 SHALL 正常进入校验与下发 payload

#### Scenario: 单位后缀

- **WHEN** 字段携带 `units: "bit/s"`
- **THEN** 输入控件 SHALL 展示 `bit/s` 后缀

### Requirement: FE-16 列表行删除（confirm→DELETE→刷新）

通用模块控制台列表 Tab 的行「删除」按钮在门禁允许（list 级 `operationExclude` 不含 delete 且非只读 Tab）时 SHALL 可用；点击 SHALL 弹出二次确认（含条目主键标识），确认后 SHALL 调用 `DELETE /config/:ip/*path?key=<主键>`；成功 SHALL 刷新列表与新鲜度并提示，失败 SHALL 如实展示后端错误且列表不变（R08/§9）。取消确认 SHALL 无任何请求。

#### Scenario: 删除成功流

- **WHEN** 用户点击某行删除并确认
- **THEN** SHALL 以该行主键调用 DELETE，成功后该行 SHALL 从列表消失（重新拉取）

#### Scenario: 取消确认

- **WHEN** 用户在确认框选择取消
- **THEN** SHALL NOT 发起任何请求，列表不变

#### Scenario: 删除失败如实透出（负路径）

- **WHEN** 后端返回错误（如设备 data-missing / 门禁 400）
- **THEN** SHALL 展示错误信息，列表 SHALL 保持原状

#### Scenario: 门禁不可用态

- **WHEN** list 级 `operationExclude` 含 delete 或 Tab 为只读
- **THEN** 删除按钮 SHALL 不可用/不渲染（沿用 FE-11/FE-14 门禁）

### Requirement: FE-17 业务网络配置菜单组与平台作用域控制台

侧边栏 SHALL 出现「业务网络配置」菜单组：由意图 YANG 模块的 task-name category 经既有分桶机制自动生成（零菜单硬编码，R05）。业务能力 SHALL 渲染为**平台作用域**控制台（一个意图实例管理多台设备，不绑定单设备上下文）：意图表单 SHALL 由意图 YANG schema 自动渲染（devices 嵌套 list 含增删改）、实例列表 SHALL 展示每实例收敛状态汇总（deviceStates 聚合：全 synced/部分 failed/pending）、实例详情 SHALL 展示每设备状态与失败原因。

#### Scenario: 菜单组自动出现
- **WHEN** 意图 YANG 模块带业务 category 注册且被 `GET /yang/modules` 返回
- **THEN** 侧边栏 SHALL 自动出现「业务网络配置」组及该能力入口，无前端菜单代码改动

#### Scenario: 意图表单模型驱动
- **WHEN** 打开「跨设备 VLAN 打通」控制台新建意图
- **THEN** 表单 SHALL 按意图 YANG 渲染（vlan-id 数字输入带 range、devices 嵌套 list 可增删改行），校验失败 SHALL 行内提示且不提交

#### Scenario: 收敛状态呈现
- **WHEN** 某意图 2 台设备中 1 台 failed
- **THEN** 实例列表 SHALL 呈现部分失败态，详情 SHALL 列出失败设备与原因

### Requirement: FE-18 原生控制台归属徽标

原生模块控制台渲染被业务意图认领的对象/路径时 SHALL 显示「由业务配置 <意图名> 管理」徽标。用户对认领路径提交手改被后端归属硬锁拒绝（信封码 409 携 intents）时，SHALL 弹阻断确认框：列出认领意图名称并警示「意图收敛会覆盖手改」；用户确认后 SHALL 携 `force=true` 重发同一请求，取消则 SHALL 中止流程且不置错误态。force 放行后的响应含 `ownershipWarning` 时 SHALL 保留非阻断提示（一期行为）。

#### Scenario: 认领对象带徽标
- **WHEN** 原生 vlan 控制台列表中某 VLAN 被意图 X 认领
- **THEN** 该行 SHALL 显示归属徽标（含意图名）

#### Scenario: 硬锁 409 触发阻断确认
- **WHEN** 提交手改收到信封码 409 且 data.intents 含意图 X
- **THEN** SHALL 弹确认框列出意图 X 与覆盖警示，SHALL NOT 直接置为下发失败

#### Scenario: 确认后 force 重发
- **WHEN** 用户在阻断确认框点击「强制下发」
- **THEN** SHALL 以 `force=true` 重发原请求，成功后按 force 分支展示非阻断归属警告

#### Scenario: 取消则中止
- **WHEN** 用户在阻断确认框取消
- **THEN** SHALL 中止提交流程，不下发、不展示错误态

