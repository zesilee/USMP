# 设计：退役旧路由重定向

## Context

FE-13 把旧配置页 `/config/interface`、`/config/vlan` 迁移到通用模块控制台 `/module/:module` 时，保留了两条 redirect 兼容老书签，并写入主 spec FE-13。现用户拍板放弃旧地址兼容。站内唯一真实依赖是 Dashboard `goConfig()` 仍 push `/config/vlan`。

## Goals / Non-Goals

**Goals:**

- 路由表只保留现役路由，删除两条 legacy redirect。
- 站内跳转全部直连 `/module/:module`，不再有任何代码路径依赖旧地址。
- 测试与 spec 同步，不留断言残留。

**Non-Goals:**

- 不新增 404/catch-all 路由（现状旧地址落空的行为与 `/config/route` 退役时一致，无匹配即空白容器，另行需求再做）。
- 不动 `/business/:module`、rpc 直达等其它路由。

## Decisions

- **旧地址直接落空而非留 404 页**：与 FE-13 已有先例（`/config/route`、`/native/:module` 退役无重定向义务）同口径，避免为两条已放弃的地址新造页面。
- **Dashboard 跳 `/module/vlan` 而非 `/module/ifm`**：保持行为不变（原重定向终点就是 `/module/vlan`），本次只消除对旧地址的依赖，不改产品行为。
- **测试策略反转**：`router.test.ts` 原断言「redirect 存在且指向正确」，改为断言「`/config/interface`、`/config/vlan` 路径不存在」（防回流守护，与 spec 的 SHALL 不存在对齐）；routerNav/schemaTree/smoke 中依赖旧地址进入页面的用例改为直接访问 `/module/:module`，原验证意图（组件重挂载、schema 派生 Tab、E2E 冒烟）全部保留。

## Risks / Trade-offs

- [老书签 404] → 用户已知悉并接受（BREAKING 已在 proposal 标注）；个人项目、无外部用户。
- [E2E 冒烟改动需真实环境验证] → 按 §6.2 门禁本地跑 `make e2e-local` 全绿后才提交。
