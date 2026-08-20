# frontend-runtime — 前端运行时与全家桶

## Purpose

前端交付产物的运行时形态（波 C 定型，2026-08-20 内网 E2E 21/21 验收）：缺省构建即 openinula 运行时（`USMP_RUNTIME=react` 显式回退为应急通道；`USMP_UI_BACKEND=antd` 的 e2e-local 口径联动强制 react）。外网开发与测试运行时恒 React 19（React 家族为 devDependencies），行为对等由内网 E2E 验收兜底。路由 API 收口 `src/router/compat`（`@app-router` 别名双实现分流）；状态层为自研 React 17 级薄层（`src/stores/createStore.ts`，双运行时可跑）；i18n 薄层自研无内核。

## Requirements

### Requirement: RT-01 运行时与全家桶锁定

前端 SHALL 运行于 openInula 运行时（构建期将 `react`/`react-dom`/`react-intl` 别名到 openinula/inula-intl），路由 SHALL 使用 inula-router（v5 API，经 `src/router/compat` 收口）、请求 SHALL 使用 inula-request、国际化承接 SHALL 使用 inula-intl（应用词表层为自研薄层）；状态层 SHALL 使用自研 React 17 级薄层（2026-08-19 拍板变更：inula-X 锁死 openinula 会使外网 React 测试体系失效——薄层双运行时可跑）。SHALL NOT 同时打包两个 UI 运行时。

#### Scenario: 构建产物单运行时
- **WHEN** 构建产出发布包
- **THEN** 产物 SHALL 只含 openinula 运行时，SHALL NOT 含 react/react-dom 实体

#### Scenario: 别名对 EviewUI 生效
- **WHEN** EviewUI 组件代码执行 `require("react")`
- **THEN** SHALL 解析到 openinula（实测其编译产物直接引 react，别名即桥）

### Requirement: RT-02 React 18+ API 禁用面与新依赖审查

业务代码 SHALL NOT 使用 openInula 未提供的 React 18+ API（`useSyncExternalStore`/`useTransition`/`useDeferredValue`/`useId`）；引入任何 React 生态第三方库前 SHALL 先审查其是否依赖上述 API，审查结论 SHALL 记录在依赖引入的提交说明中。

#### Scenario: 业务代码引用禁用 API 被拦
- **WHEN** 源码出现 `useSyncExternalStore` 等禁用 API 引用
- **THEN** 守护测试 SHALL 失败，改动 SHALL NOT 通过门禁

#### Scenario: 新依赖内部使用禁用 API（负路径）
- **WHEN** 候选第三方库内部依赖 React 18+ API
- **THEN** SHALL NOT 引入，或 SHALL 记录替代方案后另选

### Requirement: RT-03 路由离开守卫语义保持

路由层 SHALL 采用 inula-router（v5 API 形态）；攒批未提交时离开模块页的守卫 SHALL 经 `Prompt` + Router `getUserConfirmation` 桥实现，用户体验 SHALL 与既有 useBlocker 方案对等（弹确认框、取消留在原页、确认后放行且变更集保留）。

#### Scenario: 有未提交变更时切换路由
- **WHEN** 变更集非空且用户点击左树切换模块
- **THEN** SHALL 弹出确认框；取消 SHALL 停留原页且变更集不变；确认 SHALL 完成导航且变更集保留

#### Scenario: 无变更时不打扰
- **WHEN** 变更集为空
- **THEN** 导航 SHALL 直接完成，SHALL NOT 弹确认
