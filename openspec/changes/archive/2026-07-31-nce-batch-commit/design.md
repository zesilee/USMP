# Design — nce-batch-commit

## Context

一期已交付 master-detail 配置台与工具栏预留位。现状提交链路（审计 @origin/main 2026-07-31）：

- 前端：`ItemDetailPane.submit()` → `useConfigSubmit.run(ip, item)`（单条、恒包 `{listKey:[item]}`）→ `POST /config/:ip/*path` → force 回读 → 轮询对账；`ModuleFormTab` 直调 `setConfig` 无编排；行删除 `DELETE /config` 即时命令语义。变更状态散在组件局部（`formData/original`），切行即丢；`computeDiff` 表达不了删除（`now===''` 被忽略）；无任何「待提交变更」store。
- 后端：`POST /config` 为异步声明式（写 desired→失效缓存→触发对账→立即 200）；diff 引擎（`pkg/yang-runtime/diff`，纯函数，Change 带 Type/Old/New）与 XML 编码（`xmlcodec.Encode/EncodeDelete`，纯函数）可离线复用；`client.marshalChange` 的注册表分派是未导出方法；`intent.TxCoordinator` 已有跨设备 candidate 2PC（prepare→confirmed-commit→confirm，失败 discardAll、设备锁防死锁），但 `Fragment` 仅表达 merge，无删除语义；netconfsim 支持 candidate/discard/confirmed-commit 与 RFC6241 per-operation 全套；审计挂点在 config handler 成功路径。
- NCE 截图语义：变更内容=树形三列 diff+计数图例；试运行=正向/回滚报文双栏 XML + 网元数据差异对比（变更前=控制器保存的目标配置，变更后=试运行算出的目标配置——**两侧都是控制器侧计算，不是设备实时值**）。

拍板（2026-07-31）：完全改攒批（即时下发退役）；失败整体回退；导出/配置项本期不做。

## Goals / Non-Goals

**Goals:**
- 变更集：编辑/创建/删除/字段清除全部先入前端变更集（按设备隔离），可核对（变更内容）、可预览（试运行）、可反悔（重置）、一次提交（提交配置）。
- 试运行只算不下发：正向/回滚 NETCONF 报文 + 结构化 diff 树，由后端纯计算生成。
- 提交原子性：单设备内跨模块「全成或全退」（candidate 两阶段，失败 discard）。
- 字段级清除升级为真删除语义（leaf 级 `nc:operation="delete"`）。

**Non-Goals:**
- 导出、配置项按钮；跨设备联合变更集（变更集按当前设备隔离，提交只针对一台设备）；变更集持久化（刷新页面即丢，如实提示）；NETCONF `<validate>`（模拟器只会假装成功，不做伪验证）；confirmed-commit 网络级演练；RPC 执行与业务意图通道（保持现状）。

## Decisions

**D1 变更集 = 前端 Pinia store，按设备隔离，不持久化。** 新增 `changeset` store：`Map<deviceIp, ChangesetEntry[]>`，条目 = `{ op: 'create'|'update'|'delete', anchorPath, listKey?, keyValue?, payload(RFC7951 子树), baseline(编辑起点快照), clearedLeaves[] }`。替代方案「后端存变更集（CRD）」否决：变更集是单人单会话草稿，无跨实例共享需求，进 CRD 违背 R03 精神（CRD 当载体不当草稿箱）且引入清理生命周期；刷新丢草稿用离开确认 + 「请及时提交」提示条兜底，文档如实说明。

**D2 「确定」三路全部改写入变更集，即时下发退役。** `ItemDetailPane`（list 条目 create/update）、`ModuleFormTab`（非 list 表单 update）、行删除与多选批量删除（delete 条目，按 设备+锚点+主键 去重；对既有 create 条目的删除=直接移除该条目）。同一条目多次编辑合并为一条（payload 覆盖、baseline 保持首次快照）。`useConfigSubmit` 的单条即时编排退役，改为变更集提交编排（D6）。RPC、BusinessConsole 意图通道不动。

**D3 字段级清除 = leaf 删除语义。** 变更集 update 条目携 `clearedLeaves`（被清除且 baseline 有值的叶）；预览/提交时编码为条目内该叶 `nc:operation="delete"`。需要扩展 `xmlcodec`：在既有 `EncodeDelete`（条目级）之外支持**叶级删除**（条目定位键 + 目标叶打 delete 操作）。一期「清空本次不下发」tooltip 文案退役，改为「提交后从设备删除该配置项」。

**D4 试运行 = 后端纯计算接口。** `POST /api/v1/config/changeset/preview`，入参 `{device, entries[]}`，出参按设备聚合 `{forward_xml, rollback_xml, diff: DiffResult 树}`。实现：逐条目 `convertConfigAnchored` 解码 → 基线取「控制器目标态」= ConfigStore desired，缺失锚点回退 running cache（再缺失实时 GET）——与 NCE「变更前=控制器保存的目标配置」口径一致 → `diff.DefaultDiffEngine.Diff` → 正向 XML 走从 `marshalChange` 提取的导出纯函数（注册表分派 + `xmlcodec.Encode/EncodeDelete`）；**回滚 XML = 把 Change 的 ADD↔DELETE 互换、MODIFY Old/New 互换后走同一编码**。无 XML 通道的模块（如 system）如实返回「该模块不支持报文预览」降级，不猜。审计不记试运行（只读操作，与被拒请求不写审计同口径）。

**D5 提交 = 单设备原子批量接口，复用 TxCoordinator。** `POST /api/v1/config/changeset/commit`，同步执行：变更集 → `[]intent.Fragment`（**扩展 Fragment 加 `Op` 字段支持 delete**，prepare 里映射为 `DeleteChange`/叶级删除，BIO-03 既有意图链路行为不变）→ `TxCoordinator.Push`（candidate 逐条 edit-config，任一失败 discardAll 整体回退；commit 成功才落地）。**desired 写入时序：设备 commit 成功之后**才逐条 `storeConfigMerged`/`storeConfigDeleted` + 失效缓存 + 审计（复用 OA-01 挂点，每条目一条）+ 触发对账——若先写 desired 而设备提交失败，周期对账会把失败的变更重新推上去，破坏「整体回退」承诺。替代方案「前端 N 次现有 POST/DELETE」否决：无原子性，直接违背拍板 2。

**D6 前端提交编排改批量。** 「提交配置」→ 确认弹窗（复述条目数）→ 调 commit 接口（pushing）→ 成功后 force 回读 + 轮询对账（复用 `ReconcileSteps`/`reconcileProgress` 状态机，baseline 取 commit 前 last_run）→ 清空该设备变更集、刷新当前列表与新鲜度；失败如实展示后端错误、变更集原样保留（R08/§9）。「重置」→ 确认后清空当前设备变更集，已打开的详情表单 `resetForm` 回设备实际态。

**D7 工具栏与弹窗。** `ModuleConsolePage` `.header-actions` 内、设备下拉左侧插入按钮组：变更内容（徽标=当前设备未提交条目数）/试运行/重置/提交配置，无变更时后三者禁用；有变更时页顶提示条「检索到新内容变更，请及时提交」。变更内容弹窗=纯前端渲染（changeset entries：树形 属性/变更前/变更后 三列 + 增/改/删计数图例，绿/黄/红着色，复用 DiffEntry 扩展出的删除表达）；试运行弹窗=调 preview 接口（Tab① 正向/回滚双栏只读 XML；Tab② diff 树复用变更内容树组件）。切设备保留各设备变更集（徽标随设备切换）；路由离开且有未提交变更时确认。

**D8 configDiff 补删除表达。** `DiffEntry` 加 `op: 'add'|'modify'|'remove'`；`computeDiff` 识别「baseline 有值且被清除」为 remove（一期 `now===''` 忽略语义仅对无 baseline 字段保留）。DiffPreview 渲染删除行（红色删除线）。此为纯函数变更但**不属于 GD-01 派生函数枚举**（deriveTabs/deriveColumns 等不动），黄金不刷新。

**D9 string length 占位元数据（附带债）。** `field_gen` 从 goyang Entry 透出 string length 约束到 FieldDef（契约生成同步），一期 FE-22「合法长度」placeholder 自动生效。独立小任务，不与主线耦合。

## Risks / Trade-offs

- [批量 candidate 期间周期对账插入同一 candidate 混写] → 提交前提是稳态（desired 与设备已收敛，周期对账 no-op）；防御：commit 接口内先按设备取 TxCoordinator 同款设备锁、提交完成才触发对账；风险窗口与现状单模块下发相同，不新增敞口。
- [Fragment/TxCoordinator 扩展波及意图链路] → `Op` 字段默认 merge，既有意图调用零改动；BIO-03 行为用既有 B2 测试回归兜底。
- [叶级删除编码是 xmlcodec 新能力] → 先 B1 表格驱动（含嵌套条目、多叶、幂等），netconfsim per-operation delete 语义已实现可端到端验证；华为设备真机语义差异留验证口。
- [即时下发退役导致 F2/F4 大面积断言失效] → 断言与实现同 PR 改写；`make e2e-local` pre-push 硬门禁。
- [变更集不持久化，刷新即丢] → 离开确认 + 提示条 + 文档如实说明；不做 localStorage（多标签互踩，诚实优先）。
- [preview 基线取 desired 优先，desired 与设备漂移时预览失真] → Tab② 名为「网元数据差异对比」但基线是控制器口径（与 NCE 一致），弹窗内如实标注基线来源与时刻；用户可先「获取数据源」强刷再试运行。
- [无 XML 通道模块（system 等）不能报文预览] → 显式降级文案，不伪造报文；diff 树仍可用。

## Migration Plan

单 worktree 分支，按 PR ≤1000 行拆段（每段独立可合，测试先行）：
1. 后端纯函数层：marshal 分派提取导出 + 回滚反算 + xmlcodec 叶级删除（B1）。
2. 后端接口：preview + commit（Fragment 补 Op、desired 时序、审计）（B3+B2 netconfsim 端到端：试运行不改设备、提交原子、失败回退）。
3. 前端纯逻辑：changeset store + configDiff 删除表达 + API client（F1）。
4. 前端工具栏与弹窗：四按钮/徽标/提示条/变更内容/试运行（F2+F3）。
5. 前端链路切换：三路「确定」入集、删除入集、批量删除、提交/重置编排、即时路径退役、F4 全量改写。
6. 收尾：D9 string-length 债、spec sync、归档。

回滚：前端各段独立 revert；后端新增接口无存量调用方，revert 零残留；不动既有 POST/DELETE 契约（保留为兼容面，前端不再调用）。

## Open Questions

（无——提交语义、失败处理、按钮范围均已拍板；真机 leaf-delete 语义验证留待有真机窗口时执行，不阻塞本期。）
