# 左树模块级展开 — 任务清单

> TDD 红绿循环（T01/T05）：每组先写测试（红）再实现（绿）。层选取按 §5.6：后端 B1+B3、前端 F1+F2+F3+F4。

## 1. 后端：共享高危分类器（D2）

- [x] 1.1 测试先行：`rpcrisk` 共享包表格驱动测试（restart/reboot/delete 命中、reset/clear 不命中、词边界）
- [x] 1.2 抽取 rpcgen 的 `isHighRisk`/词表到共享包，rpcgen 改为引用，`make gen-rpc` regen-and-diff 验证零漂移

## 2. 后端：lefttreegen children 烘焙（LT-01）

- [x] 2.1 测试先行：lefttreegen 测试新增 children 断言（testdata demo 模块：container/rpc 节点、highRisk、res 双语烘焙、缺键回退原名、解析失败叶 children 为空、重复执行字节一致）
- [x] 2.2 实现：goyang 同循环提取 rpc（`child.RPC != nil`）、读 snd res 双语 name 烘焙 zh/en、接 rpcrisk 分类器；`LeftTreeNode` 增 `Children`（模块级）结构
- [x] 2.3 `go generate ./internal/yangschema/` 重新生成 lefttree.gen.go，抽查 huawei-ifm（1 container「通用接口」+ 10 rpc、restart-if highRisk）与另两个主力模块

## 3. 后端：left-tree API 透出（LT-02）

- [x] 3.1 测试先行：lefttree_handler 测试新增——available 叶 children 透出（container 仅已加载根容器/rpc 全量/highRisk 仅 true 携带）、不可用叶无 children、分组与模块叶无 kind 字段、LT-04 基线不回归
- [x] 3.2 实现：`LeftTreeNodeDTO` 增 `kind`/`highRisk`/name 字段，convertLeftTree 透传；swag 注释更新
- [x] 3.3 `make gen-contract` 同步 api.gen.ts（契约漂移门禁绿）

## 4. 前端：左树渲染加深（LT-03）

- [x] 4.1 测试先行（F1）：menu store LeftTreeNode 类型扩展与解析测试（children/kind/highRisk）
- [x] 4.2 测试先行（F2）：LeftTreeMenu/Sidebar 测试——available 叶变可展开 sub-menu、container/rpc 子节点路由与 data-test 锚点、highRisk 警示标识、不可用叶不可展开、locale 标签回退链（zh→en→name）
- [x] 4.3 实现：LeftTreeMenu 叶子升级 el-sub-menu + children 渲染 + 图标/锚点；menu store 类型扩展
- [x] 4.4 F3 真浏览器：SidebarTreeIndent 回归扩展到新层级（缩进单调递增、el-sub-menu 展开真实交互）

## 5. 前端：rpc 直达路由与控制台去 rpc Tab（FE-19）

- [x] 5.1 测试先行（F1/F2）：router 测试新增 `/module/:module/rpc/:rpcName`；ModuleConsolePage 测试——tabs 不含 rpc Tab、rpc 路由模式仅渲染该 rpc 执行面板（localizeRpcs 本地化）、未知 rpcName 错误态、设备上下文与面包屑
- [x] 5.2 实现：router 加路由；ModuleConsolePage 去 deriveRpcTabs 合并、增 rpc 模式（按名取 def→RpcExecuteTab）；面包屑含 rpc 本地化名
- [x] 5.3 跑全量派生黄金 GD-01 验证零漂移（理论不受影响；若红按 SF-04 人工核对后决策）

## 6. E2E 与收尾

- [x] 6.1 staging-smoke（F4）：rpc 入口断言从 Tab 栏迁移到左树（展开模块叶→点 rpc 节点→执行面板可见）；顺带确认 #233 的 teleport 污染断言随 rpc 退出常驻渲染而消解
- [ ] 6.2 `make e2e-local` 全绿；前后端全量测试 + `-race`；覆盖率不低于棘轮基线（T08，补测后上调）
- [ ] 6.3 code review（go-code-review-check）+ What/Why/How 分组提交 + 记忆更新（单独 docs commit，MEM04）
