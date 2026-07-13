# design — native-config-reposition

## Context

前端现有两组侧边菜单（explore 代码级核对，2026-07-13）：

- 「业务网络配置」菜单（`Sidebar.vue:32`）：`stores/menu.ts` 的 `businessModules`/`loadBusinessModules` 拉 `/yang/modules`，按 category 分组（FE-13），指向 `/module/:name` 通用模块控制台 = **Stack B 生产链路**。
- 「原生配置」菜单（`Sidebar.vue:68`）：`nativeModels`/`loadNativeModels` 同样拉 `/yang/modules` 但指向 `/native/:module` → `ConfigPage.vue` → `useConfigPage` → `useK8sCRD('core.usmp.io','v1','nativedeviceconfigs')` 调 K8s API = **Stack A 死路**（Stack A 已退产、无 K8s proxy、NativeDeviceConfig 下发为 TODO stub，见 business-crd spec 头注与 [[dual-stack-migration]]）。
- `/config/route` → `ConfigPage.vue`（openconfig-route）同为死路（route 无华为模型，[[vlan-config-stackb]]）。

退役级联封闭性（已核对消费关系）：`DynamicForm.vue` 唯一消费者是 `ConfigPage.vue`；`StatusBadge.vue` 唯一消费者是 `ConfigPage.vue`，其 `ConfigPhase` 类型定义在 `useK8sCRD.ts`、除此链外无人引用；`useConfigPage` 唯一消费者是 `ConfigPage.vue`。**保留**：`crdSchemaParser`（模块控制台/DeviceConfigPage 的 YANG schema→Field 映射器）、`useConfigForm`、`configDiff`（仅注释提及 legacy，无代码依赖）。

## Goals / Non-Goals

**Goals:**
- 命名与概念分层对齐：模块控制台菜单 = 原生配置；代码标识符同步（business* → native*）。
- Stack A 前端 CRD 债清零（arch-optimization-roadmap 前端部分）：删链路、删路由、删测试。
- 「业务网络配置」未来层方向落盘（持久任务文件），不实现。

**Non-Goals:**
- 不实现业务网络配置编排层（意图模型定义、编排语义、多设备展开——未来独立 change）。
- 不动后端（`business-crd` 后端 CRD 注册等 Stack A 遗留由 arch-optimization-roadmap 管辖）。
- 不动 `crdSchemaParser` 命名（虽含 CRD 字样但已是 YANG schema 通用映射器，改名是无谓 churn）。

## Decisions

### D1 标识符对齐：menu store `business*` → `native*`，旧 `native*`（CRD 版）直接删除

`loadBusinessModules`/`businessModules`/`businessGroups`/`businessLoaded` → `loadNativeModules`/`nativeModules`/`nativeGroups`/`nativeLoaded`；CRD 时代的 `nativeModels`/`loadNativeModels`/`nativeMenuLoaded`/`nativeMenuLoading` 删除。先删后改名（同名冲突），一个 commit 内完成保原子。测试 `menu.business.test.ts` → `menu.native.test.ts` 同步改名重写断言。

- 为什么不留 business* 名：未来业务网络配置层落地时，`businessModules` 将真正表示业务模型列表——旧名不清场，届时两义冲突。

### D2 路由处理：`/native/:module` 与 `/config/route` 直接移除（无 redirect）

`/config/interface`、`/config/vlan` 有 redirect 是因为曾是**可用**入口（书签/习惯延续）；`/native/*` 与 `/config/route` 在生产中从未可用（K8s API 不可达即错误页），无兼容义务。命中即落 404/首页由 router 默认行为处理。

### D3 「业务网络配置」方向留痕形态：持久任务文件（pending），非 OpenSpec change

未来层尚无可执行的需求粒度（意图模型怎么定义、编排语义、收敛策略全部待 explore），建 change 会立即腐烂；`openspec/tasks/business-network-config.md`（status: pending）记录概念定义、架构落位思路（意图模型也走 yang-controller-runtime Reconciler、R05 自动渲染同样适用于业务模型、编排=意图模型→原生模型展开）与启动指令（届时 `/opsx:explore`）。这与 optimize-frontend-nce-insights 的跨会话跟踪器模式一致。

### D4 spec 措辞策略：FE-13 标题去「业务」、Purpose 重写为单链路

FE-13「模型驱动业务导航与路由迁移」→「模型驱动原生配置导航与路由迁移」；Purpose 删除「两代下发链路并存/legacy 链路应逐步收敛」段（收敛已完成），改述：前端 = 原生配置（YANG 模型驱动直连）单链路 + 业务网络配置为未来扩展层（一句话前瞻，指向任务文件）。FE-01/FE-04 仅剔除 CRD/`NativeDeviceConfig` 字样，行为语义不变。

## Risks / Trade-offs

- **[R1] 前端覆盖率棘轮（T08）**：删除的 legacy 代码带测试（useConfigPage.test 等），删除后全局覆盖率比率可能上下波动；若下降属「分母重算」而非测试缺失 → 随 PR 重算基线并在提交说明中给出前后数字。若上升则上调棘轮。
- **[R2] 隐藏消费者**：级联封闭性靠 grep 核对，可能有动态引用遗漏 → `vue-tsc` typecheck 门禁 + 全量 F1/F2 测试 + Storybook 构建（DynamicForm.stories 删除后）兜底。
- **[R3] E2E 冒烟引用死路**：`staging-smoke.spec.ts` 若断言旧菜单文案/路由会红 → apply 先 grep e2e 规格，含 `frontend/` 改动按 §6.2 跑 `make e2e-local`。
- **权衡：不做 /native redirect** —— 死路无流量，redirect 是维护假兼容；D2 已述。

## Migration Plan

单 PR：spec delta → 前端改动（更名 commit + 退役 commit 分开）→ 任务文件。合入即完成；回退 = revert。

## Open Questions

（无——概念定义已由用户拍板，退役面已代码级核对封闭。）
