# frontend — 行为契约（反向还原）

> 反向还原自 `frontend/src/`，忠实 as-built。详见 `design.md`。CRD 驱动为活跃路径。

## 能力概述

Vue3 + Element Plus 平台前端：由 CRD/YANG 模型自动渲染表单/表格/分组，编辑→提交→联动后端下发，展示设备/状态。

## 行为契约

### FE-01 schema 驱动渲染
- **Given** CRD OpenAPIV3Schema
- **When** `parseCRDSchemaToFields(schema)`
- **Then** 逐属性映射为 `Field[]`，类型 enum→select、boolean→switch、number→input-number、object→group；经 `DynamicForm`/`FieldRenderer` 渲染为 Element Plus 控件（R05）

### FE-02 分组与校验
- **Given** Field 带 group/pattern/min/max/required
- **When** 渲染表单
- **Then** >1 分组时 `el-collapse` 折叠；由约束生成校验 rules，校验失败不提交并行内提示（R08）

### FE-03 CRUD 联动
- **Given** 用户在 ConfigPage 增改删
- **When** 提交
- **Then** 经 `useConfigPage`→`useK8sCRD` 调 K8s API（create/replace/delete custom object）；名字 `${device}-${module}-${timestamp}`

### FE-04 原生模块 schema
- **Given** 原生模块（NativeDeviceConfig）
- **When** `getSchema`
- **Then** 从后端 `GET /api/v1/yang/schema/${module}` 取预建 fields 渲染

### FE-05 实时同步
- **Given** CR 列表页
- **When** 挂载
- **Then** `useK8sCRD` List + Watch(NDJSON 流)，3s 自动重连；`stores/device`、`stores/menu` 承载设备/菜单态

## 契约缺口（详见 design.md §8）

- 两代动态表单并存，旧 `components/yang/*` 静态路径未接路由；K8sClient 依赖外部 proxy（kubectl / 后端 `/api/k8s`）。

## 关联
- `design.md`、`business-crd/spec.md`（schema 源）、`yang-api`/`devices-api`/`config-api`、`frontend-yang-dynamic-form` 技能。
