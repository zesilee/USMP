# 翻转路线判定实验（2026-08-17，组 2 先行波后）

## 背景：发现计划级硬约束

原计划"翻转波一次切运行时、组件库后换"存在死锁：**antd 6 依赖 React 18 API**
（useId/useSyncExternalStore/useTransition 遍布 rc-*），openinula 只有 17 级——
antd on openinula 会炸；反之 EviewUI 生态只认 openinula。运行时与组件库互相锁。

## 三个在线实验（全部落档 test/{rev,flip}/ 与 src/runtime/react-shim.ts）

| 实验 | 结论 |
|------|------|
| **React 19 是否仍调用无前缀 componentWillReceiveProps** | ✅ **仍调用**（断言实证：cWRP 触发且 state 同步生效）——"EviewUI on React 19"的头号风险解除 |
| **inula-intl/router 反向 alias（openinula→react）on React 19** | ❌ 测试链路不可行：两件套 ESM 构建**内嵌 openinula hook 运行时**（alias 改不了内嵌代码）；CJS 构建虽 external require("openinula")，但 vitest 对 inline CJS 的 require 解析绕过 alias（与 gate 阶段 testing-library 同源机制）。VueI18n 纯类（无 hook）可用 ✓。生产构建（Vite）下 alias 理论可行但无法在测试层验证=不可依赖 |
| **antd on openinula+18 级垫片（react-shim）** | ⚠️ vitest 探针失真（CJS 逃逸致 React/inula 元素标记混装，"Component type is invalid"）——**此路线的真判定载体只能是 Vite 真构建+浏览器**，验证成本≈做完整个翻转波，赌注过大 |

## 路线定案：乙——「组件库先换（on React 19），运行时最后切」

```
现在（React 19 + antd）
  └─ 组 3~5：适配层 EviewUI 化 + antd 逐步退场   ← on React 19（cWRP ✓ 已验，EviewUI 编译产物 require('react') 天然兼容）
  └─ 波 C（终局）：运行时切 openinula + i18n 内核 inula-intl + 路由 inula-router + zustand→inula-X
                                                  ← 终态组合 = 闸门离线实测过的组合 ✓
```

**理由**：路线乙每一步的前提都已实证（cWRP/EviewUI×openinula/测试基建），无赌注步骤；
路线甲（运行时先切）唯一未验前提（antd on openinula+垫片）的验证成本等于全量翻转。
组 2 剩余（2.1b i18n 内核 / 2.3 store / 2.4 路由）**全部改挂波 C**；组 3~5 提前为当前主线。
react-shim 垫片与 flip/rev 探针留档备用（波 C 若需垫片支持第三方库时复用）。

## 连带现实问题（需用户拍板）

切换后 USMP 前端**只能在内网构建**（@nce 系包仅内网 registry 有）——外网开发机
无法构建最终产物。适配层往后的开发主战场与协作模式需要用户决策（见对话）。
