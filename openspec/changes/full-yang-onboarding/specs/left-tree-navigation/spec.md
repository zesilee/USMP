# left-tree-navigation — delta（full-yang-onboarding）

## ADDED Requirements

### Requirement: LT-04 全量叶可用基线

左树 61 个能力叶 SHALL 有 60 个 `available:true`（模块已生成、schema 已加载、
描述符已注册、控制台可路由）；唯一例外 `huawei-pic`（goyang 无法解析其跨模块
submodule typedef 引用，延期项，CG-04）SHALL 保持 `available:false` 占位。
该基线 SHALL 由测试锁定：可用叶集合缩水（回归）或延期清单外新增不可用叶
SHALL 使测试失败。

#### Scenario: 全量可用基线

- **WHEN** 请求 `GET /api/v1/yang/left-tree`
- **THEN** `available:true` 的叶 SHALL 恰为全部叶减去 `huawei-pic`，每个可用叶
  SHALL 携带 `module` 可路由到 `/module/<module>`

#### Scenario: 延期叶占位（负路径）

- **WHEN** 请求左树查看 `huawei-pic` 叶
- **THEN** SHALL `available:false` 且仍在树中占位（全树+占位拍板不回退）
