## Context

- left-tree.json：14 顶层组/最大 3 层 group/65 叶（`xpath:"/huawei-xxx"` = 源模块名），双语 zh-cn/en-us，65 个 xpath 全部有对应 .yang（①期探索实测）。
- 运行镜像只含二进制（Dockerfile runtime 层 `COPY --from=builder /manager`），snd/ 不进镜像 → 运行期读 left-tree.json 不可行，必须构建期 codegen（与 tasknamegen/blacklistgen 同构，R04 门禁守护）。
- 模块键歧义：left-tree 叶用**源模块名**（huawei-vlan），运行期 schema/路由用**根容器名**（vlan），Module.Namespace() 恒空（②期实测）→ 映射必须构建期由 goyang 解析产出。
- 现左树：menu store category 分组（taskname）+ Sidebar 分组/平铺双态；staging smoke 依赖侧栏可点导航（vlan/ifm）。

## Goals / Non-Goals

**Goals:**
- lefttreegen 生成物：树 + 叶级 `sourceModule`/`rootContainers` 映射
- `GET /api/v1/yang/left-tree`（含 `?device=` 能力叠加）
- 前端 14 组/3 层树渲染、未接入禁用占位、接口失败回退 category 分组
- 双语字段随载荷透出（④期消费）

**Non-Goals:**
- 不为 65 模块生成 structs（渐进批次，按需另开 change）
- 不做语言切换（④期）；本期界面文案取 zh-cn
- 不动 blacklist 语义（注解已在②期）
- 不删 category 分组代码（作为降级路径保留）

## Decisions

**D1：构建期 codegen 而非运行期读文件/go:embed。** 运行镜像无 snd；go:embed 无法跨模块根引用 ../snd。生成物入库由 regen-and-diff 守护，升级包=替换目录+重跑生成（SP-01 语义）。

**D2：叶子可用性运行期计算，不进生成物。** `available` = 叶 rootContainers ∩ 已加载模块 ≠ ∅，随渐进生成自动变真；生成物只含静态映射。`?device=` 时对 available 叶叠加 `supported`（CN-02 协商子集含其根容器）；协商不可得时省略 supported 字段而非置 false（诚实：unknown ≠ 不支持）。

**D3：前端一次拉取整树（≈65 叶 JSON 数量级 KB），store 缓存；失败回退现 category 分组。** 树是静态结构无需分页/懒加载；回退保证 left-tree 端点异常时导航不消失（R08）。Sidebar 渲染：el-sub-menu 递归组件（3 层内），叶 available→el-menu-item（路由 `/module/<首个已加载 rootContainer>`），不可用→disabled + tooltip「未接入」。

**D4：路由目标取叶子 rootContainers 中**已加载**的第一个。** 少数模块有多顶层容器（如 huawei-dsa 可能多容器）；已加载优先保证点击必有表单；全部未加载则叶子本就禁用。

**D5：smoke 断言改 data-test 锚点。** 现 smoke 按文案/结构断言侧栏，重构后统一挂 `data-test="lefttree-leaf-<module>"`，E2E 与 F2 共用选择器，后续树调整不再脆断。

## Risks / Trade-offs

- [65 叶 goyang 解析拉长 go:generate 时间] → 一次性构建期成本（blacklistgen 已解析 23 模块无感）；lefttreegen 复用同一 yang.Modules 实例单次 Process
- [部分 .yang 解析告警（如 qos-bd unknown type）致根容器缺失] → 与 blacklistgen 同策略：告警不阻断、缺失容器的叶子恒不可用（禁用态），不崩（R08）；生成日志留痕
- [staging smoke 断言漂移] → D5 data-test 锚点 + 本地 make e2e-local 全绿后才推
- [前端树渲染性能] → 65 叶静态树一次渲染，量级可忽略
- [category 分组与 left-tree 并存的维护双轨] → 明确降级定位：left-tree 为主路径，category 仅在端点失败时兜底；④期后评估退役

## Migration Plan

单 PR 目标（手写面预估 <1000 行；超限拆 后端/前端 两 PR）。顺序：spec 已先行 → lefttreegen + B1（testdata）→ B3 red→green（API）→ 前端 store/Sidebar F1/F2 red→green → smoke 适配 + make e2e-local → contract regen → 棘轮校验 → 收官。回滚 = revert（前端回退路径本就保留）。

## Open Questions

（无。）
