---
name: nce-console-redesign
description: NCE 风格配置台改版一期全交付：master-detail/工具区/控件规范/左树搜索；二期=攒批提交；含 6 个前端测试踩坑
metadata:
  type: project
---

# NCE 配置台改版（一期已交付 2026-07-31）

**拍板**（用户确认）：一期只做界面对齐；「变更内容/试运行/重置/提交配置」**攒批提交是二期**（一期不渲染死按钮，右上区域预留）；设备入口保持顶部设备下拉；只动配置台。目标截图 7 张在 `~/ui/`，研究底稿 [[imaster-nce-ux-insights]]。

**交付**（PR#238 派生层 → #240 ItemDetailPane → #239 master-detail → #241 控件规范 → PR-4 左树搜索）：

- 列表详情同屏（FE-21）：点行/编辑=行高亮+下方 `ItemDetailPane`（面包屑/关闭/二级 Tab/未提交草稿切行确认），**抽屉退役**；创建同屏同构，成功后保持展开为新条目。
- 派生新函数：`deriveDetailTabs`（标量→__main__ 主 Tab、嵌套 group→子表单、嵌套 list→子表格）、`deriveAllColumns`（=deriveColumns(∞)，默认 9 列恒为前缀）——**均入 GD-01 派生黄金**（cap=9 截断的旧 follow-up 由列设置解决）。
- 列表工具区（FE-11）：创建/刷新、列设置齿轮（全集勾选、会话内生效）、多选列（**动作二期**）、全列排序、enum/boolean 列头筛选（文本筛选走高级搜索——D5 决策）、时间戳+总记录数、跳页。
- 「获取数据源」= 列表级 `force_refresh`（BR-04），**无单行粒度，文案不伪造**。
- 控件规范（FE-22/FE-01/FE-02）：boolean→打开/关闭 radio；数值 range 合成占位（**string length 契约无元数据=已知债**，后端透出后自动生效）；key 钥匙图标+编辑态只读；字段级清除=**本地不下发该叶**（非设备删除，tooltip 明示；leaf 删除语义属二期）；三列栅格（when 隐藏不占位）；ModuleFormTab 嵌套 group→二级 Tab（复用 deriveDetailTabs，面板常驻隐藏非销毁）。
- 左树（LT-05）：搜索（filterLeftTree 纯函数，zh/en/name 三口径、祖先链保留）+展开/收起全部（subIndexes 与 LeftTreeMenu index 规则同构）；分类分组**本来就有**（snd 14 组与 NCE 同源）。

**Why:** 商用网管操作密度；派生驱动零模块硬编码（R05）。

**How to apply / 踩坑（改前端测试必读）：**
- `enableAutoUnmount` 只能挂 happy-dom（browser 模式用例自带 unmount，双卸载 removeChild 崩）；不卸载重表格会让单文件后段用例递增变慢到超时。
- `vi.clearAllMocks` 不清 `mockResolvedValueOnce` 队列——跨用例残留会错位，用 `resetAllMocks`。
- Playwright `getByRole` 走无障碍树**自动排除 display:none 的隐藏 Tab 面板**，换 CSS 选择器会命中隐藏面板同名按钮（点不着超时）。
- el-input 非 prop 属性透传到内层 `<input>` 本体：`[data-test] input` 后代选择器落空，直接 `input[data-test=...]`。
- **coverage/e2e 并行跑会互相抢资源出假失败**（覆盖率虚低、用例超时、测试数变少），验证必须串行干净跑；本地覆盖率还可能被同机 staging 后端灌水（vitest.config 注释有账）。
- happy-dom testTimeout 已放宽 15s（CI 慢跑道）；覆盖率棘轮现值 **85/79/79/86**。
- commit ≤500 行、PR ≤1000 行：大改版按「纯逻辑派生层→组件→接入」切 PR，测试先行 commit 可先着地。

**二期已交付**（见 [[nce-batch-commit]]，2026-07-31）。原二期待办：攒批变更集+变更内容 diff+试运行（正向/回滚报文、网元差异对比）+重置/提交配置/导出/配置项；多选批量操作；「更多▾」列表批量菜单；string length 占位元数据（后端 schema 透出）。

**真机接入回归（2026-08-04）**：编辑态入集载荷已由「整行 visiblePayload」改为 **changedPayload = 主键+相对基线改动字段**（ItemDetailPane/ModuleFormTab 双切；BusinessConsolePage 意图表单有意保留全量）。因：真机按接口类型裁剪 config 叶能力（statistic-mode 等），回读整行原样回推被 rpc-error unknown-element 拒绝。正确性前提=基线**深快照**（snapshotBaseline）：浅拷贝下嵌套对象与 formData 共享引用，原位编辑同步污染基线→嵌套改动被误判未变。配套：2PC prepare 已修为 per-change 错误优先透出设备 rpc-error 细节（与 pushDeleteToDevice 同口径），界面不再只见「one or more changes failed」。
