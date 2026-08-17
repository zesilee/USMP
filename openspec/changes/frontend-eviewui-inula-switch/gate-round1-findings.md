# 闸门第一轮报告判读（2026-08-17，gate-report.txt 摆渡 R1）

环境：离线机 Node 18.19.1 / eview-react 3.10.28 / react 已钩到工具包 openinula ✓ / locales zh 140 键载入 ✓。

## 已定案（正面，闸门核心）

| 项 | 结论 |
|----|------|
| **EviewUI × openInula 可运行** | V0 PASS——真实 Button 挂载+点击成功（ev_button DOM 输出真实），组合可行性从推断变实测 ✅ |
| **R05 动态列** | V7 PASS——真实 Table 上自定义 render 函数出列成功 ✅ |
| **InputSelect** | V2 全收：onChange 触发、参数序=(新值, 旧值, 'select')；**弹层为 body teleport**（矩阵"疑似就地渲染"推断被推翻，已修正认知——F3/E2E 选择器按 body 层写） |
| **TextField 半受控实锤** | 敲入后父级拒写、DOM 停留输入值（内部自改）；**cWRP 受控回写通道有效**（父级改值/程序化清空均压回 DOM）→ 三板斧第②档（受控回写）可用 ✅ |
| **Loading 无 iconUrl** | V5 正常渲染（图标是 CSS 类非 img），矩阵"iconUrl 必填"担忧解除 ✅ |
| **裸 input 受控语义** | 在线补测：openinula 对原生 input 的受控压制完整（设值后无 setState 即压回）——运行时受控地基没问题，半受控是 EviewUI 组件层的实现选择 |

## 第一轮假阴性（判定方式问题，非组件能力问题，v2 已修）

| 项 | 根因 | v2 修法 |
|----|------|---------|
| V1 onChange"未触发" | ① inula 给受控 input 的 value 装 tracker，直接赋值骗过变更检测→onChange 不合成（须原型链原生 setter，React 同款姿势）；② inula 合成事件在 happy-dom 下 `e.target` 为 null，回调取 target.value 抛异常被 happy-dom 吞掉 | typeInto 换原生 setter；回调统一 safeArgs 防御序列化（在线 mock 复验：args=["AB","A","<event>"] 完整捕获） |
| V3 展开判定失真 | Tree 收起是 CSS 类（li.ev_tree_collapsed），textContent 恒在——文本口径恒 true 无信息 | 判定改 li class 序列；点击目标锁定 `.ev_tree_hit`（R1 DOM 快照确认） |
| V4 radio 找不到 | EviewUI Radio 是 `div[role=radio]`+aria-checked，无原生 input | 按 role 点击 + aria 变化断言 |
| V7 checkbox 找不到 | 同上，`div[role=checkbox]` | 同上 |
| V6 未观测到消失 | 可能 CSS 隐藏而非卸载 | 补 class/style 观测；**非阻塞**（适配层 toast 自管卸载、常驻横幅显式关自动消失） |

## 状态

- 闸门六项：2 项定案通过（测试基建、半受控+回写通道）+ 2 项收获定案数据（InputSelect 弹层/参数序、Loading）；Tree 受控/Radio 参数序/DivMessage 待 R2 报告。
- 工具包 v2（eview-gate-kit-v2.tgz）已交付摆渡。
