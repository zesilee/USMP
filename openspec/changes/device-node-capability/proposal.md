# device-node-capability — 节点级设备能力协商

## Why

真机（华为交换机）软件版本与平台内置 YANG 模型存在**节点级**差异：设备对 `<get-config>`/`<get>` 过滤器中它没有的节点回 rpc-error `unknown-element`（error-info-code 313，实测 devm `cards`/`physical-entitys`/`schedule-reboot`，且用户报告多模块中招）。既有模块级协商（CN-01~CN-03）只能过滤整模块——hello capabilities 不含节点级信息，模块内的容器差异无从预知，导致控制台页签把原始 rpc-error 裸暴露给用户，且每次进入重复打设备、重复报错。错配面随设备型号/软件版本波动、无法预枚举（模拟网元不校验故测试拦不住），必须由平台在运行时按设备学习。

## What Changes

- 后端识别 `unknown-element` 类 rpc-error 并归因到请求路径，按设备维护**节点级不支持集**（内存、连接生命周期、断线重连清空——对齐 CN-01 不持久化口径；学到的事实可随时重学，设备升级后自然刷新）。
- `GET/POST /config` 命中已知不支持路径时**快速失败**（不再打设备），错误响应携带结构化 `reason: "node-unsupported"`；`force_refresh=true` 允许绕过重试，成功即清除该标记（重测逃生门）。
- 新增按设备查询已学习不支持集的北向透出（供前端渲染前标记页签）。
- 前端模块控制台：命中 node-unsupported 的页签/子 Tab 显示「当前设备不支持此功能」占位态（诚实透出、不隐藏，与 CN-03 仅注解不硬裁剪同一口径），无编辑/下发入口，不再弹裸错误；进入控制台时按已学习集预标记，未学习页签照常取数、错误即入集并即时转占位态。
- 写路径同门禁：变更集提交前命中不支持路径的项拒绝入集/下发，给出同款友好错误。

## Capabilities

### New Capabilities

（无——全部落在既有能力的增量上）

### Modified Capabilities

- `device-capability-negotiation`: 新增 CN-04 节点级不支持集学习（unknown-element 归因、内存缓存、重连清空、force 重试清除）、CN-05 按设备透出已学习集、CN-06 hello capabilities 原文透出（诊断 + deviations 二期捷径侦察）。
- `config-api`: 新增 BR-12 节点不支持快速失败与结构化错误（reason=node-unsupported、force_refresh 逃生、写路径同门禁）。
- `frontend`: 新增 FE-24 设备不支持节点占位降级态（页签占位/预标记/即时转态，不裸报错）。

## Impact

- 后端：`pkg/yang-runtime/client/netconfcore`（rpc-error 已结构化，补 tag/element 归因判定）、`pkg/yang-runtime/client`（连接层不支持集，毗邻 hello caps 缓存）、`internal/api/config_handler.go`（快速失败+错误语义）、`internal/api`（透出端点）。
- 前端：`utils/moduleConsole.ts`（页签预标记）、`components/config/ModuleListTab.vue`/`ModuleFormTab.vue`（占位态）、`api/index.ts`（错误 reason 解析）、changeset 提交链路（写门禁提示）。
- 不引入新依赖；不持久化（R03/CN-01 口径）；多实例各自独立学习（学习成本=一次失败请求，无共享需求）。
- 测试：B1（学习集并发/race）、B3（API 错误语义）、B2（模拟网元需可注入 unknown-element 应答）、F1/F2（占位态与预标记）。netconfsim 需新增「按路径拒绝」注入能力——这是让此类真机行为从此可被测试拦住的关键。
