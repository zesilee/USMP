# Design: dynamic-form-constraints

## 0. 现状与地基（已核验）

- 前端已有**通用 YANG→UI 引擎**：`useDeviceConfig(opts)` + `DeviceConfigPage.vue`（props 驱动）+ 递归 `FieldRenderer.vue`；「加模块=加一条路由参数」。IFM 已接入（`module:'ifm'`, path `ifm:ifm/ifm:interfaces`），**纯数据驱动、零 IFM 硬编码**。
- schema 来源：后端 `GET /api/v1/yang/schema/:module?form=nested`，`field_gen.go:buildYangSchemaNested` 遍历框架 node model（由 `schema/entry.go` 从 ygot 内嵌的 goyang `yang.Entry` 树构建）。
- **实证结论（解压 ygot 内嵌 schema blob 核验）**：`when`(18)/`must`(18)/`Pattern`(14)/`Range`(116)/`Length`(38)/`Default`(51) **都在** `yang.Entry` 里存活；**`choice`/`case` 为 0**——ygot 拍平了分组。

## 1. 三个现实约束（决定设计）

| # | 约束 | 设计应对 |
|---|------|----------|
| D1 | ygot 丢弃 choice/case | 从**原始 `.yang`** 用 goyang `Loader` 恢复分组；仅作呈现元数据叠加到 nested schema |
| D2 | IFM 无 `leaf-list`、无 `must` 的 `error-message` | leaf-list 用**合成 schema** 做单测；`must` 提示取叶 `description` 兜底，无则生成通用提示 |
| D3 | 写入链路按扁平 leaf path 消费 | choice/case **不改变子叶 path**；提交 payload 结构不变，写路径零改动 |

## 2. 后端：元数据透出

### 2.1 DTO 扩展（`yang_handler.go: FieldDef`）
```go
type MustRule struct {
    Expr    string `json:"expr"`
    Message string `json:"message,omitempty"` // 取自 description，缺省生成
}
type CaseDef struct {
    Name   string     `json:"name"`
    Label  string     `json:"label"`
    Fields []FieldDef `json:"fields"`
}
// FieldDef 新增：
//   NodeKind string     `json:"nodeKind,omitempty"` // leaf|container|list|leaf-list|choice|case
//   When     string     `json:"when,omitempty"`     // 可见性 XPath
//   Must     []MustRule `json:"must,omitempty"`
//   Cases    []CaseDef  `json:"cases,omitempty"`    // 仅 nodeKind==choice
//   Default  case 选择：choice 的 `default <case>` → DefaultCase string
// pattern/minimum/maximum：既有字段，本次开始填充
```
> `FieldDef.Pattern/Minimum/Maximum/Default` 已在 DTO 声明——**扩宽而非破坏契约**。

### 2.2 采集点（`schema/entry.go` + `schema/types.go`）
- `entryToLeaf`（entry.go:147-159 附近）：读取 `e.Extra["when"]`、`e.Extra["must"]`、`e.Type.Pattern`、`e.Type.Range`/`e.Type.Length`、`e.Type.Default`；经 `schema.LeafNode` 新增只读方法（`WhenExpr()`/`MustRules()`/`Pattern()`/`Range()`) 透出。
- `field_gen.go:leafToField`：把上述映射进 `FieldDef`。Range→`minimum/maximum`；Length→string 的 min/max 长度（用于校验，不占 number 的 min/max）。

### 2.3 choice/case 恢复（`schema/loader.go`）
- 现有 `Loader`（goyang 解析原始 `.yang`，未接入 runtime）解析出 `*yang.Entry` 时 `ChoiceEntry`/`CaseEntry` 保留。
- 新增：解析目标模块 `.yang`，产出 `path→{choiceName, caseName}` 映射；`buildYangSchemaNested` 用它把命中 case 的子叶**重组**进 `FieldDef{NodeKind:choice, Cases:[...]}`，**子叶 path 不变**。
- 找不到 `.yang`（部署缺文件）→ 跳过 choice 分组、退化为扁平字段（R08 降级）。
- 框架 node model 已定义 `ChoiceNode`/`CaseNode` 接口且 `buildPathCacheLocked` 已处理——脚手架现成，只需填充。

## 3. 前端：约束引擎（通用，零硬编码）

### 3.1 XPath 子集求值器（`utils/xpathEval.ts`）
- **支持文法**（覆盖 IFM 真实表达式）：相对路径 `../leaf`（同级取值）、字面量 `'str'`/数字、`=` `!=` `>` `<` `>=` `<=`、`and` `or`、`not(expr)`、`mod`、括号。
- 解析为 AST（小型递归下降 parser），`evaluate(ast, ctx)` 对当前表单行数据求值。**不使用 `eval`/`safe-eval`（R10）**。
- 解析失败 → 抛 `XPathParseError`；调用方按语义降级（when 失败=可见、must 失败=不阻断+告警），**不崩**（R08）。
- 纯函数、无副作用、无共享可变态（R09）。

### 3.2 `useConstraintEngine.ts`
- 输入：`fields`（含 when/must）+ 响应式 `formData`。
- 输出：`visibleMap`（computed：path→bool）、`validate(formData)`（返回 must 违例列表，接入 el-form 校验）。
- `when` 求值 = false → 字段隐藏，且隐藏字段**不参与提交/校验**（YANG 语义：when 不成立即该节点不存在）。

### 3.3 渲染层（`FieldRenderer.vue`）
- `nodeKind==choice` → `el-tabs`（多 case 多字段）或 `el-radio-group`（case 为单叶）；切 case 清空非激活分支数据（YANG choice 互斥）。
- `nodeKind==leaf-list` → 可增删的 `el-tag` 输入行（成员类型复用叶渲染）。
- `pattern` → el-form rule `{ pattern: new RegExp(field.pattern) }`；非法正则降级为不校验 + 告警。
- `visible` 由 `useConstraintEngine` 注入（`v-if`），`must` 违例注入 el-form-item error。

## 4. 分阶段交付（各为独立 PR，≤1000 行）

推荐 **垂直切片优先**：先打通一条端到端链路证明架构，再横向补齐。详见 `tasks.md`。

- **P1 垂直切片 · when 显隐**：后端透出 `when` → 前端 `xpathEval` + `useConstraintEngine` 显隐 → IFM `class=sub-interface` 显隐 `parent-name` → F4 冒烟。**一条 PR 证明 后端→引擎→渲染→E2E 全管道**。
- **P2 · must 校验**：后端透出 `must` → 引擎 `validate` → 提交拦截（suppress/reuse、mod 10）。
- **P3 · choice/case**：后端 `.yang` 恢复分组 → `FieldRenderer` Tabs/RadioGroup（IFM `choice bandwidth-type`、嵌套 `choice damping`）。
- **P4 · leaf-list + pattern + range**：渲染 + 校验补齐（leaf-list 用合成 schema 测，pattern 用 IFM `number`）。

## 5. 风险与回退

| 风险 | 缓解 |
|------|------|
| XPath 子集不覆盖某些 IFM 表达式 | 求值器对未知语法**抛错并降级**，不静默误判；新增语法增量加测 |
| `.yang` 文件部署缺失 → choice 丢失 | 降级为扁平字段（仍可用），日志告警 |
| choice 分组误伤写入链路 | 子叶 path 不变 + P3 附带写入回读集成测试兜底（B2） |
| 隐藏字段仍被提交 | 引擎明确「when=false 不参与提交」，F2 覆盖 |

## 6. 备选方案（已否决）

- **前端解析 `.yang`**：重复后端已做的解析、偏离 live-schema 原则，且需前端引 YANG parser（R10）。→ 用户已选后端产出元数据。
- **safe-eval / new Function 求值 when/must**：R10 禁无关依赖 + 注入风险；自研子集求值器可控可测。
- **React/AntD 重写**：违背 R05/R10、与既有 Vue 引擎重复。→ 用户已选扩展 Vue 引擎。
