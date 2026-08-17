---
name: eviewui-inula-migration
description: EviewUI/openInula 切换调研已定案(2026-08-17)：全家桶选型拍板、四件套实测结论、Table 矩阵、垂直切片顺序；做 EviewUI/换库/运行时相关工作前必读
metadata:
  type: project
---

**EviewUI + openInula 迁移调研定案**（2026-08-17，全文 [docs/research/eviewui-inula-migration.md](../research/eviewui-inula-migration.md)，全部结论基于 npm 拆包/d.ts 实物验证）：

- **拍板**：运行时按 openInula（=Horizon 开源版）评估；生态全家桶=inula-X(状态)/inula-router(路由)/inula-request(请求)/inula-intl(国际化)。
- **核心事实**：EviewUI(@nce/eview-react 3.10.28) peer 是 @cloudsop/horizon **不是 react**；openinula@1.0.0 有 createRoot/StrictMode/17级hooks、**没有 useSyncExternalStore/useTransition/useId**（npm latest 标签还是 1.0.0-alpha）。
- **三个惊喜**：① inula-intl 两头通吃——react-intl 同形满足 EviewUI 上下文 + 内置 VueI18n 适配器（$t/changeLanguage/on('change')）与我们 i18n 薄层同构，**t() 调用点零改动**，且 4 处 useSyncExternalStore 随之消失；② inula-request 是 axios 平替（半天）；③ EviewUI Table 受控通道齐全（checkedRows/currentPage/onColumnSort+disableEviewSort），服务端排序是一等公民，"命令式非受控"担忧被 props 实测推翻。
- **最大改造**：inula-router 是 **react-router v5 API**（Switch/Prompt/useHistory，无 Outlet/useNavigate/useBlocker）——路由层重写 2~3 天，离开守卫用 Prompt+getUserConfirmation 桥接。
- **风险**：单向门（四件套类型互 import openinula，整体决策）；测试基建 vitest+testing-library 在 openinula 下**未验证**（垂直切片首日必验，159 个测试文件存续依赖它）；EviewUI 是 Less+@hui/design-token 体系。
- **未验项**：inula-X store API 细节；EviewUI 编译产物 import 谁；其余 23 组件 d.ts。
- **方法论**：评估组件库/框架一律 npm pack 拆包看 @types 与运行时 js，不信文档转述；对照面=我们的「实际使用面」清单（每组件真正用到的 props），不是对方全量 API。
- 相关：[[react-antd-rebuild]]（适配层军规 FA-01~04 是本次切换的承接点）、[[view-ui-insights]]。
