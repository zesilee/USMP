# native-config-reposition — 「原生配置」概念重定位 + Stack A 前端 CRD 死路退役

## Why

产品概念分层拍板（用户，2026-07-13）：**「原生配置」= 直接基于 YANG 模型对交换机进行配置管理**（即现有模块控制台/config-api 生产链路）；**「业务网络配置」= 未来扩展层**——业务侧基于 YANG 模型定义网络自动化能力，USMP 将其编排为原生配置下发。当前前端命名恰好**倒挂**：生产链路的菜单叫「业务网络配置」，而挂名「原生配置」的菜单（`/native/:module`）实际是 Stack A 前端 CRD 死路（`useK8sCRD` 调 K8s API，Stack A 已退出生产、`NativeDeviceConfig` 下发是 TODO stub、依赖不存在的外部 proxy）。不纠正命名，未来业务编排层落地时概念将彻底混乱。frontend spec FE-06 本就要求 legacy 链路「SHOULD 逐步收敛到主链路」——本 change 即执行这次收敛。

## What Changes

- **更名**：侧边栏「业务网络配置」菜单 → 「**原生配置**」（仍由 `/yang/modules` 驱动、指向 `/module/:name` 模块控制台，行为不变）；代码标识符对齐（menu store `business*` → `native*`），杜绝「代码里 business 实指原生」的长期混乱。
- **退役 Stack A 前端 CRD 死路整链**（互为唯一消费者，干净级联）：旧「原生配置」菜单与 `/native/:module`、`/config/route` 路由；`ConfigPage.vue`、`useConfigPage.ts`、`useK8sCRD.ts`（含 `K8sClient`）、`DynamicForm.vue`（+stories）、`StatusBadge.vue`（其 `ConfigPhase` 类型仅此链使用）；对应测试同步删除。**保留** `crdSchemaParser`/`useConfigForm`/`configDiff`（模块控制台/DeviceConfigPage 在用）。
- **「业务网络配置」方向立项留痕**：新建持久任务文件 `openspec/tasks/business-network-config.md`（pending）记录未来扩展层定义与架构落位思路，**本 change 不实现**（届时独立 explore/propose）。
- **BREAKING**（仅对死路而言）：`/native/:module` 与 `/config/route` 路由移除——二者在生产中本不可用（K8s API 不可达即报错）。

## Capabilities

### New Capabilities

（无——不新增运行时能力；业务网络配置层留待未来独立 change 立项。）

### Modified Capabilities

- `frontend`: Purpose 重写（删除「两代下发链路并存」叙述 → 单链路 + 原生/业务概念分层）；FE-13 MODIFIED（「业务配置菜单」→「原生配置菜单」措辞与概念对齐）；FE-05、FE-06 REMOVED（legacy CRD watch / CRUD 链路收敛完成）；FE-01 MODIFIED（schema 来源去 CRD 提法，仅后端 YANG schema）；FE-04 MODIFIED（去 `NativeDeviceConfig` 提法）。
- `business-crd`: BC-05（CRD 作为前端表单 schema 来源）REMOVED——spec 头注明确说「唯一可能仍被前端使用的是 BC-05，据实保留」，本 change 后前端零 CRD 消费，据实移除并更新头注。

## Impact

- **前端**：`Sidebar.vue`、`stores/menu.ts`、`router/index.ts` 修改；`views/ConfigPage.vue`、`composables/useConfigPage.ts`、`composables/useK8sCRD.ts`、`components/config/DynamicForm.vue`（+stories）、`components/common/StatusBadge.vue` 删除；测试 `menu.business.test.ts`（改名重写）、`useConfigPage.test.ts`、`views/ConfigPage.test.ts` 等同步处理。纯删除为主，手写净增量小。
- **后端**：零改动。
- **spec**：frontend 6 处 delta、business-crd 1 处 REMOVED。
- **风险**：删除带测试的 legacy 代码可能波动前端覆盖率棘轮（vitest thresholds 74/71/67/74，T08）——apply 阶段实测，若因删除高覆盖代码导致比率下降属合理重算，随 PR 说明调整基线。
