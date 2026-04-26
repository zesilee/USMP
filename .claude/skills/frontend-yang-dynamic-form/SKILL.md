---
name: frontend-yang-dynamic-form
description: YANG模型驱动的动态表单渲染框架。基于Vue3+Element-Plus，自动将YANG Schema映射为表单、表格、分组面板。新增YANG模块零前端代码。激活条件：前端开发、YANG配置页面、VLAN/Interfaces/System等设备管理功能、硬编码表单重构。
---

# Frontend YANG Dynamic Form Skill

## 一、激活时机 (Auto-Activation Triggers)

### ✅ 自动激活场景
1. **功能开发**：需要开发 VLAN、Interfaces、System 等 YANG 配置页面时
2. **重构需求**：现有硬编码表单需要改为模型驱动架构时
3. **新增模块**：后端新增 YANG 模块，需要配套前端页面时
4. **关键词触发**：动态表单、YANG页面、配置管理、设备管理、网元配置

### ❌ 不激活场景
- 纯静态展示页面
- 与 YANG 模型无关的业务表单
- 简单的 CRUD 列表（无需 YANG 模型支持）

---

## 二、核心架构原则

### 数据流模型
```
┌───────────────────────────────────────────────────────────┐
│                    前端应用层                                │
│  ┌─────────────────────────────────────────────────────┐  │
│  │  YangRenderer (统一入口)                              │  │
│  │    ├─ YangPanel  (container → 分组卡片)              │  │
│  │    ├─ YangTable  (list → 表格 + 增删)                │  │
│  │    └─ YangField  (leaf → 字段组件分派)               │  │
│  │         ├─ YangSwitch   (boolean)                    │  │
│  │         ├─ YangSelect   (enum)                       │  │
│  │         └─ YangInput    (string/int/uint)            │  │
│  └─────────────────────────────────────────────────────┘  │
│                              ↓                              │
│  ┌─────────────────────────────────────────────────────┐  │
│  │              Yang Schema 类型系统                     │  │
│  │  src/types/yang-schema.ts                             │  │
│  │  • YangNode 节点定义                                   │  │
│  │  • SchemaRegistry 注册表                                │  │
│  │  • 字段验证、默认值、类型转换工具                      │  │
│  └─────────────────────────────────────────────────────┘  │
│                              ↓                              │
└───────────────────────────────────────────────────────────┘
                              ↓ HTTP
┌───────────────────────────────────────────────────────────┐
│                    后端 API 层                              │
│  GET  /api/v1/config/{deviceIp}/{yangPath}                │
│  POST /api/v1/config/{deviceIp}/{yangPath}                │
│  GET  /api/v1/schema/{yangPath}  (可选)                   │
└───────────────────────────────────────────────────────────┘
```

### 类型自动映射规则

| YANG 类型 | 前端组件 | 验证规则 |
|-----------|---------|---------|
| `boolean` | ElSwitch 开关 | - |
| `enumeration` | ElSelect 下拉框 | 枚举值范围校验 |
| `string` | ElInput 输入框 | 长度范围校验 |
| `int8/16/32/64` | ElInputNumber 数字输入 | 数值范围校验 |
| `uint8/16/32/64` | ElInputNumber 数字输入 | 非负 + 范围校验 |
| `list` | ElTable 表格 + 新增/删除按钮 | 子节点递归渲染 |
| `container` | ElCard 分组面板 | 子节点递归渲染 |
| `leafref` | ElSelect 关联选择 | 关联数据完整性校验 |
| `empty` | ElCheckbox | - |

### 字段命名自动适配

自动处理 YANG `kebab-case` 到 JavaScript `camelCase` 的转换：

```
YANG字段: admin-status → 转换为 adminStatus
YANG字段: tagged-ports → 转换为 taggedPorts
```

**注意**：Schema 定义使用 YANG 原生命名，数据从 API 返回时自动兼容。

---

## 三、标准开发流程

### 前置条件检查
执行前确认：
- ✅ 后端 YANG 模型已定义
- ✅ 后端 Reconciler API 已实现
- ✅ 配置 API 接口已就绪

### 开发步骤 (新增 YANG 模块)

#### Step 1: 注册 Schema (5分钟)

**文件**：`web/src/types/yang-schema.ts`

```typescript
// 1. 新增 Schema 定义
export const YOUR_MODULE_SCHEMA: YangNode = {
  path: '/your-module',
  name: 'yourModule',
  type: 'container',
  description: '模块描述',
  config: true,
  children: [
    // 子节点定义...
  ]
}

// 2. 注册到注册表
export const SCHEMA_REGISTRY: Record<string, YangNode> = {
  '/vlans': VLAN_SCHEMA,
  '/interfaces': INTERFACES_SCHEMA,
  '/your-module': YOUR_MODULE_SCHEMA,  // ← 新增
}
```

#### Step 2: 页面路由 (2分钟)

**文件**：`web/src/App.vue`

```vue
<!-- 在 content-wrapper 内添加 -->
<YangRenderer
  v-if="currentYangPath === '/your-module'"
  :yang-path="currentYangPath"
  :device-ip="currentDevice.ip"
/>
```

#### Step 3: 验证测试 (3分钟)

1. 运行单元测试：`npm run test`
2. 运行 E2E 测试：`npm run e2e`
3. 手动验证页面渲染

**完成！** 总计约 10 分钟，**零业务代码**。

---

## 四、核心组件参考

### 组件目录结构
```
web/src/components/yang/
├── YangRenderer.vue      # 统一入口调度器 ✨
│   ├── 职责：加载 schema、获取数据、分派渲染
│   ├── 支持：嵌套递归渲染、加载/错误状态
│   └── API：props { yangPath, deviceIp, rootSchema, rootData }
│
├── YangPanel.vue         # container 类型 → 分组卡片
│   ├── 职责：提供分组标题、只读标签、卡片样式
│   └── API：props { node }
│
├── YangTable.vue         # list 类型 → 数据表格
│   ├── 职责：表格渲染、新增行、编辑行、删除行
│   ├── 内置：编辑弹窗、表单验证、字段分派
│   └── API：props { node, modelValue, editable? }
│
├── YangField.vue         # 通用字段包装器
│   ├── 职责：根据节点类型分派到具体组件
│   ├── 内置：错误信息展示、禁用状态处理
│   └── API：props { node, modelValue, errorMessage? }
│
├── YangSwitch.vue        # boolean 类型组件
├── YangInput.vue         # string/int/uint 类型组件
└── YangSelect.vue        # enum 类型组件
```

### YangRenderer 组件使用示例

```vue
<template>
  <!-- 最简用法 -->
  <YangRenderer
    yang-path="/vlans"
    :device-ip="deviceIp"
  />

  <!-- 嵌套渲染（内部使用） -->
  <YangRenderer
    :root-schema="childNode"
    :root-data="formData[childNode.name]"
    :device-ip="deviceIp"
  />
</template>
```

---

## 五、与其他技能联动

| 技能 | 联动方式 | 输出物 |
|------|---------|--------|
| **yang-controller-runtime-dev** | 后端完成 Reconciler 后，此技能自动激活 | 配套前端页面 |
| **netconf-sim-integration-test** | 使用模拟器数据完成 E2E 测试验证 | 前后端集成测试用例 |
| **e2e-testing** | 完成页面开发后，自动触发 E2E 测试编写 | Playwright 测试用例 |
| **golang-testing** | 后端 API 测试通过后，触发前端测试 | 完整测试链路 |

---

## 六、开发规范

### 命名规范
- Schema 常量：`MODULE_NAME_SCHEMA` (大写蛇形)
- 路径常量：`/module-name` (kebab-case，与 YANG 一致)
- 文件名：`YangXxx.vue` (帕斯卡命名，前缀 Yang)

### 代码风格
- 使用 `<script setup lang="ts">` 语法
- TypeScript 类型严格，禁止 `any`
- Props 必须提供默认值
- 组件内部状态使用 `ref` / `computed`

### 测试要求
- 新增组件必须有对应的单元测试
- 新增 YANG 模块必须有 E2E 测试覆盖
- 测试必须覆盖：渲染、正常流程、异常场景

---

## 七、验收标准

### 代码质量
✅ **零业务代码**：新增 YANG 模块只需定义 Schema，无需编写表单代码
✅ **类型安全**：所有组件有完整 TypeScript 类型定义
✅ **无重复代码**：通用逻辑完全收敛到 Yang 组件库

### 功能完备
✅ **渲染正确**：所有 YANG 类型能正确映射到对应组件
✅ **表单验证**：基于 Schema 的自动验证生效
✅ **CRUD 完整**：支持新增、编辑、删除、查询操作
✅ **状态展示**：加载中、错误、空数据状态处理完善

### 测试要求
✅ **单元测试**：每个 Yang 组件有单元测试覆盖
✅ **E2E 测试**：每个 YANG 模块有端到端测试
✅ **CI 验证**：所有测试在 CI 环境稳定通过

### 可维护性
✅ **文档完整**：Schema 有详细描述和注释
✅ **向后兼容**：不破坏现有功能，支持渐进迁移
✅ **错误边界**：组件能优雅处理异常数据

---

## 八、常见问题处理

### Q: 后端 API 返回字段名与 Schema 不匹配？
**A**：使用 `kebabToCamel` 自动转换，已内置在 `YangTable` 中。如需自定义转换，扩展 `getFieldValue` 工具函数。

### Q: 需要自定义特殊字段渲染？
**A**：
1. 在 `YangField.vue` 中新增分支逻辑
2. 或在 `YangNode` 类型中添加 `customComponent` 字段
3. 保持向后兼容，不影响现有逻辑

### Q: 如何处理复杂嵌套的 list / container？
**A**：`YangRenderer` 支持递归渲染。复杂嵌套时使用 `rootSchema` / `rootData` 属性传入子节点和对应数据。

### Q: 性能优化建议？
**A**：
- Schema 定义移至单独文件，按需导入
- 大列表使用 el-table 的 `virtual-scroll`
- API 数据缓存策略：30秒内不重复请求
- 考虑使用 `v-memo` 优化表格行渲染
