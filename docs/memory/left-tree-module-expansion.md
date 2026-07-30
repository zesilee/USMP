---
name: left-tree-module-expansion
description: 改左树/加模块/加rpc前必读——左树已加深到模块级(container+rpc平铺入树)、rpc入口唯一在左树(Tab栏已退出)、rpcgen清单=taskname超集有守护测试钉死
metadata:
  type: project
---

**左树模块级展开已交付**（change `left-tree-module-expansion`，2026-07-30）：左树在模块叶下再展开一层，把 YANG 模块顶层同级的 container（如 ifm→「通用接口」）与全部 rpc（如「按接口名清除统计」）**平铺**为树节点（用户拍板：不加"运维操作"分组层）。点 container → `/module/:module` 控制台；点 rpc → 新路由 `/module/:module/rpc/:rpcName` 直达执行页。**rpc Tab 已退出控制台 Tab 栏**（FE-19 导航落点=左树）——#233 记录的「rpc 面板常驻污染页面级 E2E 断言」随之消解。

**链路与关键决定（改这块前必读）**：
- **children 构建期烘焙**（design D1）：lefttreegen 同一 goyang 循环提取 rpc（`child.RPC != nil`），双语标签构建期读 `snd/resources/i18n/{zh-cn,en-us}/<module>-res.json` 键 `/<module>:<node>` 烘焙进 lefttree.gen.go，缺键回退原名。运行期零 res 请求、零标签闪变。lefttreegen 新增 `-res` 必填 flag。
- **高危分类唯一口径**：`backend/tools/internal/rpcrisk`（rpcgen 与 lefttreegen 共用），禁止复制词表。
- **rpcgen 清单 ≠ taskname 清单**：rpcgen = taskname + acl/bfd/pic/routing。守护测试 `TestLeftTreeRPCNodesBackedByModuleRPCs`（yangschema 包）钉死「凡可解析叶展出的树 rpc 节点必有 ModuleRPCs 定义」——加左树叶/加 rpc 时若红，把模块补进 load.go rpcgen -modules 并 `make gen-rpc` + `make gen-schema-fixtures`（SF-04 漂移门禁连锁）。
- **API 契约**：LeftTreeNodeDTO 增 kind/name/highRisk（全 omitempty，分组与模块叶自身不带 kind，向后兼容）；available 叶才透出 children，container 仅已加载根容器。改 DTO 后 `make gen-contract`。
- **前端**：LeftTreeMenu 按 kind 分支渲染，`moduleContext` prop 传 rpc 路由前缀；无 children 的可用叶回退直达菜单项。锚点：`lefttree-node-<module>`（container）、`lefttree-rpc-<module>-<rpcName>`（rpc）、叶锚点 `lefttree-leaf-<sourceModule>` 保留在 sub-menu 标题上。**E2E 里点模块叶=展开**（要 `.el-sub-menu__title` 再点 container 节点才导航）。
- **叶判定口径变更**：模块叶现在也有 children——遍历树时「叶=有 sourceModule」，不能再用「无 children」判叶（后端 LT-04 基线测试与 e2e walk 都已改）。

相关：[[yang-rpc-execution]]（rpc 执行链路）、[[snd-integration-program]]（左树①-④期背景）、[[schema-driven-test-harness]]（SF-04 fixture 门禁）
