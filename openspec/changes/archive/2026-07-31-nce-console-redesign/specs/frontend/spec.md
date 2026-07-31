# frontend — delta（nce-console-redesign）

## MODIFIED Requirements

### Requirement: FE-01 schema 驱动渲染

前端 SHALL 将后端 YANG nested schema 经 `crdSchemaParser` 逐属性映射为 `Field[]`，类型映射为 boolean→「打开/关闭」radio 单选组（i18n 文案，值仍为 true/false；可选 boolean SHALL 支持不选=不入 payload）、number→input-number、object→group；enum SHALL 按选项数与必填性细分：**必填且选项 ≤3 → segmented 分段控件，其余（可选或 >3 选项）→ select 下拉**（可选枚举 SHALL 保留清空能力，清空即该键不入 payload）。映射经 `FieldRenderer` 渲染为 Element Plus 控件（R05）。SHALL NOT 手写固定表单。

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

### Requirement: FE-02 分组与校验

Field 带 group/pattern/min/max/required 时，前端渲染 SHALL 按分组组织（>1 分组时 SHALL 渲染为二级 Tab，NCE 形态；Tab 集合超宽 SHALL 可横向滚动收纳（不截断）），并由约束生成校验 rules；校验失败 SHALL NOT 提交，且 SHALL 行内提示 YANG 约束（§9、R08）。

#### Scenario: 多分组二级 Tab
- **WHEN** 字段分布在 >1 个 group
- **THEN** SHALL 渲染二级 Tab 切换分组，切换 SHALL 保留各分组表单状态

#### Scenario: 校验失败不提交
- **WHEN** 存在缺失必填或数值越界（超出 min/max）
- **THEN** SHALL 阻止提交并在行内展示约束提示

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
- 表格 SHALL 含多选框列（为二期批量操作预留选择态，一期无批量动作）；
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

## ADDED Requirements

### Requirement: FE-21 列表详情同屏编辑区（master-detail）

列表 Tab 点击行或行「编辑」SHALL 高亮该行并在表格下方展开详情编辑区，SHALL NOT 使用抽屉/弹窗承载编辑表单。详情编辑区 SHALL 含：条目面包屑（`<list 标签> > <主键值>`）、关闭按钮（收起详情区并取消行高亮）、二级 Tab（由 `deriveDetailTabs` 派生：list 条目标量叶→首个主表单 Tab，嵌套 group→子表单 Tab，嵌套 list→子表格 Tab；超宽横向滚动收纳）、以及既有提交编排（差异预览、下发、对账进度——一期仍为即时下发语义）。工具栏「创建」SHALL 在详情编辑区打开空表单（创建态），与编辑同屏同构。只读 Tab（FE-14）SHALL 无详情编辑区（保持只读视图）。切换选中行 SHALL 重载详情表单并丢弃未提交草稿前 SHALL NOT 静默——存在未提交变更时 SHALL 提示确认。

#### Scenario: 点行展开详情

- **WHEN** 点击接口列表某行（或其「编辑」）
- **THEN** 该行 SHALL 高亮，下方 SHALL 展开详情编辑区：面包屑显示该条主键，主表单 Tab 展示标量叶三列表单，嵌套子容器/子 list SHALL 各为一个二级 Tab

#### Scenario: 创建态同屏

- **WHEN** 点击工具栏「创建」
- **THEN** 详情编辑区 SHALL 展开空表单（key 叶可填），提交成功 SHALL 刷新列表并保持详情区展开为新条目

#### Scenario: 关闭详情区

- **WHEN** 点击详情区「关闭」
- **THEN** 详情区 SHALL 收起，行高亮 SHALL 取消，列表状态（筛选/分页/排序）SHALL 不变

#### Scenario: 未提交草稿切行确认（负路径）

- **WHEN** 详情区存在未提交修改时点击另一行
- **THEN** SHALL 弹出确认；取消 SHALL 停留原条目且草稿保留，确认 SHALL 切换并丢弃草稿

#### Scenario: 无嵌套子节点退化（边界）

- **WHEN** 某 list 条目无嵌套 group/list 子节点
- **THEN** 详情区 SHALL 仅含单个主表单 Tab，SHALL NOT 渲染空二级 Tab

### Requirement: FE-22 NCE 表单控件规范（三列栅格/key 标识/约束占位/字段级清除）

详情编辑区与表单 Tab 的表单 SHALL 按三列栅格布局（窄视口 SHALL 降为 2/1 列；choice、leaf-list、嵌套子表格 SHALL 占整行；when 隐藏字段 SHALL NOT 占位）。key 叶 SHALL 呈现钥匙标识（真实图标，R12）且编辑态只读。未携带 `dynamicDefault` 的字段 SHALL 由**当前 schema 契约携带的**约束元数据合成 placeholder（数值 range→`整数 合法范围: <范围>`；字符串 length 现契约未透出，待后端补充元数据后按同规则自动生效；`dynamicDefault` 字段保持 FE-15 「系统自动分配」占位优先，显式 placeholder 优先级最高）。每个可编辑且已有值的字段旁 SHALL 提供清除控件：点击 SHALL 清空该字段本地值（该键不入 payload），tooltip SHALL 明示「清空后本次不下发该字段」，SHALL NOT 暗示设备侧删除语义（leaf 级删除属二期攒批变更集）。

#### Scenario: 三列栅格与整行控件

- **WHEN** 渲染含 7 个标量叶与 1 个 leaf-list 的表单
- **THEN** 标量叶 SHALL 按三列流式排布，leaf-list SHALL 独占一行

#### Scenario: key 叶钥匙标识与只读

- **WHEN** 编辑既有条目
- **THEN** key 叶 SHALL 展示钥匙图标且输入禁用；创建态 SHALL 可填

#### Scenario: 约束合成占位

- **WHEN** 数值字段携带 range `[60, 1000000]`
- **THEN** 输入框空值时 SHALL 展示 `整数 合法范围: [60, 1000000]` 占位；`dynamicDefault` 字段 SHALL 仍展示「系统自动分配」；显式 placeholder SHALL 优先于合成占位

#### Scenario: 字段级清除

- **WHEN** 点击某有值可选字段旁的清除控件
- **THEN** 该字段 SHALL 置空且该键 SHALL NOT 出现在差异预览与下发 payload；必填字段清除后 SHALL 触发必填校验拦截提交
