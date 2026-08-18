---
name: eviewui-switch-implementation
description: EviewUI/openInula 切换实施台账(2026-08-18 校准收敛)：闸门通过、路线乙、组2先行波+组3+组4全24桥、14轮内网校准收敛(CAL-R14全绿)、Table三定案、Tabs/Tree移交F3、实录坑集；进组5接线前必读
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

**原「下一步」记录（组 5 前）**：**组 5 接线**（index.ts 切 eview 后端+调用点微调+antd 退场+覆盖率分母回收重钉+findDOMNode polyfill 生产安装）→组 6 F3（含 Tabs/Tree 校准移交项）/派生黄金→组 7 E2E→组 8 收尾→波 C 运行时翻转（openinula+intl 内核+router v5 化+inula-X，制品在 tasks.md 组 2 挂起项）。

相关：[[eviewui-inula-migration]]（调研与矩阵）、[[react-antd-rebuild]]（适配层军规与上次迁移方法论）。
