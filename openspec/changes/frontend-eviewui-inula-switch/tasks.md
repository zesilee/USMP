# Tasks: frontend-eviewui-inula-switch

> 军规：每任务组 ≈ 一个可独立 PR（TM04 ≤1000 行，commit ≤500 行）；每步「旧路径保留+新路径并行+双路径验证→切换→删旧」（§5.3）；测试按 §5.6 选层缺层禁并（T06）；**组 1 闸门六项任一不过：停、回报、重估，禁止带病推进**。
>
> **e2e 豁免窗口声明**：组 4~7 期间 E2E 选择器随组件逐个迁移，staging-smoke 预期分段红；含 frontend 推送以 `USMP_SKIP_E2E=1` 豁免，依据即本声明；**至组 8.2 E2E 全绿即停用**（复刻 React 重建两段口径）。
>
> 输入档案：`docs/research/eviewui-inula-migration.md`（运行时/四件套/Table 矩阵）+ `docs/research/eviewui-component-matrix.md`（24 组件映射/系统性结论/切片清单）。

## 1. 垂直切片闸门（PR-1，worktree 隔离，先于一切）

- [x] 1.1 依赖引入与构建链（openinula+inula 四件套已装、gate 独立配置含 alias 全套；EviewUI 三内网包按方案 B 不入本机 node_modules——构建接入随组 2 开工做 Vite 主配置 alias/less/tsconfig；离线部署形态在组 8 工具链阶段定）
- [x] 1.2 **测试基建验证**：✅ 红线项通过——vitest+happy-dom+openinula 可用（探针 5/5：render/state/effect/事件/卸载/rerender/data-test 落 DOM）。**关键发现**：@testing-library/react 直接 alias 不可用（其 CJS 子模块 pure/act-compat 在 vitest inline 转换外原生 require 真 react-dom，与 inula 元素互不相认）；可用路径=自写 render 薄层（openinula createRoot+act，20 行）+ @testing-library/dom 查询事件（test/gate/inula-testing.tsx），全面迁移时 alias @testing-library/react→薄层可令存量测试零改动（需补齐 renderHook 等 API 面）
- [x] 1.3 切片验证（方案 B 离线摆渡，两轮完成）：工具包 v1/v2 交付、gate-report{,2}.txt 回传判读——EviewUI×openInula 真实可运行（V0）、R05 动态列通（V7）、Tree 受控桥全要素成立（V3）
- [x] 1.4 六项运行时行为逐项验证并记录（五项定案通过+两项定性为测试环境限制并给出免实测兜底设计；详见 gate-conclusion.md 与 gate-round1-findings.md）
- [x] 1.5 `gate-conclusion.md` 落档：**闸门通过，准入组 2**（半受控档位=受控回写②+key 重挂③；附 F2/F3 复核三项：合成事件 target 修法验证/叶子元素点击约定/Radio 参数序补实测）

## 2. 基建四件套（PR-2~3）

> **顺序重排说明（2026-08-17 开工时发现）**：inula-intl/inula-router/inula-X 只能跑在
> openinula 运行时上，无法在 React 下逐个双路径并行——组 2 拆为**先行波**
> （React 下零风险：uSES 清除 + 请求库，即 2.1a/2.2）与**翻转波**（Vite alias +
> i18n 内核 + store + 路由一次性原子切换，靠全量测试+E2E 验证；§5.3 的"旧代码
> 保留"以 git 可回退 + 窗口期双依赖体现）。

- [x] 2.1a （先行波）薄层 hook 化：新增 `useLocale()`（useState+useEffect 运行时无关实现），4 处 useSyncExternalStore 调用点清零；RT-02 守护测试落地（拦 React 18+ API 调用/导入）— **F1**
- [ ] 2.1b （翻转波）i18n 内核换 inula-intl：薄层 API 面不变、内核换 VueI18n 适配器、`on('change')` 接订阅 — **F1**（词表键名快照回归）
- [x] 2.2 （先行波）请求换 inula-request：ir.create 单实例（axios 同形零调用点改动）；filter 承载改直接拼 URL（inula-request params 不收 URLSearchParams），F1 请求形状测试跟随新契约更新（filter 不带[]/offset=0 省略/sort_dir 缺省/30s 超时验证点全保）；inula 四件套转正运行时依赖，axios 窗口期保留可回退 — 全量 577/577 绿
- [ ] 2.3 store 换 inula-X：5 个 zustand store → `createStore({state,actions,computed})` 三段式；组件外订阅点接 `$subscribe`；**FE-27 回归**：changeset payload 经响应式 Proxy 后序列化/解构删键语义不变（红灯先行） — **F1**
- [ ] 2.4 路由 v5 化：Switch/Route JSX 树 + MainLayout children 组合；`src/router/compat.ts`（useNavigate/useSearchParams 薄包装）；离开守卫 Prompt+getUserConfirmation 桥（FE-23 Scenario 回归钉住）；测试 MemoryRouter 迁移 — **F1/F2**

> **路线乙重排（2026-08-17，route-decision.md）**：组 3~5 提前为主线（on React 19，
> cWRP 已实证）；组 2 剩余 2.1b/2.3/2.4 连同运行时 alias 全部改挂**终局波 C**
> （终态组合=闸门实测组合）。协作模式=混合：外网开发（EviewUI 用 d.ts 行为替身
> 做 F2），每完成一组组件桥内网集成点摆渡一轮真实 E2E。

## 3. 适配层核心（PR-4~6）

- [x] 3.1 运行时禁用 API 守护（先行波落地）+ 桥接工具（useRemountKey 半受控③档兜底，F1 钉住 dep 变化重挂/稳定不重挂）；**vendor types 机制落地**：@nce/eview-react 在用组件 d.ts 入库 vendor/eview-types（92 文件 640K 纯文本），tsconfig paths 映射——外网 typecheck 可过、实现仅内网（混合协作模式地基，types-probe 实证）
- [ ] 3.2 FA-05 锚点约定：工具已落（anchorId/ANCHOR_SELECTOR，F1）；守护测试（逐桥断言锚点可命中）随组 4 各桥落地
- [x] 3.3 主题（部分）：src/ui/eview/theme.ts——十档色阶生成（50=主色+单调性 F1 钉住）+ 四族（brand/mint/yellow/red）覆盖 CSS 产出 + 幂等注入；UiProvider 装配随组 5 接线时切 — **F1**
- [x] 3.4 图标映射表：ICON_MAP（22 语义名→IcPublic/IcIct，2 近似已标注、实心走 type=filled）；F1 守护=与 icons.ts 语义名集合严格一致（不缺不多）；桥组件随组 4 落 — **F1**
- [ ] 3.5 feedback 自养挂载点：toast（DivMessage，info→default、常驻横幅关自动消失）/ confirm（MessageDialog Promise 化）；FA-03 测试沿用 — **F2**
- [ ] 3.6 FormItemShell（FA-06）：LabelField+错误态受控渲染+守护测试（拦 validator 下传） — **F2**

## 4. 适配层控件桥（PR-7~9，按矩阵逐个，每个含 F2 对等测试）

- [ ] 4.1 轻组（改名转换）：Tag/Badge/Crumbs/Empty/Drawer/Dialog/DropDown/Segmented/Radio/Switch/Checkbox
- [ ] 4.2 表单组：TextField（clear/prefix 自绘）/Spinner（min-max 显式+双回调合并）/InputSelect（半受控兜底按闸门档位）
- [ ] 4.3 结构组：Tab（key↔index 桥+溢出折叠）/Tree 左树桥（data 转换+受控回写+搜索接线）/Table 受控壳（矩阵全项）
- [ ] 4.4 Button（status 映射+loading/ghost 补齐）/Loading/TipBox（Popover+Tooltip 双语义）/Alert（DivMessage/PageMessage 分派）

## 5. 组件调用点跟随（PR-10~11）

- [ ] 5.1 FieldRenderer/SchemaForm/ItemDetailPane 等 config 组件随适配层 API 微调 + F2 全量回归 — **F2**
- [ ] 5.2 布局导航（Sidebar 左树/Header/MainLayout）+ 五个页面调用点微调 + F2 回归 — **F2**
- [ ] 5.3 antd 退场：依赖移除、构建产物验证零 react/antd 实体（RT-01 Scenario）、适配层守护测试改拦 eview 直接 import — **F1**

## 6. F3 真浏览器与派生黄金（PR-12）

- [ ] 6.1 F3 套件按切片实测口径改写（弹层挂载/嵌套 list 增删改/choice 切换） — **F3**
- [ ] 6.2 派生黄金全量重跑零漂移确认（GD-01：schema→控制台形态与组件库无关的铁证） — **F1**

## 7. E2E 对等（PR-13）

- [ ] 7.1 staging-smoke 选择器迁移（ant-* → EviewUI 结构 + FA-05 锚点落点）；data-test 契约 80 条逐条核销
- [ ] 7.2 三浏览器全绿；`make e2e-local` 全栈冒烟全绿；**豁免窗口停用**
- [ ] 7.3 覆盖率棘轮重钉（staging-down 干净口径 + 分母重算注明，不低于既有阈值语义）

## 8. 工具链与文档收尾（PR-14）

- [ ] 8.1 CI 工作流/钩子/Dockerfile/build-release 验证（Vue→React 收尾清单同款流程）
- [ ] 8.2 CLAUDE.md §3/§5.6/§7.2 栈口径更新；frontend/TESTING.md 更新
- [ ] 8.3 /opsx:sync（frontend-runtime 新主 spec + frontend-ui-adapter MODIFIED/ADDED 合入；frontend/ui-i18n 仅 Purpose 栈名）
- [ ] 8.4 /opsx:archive + 记忆归档（单独 commit）+ follow-up 债登记（Open Questions 1-3 + 目视验收妥协项）
