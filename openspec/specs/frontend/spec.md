# frontend — YANG/CRD 模型驱动的设备配置前端

## Purpose

frontend 是 React + TypeScript 平台前端（控件经 `src/ui` UI 适配层收口，当前实现 Ant Design，见 frontend-ui-adapter spec）：由后端 YANG schema **自动渲染**表单/表格/分组（R05，禁止手写固定表单），编辑→校验→提交→联动后端下发，并展示设备/缓存/对账状态。下发链路唯一：**Stack B 直连**（`POST /api/v1/config/:ip/*path` + 轮询对账），动态表单由 `FieldRenderer` 直渲；legacy K8s CRD 链路（ConfigPage/useK8sCRD/DynamicForm）已随 native-config-reposition 退役删除。概念分层：**原生配置** = 直接基于 YANG 模型的设备配置管理（模块控制台 `/module/:name`，本 spec 的全部范围）；**业务网络配置**为未来扩展层（业务侧 YANG 模型定义网络自动化能力，USMP 编排为原生配置下发，方向见 openspec/tasks/business-network-config.md）。
## Requirements
### Requirement: FE-01 schema 驱动渲染

前端 SHALL 将后端 YANG nested schema 经 `crdSchemaParser` 逐属性映射为 `Field[]`，类型映射为 boolean→「打开/关闭」radio 单选组（i18n 文案，值仍为 true/false；可选 boolean SHALL 支持不选=不入 payload）、number→input-number、object→group；enum SHALL 按选项数与必填性细分：**必填且选项 ≤3 → segmented 分段控件，其余（可选或 >3 选项）→ select 下拉**（可选枚举 SHALL 保留清空能力，清空即该键不入 payload）。映射经 `FieldRenderer` 渲染为 **UI 适配层（`src/ui`）导出的控件**（R05），SHALL NOT 直接依赖具体组件库（见 `frontend-ui-adapter` FA-01）。SHALL NOT 手写固定表单。

#### Scenario: 类型到控件的自动映射
- **WHEN** `getYangSchema(module, 'nested')` 返回带类型的属性
- **THEN** SHALL 生成对应 `Field[]`，并按类型渲染对应控件（boolean→打开/关闭 radio、number→input-number、object→分组）

#### Scenario: boolean radio 值语义
- **WHEN** boolean 字段选中「打开」
- **THEN** payload 中该叶 SHALL 为 true；「关闭」SHALL 为 false；可选 boolean 未选时该键 SHALL NOT 入 payload

#### Scenario: 必填短枚举分段控件
- **WHEN** enum 字段 `required=true` 且选项数 ≤3
- **THEN** SHALL 渲染分段控件展示全部选项，选中 SHALL 触发值更新；readonly/禁用态 SHALL 透传为控件禁用

#### Scenario: 可选或长枚举保持下拉（边界）
- **WHEN** enum 字段可选（`required=false`）或选项数 >3
- **THEN** SHALL 渲染 select 下拉；可选枚举 SHALL 可清空，清空后该键 SHALL NOT 进入下发 payload

#### Scenario: 无有效 schema
- **WHEN** schema 拉取失败或为空
- **THEN** SHALL NOT 崩溃（R08），页面继续可用，仅不渲染该模块字段

#### Scenario: 控件映射结论与组件库实现无关（换库锚点）
- **WHEN** 底层组件库实现发生替换
- **THEN** 上述全部类型→控件映射结论 SHALL 保持不变，派生黄金（GD-01）SHALL 零漂移

### Requirement: FE-02 分组与校验

Field 带 group/pattern/min/max/required 时，前端渲染 SHALL 按分组组织（>1 分组时 SHALL 渲染为二级 Tab，NCE 形态；Tab 集合超宽 SHALL 可横向滚动收纳（不截断）），并由约束生成校验 rules；校验失败 SHALL NOT 提交，且 SHALL 行内提示 YANG 约束（§9、R08）。

#### Scenario: 多分组二级 Tab
- **WHEN** 字段分布在 >1 个 group
- **THEN** SHALL 渲染二级 Tab 切换分组，切换 SHALL 保留各分组表单状态

#### Scenario: 校验失败不提交
- **WHEN** 存在缺失必填或数值越界（超出 min/max）
- **THEN** SHALL 阻止提交并在行内展示约束提示

### Requirement: FE-03 配置下发主链路（攒批变更集）

原生配置 SHALL 走攒批主链路：通用模块控制台（ModuleListTab/ModuleFormTab）以 YANG schema 渲染模型驱动表单，编辑→校验通过→「确定」时 SHALL 仅写入当前设备的变更集（changeset store），SHALL NOT 直接下发设备。工具栏「提交配置」SHALL 将变更集经 `POST /api/v1/config/changeset/commit` 一次提交，成功后 SHALL 以 `force_refresh` 强制回读实际态、轮询单设备 reconcile 结局，驱动 pushing→reading→converged/drifted/error/timeout 进度，并清空该设备变更集；提交失败 SHALL 降级、不误报成功、变更集 SHALL 原样保留（R08）。历史专用页 `DeviceConfigPage.vue` 已物理删除（通用模块控制台 FE-10~FE-16 取代）。

#### Scenario: 确定入集不下发

- **WHEN** 用户在详情编辑区修改字段并点「确定」（校验通过）
- **THEN** 该修改 SHALL 进入变更集（工具栏徽标计数 +1），SHALL NOT 发起任何 `/config` 写请求

#### Scenario: 提交配置触发批量下发与对账

- **WHEN** 用户点「提交配置」并确认
- **THEN** SHALL 调用 changeset commit 接口 → `force_refresh` 回读 → 轮询 `getDeviceReconcile` 直到出现推进过 baseline 的终态或超时；成功 SHALL 清空该设备变更集并刷新列表与新鲜度

#### Scenario: 提交失败降级

- **WHEN** commit 接口报错或返回失败信封
- **THEN** SHALL 置 error 相位并如实展示失败条目信息，变更集 SHALL 原样保留，不崩溃（R08）

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

`FieldRenderer` SHALL 将 `type:"choice"` 的字段渲染为互斥切换控件（任一 case 含多字段→Tabs，所有 case 均为单叶→Radio 组；控件经 UI 适配层），分支内子字段按 `cases[].fields` 递归渲染。切换到某 case 时 SHALL 清空其它非激活 case 的数据（YANG choice 互斥语义），提交 payload SHALL 只含激活 case 的字段且保持其扁平 path。

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

设备选择 SHALL 为**全局设备上下文**（device store 单一事实源，IP 口径）：控制台设备
下拉 SHALL 双向绑定全局上下文，模块间切换 SHALL 保持选中设备不变（先选设备、后做
配置管理）。设备管理「查看配置」入口与 `?device=<ip>` 深链 SHALL 写入同一全局上下文。
未选设备时 SHALL 展示引导空态（提示先选择设备），SHALL NOT 静默渲染空列表/空表单。
平台作用域业务控制台（`/business/:module`）SHALL NOT 绑定设备上下文。

#### Scenario: huawei-ifm 派生

- **WHEN** 打开 `/module/ifm`
- **THEN** Tab 集合 SHALL 含 `global`（表单）、`damp`（表单）、`auto-recovery-times`（列表或表单）、
  `interfaces`（列表）等根子节点，无任何硬编码模块名

#### Scenario: schema 加载失败降级

- **WHEN** schema API 失败
- **THEN** 页面 SHALL 展示错误提示且不崩（R08），设备选择仍可用

#### Scenario: 跨模块切换保持选中设备

- **WHEN** 在 `/module/ifm` 选中设备 192.168.1.2 后经左树切换到 `/module/vlan`
- **THEN** VLAN 控制台 SHALL 已选中 192.168.1.2，无需重新选择，配置数据按该设备加载

#### Scenario: 深链与「查看配置」写入全局上下文

- **WHEN** 从设备管理点击某设备「查看配置」（或直接打开 `/module/ifm?device=<ip>`）
- **THEN** 全局设备上下文 SHALL 更新为该设备，后续切换到其它模块页 SHALL 沿用该选中

#### Scenario: 未选设备引导空态

- **WHEN** 全局上下文无选中设备时打开任一 `/module/:module`
- **THEN** 内容区 SHALL 展示「请先选择设备」引导空态而非空数据表单/列表，选中设备后 SHALL 恢复正常渲染
### Requirement: FE-11 模型驱动列表 Tab（列派生/工具区/分页/操作门禁）

列表 Tab SHALL：
- 按分层启发式（key→operationExclude∋update 的 identity 叶→带 when 的条件叶→enum→其余标量）
  从 list 子叶派生**默认显示列**（封顶 9），并同时派生**可用列全集**（全部标量叶）；
  列设置入口 SHALL 允许在全集内勾选显隐（勾选态仅本页会话内生效），宽表 SHALL 横向滚动；
  enum 列渲染 Tag、值 up/down 类渲染状态点（值驱动）；
- 全部显示列 SHALL 支持客户端排序；enum/boolean 列 SHALL 支持列头筛选（客户端）；
- 对带 `when` 的列按行数据求值：不满足 SHALL 显示 `-`，求值失败 SHALL 降级正常渲染（R08）；
- 工具栏 SHALL 含「创建」「刷新」按钮与「高级搜索」折叠面板，搜索字段集 SHALL 仅取
  `supportFilter=true` 的叶（enum→下拉、其余→文本），支持查询/重置，客户端过滤；
- 表格 SHALL 含多选框列，并 SHALL 提供「更多▾」批量菜单：批量删除 SHALL 将选中条目
  逐条作为删除项写入变更集（受 `operationExclude` delete 门禁，越权条目 SHALL 跳过并提示）；
- 列表 SHALL 以标记合成视图展示变更集状态：待创建行（新增标记）、已修改行（修改标记）、
  待删除行（删除标记），标记 SHALL 随变更集增删即时更新；
- 表格下方 SHALL 展示「{最近一次加载完成时刻} 查询结束，总记录数: N」（N=过滤后全集数）
  与分页（总数/页码/每页条数/前往跳页）；
- 操作列 SHALL 含「编辑 / 删除 / 获取数据源」：编辑与删除受 `operationExclude` 门禁
  （list 级含 update/delete 时隐藏对应按钮）；「获取数据源」SHALL 以 `force_refresh=true`
  绕缓存回读该 list 路径并刷新列表与新鲜度，SHALL NOT 宣称单行粒度回读；
  详情编辑区中叶级 `operationExclude` 含 update 的字段 SHALL 禁用（创建态可填）。

#### Scenario: 高级搜索过滤

- **WHEN** 数据含 3 条 main-interface 与 2 条 sub-interface，按 class=sub-interface 查询
- **THEN** 表格 SHALL 仅显 2 行；重置后 SHALL 还原 5 行

#### Scenario: 行级 when 单元格

- **WHEN** 行 class=main-interface 且 parent-name 列的 when 为 `../class='sub-interface'`
- **THEN** 该行 parent-name 单元格 SHALL 显示 `-`；sub-interface 行 SHALL 显示其父接口名

#### Scenario: 编辑态 identity 字段禁用

- **WHEN** 在详情编辑区编辑一条既有记录且某叶 `operationExclude` 含 `update`
- **THEN** 该字段 SHALL 禁用；创建态同字段 SHALL 可编辑

#### Scenario: 列设置显隐

- **WHEN** list 有 15 个标量叶，打开列设置勾选默认集外的 2 列
- **THEN** 表格 SHALL 展示 11 列（默认 9 + 勾选 2），取消勾选 SHALL 即时隐藏

#### Scenario: 列排序与 enum 列头筛选

- **WHEN** 点击某显示列的排序控件 / 在某 enum 列头选择一个枚举值
- **THEN** 行序 SHALL 按该列值客户端排序 / 表格 SHALL 仅显该枚举值的行，清除筛选还原

#### Scenario: 获取数据源强制回读

- **WHEN** 点击某行「获取数据源」
- **THEN** SHALL 以 `force_refresh=true` 请求该 list 配置路径，完成后 SHALL 更新列表数据、
  查询时间戳与新鲜度指示；失败 SHALL 如实报错且列表保持原状（R08/§9）

#### Scenario: 查询时间戳与总记录数

- **WHEN** 列表加载或刷新完成，过滤后共 56 条
- **THEN** 表格下方 SHALL 展示完成时刻与「总记录数: 56」，分页 SHALL 支持跳转到指定页

#### Scenario: 多选批量删除入集

- **WHEN** 勾选 3 行（其中 1 行 list 级 `operationExclude` 含 delete）并点「更多▾ > 批量删除」确认
- **THEN** 2 条可删条目 SHALL 作为删除项进入变更集并显示待删除标记，越权条目 SHALL 被跳过并提示

#### Scenario: 变更集标记合成视图

- **WHEN** 变更集含该 list 的 1 条待创建、1 条修改、1 条待删除
- **THEN** 表格 SHALL 分别以对应标记展示三行；重置变更集后标记行 SHALL 全部还原为设备实际态

### Requirement: FE-12 presence 容器渲染与门禁

`presence=true` 的 group SHALL 渲染为开关：关闭时对应键 SHALL NOT 进入 payload
（YANG presence 语义）；容器 `must` 依赖不满足时开关 SHALL 禁用并强制关闭，
must 求值失败 SHALL 降级为可用（R08）。

#### Scenario: 条件互斥开关

- **WHEN** `ipv4-ignore-primary-sub=true`
- **THEN** `ipv4-conflict-enable` 开关 SHALL 禁用且为关；置 false 后 SHALL 恢复可用
### Requirement: FE-13 模型驱动原生配置导航与路由迁移

左侧**原生配置**菜单 SHALL 由 `/yang/modules` 返回的模块列表驱动生成（指向 `/module/:name`），
加载失败 SHALL 回退既有硬编码项（R08）。模块项携带 `category` 时菜单 SHALL 按 category
分组展示；无 `category` 的模块 SHALL 归入默认分组，分组渲染 SHALL NOT 因缺失 category 失败（R08）。
legacy 路由 `/native/:module`、`/config/route`、`/config/interface`、`/config/vlan`
SHALL 不存在（Stack A CRD 死路与旧配置页入口均已退役，旧地址兼容期结束，无重定向义务）；
站内跳转 SHALL 直接使用 `/module/:module`，SHALL NOT 依赖任何旧路由。

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

#### Scenario: 旧配置页路由退役

- **WHEN** 检视路由表
- **THEN** SHALL NOT 存在 `/config/interface`、`/config/vlan` 路径（含重定向）
- **WHEN** 用户在 Dashboard 点击「下发配置」
- **THEN** SHALL 直接导航到 `/module/vlan`
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
### Requirement: FE-16 列表行删除（confirm→入变更集）

通用模块控制台列表 Tab 的行「删除」按钮在门禁允许（list 级 `operationExclude` 不含 delete 且非只读 Tab）时 SHALL 可用；点击 SHALL 弹出二次确认（含条目主键标识），确认后 SHALL 将该条目作为删除项写入变更集并展示待删除标记，SHALL NOT 立即发起 DELETE 请求。待删除条目的行删除按钮 SHALL 变为「取消删除」，点击 SHALL 从变更集移除该删除项。对变更集中待创建条目执行删除 SHALL 直接移除该待创建项（不产生删除报文）。取消确认 SHALL 无任何变更集改动。实际删除 SHALL 随「提交配置」经 changeset commit 接口生效（CS-04），失败 SHALL 如实展示且变更集保留（R08/§9）。

#### Scenario: 删除入集与提交生效

- **WHEN** 用户点击某行删除并确认，随后点「提交配置」
- **THEN** 确认后该行 SHALL 显示待删除标记且无网络请求；提交成功后该行 SHALL 从列表消失（重新拉取）

#### Scenario: 取消删除

- **WHEN** 待删除行上点击「取消删除」
- **THEN** 该删除项 SHALL 从变更集移除，行恢复正常态，徽标计数 SHALL 减一

#### Scenario: 删除待创建条目（边界）

- **WHEN** 对变更集中一条待创建条目执行删除并确认
- **THEN** 该待创建项 SHALL 直接从变更集移除，SHALL NOT 生成删除报文

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
### Requirement: FE-19 模型驱动 rpc 渲染与执行

前端 SHALL 把模块的 rpc 与顶层配置容器**平级**呈现在左侧导航树的模块叶下（导航层级：模块 → container 与 rpc 并列平铺，LT-03）；模块控制台 `/module/:module` 的 Tab 栏 SHALL NOT 再出现 rpc Tab（rpc 入口唯一收敛到左树）。点击某 rpc 节点 SHALL 路由 `/module/:module/rpc/:rpcName`，右侧内容区 SHALL 仅渲染该 rpc 的执行面板：input 由 schema 的 FieldDef 渲染（复用既有渲染管线，含 leafref 下拉、mandatory 校验、单位后缀），rpc 名与 input 叶标签 SHALL 按 UI-03 本地化，执行后 SHALL 回显 rpc-reply 结果或错误。rpc 路由页 SHALL 沿用全局设备上下文与面包屑骨架（配置/厂商/模块/rpc 名）。`rpcName` 不存在于该模块 schema 时 SHALL 展示明确错误提示且不崩（R08）。渲染 SHALL 由 schema 驱动，SHALL NOT 为具体 rpc 硬编码表单。

带 leafref 目标的 rpc 输入 SHALL **始终**渲染为下拉，仅可从设备实际存在的目标值中选择，SHALL NOT 允许自由文本输入、SHALL NOT 在目标列表拉取失败或为空时降级为文本框。目标列表为空（设备离线、拉取失败、目标 list 无实例）时下拉 SHALL 呈空并展示明确占位提示（R08 降级=空下拉而非放开手输）；mandatory 的 leafref 输入无值时执行按钮 SHALL 维持校验拦截。

#### Scenario: rpc 与 container 平级呈现于左树

- **WHEN** 展开某含 rpc 的模块（huawei-ifm）左树叶
- **THEN** 该模块的 rpc（如「按接口名清除统计」）SHALL 与配置容器节点（「通用接口」）平级出现在左树中
- **AND** `/module/ifm` 控制台 Tab 栏 SHALL NOT 含任何 rpc Tab

#### Scenario: rpc 直达路由渲染执行面板

- **WHEN** 打开 `/module/ifm/rpc/reset-if-counters-by-name`
- **THEN** 内容区 SHALL 仅渲染该 rpc 执行面板，if-name SHALL 渲染为接口名下拉（leafref 驱动）
- **AND** 缺 mandatory input 时执行按钮 SHALL 被校验拦截（不执行）

#### Scenario: 执行回显结果

- **WHEN** 选合法 input 并执行
- **THEN** 前端 SHALL 调用执行 API 并回显 rpc-reply 结果；失败时回显错误（R08）

#### Scenario: 未知 rpc 名降级（负路径）

- **WHEN** 打开 `/module/ifm/rpc/no-such-rpc`
- **THEN** 内容区 SHALL 展示明确错误提示，SHALL NOT 崩溃或空白

#### Scenario: leafref 输入禁自由文本（负路径）

- **WHEN** 打开任一含 leafref 输入的 rpc（如 huawei-ifm `reset-if-control-flap-counts` 的 if-name），且目标列表拉取失败或为空
- **THEN** 该输入 SHALL 仍渲染为下拉（空态+占位提示），SHALL NOT 渲染为可手输的文本框
- **AND** 该输入为 mandatory 时执行按钮 SHALL 保持禁用
### Requirement: FE-20 高危 rpc 执行确认

前端 SHALL 在执行任一 rpc 前弹确认（展示 rpc 名、input 值、目标设备）。对高危 rpc（highRisk 标记，如 `restart-if`）SHALL 升级为更醒目的警示确认。用户未确认时 SHALL NOT 执行。

#### Scenario: 普通 rpc 基础确认

- **WHEN** 执行一个非高危 rpc
- **THEN** 前端 SHALL 先弹确认展示 rpc 名/input/目标设备，确认后才执行

#### Scenario: 高危 rpc 升级警示

- **WHEN** 执行高危 rpc（highRisk，如 restart-if）
- **THEN** 前端 SHALL 展示升级的高危警示确认
- **AND** 用户取消时 SHALL NOT 向设备下发
### Requirement: FE-21 列表详情同屏编辑区（master-detail）

列表 Tab 点击行或行「编辑」SHALL 高亮该行并在表格下方展开详情编辑区，SHALL NOT 使用抽屉/弹窗承载编辑表单。详情编辑区 SHALL 含：条目面包屑（`<list 标签> > <主键值>`）、关闭按钮（收起详情区并取消行高亮）、二级 Tab（由 `deriveDetailTabs` 派生：list 条目标量叶→首个主表单 Tab，嵌套 group→子表单 Tab，嵌套 list→子表格 Tab；超宽横向滚动收纳）、差异预览与「确定」——「确定」SHALL 将该条目变更写入变更集（攒批语义，FE-03），SHALL NOT 直接下发。编辑变更集中已有条目 SHALL 以该条目在变更集中的最新值回填并合并更新（同条目一份变更项）。工具栏「创建」SHALL 在详情编辑区打开空表单（创建态），确定后 SHALL 作为待创建项入集并在列表展示标记行，与编辑同屏同构。只读 Tab（FE-14）SHALL 无详情编辑区（保持只读视图）。切换选中行 SHALL 重载详情表单并丢弃未入集草稿前 SHALL NOT 静默——存在未入集变更时 SHALL 提示确认。

#### Scenario: 点行展开详情

- **WHEN** 点击接口列表某行（或其「编辑」）
- **THEN** 该行 SHALL 高亮，下方 SHALL 展开详情编辑区：面包屑显示该条主键，主表单 Tab 展示标量叶三列表单，嵌套子容器/子 list SHALL 各为一个二级 Tab

#### Scenario: 确定合并同条目变更（边界）

- **WHEN** 对同一条目先后两次编辑并确定（第二次基于第一次的值继续改）
- **THEN** 变更集 SHALL 仅含该条目一份变更项（最新值），diff 基线 SHALL 保持首次编辑前的设备实际态

#### Scenario: 创建态入集

- **WHEN** 点击工具栏「创建」，填写 key 叶后确定
- **THEN** 该条目 SHALL 作为待创建项进入变更集，列表 SHALL 出现带新增标记的行，详情区 SHALL 关闭或切换为该待创建条目的编辑态

#### Scenario: 关闭详情区

- **WHEN** 点击详情区「关闭」
- **THEN** 详情区 SHALL 收起，行高亮 SHALL 取消，列表状态（筛选/分页/排序）SHALL 不变

#### Scenario: 未入集草稿切行确认（负路径）

- **WHEN** 详情区存在未确定的修改时点击另一行
- **THEN** SHALL 弹出确认；取消 SHALL 停留原条目且草稿保留，确认 SHALL 切换并丢弃草稿

#### Scenario: 无嵌套子节点退化（边界）

- **WHEN** 某 list 条目无嵌套 group/list 子节点
- **THEN** 详情区 SHALL 仅含单个主表单 Tab，SHALL NOT 渲染空二级 Tab

### Requirement: FE-22 NCE 表单控件规范（三列栅格/key 标识/约束占位/字段级清除）

详情编辑区与表单 Tab 的表单 SHALL 按三列栅格布局（窄视口 SHALL 降为 2/1 列；choice、leaf-list、嵌套子表格 SHALL 占整行；when 隐藏字段 SHALL NOT 占位）。key 叶 SHALL 呈现钥匙标识（真实图标，R12）且编辑态只读。未携带 `dynamicDefault` 的字段 SHALL 由 schema 契约携带的约束元数据合成 placeholder（数值 range→`整数 合法范围: <范围>`；字符串 length→`合法长度: <范围>`；携带 `default` 时 SHALL 在范围/长度段后追加`，默认值: <值>`，仅有 default 时 SHALL 单独展示`默认值: <值>`，enum 下拉空值 SHALL 同规展示默认值占位，default 值本身 SHALL 原样展示不本地化；元数据由后端契约透出；`dynamicDefault` 字段保持 FE-15 「系统自动分配」占位优先，显式 placeholder 优先级最高）。每个可编辑且已有值的字段旁 SHALL 提供清除控件：对基线（设备实际态）有值的字段，清除 SHALL 记录为该叶的删除意图（随条目入变更集，提交时经叶级删除报文生效，CS-05），tooltip SHALL 明示「提交后将从设备删除该配置项」；对基线无值的字段，清除 SHALL 仅置空本地值（该键不入 payload）。必填字段清除后 SHALL 触发必填校验拦截「确定」。

#### Scenario: 三列栅格与整行控件

- **WHEN** 渲染含 7 个标量叶与 1 个 leaf-list 的表单
- **THEN** 标量叶 SHALL 按三列流式排布，leaf-list SHALL 独占一行

#### Scenario: key 叶钥匙标识与只读

- **WHEN** 编辑既有条目
- **THEN** key 叶 SHALL 展示钥匙图标且输入禁用；创建态 SHALL 可填

#### Scenario: 清除基线有值字段（删除语义）

- **WHEN** 点击某基线有值可选字段旁的清除控件并确定
- **THEN** 该叶 SHALL 以删除意图入变更集，差异预览与变更内容 SHALL 展示为删除行（红），必填字段清除 SHALL 被校验拦截

#### Scenario: 清除基线无值字段（边界）

- **WHEN** 清除一个本次新填、基线无值的字段
- **THEN** 该键 SHALL 仅从本地值与 payload 移除，SHALL NOT 产生删除意图

#### Scenario: 约束合成占位

- **WHEN** 数值字段携带 range `[60, 1000000]` / 字符串字段携带 length `[1..31]`
- **THEN** 输入框空值时 SHALL 展示 `整数 合法范围: [60, 1000000]` / `合法长度: [1..31]` 占位；`dynamicDefault` 字段 SHALL 仍展示「系统自动分配」；显式 placeholder SHALL 优先于合成占位

#### Scenario: 默认值并入合成占位（NCE waterMark 对齐）

- **WHEN** 数值字段携带 range `[10..600]` 且 `default=300`
- **THEN** 占位 SHALL 为 `整数 合法范围: [10, 600]，默认值: 300`

#### Scenario: 仅默认值字段占位（边界）

- **WHEN** 字段无 range/length 但携带 `default`（含 enum 下拉）
- **THEN** 空值时 SHALL 展示 `默认值: <值>` 占位，值原样不本地化

#### Scenario: dynamicDefault 优先于 default（边界）

- **WHEN** 字段同时携带 `dynamicDefault=true` 与 `default`
- **THEN** SHALL 展示「系统自动分配」占位，SHALL NOT 展示默认值段

### Requirement: FE-23 攒批工具栏与变更集交互

模块控制台页头 SHALL 呈现攒批工具栏（一期预留区域）：「变更内容」（含当前设备未提交条目数徽标）、「试运行」、「重置」、「提交配置」；当前设备变更集为空时后三者 SHALL 禁用。存在未提交变更时页面 SHALL 展示「检索到新内容变更，请及时提交」提示条（可关闭）。变更集 SHALL 按设备隔离：切换设备 SHALL 各自保留变更集，徽标与工具栏状态 SHALL 随之切换。「变更内容」SHALL 弹窗展示变更集树形三列（属性/变更前/变更后）与 增加/修改/删除 计数图例（绿/黄/红着色，纯前端渲染）。「试运行」SHALL 调用 preview 接口（CS-01）弹窗展示两个 Tab：待下发设备数据（按设备的正向/回滚报文双栏只读 XML，无 XML 通道模块展示降级说明）与网元数据差异对比（diff 树，标注基线来源与时刻）；预览失败 SHALL 如实报错且不影响变更集。「重置」SHALL 二次确认后清空当前设备变更集，已打开的详情表单 SHALL 恢复为设备实际态，列表标记行 SHALL 还原。存在未提交变更时离开模块控制台路由 SHALL 提示确认。变更集 SHALL 为会话态（刷新页面即丢），SHALL NOT 宣称持久化。

#### Scenario: 工具栏状态随变更集联动

- **WHEN** 变更集从空到有 2 条变更
- **THEN** 徽标 SHALL 显示 2，试运行/重置/提交配置 SHALL 由禁用变为可用，提示条 SHALL 出现

#### Scenario: 变更内容弹窗渲染

- **WHEN** 变更集含 增 1/改 1/删 1 三条，打开「变更内容」
- **THEN** 弹窗 SHALL 以树形三列展示三条变更（新值绿、改前后黄、删除红），图例 SHALL 显示 增加(1) 修改(1) 删除(1)

#### Scenario: 试运行弹窗双 Tab

- **WHEN** 点击「试运行」且 preview 接口成功返回
- **THEN** Tab① SHALL 按设备展示正向/回滚报文双栏 XML，Tab② SHALL 展示 diff 树与基线标注；期间设备与变更集 SHALL 无任何变化

#### Scenario: 试运行失败如实报错（负路径）

- **WHEN** preview 接口返回 400/500
- **THEN** SHALL 展示后端错误信息，变更集 SHALL 保留，弹窗 SHALL NOT 展示伪造报文

#### Scenario: 重置清空当前设备

- **WHEN** 设备 A 有 2 条、设备 B 有 1 条变更，在设备 A 上点「重置」并确认
- **THEN** 设备 A 变更集 SHALL 清空（徽标归零、标记行还原、表单回实际态），设备 B 的 1 条 SHALL 保留

#### Scenario: 切设备变更集隔离

- **WHEN** 设备 A 攒了 3 条变更后切换到设备 B
- **THEN** 徽标 SHALL 显示设备 B 的计数（0），切回设备 A SHALL 恢复显示 3

#### Scenario: 路由离开确认（负路径）

- **WHEN** 存在未提交变更时导航离开模块控制台
- **THEN** SHALL 弹出确认；取消 SHALL 停留当前页且变更集保留

### Requirement: FE-25 列表 Tab 双模式分页（阈值自适应服务端分页）

列表 Tab（含 FE-14 只读状态 list）首次读取 SHALL 携 `limit=200&offset=0`（只读状态 Tab 另携 `include_state=true`）自适应选择模式：

- `total ≤ 200`：SHALL 进入**纯前端模式**——已获全量行，过滤/分页/排序在本地完成，交互与 FE-11 现状完全一致，SHALL NOT 产生额外网络往返；
- `total > 200`：SHALL 进入**服务端模式**——翻页、每页条数、高级搜索（support-filter 字段集映射为 `filter` 参数，等值/包含语义与后端一致）与排序 SHALL 映射为查询参数重新请求；表格下方总记录数 SHALL 取响应 `total`；请求期间表格 SHALL 呈现 loading 态（覆盖快照过期重拉的延迟尖刺）。

两种模式下 FE-11 既有 UI 元素（工具区/列设置/操作列/分页器/完成时刻）SHALL 保持不变；变更集攒批的 pending create 行 SHALL 仍在本地叠加展示且 SHALL NOT 计入服务端 `total`；「获取数据源」SHALL 以 `force_refresh=true` 全量重拉并复位到第一页。

#### Scenario: 小表维持纯前端零回归

- **WHEN** list 总行数 66（≤200），用户翻页与本地搜索
- **THEN** SHALL NOT 发起新的 /config 请求，行为与现状一致

#### Scenario: 大表翻页走服务端

- **WHEN** total=12000 的状态 list 处于服务端模式，用户跳转第 5 页
- **THEN** SHALL 携 `limit/offset` 重新请求并渲染返回的 rows，总记录数展示 12000

#### Scenario: 服务端模式高级搜索下推

- **WHEN** 服务端模式下在高级搜索面板提交「name 包含 GE」
- **THEN** SHALL 携 `filter=name~=GE` 重新请求且页码复位第一页，SHALL NOT 在本地对当前页再过滤

#### Scenario: pending create 行本地叠加（边界）

- **WHEN** 服务端模式下变更集含该 list 的未提交新建行
- **THEN** 新建行 SHALL 叠加展示于当前页且带待提交标识，服务端 total SHALL 不含该行

#### Scenario: 获取数据源复位（边界）

- **WHEN** 服务端模式第 5 页点击「获取数据源」
- **THEN** SHALL 携 `force_refresh=true` 重拉、页码复位第一页并刷新新鲜度
### Requirement: FE-26 日志操作类型模型驱动派生

操作日志的「操作类型」标签 SHALL 由审计路径首段模块名派生（剥命名空间前缀）：
已知模块 SHALL 显示其菜单标题（与左侧导航称谓一致），未知模块 SHALL 回退显示段名，
空路径 SHALL 回退通用标签。派生 SHALL NOT 按具体模块名硬编码分支（R05 同口径）——
新模块接入 SHALL 零前端改动获得正确标签。模块标题映射不可用时 SHALL 降级为段名展示（R08）。

#### Scenario: 已知/未知模块与降级

- **WHEN** 审计路径为 `/vlan:vlan/vlan:vlans` 且模块标题映射含 `vlan`
- **THEN** 操作类型 SHALL 显示 vlan 模块的菜单标题
- **WHEN** 审计路径首段不在模块标题映射中（含映射整体不可用）
- **THEN** 操作类型 SHALL 显示该段名而非通用占位
- **WHEN** 审计路径为空
- **THEN** 操作类型 SHALL 显示通用标签且渲染不失败

### Requirement: FE-27 表单键存在性即节点存在性

表单状态 SHALL 以「键是否存在」表达 YANG 节点是否存在：presence 容器关闭、choice 非激活分支成员、动态缺省叶留空、字段级清除等场景，对应键 SHALL 从表单数据中真正移除，SHALL NOT 仅置为空值（`undefined`/`null`）。下发 payload SHALL NOT 包含这些键。

#### Scenario: presence 容器关闭后键消失
- **WHEN** 用户关闭 presence 容器开关
- **THEN** 表单数据中该容器键 SHALL 不存在（以「键枚举」判定，非「取值为空」判定），且 SHALL NOT 进入下发 payload

#### Scenario: choice 切换分支清空非激活成员
- **WHEN** 用户从 case A 切换到 case B
- **THEN** case A 全部成员键 SHALL 从表单数据中移除，payload SHALL 只含 case B 成员

#### Scenario: 动态缺省叶留空不下发（负路径）
- **WHEN** 带 `dynamicDefault` 的叶被清空
- **THEN** 该键 SHALL NOT 进入 payload，SHALL NOT 以空串或 null 形式下发覆盖设备缺省
