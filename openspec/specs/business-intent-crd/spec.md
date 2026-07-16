# business-intent-crd Specification

## Purpose

业务意图 CRD 公共契约：意图 YANG 是唯一 schema 源，CRD OpenAPI 构建期生成（tools/crdgen），kubectl/GitOps 直写受支持；校验分层 = apiserver OpenAPI + 控制器约束引擎兜底；status 子资源承载收敛事实（多实例唯一状态源）。

## Requirements

### Requirement: BIC-01 意图 YANG 模型是唯一 schema 源

业务意图能力 SHALL 用 YANG 模块定义（存放于版本库、纳入 gen-yang 管线生成 ygot 结构体，R04）。CRD 的 OpenAPI schema SHALL 由构建期 codegen 从意图 YANG 生成，SHALL NOT 手写或手改生成物；YANG 与 CRD 出现分歧时以 YANG 为准（regen-and-diff 门禁）。

#### Scenario: YANG→CRD 生成一致性（golden）
- **WHEN** 对意图 YANG 模块运行 YANG→CRD codegen
- **THEN** 产物 SHALL 与仓库内已提交的 CRD manifest golden 一致，漂移时 CI SHALL fail

#### Scenario: 类型映射
- **WHEN** 意图 YANG 含 uint16 range、string pattern、enum、leaf-list、嵌套 list
- **THEN** 生成的 OpenAPI SHALL 分别映射为 integer+min/max、string+pattern、enum、array、array<object>（含 list key 必填）

### Requirement: BIC-02 CRD 公共契约与版本策略

意图 CRD SHALL 归属独立 API group（`biz.usmp.io` 之外的新 group 或复位后的同 group，design 定名），每业务能力一个 Kind；SHALL 启用 status 子资源；初始版本 SHALL 为 `v1alpha1` 且为 storage version。kubectl/GitOps 直接写 CR 是受支持的接入方式：CR SHALL 携带 printer columns（关键业务字段+收敛状态）。CRD manifest 的安装 SHALL 由部署编排（helm/初始化脚本）先于 USMP 新版本滚动完成。

#### Scenario: kubectl 直写受支持
- **WHEN** 运维用 kubectl apply 创建一个字段合法的意图 CR
- **THEN** apiserver SHALL 接受写入，USMP watch 到后 SHALL 进入展开与下发流程，无需经过 USMP API

#### Scenario: 字段级非法写入被 apiserver 拒绝
- **WHEN** kubectl 写入超出 YANG range/pattern/enum 的字段值（如 vlan-id=4095）
- **THEN** apiserver SHALL 依据生成的 OpenAPI 校验直接拒绝（写入时失败，而非收敛时）

### Requirement: BIC-03 校验分层（apiserver + 约束引擎兜底）

字段级约束 SHALL 由生成的 OpenAPI 在 apiserver 侧校验；OpenAPI 无法表达的 YANG 语义（when/must 跨字段、leafref 跨对象）SHALL 由约束引擎在控制器 admission-by-watch 时校验：校验失败 SHALL 在 status 打 `Validated=False`（含原因）且 SHALL NOT 展开下发（R08 不崩溃）。

#### Scenario: 跨字段约束收敛侧兜底
- **WHEN** CR 通过 apiserver 字段校验但违反意图 YANG 的 must 跨字段约束
- **THEN** 控制器 SHALL 置 `Validated=False` + 原因，不产生任何设备下发

#### Scenario: 校验通过
- **WHEN** CR 全量校验通过
- **THEN** 控制器 SHALL 置 `Validated=True` 并进入展开流程

### Requirement: BIC-04 status 子资源结构

意图 CR 的 status SHALL 承载：`observedGeneration`、条件集（至少 `Validated`、`Converged`）、每设备状态 `deviceStates[]{device, phase(pending|synced|failed), reason, lastTransition}`、归属认领 `claims[]{device, module, path}`、上一代展开摘要（供收缩差集）。status SHALL 仅由控制器写入（spec 与 status 权限分离）。

#### Scenario: 状态可恢复
- **WHEN** USMP 实例重启
- **THEN** 意图实例、认领与每设备状态 SHALL 从 CR spec/status 完整恢复，无本地状态依赖
