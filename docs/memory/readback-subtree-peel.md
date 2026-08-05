---
name: readback-subtree-peel
description: "回读解码契约=以请求路径为根的子树（PR#281 peelToPath）：批量接入模块解码根是模块根、曾致 devm ports「一行且位置=port」；改回读形状/加消费方前必读"
metadata:
  type: project
---

配置回读（`GET /config/<ip>/<path>`）的响应契约是**「以请求路径为根的 RFC7951 子树」**——前端全部消费方（列表 Tab `normalizeRows`、表单 Tab 回填、include_state 单行状态读 `ItemDetailPane`、leafref 下拉 `extractRows`）都按此消费。2026-08-05 真机验证 devm ports 暴露：批量接入模块（[[snd-driver-registry]] 的 `registerPlain`，49 个）解码根恒为模块根容器、读子路径不剥层，返回 `{"ports":{"port":[...]}}` 多包一层 → 前端把容器键当唯一数据行（「一行且位置=port」）；vlan/ifm 手写块因锚定列表容器无此症状，所以 sim/E2E 一直绿。已修：`decodeRunningConfig` 按描述符 `EncodeAnchor` 与请求 path 段差剥层（`peelToPath`，PR#281），顺带修好表单 Tab 回填恒空、单行状态合并静默失效、`postKey` 误取容器名三个同根暗伤。

**Why:** 回读形状是前后端隐式契约，形状错了前端不报错、只默默渲染垃圾行——真机才暴露。

**How to apply:**
- 改 `decodeRunningConfig`/新增回读消费方时，锚定契约：响应恒以请求路径为根；谓词段（`[k='v']`）停剥，返回谓词段**父容器**子树（前端 `sub[listKey]` 取行）。
- `pathLocals` 按 '/' 切分不感知引号，谓词值含 '/'（`MEth0/0/0`）会错切——无害性依赖「遇 `[` 即停剥」不变量，改停剥逻辑前先改成引号感知切分（代码注释已固化）。
- 段对齐按局部名（去模块前缀）：ni 有 `/ni:` 与 `/network-instance:` 双前缀口径、前端谓词段不带前缀、RFC7951 跨模块 augment 键带 `module:` 前缀（`huawei-pic:ethernet`）。
- 症状速查：列表页「一行且主键列=list 名」= 回读多包一层；先看 decode 剥层再怀疑设备。
