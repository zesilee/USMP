---
name: yang-constraint-engine
description: "YANG 约束驱动动态表单引擎（when/must/pattern/range/leaf-list）——P1/P2/P4 已交付(PR#116)，P3 choice 待办；改约束/加渲染前必读"
metadata: 
  node_type: memory
  type: project
  originSessionId: 244f8a0f-3e42-46bb-bb0e-51c4c67f4392
---

2026-07-08 交付：把 Stack B 设备配置前端从「只消费扁平类型」升级为**消费 YANG 业务约束**，100% 元数据驱动、零厂商/模型硬编码，以华为 IFM 为演示。关联 [[vlan-config-stackb]] [[spec-format-standard]] [[dual-stack-migration]]。

**PR #116**（分支 `worktree-dynamic-form-constraints`，OpenSpec change `dynamic-form-constraints`，spec-first）交付 P1/P2/P4，**11 commits**：
- **P1 `when`**：后端 `schema/entry.go` 从 goyang `Entry.Extra["when"]` 提取 XPath→`LeafNode.WhenExpr()`→`FieldDef.When`；前端自研 `utils/xpathEval.ts`（XPath 子集求值器：`../leaf`/`= != > < >= <=`/`and or not()`/`mod`，递归下降，**不用 eval/safe-eval** R10）+ `composables/useConstraintEngine.ts`（响应式 `visibleMap`）；`DeviceConfigPage` 用 `visibleFields` 裁剪渲染/校验/下发 payload（when=false=节点不存在）。
- **P2 `must`**：`Extra["must"]`→`FieldDef.Must[{expr,message}]`（message 取叶 description 兜底）；引擎 `mustViolations`（只查可见字段）；提交拦截=**submittable + submit 双门**（EP 某些版本 `validate()` 失败是 resolve(false) 非 reject，故以引擎为权威门禁，validate 仅供行内展示）。
- **P4 `pattern`+显式`range`+`leaf-list`**：从 `Type.Pattern`/`Type.Range` 透出（用 goyang 导出默认区间 `Uint32Range` 等过滤，只暴露显式约束，避免每个整数带 0..4294967295 噪声）；pattern 前端 `^(?:…)$` 锚定校验、纳入门禁；leaf-list 后端 `IsLeafList()` 标记 + `FieldRenderer` 可增删标量行。

**关键实证（改这块前必读）**：
- when/must/pattern/range **存活**于 ygot 内嵌 goyang `Entry`（`Extra["when"]/["must"]` 元素是 `{"Name":"<xpath>"}` map 或 `*yang.Value`，提取器两者都兼容）；`e.GetWhenXPath()` 因 unzip 后 `e.Node==nil` **失效**，必须走 `Extra`。
- **⚠️ 更正（2026-07-08 P3 实测推翻旧判断）**：choice/case **并未被拍平**。`huawei.Schema()`（`ygot.GzipToSchema(ySchema)` 嵌入 gzip blob，编译进二进制、运行期零读 `.yang`）返回的 goyang `Entry` 树**完整保留** choice/case/嵌套（`IsChoice()/IsCase()` 均 true，实测 IFM `bandwidth-type` 双 case、嵌套 `damping→damp→level→manual`，路径带 `/device` fakeroot 前缀）。**旧判断「Entry 里 0 个 choice、须构建期 go:generate 生成 choice-map + go:embed」错误**——不需要生成器、不需要 `choicemap.gen.go`、不依赖 `yang-models` submodule（运行期本就不依赖）。
- 当前 `schema/entry.go` **把 choice/case 误当 container** 处理（`entryToNode` default 分支），故 choice 成员叶被渲染成嵌套 group。P3 只需：`entry.go` 识别 `IsChoice()/IsCase()` 建 `ChoiceNode/CaseNode`，且**成员叶 path 扁平化**（跳过 choice/case 段，如 `/ifm/interfaces/interface/bandwidth`，保留真实 container 段如 `damp/manual`），NETCONF 写链路不受影响。`buildPathCacheLocked`/`path.go` 已支持 ChoiceNode，`pathCache` 递归收 choice 成员的扁平 path，`Path()` 查得到。

**P3（choice/case→Tabs/RadioGroup）已合入**（PR #117 squash 为 `f031b89`；至此 P1–P4 全部在 main，change `dynamic-form-constraints` 实现完成，待 sync/archive）：无需生成器。① entry.go 建 Choice/Case 节点 + 扁平成员 path（B1 `entry_choice_test.go`：IFM `bandwidth-type` + 嵌套 `damping→damp→level→manual`，container 段保留、choice/case 段剥除）；② `field_gen` `nodeToNestedField` ChoiceNode 分支→`FieldDef{type:"choice", cases}`、`collectFields` 递归（B3）；③ B2 `bandwidth` 经前端 map 端到端**落到设备**（写链路 OK）；④ `FieldRenderer` choice→`el-radio-group`(全单叶)/`el-tabs`(多字段) + 切 case 清空非激活分支；成员**扁平存 formData**（叶名为键），`DeviceConfigPage` 用 `choiceScope/onChoiceUpdate` reconcile + `flatFields` 展开供 diff/payload/校验；F2(FieldRenderer.choice+DeviceConfigPage.choice) + F3 真 Chromium。BR-06 改为「从嵌入 ygot schema 恢复分组」。覆盖率棘轮上调（后端 54→56、前端 64/64/54/64→70/68/60/70）。**收尾已完成**（PR #119）：yang-api 主 spec 迁移标准格式 + 合入 delta（BR-03/04 MODIFIED 修正旧「硬编码 schema」契约、BR-05/06 ADDED）；frontend 主 spec 追加 FE-07/08/09；change 归档到 `openspec/changes/archive/2026-07-08-dynamic-form-constraints`。`openspec list` 无 active change。**change `dynamic-form-constraints` 全流程闭环**。手工 sync（CLI 无法迁移旧格式主 spec）+ `archive --skip-specs`。**已知 follow-up**：`bandwidth-type` choice 容器二轮对账 Changes==1 不收敛（既有 reconciler 缺口，非本改动，见 [[reconcile-convergence-3rootcauses]]）。

**测试**：后端全绿+`-race` 全绿；前端 **302 单测 + vue-tsc 全绿**；F4 staging-smoke 加「class=sub-interface 显现 parent-name」（断言限定 `.el-drawer`——左侧 SchemaTree 恒列全叶）。**完整 e2e 交 macos-staging CI**（本地 docker 全栈冒烟 USMP_SKIP_E2E 跳过）。

**待办（sync 时）**：迁移 `yang-api` 主 spec 到 OpenSpec CLI 标准格式（delta 已 MODIFIED BR-03/04 修正陈旧「硬编码 schema」契约）。
