# 左树模块级展开（container 与 rpc 入树）

## Why

左树目前只到模块叶（如 huawei-ifm）为止，模块内的配置容器与 rpc 运维操作全部塌进右侧一排 el-tabs：ifm 有 7 个配置 Tab + 10 个 rpc Tab 共 17 个横向标签，超出可视宽度、找操作靠横向滚动；且 rpc 面板常驻渲染已多次污染 E2E 断言（#233 踩坑）。用户拍板：左树在模块叶下再展开一层，把 YANG 模块顶层同级的 container 与全部 rpc **平铺**为树节点（与 YANG 模型结构一一对应），点 container 进配置控制台、点 rpc 直达该操作执行表单。

## What Changes

- **构建期左树生成扩展**（lefttreegen）：模块叶下新增 `children` —— 每个已解析根容器一个 `container` 节点、该模块全部 rpc 各一个 `rpc` 节点（携 `highRisk`）；节点 zh/en 标签构建期从 snd res（`/<sourceModule>:<root>`、`/<sourceModule>:<rpcName>` 的 name）烘焙进生成物，缺键回退原名。运行期依旧零 snd 文件依赖。
- **左树接口扩展**（GET /yang/left-tree）：available 模块叶透出 `children`（container/rpc 节点，含 kind、name、zh/en、highRisk）；不可用叶不透出 children。
- **前端左树加深一层**：模块叶由终点变为可展开分组；container 子节点路由 `/module/:module`（现控制台），rpc 子节点路由新增 `/module/:module/rpc/:rpcName`（右侧仅渲染该 rpc 执行表单，复用 RpcExecuteTab 渲染管线与确认/高危逻辑）。
- **rpc 退出右侧 Tab 栏**：ModuleConsolePage 的 Tab 集合不再合入 `deriveRpcTabs`，rpc 入口唯一收敛到左树（**BREAKING**：`/module/:module` 页 Tab 栏不再出现 rpc Tab；FE-19「导航中平级」的落点从 Tab 栏迁移到左树）。
- 全部模块按同一规律自动生效，零 per-module 前端代码（R05 口径不变）。

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `left-tree-navigation`：LT-01 构建期生成扩展（叶子携带 container/rpc children 与烘焙双语标签）；LT-02 接口透出 children；LT-03 前端渲染加深一层（container/rpc 子节点可点路由、rpc 直达执行表单）。
- `frontend`：FE-19「rpc 与 container 平级」的导航落点由控制台 Tab 栏改为左侧导航树；`/module/:module` Tab 集合不再含 rpc Tab；新增 rpc 直达路由行为。

## Impact

- 后端：`backend/tools/lefttreegen`（解析 rpc + 读 res 烘焙标签 + highRisk 分类复用 rpcgen 口径）、`backend/internal/yangschema/lefttree.gen.go`（regen，生成物 R04 口径）、`backend/internal/api/lefttree_handler.go`（children 透传）、对应 handler/lefttreegen 测试。
- 前端：`stores/menu.ts`（LeftTreeNode 契约扩展）、`components/layout/LeftTreeMenu.vue`（叶子变 sub-menu）、`router`（新增 rpc 路由）、`views/ModuleConsolePage.vue`（Tab 集合去 rpc、rpc 路由模式）、`utils/moduleConsole.ts`（deriveRpcTabs 消费点调整）。
- 契约：FieldDef 不动；LeftTree API 响应新增字段 → `make gen-contract` 同步 api.gen.ts（漂移门禁）。
- 测试：backend lefttree 表格驱动 + handler（B1/B3）；前端 menu store/Sidebar/LeftTreeMenu/router（F1/F2）、真浏览器缩进单调回归（F3）、staging-smoke rpc 入口断言迁移（F4）。派生黄金不受影响（deriveTabs/deriveRpcTabs 纯函数行为不变，仅消费点变化）——若实测漂移则按 GD-01 刷黄金并人工核对。
- E2E 收益：rpc 面板不再常驻 `/module/:module`，#233 记录的 teleport 下拉污染页面级断言问题随之消解。
