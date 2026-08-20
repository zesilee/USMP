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
- [x] 2.1b 定性核销（2026-08-19）：i18n 薄层在 React 重建期已**自研无内核**（纯词表查表+插值，src/i18n/index.ts——「换 inula-intl 内核」前提过时，17 级安全双运行时天然可跑）；inula-intl 仅在翻转日作为 eview 内部 require('react-intl') 的**构建别名承接**（归入 2.5 翻转日清单）
- [x] 2.2 （先行波）请求换 inula-request：ir.create 单实例（axios 同形零调用点改动）；filter 承载改直接拼 URL（inula-request params 不收 URLSearchParams），F1 请求形状测试跟随新契约更新（filter 不带[]/offset=0 省略/sort_dir 缺省/30s 超时验证点全保）；inula 四件套转正运行时依赖，axios 窗口期保留可回退 — 全量 577/577 绿
- [x] 2.3 状态层换**自研 React 17 级薄层**（2026-08-19 拍板变更，用户确认：inula-X 锁死 openinula 会使外网 React 测试体系失效——自研 createStore 双运行时可跑，zustand 同形 API 调用点仅换 import）：src/stores/createStore.ts（useState/useEffect/useRef 实现、selector Object.is bail、getState/setState/subscribe）+F1 五用例；5 store 换内核**全量 647 零回归**（FE-27 payload 语义在存量套件钉住）；zustand 依赖移除 — **F1 ✓**
- [x] 2.4a 路由 compat 收口（波 C 前置，2026-08-19）：src/router/compat.ts 收束 8 个路由 API（现=react-router 直通），7 文件调用点全部改经 compat，adapter-guard 扩拦业务直 import react-router；**翻转时改动收束单点** — 全量 647 零回归
- [x] 2.4a2 路由 compat 升级裸别名 @app-router（三配置接线：vite=生产/vitest+browser=测试恒 react 直通/tsconfig=类型面）——翻转后外网测试运行时不动，双实现分流与 @ui-backend 同款 — 647 零回归
- [x] 2.4b 翻转日实现（2026-08-19，开关制）：compat.inula.tsx 全量实现（createBrowserRouter/RouterProvider→BrowserRouter+递归 Switch/Route、Outlet=Context 注入、useNavigate/useSearchParams 薄包装、useBlocker=Prompt when+message 函数+getUserConfirmation 桥保 state/proceed/reset 契约、类型断言桥收 inula 类型面）；vite USMP_RUNTIME=inula 别名组（react/react-dom/jsx-runtime→openinula、react-intl→inula-intl、@app-router→compat.inula）；Dockerfile/build-release 透传。**外网测试运行时不翻转**（vitest 恒 React+compat.ts）——行为验收=内网 USMP_RUNTIME=inula 构建+E2E 21 用例全量
- [x] 2.5 内网翻转验收（2026-08-20）：openinula 运行时产物 E2E **21/21 全绿一次通过**（RT-01/RT-03 真实环境验收）；默认翻 inula（USMP_RUNTIME=react 显式回退、USMP_UI_BACKEND=antd 联动强制 react）；react 家族移 devDependencies（生产=openinula，外网测试保留）；polyfill/attachShadow 守卫复评=**保留**（polyfill 自适应跳过 inula 自带 findDOMNode、守卫幂等无害——防御纵深优于裸奔）

> **路线乙重排（2026-08-17，route-decision.md）**：组 3~5 提前为主线（on React 19，
> cWRP 已实证）；组 2 剩余 2.1b/2.3/2.4 连同运行时 alias 全部改挂**终局波 C**
> （终态组合=闸门实测组合）。协作模式=混合：外网开发（EviewUI 用 d.ts 行为替身
> 做 F2），每完成一组组件桥内网集成点摆渡一轮真实 E2E。

## 3. 适配层核心（PR-4~6）

- [x] 3.1 运行时禁用 API 守护（先行波落地）+ 桥接工具（useRemountKey 半受控③档兜底，F1 钉住 dep 变化重挂/稳定不重挂）；**vendor types 机制落地**：@nce/eview-react 在用组件 d.ts 入库 vendor/eview-types（92 文件 640K 纯文本），tsconfig paths 映射——外网 typecheck 可过、实现仅内网（混合协作模式地基，types-probe 实证）
- [ ] 3.2 FA-05 锚点约定：工具已落（anchorId/ANCHOR_SELECTOR，F1）；守护测试（逐桥断言锚点可命中）随组 4 各桥落地
- [x] 3.3 主题（部分）：src/ui/eview/theme.ts——十档色阶生成（50=主色+单调性 F1 钉住）+ 四族（brand/mint/yellow/red）覆盖 CSS 产出 + 幂等注入；UiProvider 装配随组 5 接线时切 — **F1**
- [x] 3.4 图标映射表：ICON_MAP（22 语义名→IcPublic/IcIct，2 近似已标注、实心走 type=filled）；F1 守护=与 icons.ts 语义名集合严格一致（不缺不多）；桥组件随组 4 落 — **F1**
- [x] 3.5 feedback 自养挂载点（eview 后端并行版，不接线）：toast（DivMessage 替身，info→default、自管卸载不依赖其 display:none 式消失、flushSync 立即可见）/ confirm（MessageDialog Promise 化，danger→type=risk，ok/cancel/关闭三路 resolve）；F2 七用例 — **F2**
- [x] 3.6 FormItemShell（FA-06）：适配层自绘（label/必填星/受控 error/role=alert/data-test wrapper 锚点），不依赖 EviewUI 表单容器；"拦 validator 下传"守护随组 4 各桥落 — **F2**

## 4. 适配层控件桥（PR-7~9，按矩阵逐个，每个含 F2 对等测试）

- [x] 4.1 轻组桥 11 件（对外 antd 形态零调用点改动）：Tag(色名映射+round=false)/Badge/Breadcrumb(seprator 拼写)/Empty/Drawer(open→visible、%宽折算)/Modal→Dialog(onOk→buttons、footer=null、confirmLoading 吞 onOk、movable=false)/Dropdown(key↔value)/Segmented(disable 拼写)/Radio.Group(children→data、isControlled、**参数序自适应免疫两种顺序**)/Checkbox(合成 e.target.checked)/Switch(isControlToggled+data 两态)；F2 替身 10 用例+FA-05 锚点断言 — **✅ 内网校准收敛（CAL-R14）**
- [x] 4.2 表单组桥 3 件：Input→TextField（clear 自绘 suffix/prefix 叠放/password/validator 绝不下传守护）、InputNumber→Spinner（min-max 缺省显式无界防 0/100 陷阱、无效输入不上抛）、Select→InputSelect（label→text、gate 参数序、清空=onChange(undefined) 键不入 payload、showSearch↔onlySelect）；**useSemiControlledBridge 通用机制**（②受控回写+③拒写检测重挂，F1 三路径钉住；实录坑：拒写时父不重渲染须 onEmit 主动触发检测）— **✅ 内网校准收敛（CAL-R14）**
- [x] 4.3 结构组桥 3 件：Tabs→Tab（key↔index 桥、标签栏 eview 渲染+**内容区桥自渲**绕开 TabContent 形态不确定性、observerWidthChange 溢出折叠）/Menu→Tree 左树桥（items 嵌套→data、label JSX 文本化、openKeys↔expandedKeys+onExpand 全量回写、inlineCollapsed 宿主处理）/Table 受控壳（矩阵全项：render 参数序换位、rowKey 函数→__ubkey 预计算、checkedRows 受控+强刷、rowClickDelay:0、disableEviewSort+antd sorter 合成、分页拆平、rowClassName→classStyleMap→customStyleRows）；F2 替身 10 用例；实录坑：TabItem 挂具名导出须命名空间导入 — **✅ 内网校准收敛（CAL-R14；Table 三修=renderType 运行时枚举+行序号↔rowKey 双向映射+render 行数据在第4参上下文 rawData；Tabs/Tree 交互 happy-dom 同步死循环，移交 6.1 F3）**
- [x] 4.4 收尾组桥 5 件：Button（type/danger→status、danger 优先 risk、loading=自绘 spinner+禁点、ghost=样式类、双参吞单参）/Spin→Loading(type=local)/Tooltip→TipBox(hover,title→content)/Popover→TipBox(click，display 尽力受控——matrix 已知限制集成点重点校准)/Alert（closable→DivMessage 强制关自动消失、否则 PageMessage 保 info 型）；F2 替身 7 用例。**组 4 全组收口（24 桥）** — **✅ 内网校准收敛（2026-08-18 CAL-R14：16 passed/1 skip，14 轮摆渡 PR#358-#367；判读见 docs/memory/eviewui-switch-implementation.md）**

## 5. 组件调用点跟随（PR-10~11）

- [x] 5.1 config 组件调用点跟随：SchemaForm 换 FormItemShell 外壳（Form/Form.Item 退场）、Table 桥泛型化+本地函数排序+分页/排序 onChange 合成 antd 快照、Modal footer/maskClosable、Empty children、Dropdown menu.selectedKeys；**已知债（窗口期类型收下行为暂缺，内网侦察后补映射）：①列头筛选菜单（filters/onFilter→eview embeddedFilter）②展开行（expandable→eview 展开形态）** — **F2 全量回归 ✓**
- [x] 5.2 布局导航与页面调用点微调（Devices/Logs/ModuleConsolePage/ModuleListTab Select 回调与 Key 类型退 antd）+ **@ui-backend 单点切换落地**（生产→eview 桥；外网 vitest/storybook→antd 测试镜像 src/ui/antd-backend，见其 README；EVIEW_REAL=1→全链真身）+ UiProvider eview 化（IntlProvider+locales+polyfill+主题注入）+ functions 阈值按分母重算 94.5→94.3（7.3 回填） — **F2 回归 ✓（631 绿）**
- [x] 5.3 antd 退场（生产面）：antd/@ant-design/icons 挪 devDependencies（外网测试镜像仍需，构建产物零 antd）；icons 进 @ui-backend 切换（antd 版挪镜像目录、eview 版=icon-plus 候选名自适应+缺名问号占位，实名由校准 R15 侦察用例验证）；adapter-guard 扩拦业务直连 @nce/@bridge/@ui-backend；三方语义名一致性守护。**RT-01 构建产物零 antd/react 实体验证=内网 build 时执行（脚本待组 8.1）** — **F1 ✓**

## 6. F3 真浏览器与派生黄金（PR-12）

- [x] 6.1a F3 REAL 化（校准移交项载体）：vitest.browser.config 接 EVIEW_REAL 开关（define 注入——真浏览器无 process/require 实录坑）+@nce stub 同口径；新增 test/browser/eview-real-browser.test.tsx（Tabs 受控 rerender/真实点击、Tree 首渲+展开+选中，全程打点）；内网跑法=Chromium 离线包摆渡（ms-playwright-chromium-1234.tgz）+ EVIEW_REAL=1 --config vitest.browser.config.ts — **F3 外网 10 绿 3 skip**
- [x] 6.1b F3 移交项内网实跑收敛（2026-08-18，十轮 F3-R1~R10 全绿 4/4）：Tabs 受控+点击、Tree 首渲+受控展开+叶子/父节点选中回调全通。关键定案：①findDOMNode polyfill 需静态 import（浏览器无 require）②cWRP 更新路径同步死循环→key 重挂（Tab/Tree props 路径）③节点选中→window capture 委托+名称区锚拦截（旧箭头放行规则被 ev_tree_expanded 状态类误伤=R6-R9 真凶）④同击双发→桥 50ms 去重。**存量 F3 切片口径复核（弹层/嵌套 list/choice 在 eview 后端对等）挂组 7 前置**— **F3 ✓**
- [x] 6.2 派生黄金全量重跑零漂移确认（GD-01）：组 5 接线后 test/golden 全量持续绿（631 全绿含黄金套件）——schema→控制台派生与组件库无关的铁证成立 — **F1 ✓**

## 7. E2E 对等（PR-13）

- [x] 7.1a staging-smoke 选择器迁移（外网部分）：25 处 ant-* 清零→SEL 常量收口（类名依据校准报告实证 DOM：ev_inputSelect/ev_Dialog_closeIcon/ev_table_content tr/ev_tab_title/ev_badge_content/form-item-shell/fis-label/ub-menu）；两处 TODO-E2E 待内网首跑实证（下拉弹层选项、Tab 溢出折叠形态）；**内网构建链解锁**=Dockerfile.prebuilt+docker-compose.prebuilt.yml（宿主 build dist 直装镜像，绕开容器内 @nce 安装死路）
- [x] 7.1b E2E 内网联调收敛（2026-08-19，**十三轮 R1-R13 终局 21/21 全绿**，PR#383-#396）。修复清单（原七连环之后续）：⑧FA-05 锚点回填观察器（桥 anchorId 只落 id、eview 不透传 data-test——80 条契约 DOM 全丢）⑨设备 IP 参数化 E2E_DEVICE_IP（kind=K8s 服务名）⑩tab 精确匹配回归修复（hasText 子串误中动态VLAN 只读表——#385 自伤）⑪Dropdown 项 JSX 文本化（[object Object] 真 UI bug）+侧栏输入限宽（364px 溢出盖内容区）⑫Table expandable 树数据拍平（变更对话框子行）⑬confirm 改 Dialog（MessageDialog 实为顶部消息横幅无按钮——提交确认链路断死，截图实证）⑭Tabs 内容区按激活项 key 重挂（同类型组件复用旧列残留）⑮**Tabs 标签栏全自绘**（eview Tab 三连：cWRP 死循环/溢出 index 错位/可见窗口 display 类——自绘终结+补回 role=tab）⑯Popover 自绘（TipBox display 受控被忽略）+点外关闭排除 teleport 弹层⑰E2E 若干断言跟随（按钮文本 apply/点 input 本体/类名定位 dialog）。**方法论：diag 侦察用例（DOM dump/矩形/API 抓包/列清单）+失败截图人眼判读=每轮一锤定音**
- [x] 原 7.1b 记录：E2E 内网首跑（进行中，多轮联调）：**已破七连环**（按序）①icon-plus 默认导入构建炸→命名空间导入②kind 部署链=Dockerfile.prebuilt+rollout（-n usmp-system）③icon-plus attachShadow 二次挂载崩→index.html 内联幂等守卫（closed 模式 WeakMap 记账）+install-guards 入口首行+图标级 IconBoundary（R08）④Google Fonts 外链离线必败→删外链走回退栈⑤React #520/#185 恢复循环压崩页面→**根因=eview TreeNode.cWRP 无条件 setState×60+ 节点超 React19 嵌套上限（编译产物实证）→Menu 桥自绘左树**（ev_* 类名承观感、id/data-test 锚保契约、label JSX 原样渲染）⑥onRecoverableError 钩子常驻（循环类问题自曝组件栈）⑦StrictMode 暂摘（恢复评估挂 8.1）。**下轮：内网同步 main 全量 E2E（预期进入真实选择器收敛）**
- [x] 7.2 浏览器口径拍板 **chromium 单浏览器**（内网离线环境 firefox/webkit 各需 ~300MB 离线包摆渡、收益为兼容性抽查——降口径并记录，波 C 后有真实需求再补）；`make e2e-local` 全栈门禁恢复=**USMP_UI_BACKEND=antd 构建开关**（vite/Dockerfile ARG/e2e-smoke.sh 三点接线：外网无 @nce 真包 eview 构建必炸，antd 镜像链路实证零 @nce——业务流程回归门禁；eview 真身验收=内网 kind E2E 21/21 已收敛）；**豁免窗口停用**（pre-push e2e 拦截随 antd 口径恢复生效）
- [x] 7.3 覆盖率收口：src/ui/eview 桥 346 行回收入分母（组 5 承诺）+icons F1 回填（0%→全分支）；实测 93.87/80.90/94.17/95.59 按下沿重钉 93.7/80.7/94.0/95.4（分母重算先例）；display/inputs 桥分支补测挂组 8 回填爬升

## 8. 工具链与文档收尾（PR-14）

- [x] 8.1 工具链收尾：kind-deploy.sh 支持 USMP_FRONTEND_DOCKERFILE=Dockerfile.prebuilt（离线内网）；build-release.sh 加 @nce 真包前置校验（交付构建=eview 真身，缺包快速失败）；StrictMode 恢复（守卫幂等后双挂载安全）+onRecoverableError 常驻；CI 六工作流全绿复核（frontend-ci/browser/storybook/contract/e2e-staging/pr-size）
- [x] 8.2 CLAUDE.md §3 栈口径（EviewUI+@ui-backend 单点切换+三自绘）/§5.6/§7.2 antd 提法中性化；frontend/TESTING.md 新增「UI 双后端口径」章节（五场景表+FA-05 回填）
- [ ] 8.3 /opsx:sync（frontend-runtime 新主 spec + frontend-ui-adapter MODIFIED/ADDED 合入；frontend/ui-i18n 仅 Purpose 栈名）
- [ ] 8.4 /opsx:archive + 记忆归档（单独 commit）+ follow-up 债登记（Open Questions 1-3 + 目视验收妥协项）
