# EviewUI / openInula 迁移调研（2026-08-17）

> 目的：评估前端从 React 19 + antd 6 切换到 **EviewUI（@nce/eview-react）+ openInula 全家桶**的可行性、风险与工作量。
> 方法：**全部结论基于实物验证**——EviewUI 的 d.ts 与 package.json 逐行读、openInula 与三个生态包从 npm 下载拆包核对导出面，不转述文档。
> 结论有效期锚点：openinula@1.0.0（npm latest 标签仍为 1.0.0-alpha）、eview-react@3.10.28（2020 时间戳）、inula-router@1.0.16、inula-request@1.1.9、inula-intl@1.0.35。

## 0. 已拍板的决策（用户，2026-08-17）

- 运行时按 **openInula**（= 华为 Horizon 的开源版，gitee.com/openInula/inula）评估。
- 生态选型全家桶：**状态 inula-X、路由 inula-router、请求 inula-request、国际化 inula-intl**。

## 1. 关键事实链

### 1.1 EviewUI（@nce/eview-react 3.10.28）

- **peerDependencies 没有 react**，宿主是 `@cloudsop/horizon`（+ horizon-intl / @hui/design-token / @baize/wdk）→ 它是为 Horizon/openInula 运行时写的组件库。
- 2020-2022 时代：webpack 4、TS 4.4、类组件、browserslist 含 IE 11、`import './index.less'`（Less 样式，Vite 需补 less 支持）。
- 图标在 `@nce/icon-plus`、主题走 `@hui/design-token`（适配层 tokens.ts 的对接点）。
- Table 组件重度依赖无前缀 `componentWillReceiveProps` 遗留生命周期——**在 openInula 上无风险**（它本来就是为此运行时写的）；若走"标准 React 19 直跑 EviewUI"路线才需要实测。

### 1.2 openInula 1.0.0 实测 API 面（npm 拆包核对 @types + 运行时 js）

| 有 ✓ | 没有 ✗ |
|------|--------|
| `createRoot`、`StrictMode`、`createPortal`、`forwardRef`/`memo`/`lazy`/`Suspense`、React 17 级全套 hooks、jsx-runtime | `useSyncExternalStore`（Gitee master 有开发痕迹，未进发布版）、`useTransition`、`useDeferredValue`、`useId` |

- 自带 inula-X 响应式扩展：`createStore/useStore/useSelector/useDispatch`。
- 成熟度警示：npm `latest` 标签指向 **1.0.0-alpha**（2025-04 最后更新）、社区 722 星、生态小。

### 1.3 我们的 React 18+ API 暴露面（全量 grep，就这么多）

| 用点 | openInula 下的出路 |
|------|--------------------|
| `createRoot` + `StrictMode`（main.tsx） | 原生支持，零改动 |
| `useSyncExternalStore` ×4（i18n locale 订阅：Sidebar/Header/ModuleConsolePage/ui-provider） | 换 inula-intl 后挂 `on('change')` 事件，**自然消失** |
| `useBlocker`（react-router v8，FE-23 攒批离开守卫） | inula-router 的 `Prompt` + `getUserConfirmation`（见 2.3） |
| zustand v5（内部依赖 uSES） | 换 inula-X（见 2.4） |

业务组件全是函数组件 + 基础 hooks，R05 把业务逻辑压在纯函数/schema 派生里——**运行时耦合面天然很小**。

## 2. 四件套实测

### 2.1 inula-intl 1.0.35 —— 两头通吃，最省心 ✅

- 对外 **react-intl 同形**：`IntlProvider/useIntl/FormattedMessage/injectIntl/createIntl` → EviewUI 要的 intl 上下文**原生满足**（它就是 horizon-intl 开源版）。
- **内置 VueI18n 适配器**：`$t(key, values)`、`changeLanguage`、`loadMessage`、`on('change')` 事件——与我们自研 i18n 薄层（vue-i18n 同形 `i18n.global.t` + `subscribeLocale`）同构。**迁移 = 换薄层内核，全项目 t() 调用点与订阅点零改动**，词表 JSON 直喂，UI-02 键名稳定军规自动满足。
- 注意：类型 `import from 'openinula'` → 只能配套 openInula 用。

### 2.2 inula-request 1.1.9 —— 近乎平替 ✅

axios 同形：`ir.create({baseURL, timeout})`、拦截器、泛型 `ir<T>`、`paramsSerializer`。我们 api 层=单实例+28 个类型化封装、无拦截器 → **改 import + create，半天**。回归点：`filter` 重复参数序列化口径（axios 默认 `filter[]=` 后端只认 `filter=`，已有测试钉着）。

### 2.3 inula-router 1.0.16 —— **react-router v5 API**，最大单项改造 ⚠️

导出面：`BrowserRouter/HashRouter/Switch/Route/Redirect/Prompt/Link/NavLink/useHistory/useLocation/useParams/useRouteMatch/withRouter`。**没有** Routes/Outlet/useNavigate/useSearchParams/useBlocker/data router。

| 我们在用（react-router v8） | inula-router 对应 |
|------------------------------|-------------------|
| `useNavigate` | `useHistory().push` |
| `createBrowserRouter` + `<Outlet>` 嵌套布局 | `<Switch>/<Route>` JSX 树 + children 组合（MainLayout/router.tsx 重写） |
| `useSearchParams` | `useLocation().search` 手动 URLSearchParams |
| `useBlocker` 离开守卫 | **`Prompt when=… message=…` + Router `getUserConfirmation` 桥接**（异步回调，可弹自定义确认框，FE-23 语义可完整保留） |
| `useParams` | 同名保留 |

估 2~3 天含测试改造（createMemoryRouter → v5 MemoryRouter）。URL 形态不变，**E2E 冒烟不受影响**。

### 2.4 inula-X —— 内置 openinula 核心，唯一未验到细节的项 🟡

`createStore/useStore/useSelector/useDispatch` 导出确认存在；store 定义形态（state/actions/computed 结构）**待拿文档或 d.ts 确认**。我们 5 个 zustand store（485 行，computed 已是方法形态）预估 1~2 天。

## 3. 系统性风险（如实）

1. **单向门**：四件套类型互相 `import 'openinula'`，全家桶+运行时是整体决策，无"先换一件"中间态（垂直切片除外）；退出成本高于换组件库。
2. **openInula 成熟度**：latest=alpha、生态小；此后每引一个 React 生态库都要先审"用没用 React 18 API"（`useId` 在无障碍类库里极常见）——长期税。
3. **测试基建未知**：vitest + `@testing-library/react` + happy-dom 在 openinula 下能否跑**无现成答案**（EviewUI 自测用 @testing-library/react 12，说明生态可测但版本老）。159 个测试文件与 T08 棘轮的存续依赖此项——**垂直切片首日必验**。
4. **样式链**：EviewUI 是 Less（Vite 补 less 依赖）+ @hui/design-token 主题体系，tokens.ts 对接方式重设计；Table 无 size/紧凑密度 prop，密度靠 css 覆盖后目视验收。

## 4. EviewUI Table 差异矩阵（antd 用法 → EviewUI props，d.ts 逐项核对）

前提：完整接口在 `Table/interfaces/{TableProps,ColumnProps,TableStates}.d.ts`。受控通道齐全（此前"命令式非受控"的担忧被 props 实测推翻）。

| 我们在用（antd Table） | EviewUI 对应 | 适配层动作 |
|------------------------|--------------|-----------|
| `columns[].render(值,行)` ×21 | `render(cellValue, rowData, options, row, isEdit)` | 参数换序 |
| `width / fixed / sorter` | `width / freezeCol / allowSort` + 列上 `sort: 'asc'|'desc'|'origin'` | 字段名映射 |
| `ellipsis` | 无直接对应 | `showTooltipOverFlow` + `tipFormatter` 近似 |
| `dataSource` | `dataset` | 改名 |
| `rowKey`(函数) | `rowKey: string` / `keyIndex` | 函数→字段名 |
| `rowSelection`{selectedRowKeys,onChange} | `enableCheckBox` + 受控 `checkedRows` + `onRowCheck/onHeaderCheck`（`checkedRowsForceUpdate` 强刷、`preserveCheckedRows` 跨页勾选） | 受控转换 |
| `onRow` 行点击 | `onRowClick` + `rowClickDelay: 0`（默认有单双击去抖延迟） | 直连 |
| `onChange` 排序下推 | `onColumnSort(sortColumn, sortType)` + `disableEviewSort`（后台全权排序）+ `delayOnColumnSort` | 直连（服务端排序是一等公民，正对 BR-13） |
| `pagination` 对象/false | `enablePagination` + 受控 `currentPage/pageSize/recordCount/pageSizeOptions` + `onPageChange/onPageSizeChange` | 对象拆平 |
| `rowClassName`(函数，变更标记上色) | 无函数式；`customStyleRows`/`cellClassName` 按行号映射对象 | 每次 render 把函数重算成映射 |
| `locale.emptyText` | `emptyTableMsg` + `showEmptyImage` | 直连 |
| `loading` | `enableLoading` | 直连 |
| `size="small"` | 无密度 prop | css 覆盖行高 |
| `data-test` | 不透传（无 index signature；有必填 `id`） | 适配层外包 div 补 |

- 它最复杂的单元格编辑体系（renderType 十种/双击编辑/validator）我们**完全不用**（编辑在详情区），正好绕开。
- **类型坑**：TableProps 绝大多数属性声明为必填（运行时靠 defaultProps 兜底）——业务直用 typecheck 会要求传 50+ props。适配层用自己的 Props 类型（我们 14 项用法的形状）包住，坑锁死在 `src/ui`。
- 适配层预估 **150~250 行** 转换代码。

其余 23 个在用组件的「实际使用面」清单见对话期产物（Menu/Tabs/Modal/Select 等 props 逐项枚举），拿到对应 d.ts 后照 Table 方法逐个出矩阵；Table 是最难的，它能对上其余大概率更轻。

## 5. 路线与工作量总账

**路线 B（已拍板方向）：openInula 运行时 + 全家桶 + EviewUI**

| 项 | 估时 |
|----|------|
| 运行时 alias + 挂载 | 0.5 天 |
| i18n 内核换 inula-intl（调用点零改动） | 1~2 天 |
| 请求换 inula-request | 0.5 天 |
| 路由 v5 化（含 Prompt 守卫桥接） | 2~3 天 |
| 5 个 store 换 inula-X | 1~2 天（API 待确认） |
| 测试基建验证 | 首日必验，风险未知 |
| EviewUI 组件适配层 | Table 150~250 行，其余组件另计 |

（路线 A=标准 React 19 直跑 EviewUI + alias horizon→react，改动最小但未拍板，作为 B 受阻时的备选记录在案。）

**垂直切片验证顺序（一周出结论）**：
① openinula 挂载 + vitest/testing-library 测试栈跑通 → ② EviewUI Table 挂上（动态列 + 受控勾选 + 服务端分页排序）→ ③ Prompt 离开守卫桥接 → ④ inula-X store 形态确认。

## 6. 开工前还缺的材料

> 2026-08-17 更新：离线机全量 d.ts（eview-bundle.tgz）已到手，原第 2/4 项闭环——
> **全量 24 组件差异矩阵见 [eviewui-component-matrix.md](eviewui-component-matrix.md)**
> （五个系统性结论：半受控地基风险🔴 / Form 弃用 / data-test 不透传 / 无命令式反馈 / 令牌注入无通道；
> 适配层重估 1500~2500 行）。

仍缺：

1. inula-X 的 store API 文档或 d.ts（唯一未实测项）。
2. EviewUI 任一组件的**编译产物 JS**（如 Button/Button.js，看内部 require 谁——根 index.js 只是转发壳判不了 alias 方向）。
3. `@hui/design-token` **完整包**（采到的是空壳，仅 package.json；主色 #0C5EA6 等 10 令牌注入无法验证——优先级最高）。
4. 向 EviewUI 团队确认是否有比 3.10.28 更新的版本（顺带问：半受控控件是否有受控化改造、有无内联侧边导航组件）。
