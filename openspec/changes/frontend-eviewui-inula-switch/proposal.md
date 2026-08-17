# Proposal: frontend-eviewui-inula-switch

## Why

公司前端技术栈统一要求：USMP 前端从社区栈（React 19 + antd 6 + react-router 8 + zustand + axios + 自研 i18n 薄层）切换到**公司栈**——openInula 运行时（Horizon 开源版）+ EviewUI（@nce/eview-react）组件库 + Inula 全家桶（inula-X 状态 / inula-router 路由 / inula-request 请求 / inula-intl 国际化）。

选型与可行性已经两轮调研定案（全部基于 npm 拆包/d.ts 实物验证 + NCE 同类产品实证，见 `docs/research/eviewui-inula-migration.md` 与 `docs/research/eviewui-component-matrix.md`）：

- **同类产品实证**：iMaster NCE-Fabric 设备特性编程页（左树 Tree + Tab + 特性表格 + 攒批工具栏）与 USMP 模块控制台功能一比一同构，且就是 EviewUI 实现——可行性非推断而是已被同类产品验证。
- **改造面已量化**：运行时暴露面极小（createRoot ✓、4 处 useSyncExternalStore 随 inula-intl 消失、useBlocker 走 Prompt 桥）；alias 方向实测=EviewUI 编译产物 `require("react")`，构建期别名即可。
- **为什么是现在**：React 重建（PR#316-#337）刚交付的 `src/ui` 适配层（FA-01~04：业务代码禁直接 import 组件库）正是为这次切换预留的单点——业务代码理论零改动，改动收敛在适配层与基建层。

## What Changes

**遵循 §5.3 存量改造军规：旧代码保留 + 新代码并行 + 双路径验证 → 切换 → 删除旧代码；垂直切片闸门先行（六项运行时行为验证不过即停）。**

1. **垂直切片闸门（先于一切）**：openInula 挂载 + vitest/testing-library 测试栈 + EviewUI Table/TextField 半受控实测 + Tree 受控桥 + 六项验证清单（矩阵文档 §3）全绿才准进入全面切换。
2. **运行时**：react/react-dom → openinula（Vite alias），React 19 → React 17 级 API 面（我们代码已确认兼容）。
3. **基建四件套**：
   - i18n 薄层内核 → inula-intl（VueI18n 适配器同形，`t()` 调用点零改动）；
   - axios → inula-request（近平替）；
   - react-router 8 → inula-router（v5 API：Switch/Route 树重写、useNavigate→useHistory、useBlocker→Prompt+getUserConfirmation 桥）；
   - zustand 5 个 store → inula-X（state/actions/computed 三段式机械转换）。
4. **UI 适配层重写**（唯一的组件库触点，FA-01~04 军规不变）：antd 薄转发 223 行 → EviewUI 真适配层 1300~2200 行（Table 受控壳 / Tree 左树桥 / FormItemShell / feedback 自养挂载点 / 半受控兜底工具 / data-test wrapper 约定 / design-token CSS 变量主题覆盖 / icon-plus 23 语义图标映射）。
5. **测试与门禁**：F1~F4 四层与覆盖率棘轮保持；E2E data-test 契约保留、锚点落点按"不透传"现实重设计；E2E 选择器 ant-* → EviewUI 结构（沿用 el-*→ant-* 迁移剧本）。

**Non-Goals**：不改任何业务功能与后端契约（严格功能对等）；不迁移 Storybook 故事内容；不接非华为设备相关能力；EviewUI 版本锁 3.10.28（有新版另起 change）。

## Capabilities

### New Capabilities

- `frontend-runtime`: 前端运行时与基建栈约束——openInula 运行时、Inula 全家桶四件套、构建期 alias 约定、React 18+ API 禁用面（引第三方库前须审）。

### Modified Capabilities

- `frontend-ui-adapter`: FA-02 由"薄转发"改述"受控桥接层"（半受控底层不泄漏到业务面）；新增 FA-05 测试锚点跨库稳定、FA-06 FormItemShell（不接组件库表单 store 军规化）。

> 复核说明：`frontend`（FE-01 含"换库锚点"Scenario）与 `ui-i18n`（UI-01/02 已是适配层中立 + 词表键名跨框架稳定）的需求在上次 sync 已写成组件库中立形态——本次**需求不变、仅实现载体变**，不出 delta（R17 只对契约变更刷 spec）；sync 阶段仅更新主 spec Purpose 中的描述性栈名。

## Impact

- **代码**：`frontend/` 全域（适配层重写、路由层重写、store/i18n/请求内核替换、组件调用点跟随适配层微调）；业务纯函数层（utils/form/派生逻辑）零改动。
- **基建**：Vite 配置（alias + less）、package.json 依赖面大换血、6 个 CI 工作流内部命令微调、Dockerfile/发布脚本验证。
- **测试**：159 个测试文件大部分沿用（testing-library 兼容性为切片验证项）；E2E 选择器迁移；覆盖率棘轮口径随分母重算（有先例）。
- **风险**（矩阵文档五个系统性结论）：半受控地基风险🔴（切片首验）、openInula 成熟度（latest=alpha）、单向门（全家桶互相咬合）、测试基建兼容未验、Form 弃用自写 FormItemShell。任一闸门项不过即停并回报，不带病推进。
