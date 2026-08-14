---
name: react-antd-rebuild
description: 前端 Vue→React+antd 原地重建（change frontend-react-antd-switch）进行中：进度/PR 轨迹/闸门结论/拆 PR 与门禁踩坑；续作或碰前端任何文件前必读
metadata:
  type: project
---

**change `frontend-react-antd-switch`**（worktree 分支同名目录，2026-08-14 起）：Vue3+Element Plus 整体退役，React 19 + antd 6.6 + Vite 8 + zustand + react-router **原地重建**（全新项目拍板：无双轨/无灰度/无回滚窗口）。制品与 tasks 在 `openspec/changes/frontend-react-antd-switch/`（tasks.md 勾选即进度台账，rebuild-notes/ 有 data-test 80 条清单与旧组件用量表）。

**已收官（PR #316-#331 全合入 main）**：组1 准备 / 组2 清场（6 分片：docs→删测→删源→铺路，i18n 薄层与占位使沿用 utils 字面零改动）/ 组3 脚手架+src/ui 适配层（FA-01~04 守护测试硬拦直接 import antd）/ 组4=纯逻辑层 / 组5 zustand+语言联动 / 组6 表单编排（纯函数核心 src/form + hook 壳 src/hooks，FE-27 红灯先行+伪删键守护）/ **组7 垂直切片闸门通过**（68 fixture 实测动态列+运行时校验，结论在 gate-conclusion.md；架构决定=不接 antd Form store，validateStatus/help 受控）/ 组8 详情区+表单Tab+rpc+列表全功能（useListQuery 双模式/占位/变更集标记/行删除）/ 组9 布局导航+ModuleConsolePage 宿主。

**Why:** 重建横跨几十个 PR，门禁与拆分方法论是反复踩过的坑，丢了会重摔。

**How to apply:**
- **拆 PR/commit 门禁**：commit ≤500 行（纯删 ≤6000 且**新增 ≤50**）、PR ≤1000（>20 文件 3000/纯删 6000）——超限就抽 hook/组件分文件再分批 add（先例 useListQuery/listColumns）；lockfile/生成物不计数。同文件重写可 `git rm --cached` 先删后提保纯删口径。
- **e2e 豁免**：窗口期含 frontend 推送用 `USMP_SKIP_E2E=1`，依据=tasks.md 红灯声明（两段口径），**至 tasks 12.3 E2E 全绿止**；e2e-staging 工作流页面对等前预期红。
- **合入循环**：push（pre-push -race ~3min，别 tail 管道吞退出码、后台 push 会被杀须重推）→ `gh pr create` → 分支落后用 `gh api PUT .../update-branch`（gh pr update-branch 会因 Projects-classic 报错）→ required 六项绿即 merge（frontend-ci advisory）。
- **覆盖率棘轮**：现值 95.0/84.5/94.5/96.5（vitest.config 注释有演变轨迹）；分母扩容（新组件层入 include）导致的回落按「分母重算先例」重钉并注明，加测不降标。
- **测试惯例**：happy-dom 下 antd Modal 离场动画不结束→按「最新弹窗实例」定位勿等卸载；确认钮文案随 locale 用 within(modal)+正则「执\s*行|OK」；gate 套件 fake timers 收口 rc 定时器；`asyncUtilTimeout: 4000` 已全局设（CI 慢跑道）；组件测试跑全量 ~5min/次（pre-commit 每 commit 都跑，连续多 commit 命令要留 >10min 或分次跑）。
- **UI-02 守护**（test/ui/no-hardcoded-chinese）连 JSX 文本中文都拦；语言自名/诊断字符串走词表或豁免清单（仅 xpathEval 文件级豁免）。
- **待办**（tasks 10-14 组）：五个页面（Devices/Dashboard/Logs/Settings/Business，占位页在 src/views/PlaceholderPage）→ 11 组变更集批量（BatchToolbar/CommitDialog/路由离开守卫 useBlocker/consoleEpoch 接线已留口/新鲜度环 Header 挂载）→ 12 组 F3+E2E 对等（data-test 对照 rebuild-notes 清单）→ 13 组工具链（Storybook 框架包/Dockerfile 验证）→ 14 组文档+sync+archive。
- 相关：[[frontend-contract-gen]]、[[nce-console-redesign]]、[[readback-subtree-peel]]（normalizeRows 契约已平移）、[[yang-rpc-execution]]（rpc 三铁律已平移）。
