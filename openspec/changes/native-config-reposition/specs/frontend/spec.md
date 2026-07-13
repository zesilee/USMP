# frontend — 原生配置概念重定位 + legacy CRD 链路退役（delta）

> 附带 Purpose 重写（sync 时执行，见 design D4）：删除「两代下发链路并存」叙述——前端 = 原生配置（YANG 模型驱动直连 Stack B）单链路；「业务网络配置」为未来扩展层（业务侧 YANG 模型定义自动化能力、USMP 编排为原生配置下发，方向见 openspec/tasks/business-network-config.md）。

## MODIFIED Requirements

### Requirement: FE-01 schema 驱动渲染

前端 SHALL 将后端 YANG nested schema 经 `crdSchemaParser` 逐属性映射为 `Field[]`，类型映射为 boolean→switch、number→input-number、object→group；enum SHALL 按选项数与必填性细分：**必填且选项 ≤3 → segmented 分段控件，其余（可选或 >3 选项）→ select 下拉**（可选枚举 SHALL 保留清空能力，清空即该键不入 payload）。映射经 `FieldRenderer` 渲染为 Element Plus 控件（R05）。SHALL NOT 手写固定表单。

#### Scenario: 类型到控件的自动映射
- **WHEN** `getYangSchema(module, 'nested')` 返回带类型的属性
- **THEN** SHALL 生成对应 `Field[]`，并按类型渲染对应控件（boolean→switch、number→input-number、object→分组）

#### Scenario: 必填短枚举分段控件
- **WHEN** enum 字段 `required=true` 且选项数 ≤3
- **THEN** SHALL 渲染分段控件展示全部选项，选中 SHALL 触发值更新；readonly/禁用态 SHALL 透传为控件禁用

#### Scenario: 可选或长枚举保持下拉（边界）
- **WHEN** enum 字段可选（`required=false`）或选项数 >3
- **THEN** SHALL 渲染 select 下拉；可选枚举 SHALL 可清空，清空后该键 SHALL NOT 进入下发 payload

#### Scenario: 无有效 schema
- **WHEN** schema 拉取失败或为空
- **THEN** SHALL NOT 崩溃（R08），页面继续可用，仅不渲染该模块字段

### Requirement: FE-02 分组与校验

Field 带 group/pattern/min/max/required 时，前端渲染 SHALL 按分组组织（>1 分组时用 `el-collapse` 折叠），并由约束生成校验 rules；校验失败 SHALL NOT 提交，且 SHALL 行内提示 YANG 约束（§9、R08）。

#### Scenario: 多分组折叠
- **WHEN** 字段分布在 >1 个 group
- **THEN** SHALL 用 `el-collapse` 折叠分组渲染

#### Scenario: 校验失败不提交
- **WHEN** 存在缺失必填或数值越界（超出 min/max）
- **THEN** SHALL 阻止提交并在行内展示约束提示

### Requirement: FE-04 原生/预建模块 schema

需要预建 fields 的模块 SHALL 经后端 `GET /api/v1/yang/schema/${module}` 取回预建 fields 后渲染，而非在前端硬编码表单结构（R05）。

#### Scenario: 拉取预建 schema
- **WHEN** 调用 `getSchema(module)`
- **THEN** SHALL 从后端 `GET /api/v1/yang/schema/${module}` 取 fields 并据此渲染

### Requirement: FE-13 模型驱动原生配置导航与路由迁移

左侧**原生配置**菜单 SHALL 由 `/yang/modules` 返回的模块列表驱动生成（指向 `/module/:name`），
加载失败 SHALL 回退既有硬编码项（R08）；旧路由 `/config/interface`、`/config/vlan`
SHALL 重定向到对应 `/module/:module`。模块项携带 `category` 时菜单 SHALL 按 category
分组展示；无 `category` 的模块 SHALL 归入默认分组，分组渲染 SHALL NOT 因缺失 category 失败（R08）。
legacy 路由 `/native/:module` 与 `/config/route` SHALL 不存在（Stack A CRD 死路已退役，生产中从未可用，无重定向义务）。

#### Scenario: 菜单生成与回退

- **WHEN** `/yang/modules` 返回含 `ifm`（模块根名）的列表
- **THEN** 菜单 SHALL 含指向 `/module/ifm` 的项
- **WHEN** 该 API 失败
- **THEN** 菜单 SHALL 显示回退项且不崩

#### Scenario: 任务域分组

- **WHEN** 模块列表含带 `category` 与不带 `category` 的模块
- **THEN** 菜单 SHALL 按 category 分组展示带值模块，无值模块 SHALL 归入默认分组且渲染不失败

#### Scenario: 菜单命名与概念对齐

- **WHEN** 渲染左侧导航
- **THEN** 模块控制台菜单组标题 SHALL 为「原生配置」，SHALL NOT 存在指向 `/native/*` 的菜单项

## REMOVED Requirements

### Requirement: FE-05 实时同步（legacy CRD watch）

**Reason**: Stack A 前端 CRD 链路退役——`useK8sCRD` List+Watch 依赖已退出生产的 K8s API 面（外部 proxy 缺失即不可用），消费者 `ConfigPage.vue` 已删除。设备/菜单状态由 `stores/device`、`stores/menu` 承载的部分并入既有主链路要求，无独立 CRD watch 语义。
**Migration**: 配置读写与对账走 FE-03 主链路（`POST /api/v1/config/:ip/*path` + 轮询对账）；无 CRD watch 替代品（无此需求）。

### Requirement: FE-06 legacy CRD 配置 CRUD（次要，legacy）

**Reason**: 本 requirement 自述「SHOULD 逐步收敛到 FE-03 主链路」——收敛完成：`ConfigPage.vue`/`useConfigPage`/`useK8sCRD`/`K8sClient` 整链删除，`/native/:module`、`/config/route` 路由移除（生产中从未可用）。
**Migration**: 全部模块配置走 FE-03 主链路 + FE-10 通用模块控制台（`/module/:name`）。
