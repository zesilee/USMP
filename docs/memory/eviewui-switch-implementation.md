---
name: eviewui-switch-implementation
description: EviewUI/openInula 切换实施台账(2026-08-17 进行中)：闸门通过、路线乙、组2先行波+组3+组4全24桥合入、混合模式工程机制、实录坑集、当前卡点=内网集成点报告；续作前必读
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

**当前卡点与下一步**：等用户回传 bridge-report.txt（内网四套件真实校准）→判读修桥→**组 5 接线**（index.ts 切 eview 后端+调用点微调+antd 退场+覆盖率分母回收）→组 6 F3/黄金→组 7 E2E→组 8 收尾→波 C 运行时翻转（openinula+intl 内核+router v5 化+inula-X，全部制品在 tasks.md 组 2 挂起项）。

相关：[[eviewui-inula-migration]]（调研与矩阵）、[[react-antd-rebuild]]（适配层军规与上次迁移方法论）。
