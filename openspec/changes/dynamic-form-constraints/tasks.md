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

## P2 · `must` 跨字段校验
- [ ] 2.1 [红/绿] 后端透出 `must`（`e.Extra["must"]`→`FieldDef.Must[{expr,message}]`，message 取 description 兜底）；B1+B3
- [ ] 2.2 [红/绿] `xpathEval` 扩 `mod`/`>`/`<`/关系运算；`useConstraintEngine.validate()` 返回违例
- [ ] 2.3 [红/绿] F2：`suppress>reuse`、`(interval) mod 10 = 0` 违例阻断提交并行内提示；缺 error-message 用生成提示
- [ ] 2.4 review + commit → PR

## P3 · `choice`/`case` 渲染
- [ ] 3.1 [红/绿] `schema/loader.go` 解析原始 `.yang` 恢复 `choice/case` 映射；缺文件降级（R08）；B1 用 IFM `choice bandwidth-type`、嵌套 `choice damping`
- [ ] 3.2 [红/绿] `buildYangSchemaNested` 注入 `NodeKind:choice`+`Cases`，**子叶 path 不变**；B2 集成：choice 字段下发→回读收敛（写路径未破）
- [ ] 3.3 [红/绿] `FieldRenderer` choice→`el-tabs`/`el-radio-group`，切 case 清空非激活分支；F2 + F3 真浏览器（el-tabs teleport）
- [ ] 3.4 review + commit → PR

## P4 · `leaf-list` + `pattern` + `range`
- [ ] 4.1 [红/绿] 后端透出 `pattern`（IFM `number`）+ `range`→min/max；B1
- [ ] 4.2 [红/绿] `FieldRenderer` 接 `pattern` 校验（非法正则降级）；`leaf-list`→增删输入行（**合成 schema** 测，IFM 无 leaf-list）；F2 + F3
- [ ] 4.3 review + commit → PR

## 收尾
- [ ] 5.1 `/opsx:sync`：delta → 主 spec；`openspec validate --specs` 改动能力全绿
- [ ] 5.2 `/opsx:archive`：归档 change
- [ ] 5.3 覆盖率基线上调（后端 `.coverage-baseline`、前端 vitest thresholds）（T08）
