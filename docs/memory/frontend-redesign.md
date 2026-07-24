---
name: frontend-redesign
description: 商用级前端重设计已定稿（高保真原型批准）；落地到 Vue 工程是待办大工程
metadata: 
  node_type: memory
  type: project
  originSessionId: 0ec4b9ae-3cb9-4a36-9d3b-8524c8c49f29
---

2026-07-05：交换机管理前端从"验证性实现"升级为商用级设计，高保真原型经用户审视**已批准无异议**。

**原型位置**：`frontend/design/`（`USMP-prototype.html` + `prototype.css` + `prototype.js`，纯静态、不碰 src），PR **#78**（分支 `design-commercial-frontend-prototype`，未合并）。

**设计主张（落地时必须保留）**：
- 视觉：浅色「华为 iMaster NCE 气质」；深钢蓝主色 `#0C5EA6`（**弃用**旧令牌霓虹蓝 `#165DFF`）；华为红 `#C7000B` 仅品牌+告警；IBM Plex Sans + **Mono 承载所有网络事实**（IP/端口/VLAN ID/YANG 路径）。
- 签名元素①**收敛台账** desired↔actual：配置下发不是假"成功"弹窗，而是可见的对账收敛（校验→edit-config→回读对齐→已收敛四态：已收敛/收敛中/已漂移/离线）。呼应 R01。
- 签名元素②**新鲜度环**：顶栏"缓存 Xs / TTL 30s"递减环。呼应 R03 无数据库。
- **配置下发模型驱动**：YANG 面板呈现「当前所选模型」的架构树（容器/列表/叶子+类型，pyang tree 左流式排布，长类型横向滚动兜底），非全量清单；表格列/表单字段随模型切换；新增/编辑数据驱动+实时差异预览+必填/无改动禁用。呼应 R04/R05。

**待办（下一步大工程）**：把令牌固化进 `frontend/src/styles/variables.scss`，逐页重构 `MainLayout/Sidebar/Dashboard/DeviceConfigPage` 等为该设计语言。必须走 §6 worktree + TDD + PR，单 commit ≤500 行（本地 commit-msg 钩子强制，比 CI 的 800 更严）。关联 [[frontend-contract-gen]] [[vlan-config-stackb]]。
