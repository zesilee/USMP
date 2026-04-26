# VLAN 管理前端 - 设计文档

## 一、当前前端架构评估

### 1.1 技术栈现状
| 层级 | 技术选型 | 状态 |
|------|----------|------|
| 框架 | Vue 3.4 + TypeScript 5.3 | ✅ 良好 |
| 构建工具 | Vite 5.0 | ✅ 良好 |
| UI 组件库 | Element Plus 2.5 | ✅ 基础可用，需优化 |
| HTTP 客户端 | Axios 1.6 | ✅ 良好 |

### 1.2 当前架构问题
**问题 1：动态表单未真正实现**
- `DynamicForm.vue` 目前仅展示原始 JSON 文本，没有结构化渲染
- `YangNodeRenderer.vue` 仅支持基础类型，不支持 list/container 复杂结构

**问题 2：界面缺乏专业网管风格**
- 配色单一（只有 Element Plus 默认蓝色）
- 缺少状态可视化（在线/离线、同步状态、缓存状态）
- 布局过于简单，缺乏层次感和深度
- 被评价为"有 AI 味"——过于模板化，缺少产品化的细节打磨

**问题 3：缺少专用业务组件**
- VLAN 是列表型数据，需要表格+表单的复合交互
- 端口关联需要专门的多选/穿梭组件
- 没有批量操作支持

### 1.3 优化方向
- **现代化网管风格**：深色主题 + 专业数据可视化 + 微交互动效
- **结构化表单渲染**：真正实现 YANG → UI 的动态映射
- **VLAN 专用组件**：表格列表 + 侧滑表单 + 端口选择器
- **状态系统**：设备在线状态、配置同步状态、缓存状态可视化

---

## 二、VLAN 功能需求定义

### 2.1 核心功能清单

| 功能 | 优先级 | 说明 |
|------|--------|------|
| VLAN 列表展示 | P0 | 表格展示所有 VLAN，支持分页、排序、搜索 |
| 创建 VLAN | P0 | 侧滑表单，填写 VLAN ID、名称、状态 |
| 编辑 VLAN | P0 | 修改 VLAN 名称、管理状态 |
| 删除 VLAN | P0 | 单条/批量删除，二次确认 |
| 端口关联配置 | P0 | tagged/untagged 端口批量选择 |
| 配置刷新 | P0 | 强制从设备拉取最新配置 |
| 配置下发 | P0 | 提交变更到交换机 |
| 配置状态同步 | P1 | 显示 desired vs actual 差异 |
| 批量导入/导出 | P2 | JSON/Excel 格式 |

### 2.2 数据结构（基于 OpenConfig VLAN）

```typescript
// VLAN 列表项
interface VlanItem {
  id: number           // VLAN ID (1-4094)
  name: string         // VLAN 名称
  adminStatus: 'UP' | 'DOWN'  // 管理状态
  operStatus: 'ACTIVE' | 'INACTIVE' | 'SUSPENDED'  // 运行状态
  taggedPorts: string[]    // tagged 端口列表
  untaggedPorts: string[]  // untagged 端口列表
}

// VLAN 表单数据
interface VlanFormData {
  id: number | null
  name: string
  adminStatus: 'UP' | 'DOWN'
  taggedPorts: string[]
  untaggedPorts: string[]
}
```

---

## 三、界面设计规范（现代化、去 AI 味）

### 3.1 设计原则

**专业网管风格**：
- 深色主题为基础，减少视觉疲劳
- 数据可视化优先，状态一目了然
- 高密度信息展示，但保持可读性
- 减少花哨动效，增加功能性微交互

**去 AI 味关键措施**：
1. ❌ 避免过度对称、过于规整的布局
2. ✅ 增加信息层级差异（标题、辅助文字、标签样式区分）
3. ✅ 使用真实网管产品的配色体系（深蓝+青绿+橙黄告警）
4. ✅ 增加手写感的细节（图标、阴影、间距不绝对统一）
5. ✅ 状态标签使用不同形状（不全部是圆角矩形）

### 3.2 配色系统

```scss
// 主色调 - 专业深蓝系
$color-primary: #165DFF;      // 主蓝 - 操作按钮
$color-primary-light: #4080FF;
$color-primary-dark: #0E42D2;

// 功能色
$color-success: #00B42A;      // 正常/在线
$color-warning: #FF7D00;      // 告警/同步中
$color-error: #F53F3F;        // 错误/离线
$color-info: #86909C;         // 辅助信息

// 中性色 - 深色背景
$bg-page: #0F172A;            // 页面背景
$bg-card: #1E293B;            // 卡片背景
$bg-hover: #334155;           // 悬停背景
$border-color: #475569;       // 边框

// 文字
$text-primary: #F1F5F9;       // 主文字
$text-secondary: #94A3B8;     // 辅助文字
$text-disabled: #64748B;      // 禁用文字
```

### 3.3 布局结构

```
┌─────────────────────────────────────────────────────────────┐
│  [Logo] 交换机设备管理平台    [设备状态]  [用户]  [设置]    │ ← Header
├──────────┬──────────────────────────────────────────────────┤
│          │  ┌─────────────────────────────────────────────┐ │
│ 设备树   │  │  VLAN 列表                                  │ │
│ └─ 192.168.1.1  [在线]    [刷新] [新增 VLAN] [批量删除]  │ │
│    ├─ Interfaces         ┌─────────────────────────────┐  │ │
│    ├─ VLANs   [当前]     │  VLAN ID │ 名称    │ 状态   │  │ │
│    └─ System             │    1     │ default │ ACTIVE │  │ │
│                         │   10     │ Mgmt    │ ACTIVE │  │ │
│ 端口列表                │   20     │ User    │ DOWN   │  │ │
│ └─ GigabitEthernet...   └─────────────────────────────┘  │ │
│                                                             │ │
│ [设备离线提示条]                                              │ │
└──────────┴──────────────────────────────────────────────────┘
```

---

## 四、VLAN 组件设计

### 4.1 组件层级结构

```
VlanManager.vue (页面容器)
├── VlanTable.vue (VLAN 列表表格)
│   ├── StatusBadge.vue (状态标签 - 不同形状)
│   └── TableActions.vue (行内操作按钮组)
├── VlanFormDrawer.vue (侧滑表单)
│   ├── PortSelector.vue (端口选择器 - 穿梭框)
│   └── YangNodeRenderer.vue (基础字段渲染)
├── ConfigDiffViewer.vue (配置差异对比)
└── Toolbar.vue (顶部操作栏)
```

### 4.2 核心组件详细设计

#### 4.2.1 VlanTable.vue - VLAN 列表表格

**功能**：
- 支持分页（默认 20 条/页）
- 支持按 VLAN ID、名称排序
- 支持搜索过滤（按名称或 ID）
- 行内快捷操作（编辑、删除、查看端口）
- 多选支持（批量操作）
- 状态标签差异化展示

**差异化设计（去 AI 味）**：
- 运行状态 ACTIVE = 绿色小圆点 + 文字
- 运行状态 SUSPENDED = 黄色三角形 + 文字
- 管理状态 DOWN = 灰色矩形标签
- 不全部使用统一的 el-tag 样式

#### 4.2.2 VlanFormDrawer.vue - 侧滑编辑表单

**布局（从右侧滑出，宽度 600px）**：
```
┌──────────────────────────────────┐
│  新建 VLAN                  [×]  │
├──────────────────────────────────┤
│  VLAN ID  [_______]  (1-4094)   │ ← 必填，创建后不可改
│                                  │
│  名称     [________________]     │ ← 输入框
│                                  │
│  管理状态  ● UP  ○ DOWN         │ ← 单选按钮
│                                  │
│  Tagged 端口                     │
│  ┌────────────────────────────┐ │
│  │ 可用端口     │ 已选端口    │ │ ← 穿梭框
│  │ GE0/1     >> │ GE0/2       │ │
│  │ GE0/3        │ GE0/4       │ │
│  └────────────────────────────┘ │
│                                  │
│  Untagged 端口                   │
│  ┌────────────────────────────┐ │
│  │ 同上...                      │ │
│  └────────────────────────────┘ │
│                                  │
│  [取消]              [保存]      │
└──────────────────────────────────┘
```

#### 4.2.3 PortSelector.vue - 端口选择器

**功能**：
- 左侧显示设备所有可用端口
- 右侧显示已选端口
- 支持搜索、全选、反选
- 端口按类型分组（GE、10GE、Eth）
- 显示端口当前所属 VLAN 信息（悬浮提示）

---

## 五、交互流程设计

### 5.1 新建 VLAN 流程

```
1. 点击 [新增 VLAN] 按钮
   ↓
2. 侧滑表单打开，光标聚焦 VLAN ID 输入框
   ↓
3. 输入 VLAN ID (1-4094) → 实时校验重复
   ↓
4. 填写名称、选择管理状态
   ↓
5. 选择 tagged/untagged 端口（可选）
   ↓
6. 点击 [保存]
   ├─ 成功 → 提示"VLAN 创建成功"，表单关闭，列表刷新
   └─ 失败 → 表单上方显示错误信息，保留已输入内容
```

### 5.2 配置下发流程

```
1. 修改 VLAN 配置后点击 [提交配置]
   ↓
2. 显示确认弹窗，展示变更摘要
   ┌─────────────────────────────────┐
   │  确认下发配置到设备？            │
   │                                 │
   │  变更内容：                      │
   │  • VLAN 10: 名称 → "Management" │
   │  • VLAN 20: 状态 UP → DOWN      │
   │                                 │
   │      [取消]      [确认下发]     │
   └─────────────────────────────────┘
   ↓
3. 确认后显示加载状态，按钮禁用
   ↓
4. 下发完成显示结果：成功/失败详情
```

---

## 六、API 接口定义

### 6.1 VLAN 相关 API

```typescript
// 获取设备 VLAN 列表
GET /api/v1/config/{deviceIp}/vlans
Response: {
  success: boolean
  data: {
    vlans: VlanItem[]
    fromCache: boolean
    lastSync: string  // ISO 时间戳
  }
}

// 创建 VLAN
POST /api/v1/config/{deviceIp}/vlans
Body: VlanFormData

// 更新 VLAN
PUT /api/v1/config/{deviceIp}/vlans/{vlanId}
Body: Partial<VlanFormData>

// 删除 VLAN
DELETE /api/v1/config/{deviceIp}/vlans/{vlanId}

// 批量删除 VLAN
DELETE /api/v1/config/{deviceIp}/vlans/batch
Body: { ids: number[] }

// 获取设备端口列表（用于端口选择器）
GET /api/v1/devices/{deviceIp}/ports
Response: {
  success: boolean
  data: {
    ports: Array<{
      name: string
      type: 'GE' | '10GE' | 'Eth'
      status: 'UP' | 'DOWN'
      currentVlan?: number
    }>
  }
}
```

---

## 七、迭代开发计划

### 迭代 1：基础样式与布局重构（优先级：P0）
- 目标：建立现代化深色主题，重构整体布局
- 输出文件：
  - `web/src/styles/variables.scss` - 配色变量
  - `web/src/styles/theme.scss` - 主题样式
  - `web/src/App.vue` - 新布局 Header + Sider + Content
- 预计代码量：~200 行

### 迭代 2：VLAN 列表表格组件（优先级：P0）
- 目标：实现 VLAN 数据表格展示
- 输出文件：
  - `web/src/components/vlan/VlanTable.vue`
  - `web/src/components/vlan/StatusBadge.vue`
  - `web/src/types/vlan.ts` - VLAN 类型定义
- 预计代码量：~250 行

### 迭代 3：侧滑表单与基础字段（优先级：P0）
- 目标：实现 VLAN 创建/编辑表单
- 输出文件：
  - `web/src/components/vlan/VlanFormDrawer.vue`
  - `web/src/components/vlan/Toolbar.vue`
- 预计代码量：~280 行

### 迭代 4：端口选择器组件（优先级：P0）
- 目标：实现 tagged/untagged 端口批量选择
- 输出文件：
  - `web/src/components/vlan/PortSelector.vue`
  - `web/src/api/vlan.ts` - VLAN 专用 API
- 预计代码量：~200 行

### 迭代 5：交互完善与状态展示（优先级：P1）
- 目标：增加加载状态、错误提示、成功反馈
- 输出文件：
  - `web/src/components/vlan/ConfigDiffViewer.vue`
  - 各组件交互动效优化
- 预计代码量：~180 行

**总计：5 个迭代，约 1110 行，每个迭代均 ≤ 500 行**

---

## 八、验收标准

### 8.1 功能验收
- ✅ VLAN 列表可正常展示，包含所有字段
- ✅ 可创建新 VLAN，ID 范围校验（1-4094）正常
- ✅ 可编辑现有 VLAN 的名称和状态
- ✅ 可删除单个/批量删除 VLAN，有二次确认
- ✅ 端口选择器可正常选择 tagged/untagged 端口
- ✅ 配置提交可正常下发到后端
- ✅ 刷新按钮可强制从设备获取最新配置

### 8.2 设计验收
- ✅ 深色主题，配色符合网管专业风格
- ✅ 状态标签样式差异化，不全部是统一圆角
- ✅ 信息层级清晰，主次分明
- ✅ 无过度对称、模板化的 AI 风格特征
- ✅ 微交互自然流畅，不花哨
