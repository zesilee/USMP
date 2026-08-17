# EviewUI 组件差异矩阵（全量 24 组件，2026-08-17）

> 姊妹篇：[eviewui-inula-migration.md](eviewui-inula-migration.md)（运行时与全家桶调研）。
> 材料：离线机采集的 eview-react 3.10.28 **全组件 d.ts**（`eview-bundle.tgz`，types-only，组件 JS 不在包内——凡标"需实测"的行为判定须在真包/垂直切片上冒烟确认）。
> 方法：四组并行 agent 逐 d.ts 精读，对照面 = 我们的「实际使用面」清单（每组件真正用到的 props），非 antd 全量 API。
> **同类产品实证（2026-08-17）**：iMaster NCE-Fabric「设备特性编程」页截图确认——其左树（Tree）+ Tab + 特性表格（列筛选/排序/勾选/分页/行内编辑删除）+ 攒批工具栏（变更内容/试运行/重置/提交配置）与 USMP 模块控制台功能一比一同构，且就是 EviewUI 实现的。切换可行性从"接口推断"升级为"同类产品已验证"。

## 0. 五个系统性结论（比单组件差异更重要）

1. **半受控是全库地基风险 🔴**。真受控开关只有 Radio/RadioGroup（`isControlled`）与 Switch（`isControlToggled`）；TextField/Spinner/Select/InputSelect/MultipleSelect/Checkbox/Segmented 全是「props + 内部 state + componentWillReceiveProps 同步」的半受控老写法——打字/点击先改内部 state，父级拒绝回写**不还原**。我们的表单编排（configForm 纯函数核心）建立在真受控上。兜底工具箱：InputSelect `shouldRender` 夺权、Checkbox `onPreChange` 拦截、通用 key 重挂强制重建。**垂直切片必须首先验证半受控在我们链路下的实际表现**。
2. **Form/FormItem 必须弃用**。FormItem 的 `name` 必填且会收编子控件 value/onChange 进其内部 store，`validateStatus/help` 受控错误态**无对应**——与「校验权威在自研引擎」直接冲突。方案：自写 FormItemShell（LabelField 出 label+必填星 + 自绘错误 div + error class 红框）。这与我们「不接组件库表单 store」的既有架构决定一致。
3. **data-test 全库基本不透传**。仅 Tab/TabItem（index signature）、SearchInput（restProps）、LabelField（extends LabelHTMLAttributes）例外；其余全是闭合 type。**E2E/测试锚点策略整体换**：统一走各组件都有的 `id` prop 或适配层外包 wrapper div。80 条 data-test 契约的落点要重新设计。
4. **命令式反馈不存在**。全库唯一静态方法是 `MessageDialog.success()`。适配层的 `toast()`/`await confirm()` 要**自养挂载点**（body 下挂 div + createRoot 渲 DivMessage/MessageDialog，回调 resolve 后卸载），约 100 行。
5. **主题令牌注入通道已找到 ✅（2026-08-17 补采后闭环）**。首采"空壳"是提取脚本只保留 .d.ts/.md/json 误伤——整包补采后确认 `@hui/design-token` 实体是 **less 产出的 CSS 变量体系**（hdesign 1.1）：基础色板 `:root{--brand-05..90}`（默认华为蓝 #0067D1）+ 语义层 `--color-brand: var(--brand-50)` 引用；功能色同构（`--red/--yellow/--mint` 色阶 + `--color-warning: var(--yellow-50)` 等映射）；圆角/字号/间距同为变量（`--radius-*`/`--font-size-*`/`--space-*`）。**注入方案：由 #0C5EA6 生成 10 档 brand 色阶，在 design-token 之后加载一段 `:root` 覆盖样式即可——纯 CSS、零 less 重编译、零组件侵入**，10 个令牌全部有落点，适配层 tokens.ts 改为产出这段覆盖样式。版本口径：离线机 frontend 装的是 0.0.21、根目录 0.0.24（eview peer 要求 ^0.0.23，**以 0.0.24 为基准**——frontend 那份其实不满足 peer 声明，对方环境自身存在版本错配，接入时留意）。

## 1. 逐组件映射总表

图例：✅ 直接对应 / 🔧 需适配层转换 / ⚠️ 有缺口或需实测 / 🔴 硬缺口

### 表单控件

| 我们在用 | EviewUI 对应 | 关键映射与缺口 |
|----------|--------------|----------------|
| Input | **TextField**（搜索场景用 SearchInput） | ⚠️ 半受控；`onChange(新值, 旧值, e)` 换序；**无 allowClear/prefix/size**（clear 自绘挂 `suffix`、prefix 叠放容器、size 走 CSS）；自带 validator 体系**一律不传**防双校验权威 |
| InputNumber | **Spinner**（DragInput=Slider 别名、HexField=十六进制框，均排除） | ⚠️ 半受控；`min/max` 默认 0/100 **非无界必须显式传**；onChange 仅有效值回调+无效走 `onInputError` 须合并；无 controls 隐藏、无 placeholder |
| Select | **InputSelect**（Select 无搜索无 placeholder 无 onClear，排除） | 🔧 options `label→text`；`enableClear`+`onClear` ✔ 清空可区分；`onlySelect:true`≈showSearch 单选；⚠️ 半受控（`shouldRender` 可部分夺权）；**弹层不 teleport 到 body**（原地定位，F3/E2E 口径反而简化，需实测） |
| Radio/Radio.Group | **Radio + RadioGroup** | ✅ 有 `isControlled:true` 真受控（全库唯二之一）；🔧 children 写法→`data:[{value,text}]`；⚠️ onChange 形参名与注释相反（新旧值顺序需实测） |
| Checkbox | **Checkbox** | ⚠️ 半受控无开关；`onChange` 第 2 参才是 checked；`onPreChange` 返回 false 可拦截翻转（受控兜底） |
| Switch | **Switch**（Toggle=别名） | ✅ `isControlToggled:true` 真受控（唯二之二）；🔧 `checked→toggled`、`onChange→onToggle`+必传 `data={[false,true]}` |
| Segmented | **Segmented** | 🔧 options `label→text`、逐项禁用拼写是 `disable`；⚠️ 半受控 |
| Form/Form.Item | **弃用**，自写 FormItemShell + LabelField | 🔴 见系统性结论 2 |

### 布局导航

| 我们在用 | EviewUI 对应 | 关键映射与缺口 |
|----------|--------------|----------------|
| Menu（左树） | **Tree ✅（2026-08-17 翻案）** | 首判"无合格对应"是候选清单漏了 Tree（只对照了 Menu/Accordion），**NCE 真实产品截图证实其设备左树就是 EviewUI Tree 搭的**。Tree.d.ts 验证：`data` 任意深度嵌套 children；`expandedKeys`+`onExpand(全量keys,node)`、`selectedKeys`+`onSelect` 受控要素齐全（cWRP 同步，半受控形态但回调给全量 keys 可回写）；**内置搜索** `findLevelNodes(词)` 返回命中+父链并高亮（覆盖我们左树搜索自动展开场景）；`treeNodePrefix/Suffix` 节点级 ReactElement、图标体系、虚拟滚动、`focusNode` 滚动定位。适配动作：items→data 转换（label→text、双语=重建 data 与现状同构）+ expandedKeys/selectedKeys 桥 ~100 行 |
| Tabs | **Tab** | 🔧 JSX 子组件式非 items 配置（循环渲 TabItem）；**按下标寻址非 key**（key↔index 映射表，注意增删漂移）；⚠️ 半受控（点击先内切再回调，不可拦截）；✅ 溢出折叠内建（自动收纳弹层，几十个 Tab 可行）；✅ 有 index signature，data-test 可透传 |
| Breadcrumb | **Crumbs** | 🔧 `items→data`、title 仅 string；分隔符拼写 **`seprator`**（少个 a）；附赠 countLimit 超长折叠 |
| Dropdown | **DropDown**（嵌套子菜单/右键用 PopUpMenu） | 🔧 `{items,onClick}→{data,onItemClick}`、key→value；⚠️ 开合非受控（`onDropDown` 仅通知）；无嵌套子菜单 |
| Badge | **Badge** | 🔧 `count→content`；无 size（badgeStyle 固化）；附赠 max/dot/offset 对齐 antd |
| Tag | **Tag** | ✅ 语义色齐（error→**danger**、processing→**primary** 两个色名转换）；默认圆角，直角需 `round={false}` |

### 反馈弹层

| 我们在用 | EviewUI 对应 | 关键映射与缺口 |
|----------|--------------|----------------|
| Modal | **Dialog** | ✅ `open→isOpen`、`onCancel→onClose`、`destroyOnHidden→destroyOnClose`(默认 true)；🔧 okText/onOk→`buttons:[{text,onClick}]` 数组、`width→size=[w,null]`；⚠️ 无 confirmLoading（buttons 里自管）、无 maskClosable 配置；默认可拖动（movable） |
| toast/confirm | **DivMessage / MessageDialog** 自养挂载点 | 🔴 无命令式 API（见系统性结论 4）；DivMessage type **无 'info'**（映射 'default'）；MessageDialog `type:'confirm'` buttons 仅 ok/cancel |
| Drawer | **Drawer** | ✅ `open→visible`；🔧 width **仅收 number**（'50%' 须换算 px）；`isClickMask`=maskClosable ✔ |
| Popover | **TipBox**（Popup 是数据驱动下拉列表，排除） | ⚠️ 有 children 时 **display 受控失效**（d.ts 明说）、无 onOpenChange 等价物；弹层疑似就地渲染需实测 overflow 裁剪 |
| Tooltip | **TipBox** trigger="hover"（HelpTip 无 children 仅"?图标"，排除） | 🔧 antd `title`→TipBox `content`（TipBox 的 title 是弹层内标题，别映射错） |
| Alert | **DivMessage**（可关）/ PageMessage（不可关但有 info 型） | ⚠️ **默认 10 秒自动消失**——常驻横幅必传 `enableDisposeTimeOut={false}`；type 映射 warning→warn |
| Empty | **Empty** | ✅ description 直接对应 |
| Spin | **Loading** type='local'（Loader=别名、Shade=纯遮罩） | 🔧 不包裹 children（容器自套 relative+条件渲染）；⚠️ iconUrl 注释称必填，需内置默认图标实测 |

### 装配/主题/图标/按钮

| 我们在用 | EviewUI 对应 | 关键映射与缺口 |
|----------|--------------|----------------|
| ConfigProvider(locale+theme) | **ConfigProvider**（locale+messages，内含 I18nProvider 能力） | ✅ locale 切换形态兼容（内置 24 语言包，键为 `ev_*`）；✅ theme 走 design-token CSS 变量覆盖（系统性结论 5，已闭环） |
| App/message/modal 上下文 | 无对应 | 自养挂载点方案覆盖 |
| icons（23 语义名） | **@nce/icon-plus 1.0.87**（2609 个具名组件，`IconPlusIcPublic*` 家族） | ✅ 21/23 直接对应（实心变体走 `type="filled"`、`spin` 可做加载旋转）；🔴 缺纯铃铛、纯显示器（用 IcPublicNotice/IcPublicScreenWifi 近似）；✅ props extends HTMLAttributes 可透传 data-test |
| Button | **Button**（TextButton 极简、IconButton 图标钮） | 🔧 `type→status`（primary/text/risk）；🔴 **无 ghost、无 loading**（loading=塞 spin 图标+手动 disabled）、primary+danger 组合表达不了；onClick 双参签名 |

## 2. 适配层工作量重估

antd 时代适配层=223 行薄转发；切 EviewUI 后为**真适配层，预估 1300~2200 行**（2026-08-17 Tree 翻案后下调）：

| 层级 | 内容 | 预估 |
|------|------|------|
| 重 | Table 受控壳（150~250）、feedback 自养挂载点（~100）、FormItemShell（~80） | ~450 行 |
| 中 | 左树 Tree 适配（data 转换+受控桥，~100，2026-08-17 由"自研包装 200~400"下调——Tree 翻案）、TextField（clear/prefix 自绘）、Spinner、InputSelect、Tab（key↔index）、Button（loading/ghost 补）、Loading、TipBox、Alert 双组件分派 | ~600 行 |
| 轻 | Tag/Badge/Crumbs/Empty/Drawer/Dialog/DropDown/Segmented/Radio/Switch/Checkbox 改名转换 | ~300 行 |
| 系统 | 半受控兜底工具（key 重挂）、data-test wrapper 约定、密度 CSS 覆盖 | ~200 行+样式 |

## 3. 垂直切片验证清单（更新版，按风险排序）

1. **半受控实测**：TextField/InputSelect 在「父级拒绝回写」「程序化清空」场景下的实际表现（决定兜底策略成本）。
2. 测试基建：vitest + @testing-library/react + happy-dom 在 openInula 下跑通。
3. 左树原型：**Tree** 受控展开桥实测（expandedKeys 经 cWRP 同步的回写时序）+ findLevelNodes 搜索接线。
4. Radio onChange 新旧值顺序、Checkbox onPreChange 拦截语义。
5. InputSelect 弹层是否真的不 teleport（影响 F3/E2E 选择器）。
6. Loading 不传 iconUrl 的表现；DivMessage 自动消失关闭开关。
7. ~~alias 方向终判~~ **已闭环**（2026-08-17 离线机实测 Button.js = `require("react")`：构建期 alias react→openinula 即可，路线 A 直跑真 React 的前提同时成立）。

## 4. 向 EviewUI 团队追问清单（更新）

1. ~~`@hui/design-token` 完整包~~ **已闭环**（2026-08-17 离线机整包补采，CSS 变量覆盖方案确定，见系统性结论 5）。
2. ~~任一组件编译产物 JS~~ → 降级为**一行 grep 答案**（源码无需出网）：在离线机跑
   `grep -o "require(['\"]react['\"])\|require(['\"]@cloudsop/horizon['\"])" frontend/node_modules/@nce/eview-react/Button/Button.js | sort -u`
   回传输出的那一行即可判 alias 方向。
3. ~~侧边内联导航~~ **已闭环**——Tree 即对应物（NCE 产品截图实证 + Tree.d.ts 验证）。
4. 半受控控件在新版本里是否有受控化改造计划。
5. 纯铃铛/纯显示器图标的内部补充包。
