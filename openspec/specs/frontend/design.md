# frontend — 前端架构设计（反向还原）

> **权威性**：CRD/OpenAPI-schema 驱动的动态表单为**活跃路径**；旧 `components/yang/*` 静态 YANG-schema 路径**未接路由**（死代码，迁移债 D9）。
> **还原基准**：`main@b1cfbae`，代码根 `frontend/src/`。

## 1. 职责

由 YANG/CRD 模型**自动渲染**设备管理界面（R05：禁止手写固定表单）：schema → 表单/表格/分组面板；编辑 → 提交 → 联动后端下发；展示设备/缓存/下发/异常状态。

## 2. 技术栈（`package.json` / `vite.config.ts`）

Vue 3 `^3.4` + Element Plus `^2.5` + Pinia `^3` + Axios `^1.16` + vue-router `^5` + echarts `^6`；构建 Vite `^5`（dev :3000，代理 `/api`→`:8080`）；测试 Vitest + `@vue/test-utils` + happy-dom，E2E Playwright。`main.ts` 全局注册 Pinia/router/ElementPlus。

## 3. ⚠️ 两代动态表单并存

### 3.1 活跃路径：CRD/OpenAPI 驱动
```
K8s CRD OpenAPIV3Schema
  → parseCRDSchemaToFields(schema)         utils/crdSchemaParser.ts:38
       读 schema.properties.spec.properties，逐属性映射为 Field
       类型映射 mapK8sTypeToFieldType:112 (enum/boolean/number/object→group/string)
       厂商扩展 x-custom-label/group/placeholder/readonly/hidden
  → Field[] → DynamicForm.vue                components/config/DynamicForm.vue
       >1 分组时 el-collapse；由 pattern/min/max/required 生成校验 rules
  → FieldRenderer.vue（按 type→el-input/-number/switch/select，group 递归自身）
  → DynamicTable.vue（列表视图）
```
这是 router → `ConfigPage.vue` 实际走的链路。

### 3.2 legacy 路径：静态 YANG-schema（未接路由）
`components/yang/YangRenderer.vue` 由静态注册表 `types/yang-schema.ts`（`getSchemaByPath`/`getDefaultValue`/`convertKeysToKebab`）驱动，经 `YangPanel`/`YangTable`/`YangField` 渲染，直读写设备 `getConfig`/`setConfig`。**不被活跃路由引用**——待清理。

## 4. 组合式函数（composables）

- `useConfigPage.ts` — 配置页统一大脑。`useConfigPage(module)` 按硬编码 `BUSINESS_CRDS` map（`:7`，vlan/interface/route/switch→`biz.usmp.io/v1`）分支：
  - 业务模块 → 委托 `useK8sCRD`，`getSchema()` 拉 CRD schema 再 `parseCRDSchemaToFields`（`:27`）；`listByDevice` 按 `spec.deviceID` 过滤（`:43`）。
  - 原生模块 → `useK8sCRD('core.usmp.io','v1','nativedeviceconfigs')`，但 schema 来自后端 YANG API `GET /api/v1/yang/schema/${module}`（预建 fields，`:65`）。
  - `useNativeModules()`（`:105`）拉 `GET /api/v1/yang/modules` 按 vendor 分组。
- `useK8sCRD.ts` — 基于 fetch 的 `K8sClient`：list/get/create/replace/delete custom objects、`getCRD`(OpenAPI schema)、K8s watch NDJSON 流 + 3s 自动重连（`:201`）、Vue 生命周期自动 list/watch。
- `useDeviceConfig.ts` — 旧组合式（基于 `api/crd.ts`），服务于 legacy yang/ 路径。

## 5. API 层

- `api/index.ts` — Axios 实例，`baseURL = VITE_API_URL || http://localhost:8080/api/v1`；`listDevices` `GET /devices`、`getDeviceStatus /devices/{ip}/status`、`getConfig/setConfig GET|POST /config/{ip}/{path}`、`getSchema /schema/{path}`。对齐后端北向 `devices/config/yang` API。
- `api/crd.ts` — 独立 Axios `baseURL=/api/crd`，含 SSE `watchConfigs`(EventSource)（legacy 路径用）。
- `api/logs.ts` — `/api/logs`。
- 直接 `fetch`：YANG schema/modules（`useConfigPage.ts:69,113`）、K8s API（`useK8sCRD.ts`）、`stores/menu.ts:21`。

> **k8s client-node 历史**：曾集成 `@kubernetes/client-node`（commit 45ad884），后 commit 620f70c 因浏览器兼容**移除**该依赖，改写 `useK8sCRD.ts` 为浏览器原生 fetch 的 `K8sClient`，靠 kubectl proxy(dev)/后端 `/api/k8s` proxy(prod)（`getDefaultBaseUrl:140`）。**当前构建无 `@kubernetes/client-node`**。

## 6. 页面 / 路由（`router/index.ts`）

| 路由 | 视图 | 说明 |
|------|------|------|
| `/` | Dashboard.vue | StatCard + echarts StatusChart + 日志表 |
| `/devices` | Devices.vue | 设备列表 el-table，搜索/测连 |
| `/config/interface\|vlan\|route` | ConfigPage.vue | props.module=`openconfig-*`，配置编辑器 |
| `/native/:module` | ConfigPage.vue | 动态原生模块 |
| `/logs` `/settings` | Logs/Settings.vue | |

`ConfigPage.vue` = 配置编辑器：设备选择器 + `StatusBadge`(phase) + `DynamicTable`(列表) + `DetailDrawer`+`DynamicForm`(增改)；增改删经 `useConfigPage`，名字 `${device}-${module}-${Date.now()}`（`:171`）。布局 `MainLayout.vue`(Header+Sidebar+router-view)，`Sidebar.vue` 动态原生子菜单按 vendor 分组。

## 7. 状态管理（仅两个 Pinia store）

- `stores/device.ts` `useDeviceStore`：`devices/selectedDevice/isLoading`；getter `onlineCount/offlineCount`；action `fetchDevices`(`GET /api/devices`)/`testConnection`/`selectDevice`。
- `stores/menu.ts` `useMenuStore`：`nativeModels/nativeMenuLoaded/isCollapsed`；`loadNativeModels`(`GET /api/v1/yang/modules`，去重+huawei 回退)；getter `groupedByVendor`。
- 架构注记：设备/菜单态在 Pinia，但**每页 CRD 配置态在 `useConfigPage`/`useK8sCRD` 的局部 ref**，不入 store。

## 8. as-built 缺口

| 缺口 | 位置 |
|------|------|
| 两代动态表单并存，旧 yang/ 路径未接路由 | `components/yang/*`、`components/DynamicForm.vue`、`useDeviceConfig` |
| `BUSINESS_CRDS` 模块→group 硬编码 | `useConfigPage.ts:7` |
| K8sClient 依赖外部 proxy（dev kubectl / prod `/api/k8s`） | `useK8sCRD.ts:140` |

## 9. 红线对照

- **R05 YANG 自动渲染**：✅ 活跃路径 schema→表单全自动，零硬编码表单。
- **R11/R12 反 AI 陈词滥调/emoji 图标**：暗色网管风，未见紫粉蓝渐变；图标用法建议后续按 `web-design-engineer` 复核。

## 10. 关联
- `frontend-yang-dynamic-form` 技能；`business-crd/design.md`（CRD schema 来源）；`yang-api`/`devices-api`/`config-api`（后端接口）；`spec/vlan-frontend-design.md`、`docs/superpowers/specs/2026-05-03-frontend-design.md`（UX 设计参考）。
