# USMP 前端测试分层规范（权威）

> 本文件是前端测试的**唯一权威**分层规范，与 [CLAUDE.md §5.6](../CLAUDE.md)（测试军规）一致。
> 军规：每个改动**必须**按「改动类型→必补层」补齐测试，缺层=未完成、禁止合并（T06）；测试设计先于编码（T05）。

## 四层 + 契约/类型门禁

| 层 | 运行器 / 配置 | 位置 | 职责（测什么） | 跑法 |
|----|--------------|------|----------------|------|
| **F1 纯逻辑单测** | vitest（happy-dom）`vitest.config.ts` | `test/utils/`、`test/composables/`、`test/stores/` | 纯函数、composable、store：输入→输出、分支、异常/边界。无真 DOM，最快 | `npm run test` |
| **F2 组件单测** | vitest（happy-dom）+ `@vue/test-utils` | `test/components/`、`test/views/` | 组件：渲染、props、**emit**、条件分支。list/group 的 **add/edit/remove** emit、校验错误态都要测（不能只测 render） | `npm run test` |
| **F3 真浏览器** | vitest **browser mode**（真 Chromium）`vitest.browser.config.ts` | `test/browser/` | **仅**放 happy-dom 伪造不了的：Element Plus `el-select` 弹层/teleport、嵌套 list 真实交互、真实布局/测量。交互控件的 add/edit/remove 必须**全覆盖** | `npm run test:browser` |
| **F4 E2E** | Playwright（起 docker 全栈）`playwright.config.ts` | `frontend/tests/staging-smoke.spec.ts` | 部署冒烟：路由、SPA 挂载、种子数据、YANG 表单动态渲染、校验拦截。跑在 nginx :3002 + 后端 :8080 真栈 | `npm run e2e` / `make e2e-local` |
| 契约/类型 | `vue-tsc` + `openapi-typescript` | `src/**`、`src/types/api.gen.ts` | `src/` 零类型错；后端注解→契约不漂移 | `npm run typecheck` / `gen:api` |
| Storybook | `.storybook/` | `*.stories.ts` | 构建门禁 | `npm run build-storybook` |

## 何时必须上 F3 真浏览器（vs F2 happy-dom）

happy-dom 是近似实现，**测不准**这些 → 必须 F3：
- Element Plus `el-select` / 下拉 / 弹层（teleport 到 body、popper 定位）；
- 嵌套 list 子表单的真实增删改交互（点击「添加/删除」按钮后的真实 DOM 与事件）；
- 依赖真实布局/尺寸/滚动/焦点的行为。

其余（纯渲染、emit 断言、分支逻辑）用 **F2 happy-dom** 即可（更快、无需起 Chromium）。**F3 保持精简**，只打 happy-dom 打不了的点。

## 改动类型 → 必须补的层

| 改动 | 必补 |
|------|------|
| util / composable / store | F1 |
| 组件 / 页面逻辑 | F2（含 add/edit/remove/校验态） |
| el-select / teleport / 嵌套 list 增删改 | **F3 真浏览器**（add/edit/remove 全覆盖） |
| 新页面 / 路由 / 端到端用户流 | F4 staging-smoke |
| 改 API 类型 / 契约 | typecheck + 契约漂移门禁 |
| **修 Bug** | 先写复现该 Bug 的回归测试（红）再修（T07） |

## 目录结构

```
frontend/
├── test/                     # vitest 套件（happy-dom + browser）
│   ├── utils/ composables/ stores/   # F1
│   ├── components/ views/            # F2
│   └── browser/                     # F3（真 Chromium，vitest.browser.config.ts）
├── tests/                    # F4：Playwright（staging-smoke.spec.ts）
├── vitest.config.ts          # F1/F2：happy-dom，exclude test/browser/**，coverage.thresholds
├── vitest.browser.config.ts  # F3：playwright provider，仅 include test/browser/**
└── playwright.config.ts      # F4：baseURL :3002（PLAYWRIGHT_BASE_URL 可覆盖）
```

## 覆盖率棘轮（T08）

- `vitest.config.ts` 的 `coverage.thresholds`（statements/branches/functions/lines）为**只准升不准降**的棘轮，`frontend-ci.yml` 跑 `npm run test:coverage`，低于阈值即 fail。
- 补测后**同步上调阈值**到新水平，形成单向棘轮。本地自查：`npm run test:coverage`。
- 基线实测(2026-07-06)：Stmts 66.55 / Branch 66.57 / Funcs 56.67 / Lines 66.88。

## 门禁（本地 + CI）

| 时机 | 拦截 |
|------|------|
| pre-commit | 暂存含 `frontend/*.{ts,vue,…}` → `npm run test`（F1/F2 happy-dom）。`USMP_SKIP_FE_TEST=1` 跳过 |
| pre-push | 前端变更 → `scripts/e2e-smoke.sh`（F4）。`USMP_SKIP_E2E=1` 跳过 |
| CI `frontend-ci.yml` | typecheck + `npm run test:coverage`（F1/F2 + 覆盖率门禁） |
| CI `frontend-browser-tests.yml` | `npm run test:browser`（F3） |
| CI `frontend-storybook.yml` | `build-storybook` |
| CI `e2e-staging.yml`（post-merge，自托管） | 部署 → `staging-smoke.spec.ts`（F4） |

## 全部 npm scripts

`typecheck` / `gen:api` / `test` / `test:watch` / `test:ui` / `test:coverage` / `test:browser` / `storybook` / `build-storybook` / `e2e` / `e2e:ui` / `e2e:headed` / `e2e:report`
