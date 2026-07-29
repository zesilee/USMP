# rpc-label-localization

## Why

rpc 执行能力（yang-rpc，#228–#232）已交付，但模块控制台里 rpc 的**标签仍是后端原始 YANG 节点名**（如 `reset-if-counters-by-name`、输入叶 `if-name`），而配置字段早已按 snd res 查表中文化（UI-03）。同一个控制台里，配置树是中文、rpc 却是一串英文节点名，割裂且对非英文运维不友好。

关键：翻译数据**早已就位**——`make sync-snd-i18n` 同步进来的 res 副本里已含 rpc 标签与 input 叶的中英文（如 `/huawei-ifm:restart-if` →「重启接口」、`/huawei-ifm:restart-if/input/if-name` →「重启接口名」）。甚至 FE-19 的验收场景本身就以中文名「按接口名清除统计」断言，而当前实现渲染的是原始名，**与既有 spec 场景不符**。缺的只是把既有 `localizeFields` 的 res 查表机制**延伸到 rpc 路径**。

## What Changes

- **UI-03 扩展到 rpc**：字段标签本地化机制（懒加载 res + 按路径查表 + 缺失回退，R08）延伸覆盖 **rpc 标签**与 **rpc input 叶标签**。rpc 路径键规律：rpc 标签 = `/<sourceModule>:<rpcName>`（rpc 是模块顶层语句，键中**无根容器段**，区别于配置叶的 `/<sourceModule>:<root>/...`）；input 叶 = `/<sourceModule>:<rpcName>/input/<leafName>`。
- **纯前端消费既有数据**：新增 `localizeRpcs`（`useFieldLabels`）对 rpc 定义树做同款查表重标；`ModuleConsolePage` 在既有 `relabelFields`（首帧原名→就绪替换、locale 切换即时重查）里一并本地化 rpc，`deriveRpcTabs` 与 `RpcExecuteTab` 零改动即得中文标签。
- **无契约改动**：不动后端、不动 API、不动 res 数据；只在前端把已入库的翻译用起来。

### 明确不在本次范围

- 新增/修改任何 res 翻译条目（缺翻译的 rpc 仍回退原名，R08）——翻译随 snd 包升级，不在本次手工补。
- rpc `output` 结构化字段的本地化（首期 rpc 只回显 reply 原文，无 output 字段树）。

## Capabilities

### Modified Capabilities

- `ui-i18n`: 修改 Requirement UI-03——本地化覆盖面从「配置字段标签」扩展到「配置字段 + rpc 标签 + rpc input 叶标签」，明确 rpc 路径键规律与回退语义。

## Impact

- 前端：`frontend/src/composables/useFieldLabels.ts`（+`localizeRpcs`）、`frontend/src/views/ModuleConsolePage.vue`（`relabelFields` 一并本地化 rpc）。
- 测试：F1 `useFieldLabels.test.ts`（rpc 查表命中/input 叶/缺键回退/双语）、F2 `ModuleConsolePage` 组件测（rpc Tab 标签本地化）。
- 无后端、无 API、无 res 数据变更。
