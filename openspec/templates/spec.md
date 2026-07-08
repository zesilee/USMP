<!--
Change delta spec 模板 —— 见 openspec/SPEC_CONVENTIONS.md §4。
路径：openspec/changes/<change>/specs/<capability>/spec.md
增量头：## ADDED / MODIFIED / REMOVED Requirements
新增契约用 ADDED；MODIFIED/REMOVED 的标题必须与主 spec 完全一致（含 XX-NN 前缀）。
每个 ### Requirement 至少一个 #### Scenario，含 WHEN/THEN。
-->

## ADDED Requirements

### Requirement: <XX-NN> <简短陈述>

<SHALL 陈述（RFC2119 大写关键词）。说明必须满足什么契约。>

#### Scenario: <正常场景名>

- **WHEN** <触发条件>
- **THEN** <系统 SHALL <可验证结果>>

#### Scenario: <异常/边界场景名>

- **WHEN** <异常触发>
- **THEN** <系统 SHALL <降级/报错的可观察表现（R08 不崩）>>
