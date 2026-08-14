---
name: react-antd-rebuild
description: 前端 Vue→React+antd 原地重建已全量交付归档(2026-08-14,PR#316-#337)：适配层军规/E2E 对等三根因/F3 惯例/拆 PR 门禁方法论/follow-up 债；碰前端任何文件前必读
metadata:
  type: project
---

**change `frontend-react-antd-switch` 已全量交付归档**（2026-08-14，PR #316-#337，71/71 tasks；制品在 `openspec/changes/archive/`）：Vue3+Element Plus 整体退役，现栈 = **React 19 + antd 6.6 + Vite 8 + zustand + react-router 8 + 自研 i18n 薄层**（同形 `i18n.global.t` API，词表键名原样沿用）。E2E 对等以旧 staging-smoke 套件为验收标尺，chromium/firefox/webkit 63/63 全绿；覆盖率棘轮 94.5/83.3/94.5/96.1。

**Why:** 重建方法论与 antd 特有坑是数十个 PR 反复踩出来的；换栈后的军规锚点（适配层/键存在性）不写下来会被后续开发无意破坏。

**How to apply:**
- **适配层军规（FA-01~04，主 spec frontend-ui-adapter）**：业务代码禁直接 import antd——只从 `src/ui` 导入（守护测试硬拦，为换 EviewUI 留单点）；反馈用 `toast()`/`await confirm()`；图标/主题令牌经适配层收口。
- **FE-27 键存在性=节点存在性**（主 spec frontend）：删键必解构，守护测试拦 `{...prev,[k]:undefined}` 伪删。
- **antd E2E/测试三根因**（改 E2E 或组件测试前必读）：① 左树 data-test span 在 submenu title **内部**，选择器直接点 `[data-test=...]` 靠冒泡展开，别写 `[data-test] .ant-menu-submenu-title`；② 详情区 Tab 溢出折叠（接口 47 个）→ 目标 Tab 在「更多」下拉，走 `.ant-tabs-nav-more` + `.ant-tabs-dropdown-menu-item`，直点 nav 节点在视口外且不切换；③ antd 两字中文按钮自动插空格 →「确 定」「查 询」一律用 `/确\s*定/` 式正则。
- **F3 真浏览器坑**：test 环境 antd useId 恒为 `test-id`，跨 Radio.Group 撞 name 致 DOM checked 互斥失真（生产无此问题）→ 断 `ant-radio-wrapper-checked` 受控 class 而非 toBeChecked。
- **模块页本地化时序**：loadSchema 成功后 bump `schemaEpoch`，relabel effect 依赖它重跑——选设备后 schema 重拉会用原始 fields 覆盖已本地化版本（PR#335 根因）。
- **拆 PR/commit 门禁**：commit ≤500 行（纯删 ≤6000 且新增 ≤50）、PR ≤1000（>20 文件 3000/纯删 6000）；lockfile/生成物不计。pre-commit 每 commit 跑全量前端测试 ~5min。
- **Storybook**：`@storybook/react-vite` 需 ≥10.2.19 才 peer 兼容 Vite 8；storybook 的 vite-plus 可选 peer 会钉 vitest 族小版本（升 storybook 连带对齐 vitest）。npm 原地升级被旧树卡 ERESOLVE 时删 node_modules+lock 干净重算。
- **测试惯例**：happy-dom 下 antd Modal 离场动画不结束→按最新弹窗实例定位；`asyncUtilTimeout: 4000` 已全局设；UI-02 守护连 JSX 文本中文都拦（story mock 也算，用英文）。
- **follow-up 债**（登记于 2026-08-14，未闭环）：① design Open Q1 主题令牌对齐粒度待用户目视验收；② Q2 Storybook 故事内容重建（现仅框架冒烟 2 story）另开 change；③ Q3 EviewUI 接入时机与其 React 版本约束（若仅支持 React 18 有降级成本）待其 package.json/d.ts 到手评估；④ 评审低severity 四条：en 词表 `common.apply="Apply"` 与中文「查询」语义漂移、死 key `console.addConfigItem` 可清理、E2E 提交后未断言徽标清零、pickDevice 建议改用 `data-test="device-select"`；⑤ 存量债 loadSchema 无 in-flight 序号守卫（快速切模块/设备旧响应后到会覆盖，重建前已存在）。
- 相关：[[frontend-contract-gen]]、[[nce-console-redesign]]、[[test-governance-military-rules]]、[[readback-subtree-peel]]、[[yang-rpc-execution]]。
