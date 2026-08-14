## Why

USMP 前端需按团队技术栈统一为 **React 19 + Ant Design 6**。项目尚无存量继承包袱——**无需向后兼容、无需与既有 Vue 实现并行运行、无需灰度切换窗口**，因此本次按「原地重建」处理：直接以 React 实现取代 Vue 实现，不引入双轨维护成本。

同时，公司内部组件库（EviewUI，私有包）**将来可能替换 antd**，故本次须一并建立「业务代码与组件库解耦」的单点切换能力，避免下一次换库重来一遍。

产品形态不变：仍是 YANG 模型驱动的运行时渲染控制台（R05，禁手写固定表单）。换的是实现技术，不是产品。

## What Changes

- **原地重建 `frontend/`**：删除 Vue3 + Element Plus 实现，以 React 19 + antd 6.6 重建。目录路径保持不变，故全部基础设施（6 个 CI 工作流、3 个 git 钩子、`Dockerfile`、`build-release.sh`、Makefile 目标）**路径无需改动**，仅需调整其内部工具链命令。
- **新增 UI 适配层 `src/ui/`**：业务代码 SHALL NOT 直接 import 组件库；全部控件经适配层导出。antd 为当前实现，EviewUI 为潜在后继实现。
- **技术选型**：React 19 / antd 6.6（经适配层）/ zustand / react-router / 自研 i18n 薄层（与旧 vue-i18n 同形 `i18n.global.t` API，词表沿用）/ `@testing-library/react`。保留 Vite、axios、Vitest、Playwright、openapi-typescript 契约生成与漂移门禁。
- **沿用框架无关资产**（非继承包袱，是免于重写的既有正确实现）：
  - `src/utils` 派生纯函数（1368 行，零框架依赖）与其单测；
  - `src/types` 契约生成物（1532 行，由 openapi 生成）；
  - `src/api` axios 层（242 行）；
  - i18n 词表键值（zh-cn / en-us 两份）；
  - 派生黄金快照（GD-01）与 schema fixture；
  - Playwright E2E 用例与 `data-test` 属性命名。
- **重建部分**：全部页面与业务组件（原 7017 行）、组件层测试（原 6304 行）、状态层（Pinia→zustand）、表单编排（`useConfigForm` / `useConstraintEngine` 改写为 hooks）。
- **BREAKING**（对开发者）：前端源码语言与组件库全面更替；对外 HTTP 契约、后端、部署产物形态均不变。

## Capabilities

### New Capabilities
- `frontend-ui-adapter`: UI 组件适配层契约——业务代码与具体组件库解耦、适配层导出面与禁止直接 import 的门禁、组件库单点替换能力（当前 antd，后继可能 EviewUI）。

### Modified Capabilities
- `frontend`: Purpose 与 FE-01 中「Vue3 + Element Plus」「Element Plus 控件」的技术栈表述改为「React + UI 适配层」；控件类型映射结论（boolean→打开/关闭单选组、必填短枚举→分段控件、其余→下拉等）保持不变。新增 FE-27（表单键存在性即节点存在性）。
- `ui-i18n`: Purpose、UI-01、UI-02 中「vue-i18n」「ElementPlus locale / ElConfigProvider」表述改为 i18n 框架无关 + 适配层 locale 联动；双语覆盖、持久化、缺档回退等行为要求不变。

> `console-derivation-golden`（GD-01 派生黄金）不改：派生逻辑为框架无关纯函数，黄金结论在重建前后 SHALL 保持一致。

## Impact

| 面 | 影响 |
|----|------|
| 删除 | `frontend/src` 下全部 `.vue` 实现（7017 行）与组件层测试（6304 行）；vue / vue-router / pinia / vue-i18n / element-plus / @vue/test-utils / vue-tsc 依赖 |
| 沿用 | `src/utils` 1368 行 + `src/types` 1532 行 + `src/api` 242 行 + `test/{utils,stores,composables}` 约 3220 行 + E2E 414 行 + i18n 词表 + 派生黄金 |
| 新增 | React 脚手架、`src/ui/` 适配层、全部页面与业务组件、组件层测试（`@testing-library/react`） |
| 后端 | 零改动（HTTP 契约不变） |
| 基础设施 | **路径全部不变**；仅调整工具链命令（`vue-tsc`→`tsc`、Storybook 框架包、Vitest 插件、Dockerfile 内构建命令） |
| 依赖 | 新增 react/react-dom 19、antd 6.6、react-router、zustand、@testing-library/react（i18n 为自研薄层，不新增运行时依赖） |
| 空窗期 | 清场 PR 与脚手架 PR 之间仓库暂无可运行前端，前端 CI 作业临时降级为 no-op，脚手架 PR 即恢复（全新项目无线上影响） |
| 风险 | ① antd Table 运行时动态列 / Form 运行时动态 rules 撑不住则触及 R05 命门；② 表单状态改为不可变更新时丢失「键不存在=节点不存在」语义会静默下发多余字段 |
