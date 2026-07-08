# tasks — ext-ui-annotations

> TDD 红绿循环（T01/T05）：每组先写测试（红）再实现（绿）。测试层按 §5.6 标注。
> 单 commit ≤500 行、原子功能；What/Why/How 三段式。

## 1. S1 后端：config-false → Readonly 透出（B1+B3）

- [x] 1.1 B1 红：`schema/entry_readonly_test.go` 表格驱动——`config false` 子树根/后代/config-true 兄弟/混合容器叶（含 race）
- [x] 1.2 绿：`entry.go` 构树时下推继承只读（design D1），Node/LeafNode 增 `ReadOnly()` 存取器
- [x] 1.3 B3 红：`field_gen` 测试——readonly 落 FieldDef（容器/list/叶一致）、config-true 无键（omitempty）
- [x] 1.4 绿：`field_gen.go` 填充既有 `FieldDef.Readonly`

## 2. S2+S3 后端：dynamic-default + units 透出（B1+B3）

- [x] 2.1 B1 红：`entry_ext_test.go` 增 dynamic-default 用例——有/无子句形态、其他扩展不误报（负路径）
- [x] 2.2 绿：`entry.go` 增 `extDynamicDefault()`（本名匹配，同 BR-07 规约）+ LeafNode 存取器
- [x] 2.3 B3 红：field_gen 测试——`dynamicDefault`/`units` 透出与省略；units 取 `Type.Units` 兜底 `Entry.Units`
- [x] 2.4 绿：FieldDef 增 `DynamicDefault bool`/`Units string`（omitempty）+ field_gen 填充
- [x] 2.5 `make gen-contract` 同步 api.gen.ts，确认漂移门禁绿

## 3. S4 后端：task-name 构建期 codegen + /yang/modules category（B1+B3）

- [x] 3.1 B1 红：生成器测试——goyang 解析模块级 `ext:task-name`、键=根容器名、无 task-name 模块缺省
- [x] 3.2 绿：生成器（goyang 解析 8.20.10/ne40e-x8x16 的 huawei-vlan/ifm/system）+ go:generate 声明（与 huawei.go 相邻）+ 提交 `taskname.gen.go`
- [x] 3.3 B3 红：ListModules 测试——有映射附 `category`、无映射省略且不失败（R08）
- [x] 3.4 绿：`YangModuleInfo` 增 `Category`（omitempty）+ handler 查表填充；`make gen-contract`

## 4. S1 前端：只读降级（F1+F2）

- [x] 4.1 F1 红：`moduleConsole` 派生测试——整棵 readonly 子树→只读 Tab、混合容器 readonly 叶标记
- [x] 4.2 绿：`moduleConsole.ts` Tab 派生携带 readonly 标记（现有 `!f.readonly` 过滤改为降级派生）
- [x] 4.3 F2 红：只读 Tab 只读视图（无编辑/下发入口）、只读 list 表格无操作列、混合容器叶禁用态且不入 payload/校验
- [x] 4.4 绿：控制台/FieldRenderer 只读呈现路径（design D4 两层）

## 5. S2+S3 前端：动态缺省占位 + 单位后缀（F2）

- [ ] 5.1 F2 红：FieldRenderer——dynamicDefault 占位提示、空值不入 diff/不报必填、显式覆写正常下发（边界）、units 后缀展示
- [ ] 5.2 绿：FieldRenderer 消费 `dynamicDefault`/`units`；useConfigForm 空值豁免必填与 diff

## 6. S4 前端：左导航任务域分组（F1+F2）

- [ ] 6.1 F1/F2 红：分组派生纯函数 + 菜单渲染——带/不带 category 混合、缺失归默认组不失败（R08）
- [ ] 6.2 绿：左导航按 category 分组渲染

## 7. 收口

- [ ] 7.1 全量验证：后端 `go test ./...`（含 -race）、前端单测 + `vue-tsc` + gen-contract 漂移检查全绿
- [ ] 7.2 覆盖率对齐棘轮（后端 57 / 前端 73/70/65/73），补测后按需上调基线
- [ ] 7.3 `go-code-review-check` + What/Why/How 提交整理，PR ≤1000 行自检
