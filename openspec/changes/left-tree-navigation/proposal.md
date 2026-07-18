## Why

snd 融合四期计划③（用户 2026-07-18 拍板：全树展示 65 模块 + 未接入占位 + 渐进生成）。当前原生配置左树按 task-name category 单级分组、仅覆盖已生成的少数模块，与 SND 包定义的业务域组织（`snd/webui/template/left-tree.json`：14 顶层分组/3 层/65 特性模块/中英双语）不一致；用户要求把界面左侧特性树重构为 left-tree.json。

## What Changes

- 新增 `tools/lefttreegen`（构建期 codegen，与 tasknamegen/blacklistgen 同构）：解析 left-tree.json + goyang 解析每个叶子 xpath 模块的顶层数据容器名，生成 `lefttree.gen.go`（树结构 + 每叶 `sourceModule`/`rootContainers`；运行期零 snd 文件依赖，运行镜像不含 snd 目录——实测 Dockerfile 只拷贝二进制）
- 新增 `GET /api/v1/yang/left-tree`：返回双语特性树，每个叶子按当前 schema 树标注 `available`（其根容器已加载）与目标模块名（前端路由用）；可选 `device=<id>` 叠加 CN-02 能力协商标注 `supported`（设备能力含该模块；协商不可得省略，R08）
- **BREAKING（界面）** 前端原生配置左树重构：category 单级分组 → left-tree 14 组/3 层树；已接入叶子 → 路由 `/module/<root>`；未接入叶子 → 禁用态 + 「未接入」提示（拍板：全树+占位）；left-tree 接口失败时回退现有 category 分组（R08，现逻辑保留为降级路径）
- 双语字段（zh-cn/en-us）全程随树透出，④期 i18n 切换直接消费；本期界面默认 zh-cn
- F4 staging smoke 同步：路由断言适配新树结构（vlan/ifm 仍可点）

## Capabilities

### New Capabilities

- `left-tree-navigation`: 左树数据供给与渲染契约（codegen 结构、available/supported 标注语义、降级路径）

### Modified Capabilities

（无——yang-api 既有条款不变，left-tree 为新端点新能力；前端行为变更由新能力 spec 承载）

## Impact

- 后端：`tools/lefttreegen`（新）、`internal/yangschema`（生成物+接线）、`internal/api`（新 handler + 路由注册 main.go）
- 前端：`stores/menu.ts`（loadLeftTree + 降级）、`Sidebar.vue`（树渲染重构）、路由不变（仍 `/module/:module`）
- 测试：B1（lefttreegen 解析/映射 testdata 正负路径）、B3（left-tree API：available 标注/device 叠加/降级）、F1（store 树装配与降级）、F2（Sidebar 分组渲染/禁用态/可点叶）、F4（staging smoke 选择器适配）
- 契约生成物：swagger + api.gen.ts regen
- 不动：模块表单渲染链路（/yang/schema）、业务网络配置菜单、i18n 框架（④期）
