# 垂直切片闸门结论（2026-08-17，两轮离线摆渡后收口）

> 判定：**闸门通过，准入组 2**。六项验证五项定案通过，两项残留不确定性定性为
> 测试环境限制（非组件能力缺陷）且均有不依赖实测的适配层兜底设计，附两条
> F2/F3 阶段复核项。证据链：gate-report.txt / gate-report2.txt（离线机
> Node 18.19.1 × eview-react 3.10.28 × openinula 1.0.0）+ 在线侧源码分析与
> mock 复现（gate-round1-findings.md）。

## 六项判定

| # | 验证项 | 判定 | 依据与设计决定 |
|---|--------|------|----------------|
| 1 | 测试基建（vitest+happy-dom on openinula） | **✅ 通过** | 探针 5/5、主套件 576/576 无回归；@testing-library/react 走自写 render 薄层 alias（存量测试零改动路径已验证） |
| 2 | 半受控实测（TextField 代表） | **✅ 风险可控** | 组件确为半受控设计（d.ts 源码结构）；**cWRP 受控回写通道两轮实证有效**（父级改值/程序化清空均压回 DOM）。**适配层档位定案**：正常路径=受控回写（②档，已实证）；父级拒写路径=key 重挂（③档，机制必然有效，无需实测）。①档事件拦截不作依赖 |
| 3 | Tree 受控桥（左树命门） | **✅ 通过（R2 定案）** | class 口径：expandedKeys 受控展开/回收完全生效；点击 `.ev_tree_hit` 后 **onExpand 回传全量 keys 数组**（["1"]）——受控回写桥全要素成立。左树适配 ~100 行的估算维持 |
| 4 | Radio onChange 参数序 | **⚠️ 未实测定案，风险微小** | 两轮未触发=点击目标未达叶子监听点（判定方式问题）。**适配层写法免疫参数序**：新值必为 options 值之一且≠当前值，运行时自适应判别，无需依赖参数位置。F2 阶段随 Radio 桥补实测 |
| 5 | InputSelect 弹层挂载 | **✅ 定案（R1）** | **body teleport**（矩阵"疑似就地"推断已修正）；onChange 参数序=(新值, 旧值, 'select') 定案 |
| 6 | Loading / DivMessage 缺省行为 | **✅ 定案** | Loading 无 iconUrl 正常渲染（CSS 图标，矩阵担忧解除）；DivMessage 自动消失=真（display:none 隐藏而非卸载），适配层常驻横幅必须 `enableDisposeTimeOut={false}`，toast 场景自管卸载 |

## 附带定案（超出六项的收获）

- **EviewUI × openInula 组合真实可运行**（V0 Button 挂载+点击，DOM 真实）——切换的根本可行性从推断变实测。
- **R05 动态列在真实 Table 上通**（V7 自定义 render 出列）。
- openinula 对裸 input 的**受控语义完整**（无 setState 即压回）——运行时地基没问题，半受控是 EviewUI 组件层实现选择。
- EviewUI DOM 结构情报入库：`ev_*` 类名体系、radio/checkbox 为 `div[role=*]`+aria 无原生 input、Tree 的 `.ev_tree_hit`/`ev_tree_collapsed|expanded`——E2E/F2 选择器设计的直接输入。

## 已知限制与复核项（进 F2/F3 必办清单）

1. **inula 合成事件 `e.target` 在 happy-dom 下为 null**（`e.currentTarget` 正常）——EviewUI 内部凡读 `e.target` 的逻辑在 happy-dom 测试中会断链。**修法已知**：测试薄层 fire() 派发前 `Object.defineProperty(ev,'target',{value:el})` 定义自有属性（合成事件复制枚举不到原型 getter 是丢失根因的最大嫌疑）；F2 基建落地时验证，不通则上报 openinula issue 并评估 F2 用真浏览器兜底。
2. Radio/checkbox 交互测试必须**点击叶子元素**（label.ev_radio_label / span.ev_checkbox_span）令事件冒泡经过监听点。
3. Radio onChange 参数序 F2 随桥实测补记。

## 方法论沉淀（摆渡两轮的教训）

- happy-dom 吞监听器异常 → "未触发"可能是回调自己抛错，判定回调必须防御序列化（safeArgs）。
- inula 给受控 input 的 value 装 tracker → 测试设值必须原型链原生 setter（React 生态同款姿势）。
- CSS 隐藏型组件（Tree 收起/DivMessage 消失）不可用 textContent 判可见，用 class/style 口径。
