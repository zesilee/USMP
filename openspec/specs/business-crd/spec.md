# business-crd — 行为契约（反向还原）

> 反向还原自 `backend/api/{v1,biz/v1,core/v1}` + `backend/controllers/*`，忠实 as-built。详见 `design.md`。Stack A；`api/v1` = legacy。

## 能力概述

以 K8s CRD 表达厂商中立配置意图，经 controller-runtime Reconciler → 翻译引擎 → Actor 2PC → NETCONF 下发；CRD 同时充当前端表单 schema 来源。

## 行为契约

### BC-01 意图 CRD 集合
- **Given** 需声明设备配置意图
- **When** 创建 CR
- **Then** 支持 BusinessRoute/Switch/Vlan/Interface（`biz.usmp.io`）+ NativeDeviceConfig（`core.usmp.io`）

### BC-02 声明式对齐
- **Given** BusinessVlan CR 变更
- **When** Reconciler 触发
- **Then** finalizer 管理 → 翻译为 Huawei ygot → Prepare(candidate)→Commit(running) → 读回 → `Status.Phase=Synced`，requeue 5min

### BC-03 错误分类退避
- **Given** 下发失败
- **When** `handleReconcileError`
- **Then** 分类 temporary/permanent，temporary 指数退避 requeue，permanent 停止并标记 Failed

### BC-04 原生逃生舱
- **Given** 需下发原始 YANG/透传配置
- **When** 创建 NativeDeviceConfig
- **Then** `core/v1` 模型化(Module+Config map，schema 动态加载) 绕过翻译引擎；`api/v1`(legacy) 传输化(Format/Content)

### BC-05 表单 schema 来源
- **Given** 新树 CRD 带 `+custom:*` 标注
- **When** 生成 CRD YAML
- **Then** 标注转为 OpenAPI `x-` 扩展，供前端 `parseCRDSchemaToFields` 渲染

## 契约缺口（详见 design.md §7）

- 双 CRD 树抢注同 group、schema 不兼容；NativeDeviceConfig 下发为 TODO stub；生产控制器仍绑 legacy `api/v1`。

## 关联
- `design.md`、`translation-engine/spec.md`、`actor-transaction/spec.md`、`frontend/spec.md`。
