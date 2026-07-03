# Grid YANG UI Schema 设计

## 背景

当前前端已有 `YangRenderer`、`YangTable`、`YangField` 等 YANG 感知组件，并在 `frontend/src/types/yang-schema.ts` 中维护本地 `YangNode` schema、枚举、默认值和校验规则。这个模式让新增 YANG 模块时仍需要前端理解模型细节，不符合“控件数据由后端通过 YANG 对象转化”的目标。

本设计将 Interfaces 作为第一期垂直切片：后端根据 YANG/ygot、设备 capability、设备元信息和模块集合生成完整 Grid UI schema，前端只按 schema 渲染，不再推导 YANG 语义。

## 目标

- 后端生成完整控件数据：布局、控件类型、默认值、枚举、校验规则、schema 版本。
- 前端通过通用 GridRenderer 渲染 UI schema，不导入 YANG schema，不维护枚举或校验规则。
- Interfaces 页面先完成端到端链路：加载 schema、渲染接口列表、抽屉编辑、提交 apply、展示错误和结果。
- 保留旧 `YangRenderer`，不在第一期替换 VLAN/System。
- 不引入数据库，不改变现有 NETCONF、缓存和 controller-runtime 架构。

## 非目标

- 不一次性替换所有配置页面。
- 不让前端继续从 YANG path 推导控件或校验规则。
- 不在第一期实现复杂可视化设计器。
- 不把运行配置存入数据库。

## 架构边界

### 后端职责

后端新增 UI Schema Generator，负责：

- 从 YANG/ygot 模型生成 Grid UI schema。
- 根据优先级裁剪节点：设备实际 capability > 设备元信息 > 后端模块集合兜底。
- 选择控件类型、分区、默认值、枚举、必填、range、length、pattern 等规则。
- 生成稳定的 `schemaVersion`。
- 校验 `{ schemaVersion, values }`。
- 将 Grid values 反解为 ygot/YANG 对象。
- 通过现有 NETCONF/Controller 流程下发配置并失效缓存。

### 前端职责

前端新增 GridRenderer 组件族，负责：

- 按后端 UI schema 渲染布局和控件。
- 管理输入状态、抽屉编辑、提交按钮和刷新按钮。
- 展示字段级错误、页面级错误和下发结果。
- 提交 `{ schemaVersion, values }` 到后端 apply API。

前端不负责：

- 解析 YANG。
- 推导控件类型。
- 维护枚举、默认值或校验规则。
- 拼接 YANG path 或直接调用旧 `setConfig` 完成下发。

## API 设计

### 获取 Interfaces UI schema

`GET /api/v1/ui-schema/devices/:ip/interfaces`

返回示例结构：

```json
{
  "success": true,
  "data": {
    "schemaVersion": "interfaces:sha256:...",
    "module": "huawei-ifm",
    "targetPath": "/ifm:ifm/ifm:interfaces",
    "capabilitySource": "device-capability",
    "layout": {
      "type": "grid",
      "columns": 12,
      "gap": "md"
    },
    "sections": [
      {
        "id": "interfaces",
        "title": "接口配置",
        "description": "管理设备接口基础配置",
        "widgets": ["interfaces-table"]
      }
    ],
    "widgets": [
      {
        "id": "interfaces-table",
        "type": "table",
        "label": "接口列表",
        "rowKey": "name",
        "grid": { "span": 12 },
        "columns": [
          {
            "id": "name",
            "type": "text",
            "label": "接口名称",
            "readonly": true,
            "validation": { "required": true }
          },
          {
            "id": "description",
            "type": "text",
            "label": "描述",
            "validation": { "maxLength": 80 }
          },
          {
            "id": "mtu",
            "type": "number",
            "label": "MTU",
            "validation": { "min": 1280, "max": 9216 }
          },
          {
            "id": "admin-status",
            "type": "select",
            "label": "管理状态",
            "options": [
              { "label": "启用", "value": 1 },
              { "label": "禁用", "value": 0 }
            ]
          }
        ],
        "binding": {
          "target": "opaque-backend-binding"
        }
      }
    ],
    "values": {
      "interfaces-table": []
    }
  }
}
```

`binding` 是后端反解所需的 opaque metadata。前端可以随请求原样传回，但不能基于它做 YANG 逻辑判断。

### 应用 Interfaces 配置

`POST /api/v1/ui-schema/devices/:ip/interfaces/apply`

请求：

```json
{
  "schemaVersion": "interfaces:sha256:...",
  "values": {
    "interfaces-table": [
      {
        "name": "GigabitEthernet0/0/1",
        "description": "uplink",
        "mtu": 1500,
        "admin-status": 1
      }
    ]
  }
}
```

成功响应：

```json
{
  "success": true,
  "data": {
    "schemaVersion": "interfaces:sha256:...",
    "values": {},
    "lastSync": "2026-05-23T00:00:00Z"
  }
}
```

字段错误响应：

```json
{
  "success": false,
  "code": "VALIDATION_FAILED",
  "message": "配置校验失败",
  "fieldErrors": {
    "interfaces-table:row:GigabitEthernet0/0/1:mtu": ["MTU 必须在 1280 到 9216 之间"]
  }
}
```

schema 过期响应：

```json
{
  "success": false,
  "code": "SCHEMA_VERSION_MISMATCH",
  "message": "Schema 已更新，请刷新后重试"
}
```

## 前端组件设计

### GridRenderer.vue

输入：

- `schema`
- `modelValue`
- `loading`
- `errors`

输出：

- `update:modelValue`
- `submit`
- `refresh`

职责：渲染 toolbar、sections、widgets、错误状态和空状态。

### GridSection.vue

职责：渲染 section 标题、描述、折叠状态和 12 列 grid layout。根据 widget 的 `grid.span`、`grid.offset`、`grid.order` 控制布局。

### GridWidget.vue

职责：按 `widget.type` 分发 Element Plus 控件：

- `text` → `el-input`
- `number` → `el-input-number`
- `select` → `el-select`
- `switch` → `el-switch`
- `textarea` → `el-input type="textarea"`
- `table` → 表格 widget

### InterfaceGridPage.vue

职责：

- 调用 `GET /api/v1/ui-schema/devices/:ip/interfaces`。
- 初始化 Grid values。
- 渲染接口表格和抽屉编辑。
- 调用 apply API 提交 `{ schemaVersion, values }`。
- 展示后端字段级错误、schema 过期错误、设备离线和 NETCONF 下发失败。

## Interfaces 交互设计

第一期采用“接口列表 + 抽屉编辑 + 批量提交”。

- 主视图展示接口列表。
- 表格列完全来自后端 table widget schema。
- 抽屉编辑展示当前行可编辑字段。
- 后续可扩展行内编辑，但第一期不强制实现。
- 用户提交后，后端在同一 apply 入口完成 schema 版本校验、values 反解、YANG 校验、NETCONF 下发和缓存失效。

## 错误处理

- 字段校验错误：后端按 widgetId 或 rowKey + fieldId 返回，前端定位到表格单元格或抽屉字段。
- schema 过期：后端返回 `SCHEMA_VERSION_MISMATCH`，前端提示刷新 schema，保留用户当前输入直到用户确认刷新。
- capability 不支持：后端不返回该 widget，或返回 `disabled: true` 和 `disabledReason`。
- 设备离线或 NETCONF 失败：后端返回页面级错误，前端保留用户输入，不清空表单。
- apply 成功：后端失效缓存并返回最新 values 或 lastSync。

## 测试策略

### 后端单元测试

- YANG/ygot → UI schema：断言 Interfaces table widget、columns、options、validation、schemaVersion。
- capability 裁剪：覆盖实际 capability、设备元信息、模块集合兜底三种来源。
- `{ schemaVersion, values } → ygot`：覆盖正常值、非法 MTU、缺失必填、过期 schemaVersion。

### 后端集成测试

- 基于 NETCONF 模拟网元验证 Interfaces apply 成功下发。
- 覆盖设备离线或 NETCONF edit-config 失败，确认缓存不更新、错误返回清晰。

### 前端组件测试

- `GridRenderer` 能按 schema 渲染 table widget、抽屉字段、校验错误。
- `InterfaceGridPage` 能加载 schema、提交 apply、展示 fieldErrors。
- 前端测试只验证 UI schema 约定，不验证 YANG 细节。

### E2E 测试

- 打开 Interfaces Grid 页面。
- 修改一个接口描述、MTU 或管理状态。
- 提交成功并展示 lastSync。
- 模拟后端校验错误时能定位到字段。

## 迭代拆分建议

1. 后端 UI schema 类型、Interfaces generator 单元测试和最小实现。
2. 后端 Interfaces apply API、schemaVersion 校验和 values → ygot 反解测试。
3. NETCONF 模拟网元集成测试，验证 apply 成功和失败路径。
4. 前端 GridRenderer 组件族测试先行实现。
5. InterfaceGridPage 接入 schema/apply API。
6. E2E 测试覆盖 Interfaces Grid 黄金路径和校验错误。

每个迭代都是一个原子功能，遵循测试先行、小步提交。单次代码输出不超过 500 行。
