# 退役旧路由重定向 /config/interface 与 /config/vlan

## Why

FE-13 迁移时为保留书签可达性留下的两条旧路由重定向（`/config/interface` → `/module/ifm`、`/config/vlan` → `/module/vlan`）已完成过渡使命。用户拍板不再保留旧地址兼容；且首页 Dashboard「去配置」按钮至今仍跳旧地址 `/config/vlan`，靠重定向兜底才可用，属于内部代码对已退役入口的隐性依赖，应一并纠正。

## What Changes

- **BREAKING**：删除 `/config/interface`、`/config/vlan` 两条 redirect 路由，旧地址访问将落入无匹配路由（老书签失效，用户已知悉并接受）。
- Dashboard「去配置」按钮跳转目标从 `/config/vlan` 改为 `/module/vlan`（直连现役路由，不再依赖重定向）。
- 同步更新引用旧路由的测试断言（router 单测、Dashboard/Header 单测、schemaTree 契约测试、routerNav 回归测试、staging-smoke E2E）与 App.vue 注释。

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `frontend`：FE-13「模型驱动原生配置导航与路由迁移」中「旧路由 SHALL 重定向到对应 /module/:module」的要求删除，改为旧路由 SHALL 不存在（与 `/native/:module`、`/config/route` 同口径）；站内入口 SHALL 直接使用 `/module/:module`。

## Impact

- `frontend/src/router/index.ts`：删两条 redirect。
- `frontend/src/views/Dashboard.vue`：goConfig 跳转改 `/module/vlan`。
- `frontend/src/App.vue`：过时注释更新。
- 测试：`test/router.test.ts`、`test/views/routerNav.test.ts`、`test/views/schemaTree.contract.test.ts`、`test/views/Dashboard.test.ts`、`test/components/Header.test.ts`、`tests/staging-smoke.spec.ts`。
- 无后端、无 API、无依赖变更。
