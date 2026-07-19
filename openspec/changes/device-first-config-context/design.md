# Design: device-first-config-context

## Context

设备作用域配置管理的唯一入口是通用模块控制台 `/module/:module`（FE-10，左树导航驱动）。当前每次进入该页都新建局部 `selectedDevice = ref('')`（仅 `?device=` 深链可预填），模块切换即丢失选中。device store 已有 `selectedDevice: Device | null` + `selectDevice/clearSelection`，但全仓无消费者（死状态）。业务控制台 `/business/:module` 为平台作用域（一个意图实例管 N 台设备），无设备上下文概念。

## Goals / Non-Goals

**Goals:**
- 设备选择一次生效、跨模块保持（先选设备、后配置）
- 深链 `?device=` 与设备管理「查看配置」入口写入同一全局上下文
- 未选设备时引导空态，不再静默渲染空数据

**Non-Goals:**
- 不做刷新持久化（localStorage/sessionStorage）——SPA 会话内保持已满足诉求，刷新后深链或引导空态兜底
- 不改造业务控制台、后端 API、YANG 模型
- 不做多设备并排/对比视图

## Decisions

**D1 上下文承载：device store `selectedDeviceIp: ref<string>`（IP 口径）**
替换无人消费的 `selectedDevice: Device | null`（连同 `selectDevice/clearSelection` 语义收敛为 IP 口径）。IP 是控制台下拉 value、配置 API 与对账链路的统一设备标识（store `id` 即 `ip`）。
备选：provide/inject（作用域不跨路由）、路由 query 全程携带（每次导航都要改写 URL，左树 `:index` 静态路径不可行）——store 最小且已存在。

**D2 入页初始化优先级：query > store**
`?device=` 存在 → 写入 store（深链/「查看配置」语义：显式指定覆盖旧上下文）；不存在 → 沿用 store 现值。「查看配置」跳转同时写 store 并带 query（URL 可分享，双写幂等）。

**D3 未选设备空态：Tab 区域整体替换为引导 `el-empty`**
页头（面包屑 + 设备下拉）保持可用，schema 照常加载（Tab 派生不依赖设备，选中后即时呈现）。仅内容区按「未选设备」分支渲染引导，提示到页头下拉选择。

**D4 控制台下拉双向绑定 store**
`ModuleConsolePage` 删除局部 ref，`el-select` 直接 `v-model` store 的 `selectedDeviceIp`（Pinia setup store ref 可写）。既有 `loadOwnership` watch 源改为 store 值，行为不变。

## Risks / Trade-offs

- [刷新 `/module/vlan` 丢上下文] → 引导空态兜底 + 「查看配置」深链带 query；持久化留待真实诉求
- [选中设备被删除/离线后上下文悬空] → 下拉 options 来自 `store.devices`，悬空值显示原 IP 文本；配置 API 对不可达设备已有明确错误降级（§9），不新增拦截
- [既有 F2 测试以 `vm.selectedDevice` 直改局部 ref] → 同步迁移为 store 写入，测试语义不变
