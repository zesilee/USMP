# Tasks: generic-module-console

> TDD 红绿循环（T01/T05）：每项先测试后实现。每 commit ≤500 行；按阶段分 PR（TM04）。
> legacy→新架构：`DeviceConfigPage` 旧路由重定向保留、组件暂存，新控制台并行落地后再删旧。

## P0 · spec-first 地基
- [x] 0.1 change 三件制品（proposal/design/tasks）+ delta spec（yang-api BR-07/08、frontend FE-10~13）
- [x] 0.2 `openspec validate 2026-07-08-generic-module-console` 通过

## P1 · 后端扩展元数据透出（PR-1，B1+B3）
- [x] 1.1 [红] `schema/entry_ext_test.go`：合成 IFM 形 schema 断言 `SupportFilter()`/
      `OperationExcludes()`；覆盖异前缀（`hw-ext:`）、参数大小写、`|`/`,` 混合切分、无参数降级
- [x] 1.2 [绿] `schema/types.go` LeafNode += `SupportFilter()/OperationExcludes()`；
      `entry.go` 从 `Entry.Exts` 按关键字本名采集；`schema.go` 默认实现
- [x] 1.3 [红] 同文件：ContainerNode `WhenExpr()/MustExprs()` 与 `IsPresence()`（Extra["presence"]）
      联动断言；list 级 `OperationExcludes()`
- [x] 1.4 [绿] ContainerNode/ListNode 接口扩展 + entry.go 容器/list 级 Extra/Exts 采集
- [x] 1.5 [红] `api/field_gen_ext_test.go`：真实 IFM `class` 透出 supportFilter+operationExclude、
      `ipv4-conflict-enable` group 透出 presence+must；合成 list 级 operationExclude；omitempty 不污染
- [x] 1.6 [绿] `FieldDef` 三字段 + `nodeToNestedField`/`leafToField` 填充；`make gen-contract` 同步 api.gen.ts
- [x] 1.7 review（go-code-review-check 通过）+ commit（What/Why/How）

## P2 · 前端列表纯逻辑 + 通用控制台骨架（PR-2 起步，F1+F2）
- [x] 2.1 [红] `test/utils/moduleConsole.test.ts`：`deriveColumns`（分层取列/封顶/层内序稳定）、
      `filterRows`（enum 全等/子串/空条件/大小写）、`cellVisible`（行级 when、求值失败降级可见）
- [x] 2.2 [绿] `utils/moduleConsole.ts` 三纯函数 + `Field` 类型扩宽（supportFilter/operationExclude/presence）
- [x] 2.3 [红] F2 `test/views/ModuleConsolePage.test.ts`：根子节点→Tab 派生（list/group/散叶聚合）、
      面包屑随激活 Tab、schema 加载失败降级
- [x] 2.4 [绿] `views/ModuleConsolePage.vue` + 路由 `/module/:module`

## P3 · 列表 Tab 全功能（PR-2 主体，F2）
- [x] 3.1 [红] F2 `ModuleListTab` 用 5 条 IFM 形 fixture（3 main/200GE/up + 2 sub/Vlanif/down）：
      模型列渲染、enum Tag、up/down 状态点、sub 行显示 parent-name / main 行 `-`
- [x] 3.2 [红] 高级搜索：默认折叠、supportFilter 字段成控件、class=sub-interface 过滤 2 行、重置还原
- [x] 3.3 [红] 分页：>pageSize 分页生效、切页、总数
- [x] 3.4 [红] 操作门禁：编辑抽屉中 operationExclude∋update 字段禁用（新增态可编）；
      list 级 exclude 隐藏对应按钮
- [x] 3.5 [绿] `components/config/ModuleListTab.vue`（drawer/对账流复用既有 composable）
- [ ] 3.6 review + commit

## P4 · 表单 Tab + presence（PR-2 收尾或 PR-3，F2）
- [x] 4.1 [红] presence：`ipv4-conflict-enable` 形 group→开关、关=键不存在、
      `ipv4-ignore-primary-sub=true` 时开关禁用并强制关、求值失败降级可用
- [x] 4.2 [红] 全局属性表单：statistic-interval range 10..600 + must mod10 违例行内报错并拦截提交
- [x] 4.3 [绿] `FieldRenderer` presence 分支 + `components/config/ModuleFormTab.vue`（GET/POST 路径派生、后端不支持路径错误透出）
- [ ] 4.4 review + commit

## P5 · 导航/路由迁移 + 种子 + 冒烟（PR-3）
- [x] 5.1 [红/绿] Sidebar 业务菜单模型驱动（menu store businessModules；失败回退硬编码，F2）
- [ ] 5.2 [绿] `/config/interface`、`/config/vlan` redirect `/module/…`
- [ ] 5.3 [绿] netconfsim ifm 种子 5 条（design §3）
- [ ] 5.4 [红/绿] F4 staging-smoke：/module/huawei-ifm 渲染 Tab、种子行、class 过滤、
      全局属性 Tab must 拦截
- [ ] 5.5 覆盖率棘轮上调（T08）+ review + commit

## 收尾
- [ ] 6.1 `/opsx:sync` delta→主 spec；`openspec validate` 改动能力全绿
- [ ] 6.2 `/opsx:archive` 归档
