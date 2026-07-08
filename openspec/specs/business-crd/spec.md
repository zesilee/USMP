# business-crd — K8s CRD 意图声明与前端表单 schema 来源

## Purpose

> **LEGACY / 已退出生产。** 本能力属 Stack A（K8s CRD + controller-runtime Reconciler → 翻译引擎 → Actor 2PC → NETCONF 下发）。Stack A 已退出生产：`cmd/controller` 入口已删除，`backend/main.go`（Stack B）为唯一进程入口。本 spec 作历史契约保留，**不代表当前生产配置面**——生产配置读写走 [[config-api]] / [[yang-controller-runtime]] 直连链路。唯一可能仍被前端使用的是 BC-05（CRD 作为前端表单 schema 来源，route/native 的 ConfigPage + useK8sCRD），据实保留。

business-crd 以 K8s CRD 表达厂商中立的设备配置意图，经 controller-runtime Reconciler → 翻译引擎 → Actor 2PC → NETCONF 下发；CRD 同时充当前端表单 schema 来源。

> **契约缺口（详见 design.md §7）**：双 CRD 树抢注同 group、schema 不兼容；NativeDeviceConfig 下发为 TODO stub；生产控制器仍绑 legacy `api/v1`。关联制品：`design.md`、`translation-engine/spec.md`、`actor-transaction/spec.md`、`frontend/spec.md`。

## Requirements

### Requirement: BC-01 意图 CRD 集合

系统 SHALL 支持以一组 K8s CRD 声明设备配置意图，覆盖 `biz.usmp.io` 组下的 BusinessRoute/BusinessSwitch/BusinessVlan/BusinessInterface，以及 `core.usmp.io` 组下的 NativeDeviceConfig。

#### Scenario: 创建业务意图 CR
- **WHEN** 用户创建 BusinessVlan/BusinessRoute/BusinessSwitch/BusinessInterface 等 CR
- **THEN** 系统 SHALL 接受 `biz.usmp.io` 组下的 CR，并 SHALL 支持 `core.usmp.io` 组下的 NativeDeviceConfig CR

### Requirement: BC-02 声明式对齐

BusinessVlan CR 变更时 Reconciler SHALL 执行声明式对齐：以 finalizer 管理生命周期，将意图翻译为 Huawei ygot 结构，经 Prepare(candidate)→Commit(running) 下发并读回，成功后 SHALL 置 `Status.Phase=Synced` 并 requeue（5min 周期）。

#### Scenario: BusinessVlan 变更触发对齐
- **WHEN** BusinessVlan CR 变更且 Reconciler 触发
- **THEN** 系统 SHALL 经 finalizer 管理 → 翻译为 Huawei ygot → Prepare(candidate)→Commit(running) → 读回，并 SHALL 置 `Status.Phase=Synced`、requeue 5min

### Requirement: BC-03 错误分类退避

下发失败时 `handleReconcileError` SHALL 将错误分类为 temporary 或 permanent：temporary 错误 SHALL 以指数退避 requeue，permanent 错误 SHALL 停止重试并标记为 Failed。

#### Scenario: 临时错误指数退避
- **WHEN** 下发失败且错误被分类为 temporary
- **THEN** 系统 SHALL 以指数退避 requeue 重试

#### Scenario: 永久错误停止
- **WHEN** 下发失败且错误被分类为 permanent
- **THEN** 系统 SHALL 停止重试并 SHALL 将状态标记为 Failed

### Requirement: BC-04 原生逃生舱

需下发原始 YANG / 透传配置时，系统 SHALL 提供原生逃生舱：`core/v1` 的 NativeDeviceConfig SHALL 以模型化方式（Module + Config map，schema 动态加载）绕过翻译引擎；legacy `api/v1` SHALL 以传输化方式（Format/Content）承载。

#### Scenario: core/v1 模型化透传
- **WHEN** 创建 `core/v1` NativeDeviceConfig（Module + Config map）
- **THEN** 系统 SHALL 动态加载 schema 并绕过翻译引擎下发

#### Scenario: legacy api/v1 传输化透传
- **WHEN** 通过 legacy `api/v1` 提交 Format/Content 透传配置
- **THEN** 系统 SHALL 以传输化方式承载该原始配置

### Requirement: BC-05 表单 schema 来源

带 `+custom:*` 标注的新树 CRD 在生成 CRD YAML 时，标注 SHALL 转为 OpenAPI `x-` 扩展，供前端 `parseCRDSchemaToFields` 渲染表单。此为当前可能仍被前端（route/native 的 ConfigPage + useK8sCRD）使用的能力面。

#### Scenario: 标注转 OpenAPI 扩展供前端渲染
- **WHEN** 生成带 `+custom:*` 标注的 CRD YAML
- **THEN** 系统 SHALL 将标注转为 OpenAPI `x-` 扩展，前端 SHALL 可经 `parseCRDSchemaToFields` 据此渲染表单字段
