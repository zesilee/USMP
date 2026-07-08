# Tasks: dynamic-form-constraints

> TDD 红绿循环：每项**先写测试**再实现（T01/T05）。每 commit ≤500 行，每阶段一个 PR ≤1000 行。
> legacy→新架构：`FieldDef`/`Field` 契约**扩宽不破坏**，旧字段/旧渲染并行保留，逐步叠加。

## P0 · spec-first 地基（本 change）
- [ ] 0.1 迁移 `yang-api` 主 spec 到 OpenSpec CLI 标准格式（Purpose/Requirements/Requirement/Scenario），修正 BR-03/BR-04 陈旧「硬编码 schema」契约
- [ ] 0.2 写 change delta：`specs/yang-api/spec.md`（MODIFIED BR-03/04 + ADDED BR-05/06）、`specs/frontend/spec.md`（ADDED FE-07/08/09）
- [ ] 0.3 `openspec validate yang-api --type spec` / `frontend --type spec` 通过；`openspec validate dynamic-form-constraints` 通过

## P1 · 垂直切片：`when` 显隐（后端→引擎→渲染→E2E）
### 后端（B1 + B3）
- [ ] 1.1 [红] `schema/entry_test.go`：断言 leaf 透出 `WhenExpr()`（用 IFM `parent-name` `when "../class='sub-interface'"`）
- [ ] 1.2 [绿] `schema/types.go` `LeafNode` 加 `WhenExpr()`；`entry.go` 采集 `e.Extra["when"]`
- [ ] 1.3 [红/绿] `field_gen_test.go`：`FieldDef.When` 被填充；`FieldDef` 加 `When`/`NodeKind` 字段
- [ ] 1.4 [红/绿] `yang_handler_test.go`(B3)：`GET /yang/schema/ifm?form=nested` 响应含 `parent-name` 的 `when`
### 前端（F1 + F2 + F4）
- [ ] 1.5 [红] `test/utils/xpathEval.test.ts` 表驱动：`../class='sub-interface'`、复合 `and`（`down-delay-time` 的 `../type='Eth-Trunk' and ../class='main-interface'`）、字面量、`not()`、语法错误降级
- [ ] 1.6 [绿] `utils/xpathEval.ts`：递归下降 parser + evaluate（不引依赖，R10）
- [ ] 1.7 [红/绿] `test/composables/useConstraintEngine.test.ts`：`visibleMap` 随 formData 变化；when 缺省=可见；解析失败=可见+告警
- [ ] 1.8 [绿] `composables/useConstraintEngine.ts` + `Field` 类型加 `when`/`nodeKind`
- [ ] 1.9 [红/绿] `test/components/FieldRenderer.test.ts`(F2)：when=false 字段不渲染、不入提交
- [ ] 1.10 [绿] `DeviceConfigPage.vue` 接入 `useConstraintEngine`；隐藏字段排除出 payload/校验
- [ ] 1.11 [红/绿] `staging-smoke.spec.ts`(F4)：进 `/config/interface`→新增→设 `class=sub-interface`→断言 `parent-name` 出现，切回 `main-interface`→消失
- [ ] 1.12 review（`go-code-review-check`）+ 覆盖率不降（T08）+ commit（What/Why/How）→ PR

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
