<!--
change delta：frontend。主 spec 已是 CLI 标准格式，本次仅 ADDED 新需求。
-->

## ADDED Requirements

### Requirement: FE-07 约束引擎（when 显隐 / must 校验）

前端 SHALL 提供**通用**约束引擎（`utils/xpathEval` + `composables/useConstraintEngine`），把 schema 中的 `when`/`must` XPath 表达式求值为运行时行为，SHALL NOT 硬编码任何厂商/模型/字段名。求值器 SHALL 为自研 YANG XPath 子集解析器（相对路径 `../leaf`、`= != > < >= <=`、`and`/`or`/`not()`、`mod`、字面量），SHALL NOT 引入 `eval`/`safe-eval` 等依赖（R10）。表达式解析失败 SHALL 降级（when 失败=字段可见、must 失败=不阻断）并记录告警，SHALL NOT 崩溃（R08）。

#### Scenario: when 驱动显隐
- **WHEN** 字段带 `when`（如 `../class='sub-interface'`），用户改动被引用字段的值
- **THEN** 引擎 SHALL 实时重算该字段 `visible`；`visible=false` 的字段 SHALL 隐藏且 SHALL NOT 参与提交与校验

#### Scenario: must 阻断非法提交
- **WHEN** 字段带 `must`（如 `(../suppress>../reuse)` 或 `(../interval) mod 10 = 0`）且当前表单违反该约束
- **THEN** 引擎 SHALL 返回违例，前端 SHALL 阻止提交并行内提示（message 取 YANG `description` 或生成的通用提示）

#### Scenario: 表达式语法错误降级
- **WHEN** `when`/`must` 表达式无法被求值器解析
- **THEN** SHALL 降级（可见 / 不阻断）并记录告警，页面 SHALL NOT 崩溃（R08）

### Requirement: FE-08 choice/case 渲染

`FieldRenderer` SHALL 将 `nodeKind:"choice"` 的字段渲染为互斥切换控件（多字段 case→`el-tabs`，单叶 case→`el-radio-group`），分支内子字段按 `cases[].fields` 递归渲染。切换到某 case 时 SHALL 清空其它非激活 case 的数据（YANG choice 互斥语义），提交 payload SHALL 只含激活 case 的字段且保持其扁平 path。

#### Scenario: choice 渲染为切换控件
- **WHEN** schema 含 `nodeKind:"choice"` 节点（如 IFM `bandwidth-type` 的 mbps/kbps 两 case）
- **THEN** SHALL 渲染为 Tabs/RadioGroup，可切换不同 case 的配置块

#### Scenario: 切换 case 清空非激活分支
- **WHEN** 用户从 case A 切到 case B
- **THEN** SHALL 清空 case A 字段值，提交时 SHALL 只携带 case B 字段（扁平 path）

### Requirement: FE-09 leaf-list 与 pattern 校验

`FieldRenderer` SHALL 支持 `nodeKind:"leaf-list"`（可增删的多值输入行，成员复用叶渲染），并 SHALL 对带 `pattern` 的 string 字段绑定正则校验；非法正则 SHALL 降级为不校验并告警（R08），SHALL NOT 崩溃。

#### Scenario: leaf-list 增删多值
- **WHEN** 字段为 `nodeKind:"leaf-list"`
- **THEN** SHALL 渲染可增删的多值输入，提交为数组

#### Scenario: pattern 校验
- **WHEN** string 字段带 `pattern`（如 IFM `number` 的接口编号正则）
- **THEN** SHALL 以该正则校验输入，不匹配时行内报错、阻止提交
