# tasks — native-config-reposition

> TDD（T01/T05/T06）：改动类型=前端组件/页面逻辑+store → F1/F2 必补；测试先行。含 `frontend/` 改动 → 提交前 `make e2e-local`（§6.2）。commit 拆分：更名与退役分开。

## 1. 前置侦察与测试设计（T05）

- [x] 1.1 grep `frontend/tests/staging-smoke.spec.ts` 等 e2e/Storybook 是否引用「业务网络配置」文案、`/native/*`、`/config/route`、ConfigPage/DynamicForm/StatusBadge——列出需同步改动的断言清单
- [x] 1.2 【测试先行】改写 `menu.business.test.ts` → `menu.native.test.ts`（红）：`nativeModules`/`nativeGroups`/`loadNativeModules` 新契约（拉 `/yang/modules`、category 分组、失败回退）；Sidebar F2 断言「原生配置」标题 + 无 `/native/*` 菜单项（红）

## 2. 更名（D1/D4，FE-13）

- [x] 2.1 `stores/menu.ts`：删除 CRD 时代 `nativeModels`/`loadNativeModels`/`nativeMenuLoaded`/`nativeMenuLoading`；`business*` → `native*` 改名（同 commit 保原子）
- [x] 2.2 `Sidebar.vue`：菜单标题「业务网络配置」→「原生配置」、删除旧 native-config 子菜单块、标识符随 store 对齐；测试转绿
- [x] 2.3 全仓 grep「业务网络配置/业务配置」残留（含注释/文档/Storybook 文案）清理为「原生配置」措辞

## 3. 退役 Stack A 前端 CRD 死路（D2，FE-05/06 REMOVED）

- [x] 3.1 `router/index.ts` 移除 `/native/:module`、`/config/route`
- [x] 3.2 删除 `views/ConfigPage.vue`、`composables/useConfigPage.ts`、`composables/useK8sCRD.ts`、`components/config/DynamicForm.vue`（+`.stories.ts`）、`components/common/StatusBadge.vue` 及其测试（`useConfigPage.test.ts`、`views/ConfigPage.test.ts` 等）
- [x] 3.3 `vue-tsc` typecheck + 全量 F1/F2 + Storybook 构建全绿（R2 隐藏消费者兜底）；覆盖率对比棘轮（T08/R1），波动随 PR 说明处理

## 4. 业务网络配置方向留痕（D3）

- [x] 4.1 新建 `openspec/tasks/business-network-config.md`（status: pending）：概念定义（用户拍板原文）、架构落位思路（意图模型走 yang-controller-runtime / R05 渲染复用 / 编排=意图→原生展开）、启动指令（届时 /opsx:explore）

## 5. 收尾

- [ ] 5.1 `make e2e-local` 全绿（frontend 改动强制，§6.2）；`go-code-review-check`/前端自审通过；commit What/Why/How
- [ ] 5.2 推送 + PR，CI 全绿自助 merge
- [ ] 5.3 合入后：`/opsx:sync`（frontend/business-crd delta + Purpose/头注重写）+ `/opsx:archive` + `/task sync`
