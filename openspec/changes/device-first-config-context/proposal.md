# Proposal: device-first-config-context

## Why

设备作用域的配置管理页（`/module/:module` 通用模块控制台）每页各自维护局部 `selectedDevice`，模块切换即丢失——用户每进一个配置页都被迫重新选择设备，与「先选设备、后做配置管理」的自然操作模型相悖。device store 中已有的全局 `selectedDevice` 状态是无人消费的死状态。

## What Changes

- 设备选择提升为**全局设备上下文**：Pinia device store 作为唯一事实源，所有设备作用域配置页共享，跨模块切换保持选中不丢
- 模块控制台（`/module/:module`）改为消费全局上下文：页内设备下拉双向绑定 store（选一次全局生效），不再各自维护局部状态
- 设备管理「查看配置」与 `?device=<ip>` 深链写入同一全局上下文（深链仍可用且与全局态一致）
- 未选设备时配置页展示**引导空态**（提示先选择设备），替代当前静默空列表/空表单
- 平台作用域业务控制台（`/business/:module`）不涉及设备上下文，不受影响

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `frontend`: FE-10 通用模块控制台新增设备上下文要求——设备选择为全局共享状态、跨模块保持、深链写入同一上下文、未选设备时引导空态

## Impact

- `frontend/src/stores/device.ts`：`selectedDevice` 语义收敛为「全局设备上下文（IP 口径）」，激活既有死状态
- `frontend/src/views/ModuleConsolePage.vue`：局部 `selectedDevice` ref → store 绑定；`?device=` 初始化改写 store；新增未选设备引导空态
- `frontend/src/views/Devices.vue`：「查看配置」跳转同时写入全局上下文（行为不变，多一处同步）
- 测试：F1（store）、F2（ModuleConsolePage/Devices 组件）、F4（staging smoke 跨模块保持场景）
- 不涉及后端 API、YANG 模型、业务控制台
