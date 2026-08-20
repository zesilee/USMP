---
name: eviewui-switch-implementation
description: EviewUI/openInula 切换**全量交付归档(2026-08-20,PR#342-#404,archive/2026-08-20-frontend-eviewui-inula-switch)**：校准14轮+F3十轮+E2E十三轮方法论、三大自绘、双后端/双运行时开关体系、波C自研store拍板；碰前端运行时/桥/测试口径前必读
metadata:
  type: project
---

**change `frontend-eviewui-inula-switch` 实施进行中**（2026-08-17，PR #342-#353 已合入；台账=该 change 的 tasks.md，制品含 gate-conclusion/route-decision/gate-round1-findings）。

**已完成**：
- **闸门通过**（两轮离线摆渡）：EviewUI×openInula 可运行、R05 动态列通、Tree 受控桥成立、半受控档位定案（②cWRP 回写[实证]+③key 重挂）；两残留定性为测试环境限制（inula 合成事件 e.target 在 happy-dom 为 null——修法=派发前 defineProperty；Radio 参数序→桥自适应免疫）。
- **路线乙定案**（route-decision.md）：antd(React18+)⟂openinula(17级) 死锁 → 组件库先换 on React 19（cWRP 实证仍被调用）、运行时+inula 全家桶（intl/router/inula-X）全部推终局波 C（波 C 终态=闸门实测组合）。inula 两件套反向 alias 在测试链路不可行（ESM 内嵌 hook 运行时+vitest CJS 逃逸）。
- **组 2 先行波**：uSES 清零（useLocale 薄层 hook）+RT-02 守护；请求 axios→inula-request（filter 拼 URL，真实 E2E 验证）。
- **组 3 适配层核心**：vendor/eview-types（92 d.ts 入库+tsconfig paths=外网 typecheck 地基）、useRemountKey/useSemiControlledBridge、主题十档色阶 CSS 变量段、iconMap 22 语义名、feedback 自养挂载点（flushSync 立即可见）、FormItemShell。
- **组 4 全 24 桥**：4.1 轻组 11/4.2 表单 3/4.3 结构 3（Tabs key↔index+内容区自渲、Menu→Tree 左树、Table 受控壳全矩阵项）/4.4 收尾 5。对外恒 antd 形态（接线零改动）。

**混合协作模式工程机制**（外网开发+内网集成点）：
- vendor types 外网 typecheck；vitest `test.alias` 把 @nce 按**子路径映射独立 stub 文件**（catch-all 单文件会让多组件 vi.mock 工厂共享模块身份互相覆盖——实录坑）；替身规格=vendor d.ts+matrix+gate 实测非虚构；覆盖率分母窗口期排除 src/ui/eview/**（组 5 接线时纳入重钉，vitest.config 注释有回收点）。
- 内网集成点操作：同步 main→注释 vitest.config 的 @nce stub alias 两行→跑四份 eview-bridge-*.test.tsx→回传报告。

**实录坑集**（续作防重摔）：worktree 钩子须 `-c core.hooksPath` 绝对路径；vi.mock 工厂提升（辅助进 vi.hoisted+工厂内动态 import react）；TabItem 挂具名导出须命名空间导入；React 19 createRoot.render 异步提交（命令式反馈用 flushSync）；useSemiControlledBridge 拒写场景父不重渲染须 onEmit 主动 force；eview Spinner min/max 默认 0/100 非无界；DivMessage 默认 10s 自动消失；openinula 包 exports 缺 types 条件（tsconfig paths 映射修）。

**内网真实校准已收敛**（2026-08-18，14 轮摆渡 R1-R14/PR#358-#367，终局 CAL-R14=16 passed/1 skip 全绿）。载体=test/integration/eview-real.test.tsx（EVIEW_REAL=1 启用；含版本指纹 CAL-Rn+beforeEach 用例入口标记——挂死轮次也能精确定位，R8 教训）。核心定案：
- **Table 桥三定案**：①自定义列渲染必须 renderType（取运行时枚举 ColumnRenderType.CUSTOM，实值='custom'）；②勾选通道 checkedRows 双向均为**行序号**语义（不设 keyIndex 时；keyIndex 又是数字列序号对不上对象行）——桥内做行序号↔rowKey 双向映射闭环；③render 实参=(cellValue, [cellValue], options, **{id,data,rawData}**, {isEdit})——行数据在第 4 参上下文对象 rawData/data 字段内，d.ts 五参签名与真实调用完全不符（探针一轮定案，桥判别链 [ctx.rawData, ctx.data, a4, a2] 取首个对象非数组）。
- **Tabs/Tree 交互移交 F3**（tasks 6.1）：eview Tab 的 activeKey 更新路径与 Tree 的**首次渲染**均在 happy-dom 同步死循环（内部布局循环依赖真实元素宽度，happy-dom 恒 0 不收敛；同步循环连 vitest 单用例超时都中断不了——五轮挂死的总根因）。Tabs 静态渲染已实证正确；真 Chromium 真实布局下循环可收敛。
- **修桥方法论**（14 轮提炼）：d.ts 是形态线索不是行为契约（renderType 缺失/参数位/枚举实值三连翻车）；猜两轮不中就**放探针拿实参真面目**（(...args)=>类型摘要，一轮换确定性）；交互点击类用例排文件最末+挂死嫌疑项拆独立用例（保证其余数据先落袋）；findDOMNode polyfill+raf 异步化+IntlProvider 包裹=REAL 前置三件套。

**组 5 已全量交付**（2026-08-18，PR#368-#370）：@ui-backend 裸别名单点切换（生产→eview 桥/外网测试与 storybook→src/ui/antd-backend antd 镜像/EVIEW_REAL=1→全链真身；Vite alias 不作用于相对导入是首版翻车根因）；SchemaForm 换 FormItemShell；antd 家族挪 devDeps；icons 切 icon-plus（**R16 实证：2608 图标、命名=IconPlus 前缀、22 语义名全命中**）；修真 flaky=eview provider 无 antd feedback 绑定时静态 message 3s 定时器 teardown 后炸（provider 对称入切换根治）；已知债=列头筛选菜单/展开行映射（tasks 5.1）。**校准终态 CAL-R16=17 passed/1 skip 全绿。**

**组 6.1 F3 真浏览器校准已收敛**（2026-08-18，十轮 F3-R1~R10 终态 4/4 全绿）。四大定案（改 Tab/Tree 桥前必读）：
- findDOMNode polyfill 须**静态 import react-dom default**（真浏览器 ESM 无 require——F3-R1）；REAL 开关在 browser 配置须 define 注入（无 process）。
- **cWRP 更新路径同步死循环**（happy-dom 与真浏览器均实证，真实布局救不了）→ Tab（selectedIndex 变化）/Tree（openKeys/selectedKeys 变化）一律 **key 重挂**恒走首渲路径（首渲两环境实证正常）；受控数据 props 喂回语义无损，代价=无过渡动画（波 C 退役）。
- **Tree 节点选中=window capture 委托+名称区锚拦截**：eview 选中监听在节点容器、点容器触发内部 setState 踩同一死循环；委托拦截边界=<a id=ev_tree_node_id{key}>（箭头天然在锚外自然放行走受控展开链）。**血泪坑：旧「箭头放行」规则 [class*="expand"] 被祖先 ev_tree_expanded 状态类误伤——R6-R9 四轮零回调+挂死的真凶，事件层级假设（React 合成层/document 层）皆误诊；全景探针（三层监听+桥打点）一轮破案**。同击双发（真浏览器实测）→桥 50ms 同锚去重。
- 真浏览器 interop：pickDefault 须**多层剥 default**（vite optimizer CJS 双层包装，happy-dom 单层即中）；具名导出（TabItem）用候选链逐层找。
配套：Chromium 离线包摆渡（ms-playwright tgz→~/.cache）；F3 报告减脂=套件 console 节流+grep 过滤命令。

**组 7 E2E 对等已收敛**（2026-08-19，十三轮 R1-R13 终局 **21/21 全绿**，PR#382-#396，kind 全栈+真浏览器）。核心定案（改桥/查「组件不响应」前必读）：
- **eview 三大组件全自绘**：Tree（TreeNode.cWRP 无条件 setState×大树超 React19 嵌套上限 #185）、Tabs 标签栏（cWRP 死循环/溢出 index 错位/「可见窗口」只给部分标签 display 类三连）、Popover（TipBox display 受控被忽略）——桥自绘=ev_* 类名承观感+role 语义补回+ub- 自有类；自绘先例可复制（点外关闭 window capture+teleport 弹层排除）。
- **FA-05 锚点回填观察器**（bridge.ts installAnchorAttrObserver）：桥 anchorId 只落 id=dt-*、eview 不透传 data-test——MutationObserver 全局回填，80 条契约零改造。
- **confirm=Dialog 非 MessageDialog**（后者实为顶部消息横幅、无按钮——提交确认曾断死，截图实证）；Table expandable=树数据拍平；Tabs pane 按激活项 key（同类型 children 复用残留）；宽 Select 点容器空白不弹（eview 只认 input 本体——E2E 点 input）。
- **环境口径**：kind 部署=Dockerfile.prebuilt（宿主 build dist 直装，容器 npm 装不到 @nce）+kind load+rollout（-n usmp-system）；E2E_DEVICE_IP=netconf-sim.default（K8s 服务名≠compose 的 192.168.1.1）；Google Fonts 外链已删（离线必败）。
- **E2E 侦察方法论**：diag.spec 用例（DOM dump/boundingRect/response 抓包/列 col-id 清单/弹层计数）+失败截图人眼判读+精确 tab 正则（hasText 子串曾误中「动态VLAN列表」）——每轮一锤定音、十三轮无一轮空转。

**波 C 已全量交付、change 已归档**（2026-08-20，PR#400-#404）：
- 状态层=自研 React 17 级薄层 createStore（拍板变更弃 inula-X：锁死 openinula 会废外网测试体系；zustand 同形 API、647 零回归）；路由收口 @app-router 别名双实现（compat.ts=react 直通/compat.inula.tsx=inula-router v5：递归 Switch/Route+Outlet Context+useBlocker=Prompt+getUserConfirmation 桥）；i18n 薄层本就自研无内核（2.1b 定性核销）。
- **翻转开关体系**（终态）：USMP_RUNTIME（缺省 inula、react 显式回退）×USMP_UI_BACKEND（antd=e2e-local 口径、联动强制 react）×EVIEW_REAL（内网真桥测试）——外网开发/测试运行时恒 React 19（react 家族=devDependencies），交付产物=openinula+EviewUI。内网验收：openinula 产物 E2E 21/21 一次通过。
- polyfill/attachShadow 守卫保留（自适应/幂等，防御纵深）；三大自绘保留（inula 上纯组件安全）。
- spec 已 sync（frontend-runtime 新建含 RT-01 按拍板修正、FA-02/05/06 合入、frontend/ui-i18n Purpose 栈名）；change 归档 archive/2026-08-20-frontend-eviewui-inula-switch。

**原「下一步」记录（组 5 前）**：**组 5 接线**（index.ts 切 eview 后端+调用点微调+antd 退场+覆盖率分母回收重钉+findDOMNode polyfill 生产安装）→组 6 F3（含 Tabs/Tree 校准移交项）/派生黄金→组 7 E2E→组 8 收尾→波 C 运行时翻转（openinula+intl 内核+router v5 化+inula-X，制品在 tasks.md 组 2 挂起项）。

**归档后收尾波（2026-08-20，PR#406）——两债销账**：
- **列头筛选债（tasks 5.1）已清**：eview 列 filter/embeddedFilter 形状 d.ts 未暴露（仅 object）→ 不猜其 API，桥自绘 ColFilter（列 title 收 ReactNode 实证可用；漏斗触发器+portal 弹层+确定/重置），谓词过滤在排序前、onChange 合成 antd filters 快照、点外关闭弃草稿；F2 5 用例 TDD、内网 E2E 21/21 验证。棘轮上调 94.2/81.0/94.5/95.8。
- **外网 e2e 门禁空转债已清**（此前只能 USMP_SKIP_E2E 跳过）：①e2e-smoke `up -d --build` 用缺省参二次重建前端（eview 口径外网必炸，靠切换前旧层缓存掩盖，builder prune 后现形）→构建/启动分离；②规格类名全 ev_* 对 antd DOM 恒不匹配→SEL 表双口径（E2E_UI_BACKEND=antd），antd6 类名探针实证：对话框=.ant-modal（无 -content 层）、抽屉=.ant-drawer-section、Tab 溢出走 .ant-tabs-nav-more→.ant-tabs-dropdown-menu-item、Dropdown 项=.ant-dropdown-menu-item。外网 pre-push 21/21 真实拦截恢复。
- 剩余债（有触发再启）：展开行拍平缩进视觉、Dropdown label 锚点债、CI Install Chromium 缓存、axios 清退评估、firefox/webkit 按需补、三大自绘 inula 上复评回退真组件（纯观感）。
- 环境坑追加：内网新终端丢 Node22 PATH→rolldown 炸 styleText（node:util，需≥20.12）；后台 push 会被杀（再证）；pre-commit 前端单测偶发假失败重试即过（并行前科同款）。

相关：[[eviewui-inula-migration]]（调研与矩阵）、[[react-antd-rebuild]]（适配层军规与上次迁移方法论）。
