# left-tree-navigation — delta（full-yang-onboarding）

## ADDED Requirements

### Requirement: LT-04 全量叶可用基线

左树 65 个能力叶 SHALL 有 60 个 `available:true`（模块已生成、schema 已加载、
描述符已注册、控制台可路由）。例外恰为五叶：`huawei-pic`（goyang 无法解析其
跨模块 submodule typedef 引用，延期项，CG-04）与四个 augment-only 叶
（`huawei-ethernet`/`huawei-ip`/`huawei-gre`/`huawei-nvo3-statistics`——无自有
根容器，模块 SHALL 入生成闭包使其 augment 面并入宿主模块树呈现，但叶本身
无独立控制台语义、保持 `available:false` 占位）。该基线 SHALL 由测试锁定：
可用叶集合缩水（回归）或例外清单外新增不可用叶 SHALL 使测试失败。

#### Scenario: 全量可用基线

- **WHEN** 请求 `GET /api/v1/yang/left-tree`
- **THEN** `available:true` 的叶 SHALL 恰为全部叶减去五个例外叶，每个可用叶
  SHALL 携带 `module` 可路由到 `/module/<module>`

#### Scenario: 延期叶占位（负路径）

- **WHEN** 请求左树查看 `huawei-pic` 叶
- **THEN** SHALL `available:false` 且仍在树中占位（全树+占位拍板不回退）
