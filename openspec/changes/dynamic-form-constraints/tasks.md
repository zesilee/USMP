# Tasks: dynamic-form-constraints

> TDD 红绿循环：每项**先写测试**再实现（T01/T05）。每 commit ≤500 行，每阶段一个 PR ≤1000 行。
> legacy→新架构：`FieldDef`/`Field` 契约**扩宽不破坏**，旧字段/旧渲染并行保留，逐步叠加。

## P0 · spec-first 地基（本 change）
- [ ] 0.1 迁移 `yang-api` 主 spec 到 OpenSpec CLI 标准格式（Purpose/Requirements/Requirement/Scenario），修正 BR-03/BR-04 陈旧「硬编码 schema」契约 — 留到 sync 前做
- [x] 0.2 写 change delta：`specs/yang-api/spec.md`（MODIFIED BR-03/04 + ADDED BR-05/06）、`specs/frontend/spec.md`（ADDED FE-07/08/09）
- [x] 0.3 `openspec validate dynamic-form-constraints` 通过

## P1 · 垂直切片：`when` 显隐（后端→引擎→渲染→E2E）✅ 已交付（commit d60bb64 / 78cbe06 / 6a45a0f）
### 后端（B1 + B3）
- [x] 1.1 [红] `entry_when_test.go`：leaf 透出 `WhenExpr()`（IFM `parent-name` `when "../class='sub-interface'"` + *yang.Value 双形态）
- [x] 1.2 [绿] `schema/types.go` `LeafNode` 加 `WhenExpr()`；`entry.go` 采集 `e.Extra["when"]`（兼容 map / *yang.Value）
- [x] 1.3 [红/绿] `field_gen_when_test.go`：`FieldDef.When` 被填充；`FieldDef` 加 `When` 字段（NodeKind 留到 P3）
- [x] 1.4 [绿] `buildYangSchemaNested` 经 `leafToField` 透出真实 IFM when（实证：GetWhenXPath 失效走 Extra）
### 前端（F1 + F2 + F4）
- [x] 1.5 [红] `test/utils/xpathEval.test.ts` 14 例：相等/复合 and/or/not()/mod/关系/语法错误降级（语料全取自真实 IFM）
- [x] 1.6 [绿] `utils/xpathEval.ts`：递归下降 parser + evaluator（不引依赖，R10）
- [x] 1.7 [红/绿] `test/composables/useConstraintEngine.test.ts`：`visibleMap` 随 formData 响应式变化；缺省可见；解析失败可见+告警
- [x] 1.8 [绿] `composables/useConstraintEngine.ts`（toValue 归一，兼容 reactive）+ `Field.when`
- [x] 1.9 [红/绿] `test/views/DeviceConfigPage.visibility.test.ts`(F2)：when=false 字段不渲染、不入 payload
- [x] 1.10 [绿] `DeviceConfigPage.vue` 接入引擎；visibleFields 驱动 渲染/rules/diff/submittable/payload
- [x] 1.11 [绿] `staging-smoke.spec.ts`(F4)：`/config/interface`→新增→class=sub-interface 显现 parent-name（限定 .el-drawer；playwright --list 通过，全量待 make e2e-local）
- [x] 1.12 review（go-code-review-check 通过）+ commit（What/Why/How）；覆盖率上调留到 P4 收尾一并处理

## P2 · `must` 跨字段校验 ✅ 已交付（commit 795df5e / e981fea）
- [x] 2.1 [红/绿] 后端透出 `must`（`e.Extra["must"]`→`FieldDef.Must[{expr,message}]`，message 取 description 兜底）；B1+B3（真实 IFM down-delay-time mod 100）
- [x] 2.2 [绿] `xpathEval` 已含 `mod`/关系/逻辑；`useConstraintEngine.mustViolations` 对可见字段逐条求值返回违例
- [x] 2.3 [红/绿] F2：`suppress>reuse` 违例阻断提交（submittable+submit 双门）并行内提示；缺 error-message 生成含标签提示
- [x] 2.4 review + commit（覆盖率上调留 P4 收尾）

## P3 · `choice`/`case` 渲染
> ⚠️ 修订：实测 `huawei.Schema()` 内嵌 schema **已完整保留 choice/case**（`IsChoice()/IsCase()`），运行期零读 `.yang`。**去除**「构建期生成 choice-map + go:embed」旧计划——不需生成器、不依赖 yang-models submodule。仅需让 `schema/entry.go` 识别 choice/case 并**扁平化成员 path**。
- [x] 3.1 [红/绿] `schema/entry.go` 识别 `IsChoice()/IsCase()` 建 `ChoiceNode/CaseNode`，成员子字段 **path 扁平化**（剥 choice/case 段、留 container 段）；`defaultChoice/defaultCase` 实现；B1 `entry_choice_test.go` 用 IFM `bandwidth-type`（双单叶 case）+ 嵌套 `damping→damp→level→manual`（container 段保留）（commit b690219）
- [x] 3.2 [红/绿] `field_gen` `nodeToNestedField` 加 `ChoiceNode` 分支→`FieldDef{type:"choice", cases:[{name,label,fields}]}`；`collectFields` 递归 choice cases；`FieldDef.Cases` + `CaseDef`；B3 `field_gen_choice_test.go` 真实 IFM 透出；B2 集成 `bandwidth` 经前端 map 端到端落到设备（commit 8b6d22e）。注：`bandwidth-type` choice 容器二轮对账收敛属既有 reconciler 缺口、与本呈现改动无关，独立 follow-up（B2 只断言落到设备）
- [x] 3.3 [红/绿] `FieldRenderer` choice→`el-tabs`(含多字段 case)/`el-radio-group`(全单叶)，切 case 清空非激活分支、payload 仅激活 case 扁平 path；F2（FieldRenderer.choice + DeviceConfigPage.choice）+ F3 真浏览器（el-tabs/el-radio 真实交互）（commit 816e5c1）
- [ ] 3.4 review（go-code-review-check 通过）+ 覆盖率棘轮上调 + commit → PR

## P4 · `leaf-list` + `pattern` + `range` ✅ 已交付（commit 8d401df / 4d4dae1）
- [x] 4.1 [红/绿] 后端透出 `pattern`（真实 IFM `number`）+ 显式 `range`→min/max（真实 IFM `statistic-interval` 10..600，跳过类型全域默认区间避免噪声）；B1/B3
- [x] 4.2 [红/绿] DeviceConfigPage 接 `pattern` 校验（^(?:…)$ 锚定、非法正则降级 R08、纳入 submit 权威门禁）；`FieldRenderer` `leaf-list`→增删标量行（合成 schema，IFM 无 leaf-list）；F2（pattern 拦截 + leaf-list add/edit/remove/枚举 5 例）
- [x] 4.3 review + commit（F3 真浏览器留待与 P3 el-tabs 合并跑）

## 收尾
- [ ] 5.1 `/opsx:sync`：delta → 主 spec；`openspec validate --specs` 改动能力全绿
- [ ] 5.2 `/opsx:archive`：归档 change
- [ ] 5.3 覆盖率基线上调（后端 `.coverage-baseline`、前端 vitest thresholds）（T08）
