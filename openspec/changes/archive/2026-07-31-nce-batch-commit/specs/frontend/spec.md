# frontend — delta（nce-batch-commit）

## MODIFIED Requirements

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

详情编辑区与表单 Tab 的表单 SHALL 按三列栅格布局（窄视口 SHALL 降为 2/1 列；choice、leaf-list、嵌套子表格 SHALL 占整行；when 隐藏字段 SHALL NOT 占位）。key 叶 SHALL 呈现钥匙标识（真实图标，R12）且编辑态只读。未携带 `dynamicDefault` 的字段 SHALL 由 schema 契约携带的约束元数据合成 placeholder（数值 range→`整数 合法范围: <范围>`；字符串 length→`合法长度: <范围>`，元数据由后端契约透出；`dynamicDefault` 字段保持 FE-15 「系统自动分配」占位优先，显式 placeholder 优先级最高）。每个可编辑且已有值的字段旁 SHALL 提供清除控件：对基线（设备实际态）有值的字段，清除 SHALL 记录为该叶的删除意图（随条目入变更集，提交时经叶级删除报文生效，CS-05），tooltip SHALL 明示「提交后将从设备删除该配置项」；对基线无值的字段，清除 SHALL 仅置空本地值（该键不入 payload）。必填字段清除后 SHALL 触发必填校验拦截「确定」。

#### Scenario: 三列栅格与整行控件

- **WHEN** 渲染含 7 个标量叶与 1 个 leaf-list 的表单
- **THEN** 标量叶 SHALL 按三列流式排布，leaf-list SHALL 独占一行

#### Scenario: key 叶钥匙标识与只读

- **WHEN** 编辑既有条目
- **THEN** key 叶 SHALL 展示钥匙图标且输入禁用；创建态 SHALL 可填

#### Scenario: 约束合成占位

- **WHEN** 数值字段携带 range `[60, 1000000]` / 字符串字段携带 length `[1..31]`
- **THEN** 输入框空值时 SHALL 展示 `整数 合法范围: [60, 1000000]` / `合法长度: [1..31]` 占位；`dynamicDefault` 字段 SHALL 仍展示「系统自动分配」；显式 placeholder SHALL 优先于合成占位

#### Scenario: 清除基线有值字段（删除语义）

- **WHEN** 点击某基线有值可选字段旁的清除控件并确定
- **THEN** 该叶 SHALL 以删除意图入变更集，差异预览与变更内容 SHALL 展示为删除行（红），必填字段清除 SHALL 被校验拦截

#### Scenario: 清除基线无值字段（边界）

- **WHEN** 清除一个本次新填、基线无值的字段
- **THEN** 该键 SHALL 仅从本地值与 payload 移除，SHALL NOT 产生删除意图

## ADDED Requirements

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
