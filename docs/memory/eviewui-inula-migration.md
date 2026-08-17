---
name: eviewui-inula-migration
description: EviewUI/openInula 切换调研已定案(2026-08-17)：全家桶拍板、四件套实测、全量24组件矩阵(半受控地基风险/Menu无对应/令牌无通道)、垂直切片顺序；做 EviewUI/换库/运行时相关工作前必读
metadata:
  type: project
---

**EviewUI + openInula 迁移调研定案**（2026-08-17，全文 [docs/research/eviewui-inula-migration.md](../research/eviewui-inula-migration.md)，全部结论基于 npm 拆包/d.ts 实物验证）：

- **拍板**：运行时按 openInula（=Horizon 开源版）评估；生态全家桶=inula-X(状态)/inula-router(路由)/inula-request(请求)/inula-intl(国际化)。
- **核心事实**：EviewUI(@nce/eview-react 3.10.28) peer 是 @cloudsop/horizon **不是 react**；openinula@1.0.0 有 createRoot/StrictMode/17级hooks、**没有 useSyncExternalStore/useTransition/useId**（npm latest 标签还是 1.0.0-alpha）。
- **三个惊喜**：① inula-intl 两头通吃——react-intl 同形满足 EviewUI 上下文 + 内置 VueI18n 适配器（$t/changeLanguage/on('change')）与我们 i18n 薄层同构，**t() 调用点零改动**，且 4 处 useSyncExternalStore 随之消失；② inula-request 是 axios 平替（半天）；③ EviewUI Table 受控通道齐全（checkedRows/currentPage/onColumnSort+disableEviewSort），服务端排序是一等公民，"命令式非受控"担忧被 props 实测推翻。
- **最大改造**：inula-router 是 **react-router v5 API**（Switch/Prompt/useHistory，无 Outlet/useNavigate/useBlocker）——路由层重写 2~3 天，离开守卫用 Prompt+getUserConfirmation 桥接。
- **风险**：单向门（四件套类型互 import openinula，整体决策）；测试基建 vitest+testing-library 在 openinula 下**未验证**（垂直切片首日必验，159 个测试文件存续依赖它）；EviewUI 是 Less+@hui/design-token 体系。
- **全量 24 组件矩阵已完成**（同日二期，全文 [docs/research/eviewui-component-matrix.md](../research/eviewui-component-matrix.md)）五个系统性结论：① **半受控是全库地基风险**——真受控开关只有 Radio(isControlled)/Switch(isControlToggled)，其余控件全是 props+内部state+cWRP 老写法（兜底：InputSelect.shouldRender/Checkbox.onPreChange/key 重挂）；② Form/FormItem 必弃用（name 必填收编 onChange、无 validateStatus 注入）自写 FormItemShell+LabelField；③ **data-test 全库不透传**（仅 Tab/SearchInput/LabelField 例外）→锚点整体换 id/wrapper 方案；④ 无命令式反馈，toast/confirm 自养挂载点；⑤ ~~主题令牌注入无通道~~**已闭环**：design-token 整包补采后确认是 CSS 变量体系（:root{--brand-05..90}+语义层引用），注入=#0C5EA6 生成 10 档色阶后加载覆盖样式，零 less 编译（首采"空壳"是提取脚本只留 .d.ts/.md 误伤——**采集非类型包勿过滤**）；版本以 0.0.24 为基准（对方 frontend 装 0.0.21 不满足 peer ^0.0.23，自身有错配）。组件级：**Menu 无合格对应**（悬停级联非内联，Accordion 两层+展开不可控，左树要自研包装=最重工作量）；Tab 按下标寻址+溢出折叠内建；Dialog 受控好但无 confirmLoading/maskClosable；Alert=DivMessage 有默认 10 秒自动消失陷阱；图标 21/23 直接对应（缺纯铃铛/显示器）；Button 无 ghost/loading。适配层从 223 行薄转发重估为 **1500~2500 行真适配层**。
- **缺料全闭环(2026-08-17 终态)**：① inula-X=state/actions/computed 三段式（actions 可变修改响应式代理、computed 真计算、$subscribe 组件外订阅、无需 selector），与 zustand 三段同构机械转换，新增切片验证项=响应式 Proxy 与 FE-27 不可变语义边界；② alias 方向实测=EviewUI 编译产物 `require("react")`，构建期 alias react→openinula 即可（路线 A 直跑真 React 的前提也因此成立，备选保留）；③ design-token CSS 变量方案已定。调研完备，可直接作 /opsx:explore 输入。
- **方法论**：评估组件库/框架一律 npm pack 拆包看 @types 与运行时 js，不信文档转述；对照面=我们的「实际使用面」清单（每组件真正用到的 props），不是对方全量 API。
- 相关：[[react-antd-rebuild]]（适配层军规 FA-01~04 是本次切换的承接点）、[[view-ui-insights]]。
