# 任务：退役旧路由重定向

> TDD（T05/T07 口径）：先反转测试断言（红），再改路由/跳转（绿）。改动类型=前端组件/页面逻辑+路由 → F2 单测 + F4 staging-smoke（§5.6）。

## 1. 测试先行（红）

- [ ] 1.1 `test/router.test.ts`：断言反转——`/config/interface`、`/config/vlan` 路径 SHALL 不存在于路由表（防回流守护）
- [ ] 1.2 `test/views/Dashboard.test.ts`：「去配置」断言跳转 `/module/vlan`；桩路由表移除旧路径
- [ ] 1.3 `test/views/routerNav.test.ts` 与 `test/views/schemaTree.contract.test.ts`：入口改为直接访问 `/module/vlan`、`/module/ifm`，保留原验证意图（重挂载/schema 派生 Tab）
- [ ] 1.4 `test/components/Header.test.ts`：桩路由表移除 `/config/vlan`，改用现役路径
- [ ] 1.5 `tests/staging-smoke.spec.ts`：E2E 直接访问 `/module/vlan`、`/module/ifm`，删除旧地址重定向断言

## 2. 实现（绿）

- [ ] 2.1 `src/router/index.ts`：删除两条 redirect 及过时注释
- [ ] 2.2 `src/views/Dashboard.vue`：`goConfig()` 改跳 `/module/vlan`
- [ ] 2.3 `src/App.vue`：更新提及旧路由的注释

## 3. 门禁与收尾

- [ ] 3.1 前端单测全绿（happy-dom），覆盖率不低于棘轮阈值（T08）
- [ ] 3.2 `make e2e-local` staging smoke 全绿（§6.2 前端改动门禁）
- [ ] 3.3 code review + What/Why/How 提交，推 PR
