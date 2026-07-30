# 左树模块级展开 — 设计

## Context

- 左树链路现状：`snd/webui/template/left-tree.json` →（构建期 `tools/lefttreegen`，goyang 解析根容器）→ `internal/yangschema/lefttree.gen.go` →（运行期 `lefttree_handler.convertLeftTree` 算 available/module）→ `GET /yang/left-tree` → 前端 `stores/menu.ts` + `LeftTreeMenu.vue` 递归渲染。树深 3 层，叶子=模块，点击路由 `/module/:module`。
- 模块内导航现状：`ModuleConsolePage` 的 `tabs = [...deriveTabs(fields), ...deriveRpcTabs(rpcs)]`，container Tab 与 rpc Tab 同排（FE-19 的"平级"落在 Tab 栏）。ifm 为 7+10=17 个 Tab，超宽；rpc 面板 el-tabs 常驻渲染，teleport 下拉曾污染 E2E 断言（#233）。
- rpc 事实源：构建期 `tools/rpcgen` → `internal/yangschema/rpc.gen.go`（`ModuleRPCs`，键=根容器名；`isHighRisk` 词表分类）。ygot 运行期 schema 不含 rpc。
- 本地化事实源：`snd/resources/i18n/{zh-cn,en-us}/<sourceModule>-res.json`；根容器键 `/<sourceModule>:<root>`（如 `/huawei-ifm:ifm`→通用接口/Common Interface）、rpc 键 `/<sourceModule>:<rpcName>`（已实测双语齐全）。运行镜像不含 snd 目录（LT-01 红线）。

## Goals / Non-Goals

**Goals:**
- 左树模块叶下平铺展开：每个已加载根容器一个 container 节点 + 该模块全部 rpc 节点，层级与 YANG 模块顶层同级结构一一对应（用户拍板：全部平铺，不加"运维操作"分组层）。
- container 节点 → 既有 `/module/:module` 控制台；rpc 节点 → 新路由 `/module/:module/rpc/:rpcName`，右侧只渲染该 rpc 执行表单（复用 RpcExecuteTab 全部渲染/确认/高危/leafref 逻辑）。
- rpc 退出右侧 Tab 栏（入口唯一收敛到左树）。
- 树节点标签双语，构建期烘焙，缺键回退原名（R08）。
- 全模块同一规律自动生效，零 per-module 代码（R05）。

**Non-Goals:**
- 不把 container 的**下级**（ifm 的 7 个二级容器）入树——它们仍是控制台 Tab；树只加深"模块顶层"这一层。
- 不改 rpc 执行链路（RPC-01~05、DP-10）与 FieldDef 契约。
- 不做树节点按设备能力过滤（supported 语义停留在模块叶，不下沉）。
- 不动未接入叶（available:false 仍无 children、禁用占位）。

## Decisions

**D1：children 构建期烘焙进 lefttree.gen.go，而非前端运行时拼装。**
备选 A（前端拼装）：叶子展开时拉 `/yang/schema/:module` 取 rpcs、再懒加载 res 副本本地化——66 个模块最多 66 次 schema 请求 + 66 次 res 请求，且标签异步闪变。备选 B（构建期烘焙）：lefttreegen 已 goyang 解析全部叶模块（根容器提取处 `child.RPC == nil` 的同一循环里 `child.RPC != nil` 即 rpc），再读 snd res 两个 locale 的 name 烘焙 zh/en——运行期零额外请求、标签静态稳定、与既有分组节点 zh/en 同构。选 B。树标签走生成物、控制台页内标签仍走 UI-03 前端查表：两者同源（同一 res 文件），口径一致。

**D2：highRisk 分类复用 rpcgen 词表，抽为共享包。**
`isHighRisk` 当前私有在 `tools/rpcgen/main.go`。抽到 `backend/tools/internal/rpcrisk`（或 `internal/yangschema` 下非生成文件）供 lefttreegen/rpcgen 共用，两生成器口径永不漂移。禁止复制词表（否则新增高危动词只改一处、树与执行确认分级不一致）。

**D3：DTO 扩展而非新端点。**
`LeftTreeNodeDTO` 新增 `kind`（`"container"`/`"rpc"`，既有分组/叶子省略该字段保持兼容）与 `highRisk`（仅 rpc 节点、true 时携带）。convertLeftTree 对 available 叶把生成物 children 中**已加载根容器**的 container 节点与 rpc 节点透传；不可用叶不透出 children（保持禁用占位语义）。rpc 节点跟随模块可用性——模块 available 才透出（rpc 执行也依赖模块 schema 校验 mandatory）。改 DTO 后 `make gen-contract` 同步 api.gen.ts。

**D4：前端路由 `/module/:module/rpc/:rpcName`；rpc 页复用 ModuleConsolePage 骨架。**
ModuleConsolePage 读路由：无 `rpcName` → tabs = `deriveTabs(fields)`（不再并入 deriveRpcTabs）；有 `rpcName` → 内容区仅渲染该 rpc 的 RpcExecuteTab（从 `/yang/schema/:module` 的 rpcs 中按名取 def，localizeRpcs 本地化 input 叶，找不到该 rpc 名 → 明确错误提示不崩，R08）。备选（独立 RpcPage 组件）会复制 schema 拉取/设备上下文/面包屑逻辑，弃。`deriveRpcTabs` 保留（RpcExecuteTab 的 props 形状不变），仅 ModuleConsolePage 的合并消费点移除。
面包屑：rpc 模式为 配置/厂商/模块/rpc 本地化名。设备上下文沿用全局 device store（FE-10 不变）；rpc 深链 `?device=` 行为与控制台一致。

**D5：LeftTreeMenu 叶子升级为 el-sub-menu，子节点带类型锚点。**
available 模块叶从 `el-menu-item` 变 `el-sub-menu`（默认折叠，不自动展开 66 个模块）；children 里 container → `el-menu-item` 路由 `/module/:module`，rpc → `el-menu-item` 路由 `/module/:module/rpc/:rpcName`。锚点：container `data-test="lefttree-node-<module>"`、rpc `data-test="lefttree-rpc-<module>-<rpcName>"`（既有 `lefttree-leaf-<sourceModule>` 留在模块 sub-menu 标题上，E2E 迁移成本最小）。高危 rpc 节点渲染警示色图标（区别温和 rpc；R12 用真实图标/规范占位符，禁 emoji）。标签按 locale 取 zh/en，en 缺失回退 zh 再回退 name。
菜单收敛宽度：树加深到 5 层可见缩进，沿用既有 `--el-menu-level-padding: 12px` 方案，F3 缩进单调递增回归测试同步覆盖新层级。

**D6：多根容器与 rpc 的归属。**
生成物 children 按根容器逐个出 container 节点；rpc 节点取 `ModuleRPCs` 中该模块**各根容器键**下的并集（rpcgen 键=根容器名，与 schema module key 同构）。lefttreegen 侧直接从 goyang entry 提取（不 import rpc.gen.go，避免生成物互相依赖），但 highRisk 走 D2 共享分类器，rpc 集合与 rpc.gen.go 天然同源（同一 YANG 输入）。

## Risks / Trade-offs

- [BREAKING：rpc Tab 从控制台消失，存量用户习惯/E2E 断言失效] → FE-19 spec 同步改（导航落点=左树）；staging-smoke 断言迁移到左树入口；PR 描述明确标注。
- [生成物膨胀：66 模块 × (根容器+rpc) ≈ 数百节点入 lefttree.gen.go] → 实测 149 rpc + ~70 容器，量级可控；regen-and-diff 门禁不变。
- [res 缺键 → 树节点英文/原名] → 构建期回退原名（LT-01 R08 同款：不阻断生成，留日志）；上线前抽查主力模块。
- [el-sub-menu 层级加深致缩进溢出窄侧栏] → F3 真浏览器缩进回归 + 复用 12px 层级缩进变量；必要时叶层截断+title 提示。
- [黄金漂移风险] → deriveTabs/deriveRpcTabs 纯函数不动，GD-01 黄金理论无漂移；apply 阶段跑全量黄金验证，若红按 SF-04 人工核对。
- [旧路由 `/module/:module` 上直接输 rpc 名的深链不存在] → 新路由仅前端 SPA 路由，无后端路由面；未知 rpcName 明确错误态（R08）。

## Migration Plan

单 PR 交付（预估 <1000 行，TM04）：后端生成器+regen+handler 先行（契约向后兼容，children 新增字段旧前端忽略），前端随后消费。回滚=revert 单 PR。无数据迁移。

## Open Questions

（无——运维操作平铺 vs 分组已由用户拍板：全部平铺。）
