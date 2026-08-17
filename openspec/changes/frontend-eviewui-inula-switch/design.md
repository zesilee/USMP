# Design: frontend-eviewui-inula-switch

## Context

现栈：React 19.2 + antd 6.6 + Vite 8 + react-router 8 + zustand 5 + axios + 自研 i18n 薄层（vue-i18n 同形 API）。业务代码经 `src/ui` 适配层隔离组件库（FA-01~04 守护测试硬拦直接 import），业务逻辑集中在纯函数层（`src/form`、`src/utils` 派生逻辑，R05 schema 驱动）。测试 F1~F4 四层 + 覆盖率棘轮 94.3/83.2/94.5/96.0 + 80 条 data-test E2E 契约。

目标栈与全部可行性结论见调研档案（**本 design 不复述，直接引用**）：
- `docs/research/eviewui-inula-migration.md`——运行时/四件套/Table 矩阵/工作量总账
- `docs/research/eviewui-component-matrix.md`——24 组件映射/五个系统性结论/切片验证清单

## Goals / Non-Goals

**Goals:**
- 严格功能对等切换到公司栈（openInula + EviewUI + Inula 全家桶），用户可感知行为不变。
- 保住三样资产：业务纯函数层零改动、80 条 data-test E2E 契约（锚点落点可变、语义不变）、覆盖率棘轮不倒退（分母重算按先例注明）。
- FA-01~04 适配层军规原样承接：换库后业务代码依旧禁直接 import 组件库。

**Non-Goals:**
- 不做任何业务功能变更、不动后端契约。
- 不迁 Storybook 故事内容（Non-Goal 延续）。
- 不追 EviewUI 新版本（锁 3.10.28，有新版另起 change）。
- 不做视觉重设计（密度/观感以"接近现状"为准，目视验收）。

## Decisions

### D1 闸门先行，六项不过即停
垂直切片先打通「模块控制台列表页」最小环（openInula 挂载 → Tree 左树 → Table 列表 → TextField/InputSelect 编辑）并逐项验证矩阵 §3 六项运行时行为（半受控实测/测试栈兼容/Tree 受控桥时序/Radio 参数序/InputSelect 弹层挂载/Loading iconUrl）。**任一项不过：停、回报、重估**——不带病进入全面切换。闸门结论写 `gate-conclusion.md`（复刻 React 重建 R05 闸门惯例）。

### D2 并行目录策略：原地改造而非清场重建
与上次 Vue→React"全新项目清场"不同，本次**业务层大头兼容**（纯函数/组件结构/测试断言大部分沿用），采用**原地渐进改造**：
- 基建四件套逐个替换（i18n→请求→store→路由，依赖由浅入深），每步旧路径保留可回退、双路径测试验证后删旧；
- 适配层 `src/ui` 内部按组件逐个从 antd 转发改为 EviewUI 桥接，**业务调用点 API 形状保持 antd 形态**（适配层抹平），把改动锁死在 ui 目录；
- antd 与 EviewUI 在窗口期共存于 node_modules（构建产物只引其一），全部组件切完后一次性移除 antd 依赖。

### D3 适配层桥接总纲（矩阵结论的实现口径）
- **半受控兜底三板斧**（按成本升序取用）：① 事件拦截（Checkbox.onPreChange / InputSelect.shouldRender）；② 受控回写（onChange 后 props 回灌，依赖 cWRP 同步）；③ key 重挂强制重建（通用兜底，适配层提供 `useRemountKey` 工具）。切片实测决定每控件用哪档。
- **FormItemShell**：LabelField(label+必填星) + 控件 + 自绘错误 div；错误红框走 error class；绝不给 EviewUI 控件传 validator/required/rules（校验权威在自研引擎不动摇）。
- **feedback 自养挂载点**：body 下常驻容器 + createRoot 渲 DivMessage（toast，映射 info→default、常驻横幅关自动消失）/ MessageDialog（confirm，ok/cancel resolve Promise）。
- **data-test 锚点两级约定**：能透传的（Tab/SearchInput/LabelField/icon-plus）直传；其余适配层外包 `<div data-test=…>` 或映射到组件 `id` prop——**约定写进适配层 README 并加守护测试**（拦"business 代码直接给 EviewUI 组件传 data-test 却被吞"的静默失效）。
- **主题**：tokens.ts 改为产出 design-token CSS 变量覆盖段（#0C5EA6 生成 10 档 --brand-* 色阶 + 功能色/密度覆盖），在 design-token 样式之后加载。
- **Tree 左树桥**：leftTree→Tree.data 转换（双语重建 data 与现状同构）、expandedKeys/selectedKeys 受控回写、搜索优先沿用自研 filterLeftTree（Tree 内置 findLevelNodes 作备选，切片比较后取一）。
- **Table 受控壳**：按已定矩阵（rowSelection→checkedRows 桥、onChange→onColumnSort+disableEviewSort、pagination 拆平、rowClassName→customStyleRows 重算、rowClickDelay:0）。

### D4 路由 v5 化
- 路由表从 createBrowserRouter 数据形态改 `<Switch>/<Route>` JSX 树；MainLayout 从 Outlet 改 children 组合。
- `useNavigate/useSearchParams` → 适配薄函数（`src/router/compat.ts`）包 useHistory/useLocation，调用点机械替换。
- 离开守卫：Router `getUserConfirmation` 全局桥（异步弹适配层 confirm）+ `Prompt when={dirty}`，语义与 useBlocker 对等（FE-23 Scenario 回归钉住）。

### D5 测试策略
- F1（纯函数/表单核心/派生黄金）：**零改动直接沿用**——这是本次改造最大的安全网。
- F2：断言以文案/role/data-test 为主的沿用；`ant-*` 类名断言按映射表批量替换为 EviewUI 结构。
- F3 真浏览器：弹层挂载口径按切片实测结果改写（InputSelect 若不 teleport，选择器反而简化）。
- F4 E2E：data-test 契约不变；`ant-*` 选择器迁移沿用 el-*→ant-* 剧本；窗口期 e2e 豁免按 tasks 红灯声明两段口径（复刻上次）。
- 覆盖率棘轮：分母重算（适配层扩容）按先例重钉并注明，加测不降标。

### D6 运行时与依赖
- Vite alias：`react`/`react-dom` → openinula（jsx-runtime 同步）；`react-intl` → inula-intl（EviewUI 内部 contextType 用）。
- tsconfig types 加 node（EviewUI d.ts 依赖 NodeJS.Timeout）；@types/react 移除、openinula 自带类型接管（typecheck 清错潮预期在切片阶段暴露）。
- Vite 补 less（EviewUI 样式）；design-token 0.0.24 为基准版本。
- **新依赖审查军规**：openInula 时代引任何 React 生态库前先审其是否用 React 18+ API（useId/useSyncExternalStore 高频），写进 frontend-runtime spec。

## Risks / Trade-offs

- **半受控地基**（🔴 切片首验）：若三板斧都压不住关键控件（表单回显/程序化清空失真），全案重估——这是唯一预设的"停止线"。
- **openInula 成熟度**：latest=alpha、社区小；换来公司栈统一。缓解：F1 安全网 + E2E 契约 + 闸门制。
- **单向门**：全家桶互相咬合，退出成本高；决策已由用户拍板，风险如实留档。
- **Tab 下标寻址**：key↔index 映射在 Tab 动态增删时有漂移风险——我们 Tab 集在设备/模块选定后稳定，风险可控，测试钉住。
- **密度/观感**：无 size prop，CSS 覆盖后目视验收，可能有细节妥协（如实呈现给用户拍板）。

## Open Questions

1. EviewUI 是否有 >3.10.28 版本（半受控受控化/新组件）——非阻塞，有新版另起 change 升级。
2. E2E 锚点最终形态（wrapper div vs id 映射）——切片阶段按实测选定后统一。
3. 纯铃铛/显示器两个图标用近似还是内部补充包——目视验收时定。
