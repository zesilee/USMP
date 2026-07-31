## Context

目标形态 = 华为 NCE 设备配置台（`~/ui/` 7 张截图，2026-07-31）。现状（FE-10/FE-11）：
`ModuleConsolePage` 一级 Tab 由 `deriveTabs` 派生；list Tab（`ModuleListTab`）已有 新增按钮、
support-filter 高级搜索、客户端分页、DELETE 行删除、el-drawer 抽屉编辑（内嵌
useConfigForm + DiffPreview + ReconcileSteps）；`deriveColumns` cap=9 截断；boolean 渲染为开关；
约束（range/length/pattern）已数据驱动但不以 placeholder 呈现；左树（LT-01~04）已是
snd 14 业务域分组 → 模块叶 → container+rpc 平铺（与 NCE 截图同源同构），缺节点搜索与
一键展开/收起。

拍板范围：一期界面对齐，攒批提交（变更内容/试运行/重置/提交配置）二期；设备入口保留顶部
设备下拉；只动配置台，其他页面不动布局。后端零接口变更（`force_refresh` BR-04、DELETE BR-09
均已交付可复用）。

## Goals / Non-Goals

**Goals:**
- 列表→详情同屏 master-detail：点行/点编辑 = 行高亮 + 下半屏详情编辑区（面包屑 + 二级 Tab + 三列表单 + 关闭），抽屉退役。
- 列表工具区对齐：创建/刷新按钮、高级搜索折叠区（保留）、列头排序、enum 列头筛选、列设置（显示/隐藏列，解除 cap=9 对可用列的限制）、多选框列、查询时间戳+总记录数、分页增强（页码+前往）。
- 行操作三件套：编辑 / 删除 / 获取数据源（= force_refresh 回读）。
- 表单 NCE 控件规范：三列栅格、key 钥匙图标+编辑态只读、boolean「打开/关闭」radio、约束 placeholder、字段级清除（垃圾桶）。
- 左树升级：树内搜索（命中保留祖先链并自动展开）、一键展开/收起（分组层已与 NCE 同源同构，不动）。

**Non-Goals:**
- 攒批变更集与 变更内容/试运行/重置/提交配置/导出/配置项（二期；一期只预留右上工具栏布局区域，不渲染死按钮）。
- 后端 API 变更；单行粒度设备回读（API 无此粒度，见 D8）。
- Dashboard/Devices/Logs/Settings 布局改版。

## Decisions

**D1 左树增强而非搬迁。** 左树仍在 MainLayout 全局侧栏（搬进页面 = 全站布局重排，超出拍板范围）。`LeftTreeMenu` 增加：搜索框（按节点名过滤，命中自动展开祖先）、展开/收起全部按钮、顶层业务分类层。

**D2 树搜索为纯客户端过滤。** 左树数据已全量在前端（menu store），搜索按 zh/en/name 三口径递归匹配节点名，命中节点保留祖先链并自动展开；清空恢复全树与默认折叠态。展开/收起全部 = 操作树的 open 状态集。SHALL NOT 改 LT 生成物与查询接口。替代方案「后端搜索接口」否决：数据量小（14 组/65 叶/模块级 children），客户端过滤零延迟且无契约变更。

**D3 master-detail 取代抽屉。** `ModuleListTab` 拆为列表区 + 详情区（新组件 `ItemDetailPane`）：点行/点「编辑」→ 行高亮 + 详情区展开该条；「创建」→ 详情区空表单（create 模式）；关闭按钮收起。提交编排（useConfigForm/useConfigSubmit/DiffPreview/ReconcileSteps）原样迁入详情区，仍为即时下发（一期语义不变）。el-drawer 路径删除。

**D4 详情二级 Tab 新派生 `deriveDetailTabs`。** list 条目的标量叶 → 首个主 Tab（表单三列）；每个嵌套 group → 子表单 Tab；嵌套 list → 子表格 Tab（复用既有嵌套 list 编辑能力）；超宽 Tab 集合以「更多」下拉溢出。`ModuleFormTab` 的嵌套 group 同样改为二级 Tab 呈现（对齐截图「全局配置属性→IPv4地址冲突检测」形态）。新派生函数纳入 GD-01 派生黄金。

**D5 列全集 + 列设置，默认仍 9 列。** `deriveColumns` 派生全部标量叶为「可用列全集」，默认显示集 = 现分层规则前 9（黄金语义兼容：默认集不变，新增全集维度）；列设置齿轮控制显隐，宽表横向滚动。排序：全列客户端排序（el-table sortable）。列头筛选：enum/boolean 用 el-table 原生 filters；文本列筛选继续走高级搜索（一期不自绘全列文本筛选弹层，降复杂度）。

**D6 控件规范落在 FieldRenderer。** boolean → radio「打开/关闭」（i18n 文案，值仍 true/false）；placeholder 由既有约束元数据合成（range→`整数 合法范围: …`、length→`合法长度: …`、pattern 有 description 则用之）；key 叶加钥匙图标、编辑态只读（复用 isCreateOnly/isKey）；每可编辑字段旁垃圾桶 = 本地清空该字段值（payload 中剔除该叶）。**诚实边界**：一期「清除」是本地不下发该叶，不产生设备侧 leaf 删除语义（那属于二期攒批变更集），文档与 tooltip 如实说明。

**D7 表单三列栅格。** 详情/表单区 el-row 3×el-col-8（窄屏降 2/1 列）；label 上置改为 NCE 式 label 顶部+控件下置的紧凑布局。choice/leaf-list 等宽控件占整行。

**D8 「获取数据源」= 列表级 force_refresh。** 行操作触发 `getConfig(..., force_refresh=true)` 绕缓存回读整个 list 路径（BR-04），刷新后保持该行选中与详情区打开。API 无单行读取粒度，不伪造「单行回读」文案。

**D9 时间戳与记录数。** 表格下方展示「{查询完成时刻} 查询结束，总记录数: N」（N = 过滤后全集数），数据来源为最近一次 load/force_refresh 完成时刻。

## Risks / Trade-offs

- [派生黄金大面积震动：deriveDetailTabs 新增 + deriveColumns 全集维度] → 按 GD-01 全模块刷新黄金并逐模块人工核对（SF-04）；默认列集不变以缩小 diff 面。
- [F4 staging-smoke 与 F2/F3 大量断言基于抽屉交互] → 断言随 D3 全部改写为 master-detail 交互；`make e2e-local` 全绿是 pre-push 硬门禁，改写与实现同 PR。
- [68 模块形态各异（纯 form 模块、无嵌套 list 模块）] → deriveDetailTabs 空子集时详情区退化为单主 Tab；派生黄金全模块核对兜底。
- [三列栅格 + when 显隐 → 空洞与跳变] → 隐藏字段不占位（栅格流式补位）；F2 校验态用例覆盖。
- [字段级清除被误解为设备侧删除] → tooltip 明示「清空本次不下发该字段」；二期攒批上线后升级语义。
- [worktree 前端踩坑（node_modules 勿 symlink、勿 npx vitest、Playwright 双版本）] → 按既有记忆 checklist 执行。

## Migration Plan

单 worktree 分支 `nce-console-redesign`，按 PR ≤1000 行（TM04）拆四段，每段独立可合：
1. 纯逻辑派生层：deriveDetailTabs + deriveColumns 全集/默认集双维度 → 派生黄金刷新（F1 + 黄金，GD-01 人工核对）。
2. master-detail 重构：ItemDetailPane、抽屉退役、二级 Tab、时间戳/分页/多选/刷新/创建/获取数据源（F2 + F3 + F4 改写）。
3. 控件规范：FieldRenderer boolean-radio/约束 placeholder/钥匙图标/字段级清除 + 三列栅格（F2/F3）。
4. 左树：搜索 + 展开/收起（过滤纯函数 F1 + 组件 F2；LT 生成物与接口不动）。

回滚：各段独立 revert；无后端与数据迁移，回滚零残留。

## Open Questions

（无——设备入口、范围、二期边界均已与用户拍板。）
