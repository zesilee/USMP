# frontend — YANG/CRD 模型驱动的设备配置前端

## Purpose

frontend 是 Vue3 + Element Plus 平台前端：由 YANG/CRD schema **自动渲染**表单/表格/分组（R05，禁止手写固定表单），编辑→校验→提交→联动后端下发，并展示设备/缓存/对账状态。当前存在两代下发链路并存：**主链路为 Stack B 直连**（interface/vlan：`DeviceConfigPage.vue` 经 `useDeviceConfig`/`useConfigSubmit` 直接 `POST /api/v1/config/:ip/*path` 并轮询对账，非 K8s CRD）；**legacy 链路为 K8s CRD**（route/native：`ConfigPage.vue` 经 `useConfigPage`→`useK8sCRD` 调 K8s API，依赖外部 proxy）。两代动态表单（Stack B 的 `FieldRenderer` 直渲 vs legacy 的 `DynamicForm`）暂并存，legacy 链路应逐步收敛到主链路。

## Requirements

### Requirement: FE-01 schema 驱动渲染

前端 SHALL 将 schema（CRD `OpenAPIV3Schema` 或后端 YANG nested schema）经 `crdSchemaParser` 逐属性映射为 `Field[]`，类型 enum→select、boolean→switch、number→input-number、object→group，并经 `DynamicForm`/`FieldRenderer` 渲染为 Element Plus 控件（R05）。SHALL NOT 手写固定表单。

#### Scenario: 类型到控件的自动映射
- **WHEN** `parseCRDSchemaToFields(schema)` 或 `getYangSchema(module, 'nested')` 返回带类型的属性
- **THEN** SHALL 生成对应 `Field[]`，并按类型渲染对应控件（enum→select、boolean→switch、number→input-number、object→分组）

#### Scenario: 无有效 schema
- **WHEN** schema 拉取失败或为空
- **THEN** SHALL NOT 崩溃（R08），页面继续可用，仅不渲染该模块字段

### Requirement: FE-02 分组与校验

Field 带 group/pattern/min/max/required 时，前端渲染 SHALL 按分组组织（>1 分组时用 `el-collapse` 折叠），并由约束生成校验 rules；校验失败 SHALL NOT 提交，且 SHALL 行内提示 YANG/CRD 约束（§9、R08）。

#### Scenario: 多分组折叠
- **WHEN** 字段分布在 >1 个 group
- **THEN** SHALL 用 `el-collapse` 折叠分组渲染

#### Scenario: 校验失败不提交
- **WHEN** 存在缺失必填或数值越界（超出 min/max）
- **THEN** SHALL 阻止提交并在行内展示约束提示

### Requirement: FE-03 配置下发主链路（Stack B 直连）

interface / vlan 配置 SHALL 走 Stack B 直连主链路：`DeviceConfigPage.vue` 用 YANG schema 渲染模型驱动表单，编辑→校验通过→提交时 SHALL 经 `useConfigSubmit` 调 `setConfig` 直接 `POST /api/v1/config/:ip/*path`（**非 K8s CRD**），随后 SHALL 以 `force_refresh` 强制回读实际态、轮询单设备 reconcile 结局，驱动 pushing→reading→converged/drifted/error/timeout 进度。下发失败 SHALL 降级、不误报成功（R08）。

#### Scenario: 编辑并下发触发对账
- **WHEN** 用户在 `DeviceConfigPage` 提交一条合法（校验通过）配置
- **THEN** SHALL `POST /config` 下发 → `force_refresh` 回读 → 轮询 `getDeviceReconcile`，直到出现推进过 baseline 的终态（收敛/漂移/失败）或超时

#### Scenario: 下发失败降级
- **WHEN** `setConfig` 报错
- **THEN** SHALL 置 error 相位、SHALL NOT 重读列表、保留原表单，不崩溃（R08）

#### Scenario: 对账超时
- **WHEN** 轮询达到上限仍无终态
- **THEN** SHALL 标注 `timedOut` 停在 reading 相位，SHALL NOT 误报成功

### Requirement: FE-04 原生/预建模块 schema

原生模块（`NativeDeviceConfig`）及需要预建 fields 的模块 SHALL 经后端 `GET /api/v1/yang/schema/${module}` 取回预建 fields 后渲染，而非在前端硬编码表单结构（R05）。

#### Scenario: 拉取预建 schema
- **WHEN** 调用 `getSchema(module)`
- **THEN** SHALL 从后端 `GET /api/v1/yang/schema/${module}` 取 fields 并据此渲染

### Requirement: FE-05 实时同步（legacy CRD watch）

legacy CRD 列表页 SHALL 在挂载时经 `useK8sCRD` 执行 List + Watch（NDJSON 流），watch 断线时 SHALL 3s 自动重连；`stores/device`、`stores/menu` SHALL 承载设备/菜单状态。

#### Scenario: 挂载即 List+Watch
- **WHEN** legacy CR 列表页挂载
- **THEN** SHALL 先 List 拉取快照，再建立 Watch 订阅增量事件（ADDED/MODIFIED/DELETED）

#### Scenario: watch 断线重连
- **WHEN** watch 连接报错中断
- **THEN** SHALL 在 3s 后自动重建 watch，SHALL NOT 崩溃（R08）

### Requirement: FE-06 legacy CRD 配置 CRUD（次要，legacy）

route / native 配置 SHALL 走 legacy K8s CRD 链路：`ConfigPage.vue` 经 `useConfigPage`→`useK8sCRD` 调 K8s API（create/replace/delete custom object），对象名 SHALL 为 `${device}-${module}-${timestamp}`；`K8sClient` 依赖外部 proxy（kubectl proxy / 后端 `/api/k8s`）。此链路为过渡遗留，SHOULD 逐步收敛到 FE-03 主链路，SHALL NOT 扩展为新模块的默认下发路径。

#### Scenario: CRD 增改删
- **WHEN** 用户在 `ConfigPage` 提交增/改/删
- **THEN** SHALL 经 `useK8sCRD` 调用对应 K8s custom object 的 create/replace/delete，并按 `${device}-${module}-${timestamp}` 命名新对象

#### Scenario: 缺 proxy 降级
- **WHEN** K8s API 不可达（外部 proxy 缺失或返回非 2xx）
- **THEN** SHALL 展示错误、SHALL NOT 崩溃（R08）

### Requirement: FE-07 约束引擎（when 显隐 / must 校验）

前端 SHALL 提供**通用**约束引擎（`utils/xpathEval` + `composables/useConstraintEngine`），把 schema 中的 `when`/`must` XPath 表达式求值为运行时行为，SHALL NOT 硬编码任何厂商/模型/字段名。求值器 SHALL 为自研 YANG XPath 子集解析器（相对路径 `../leaf`、`= != > < >= <=`、`and`/`or`/`not()`、`mod`、字面量），SHALL NOT 引入 `eval`/`safe-eval` 等依赖（R10）。表达式解析失败 SHALL 降级（when 失败=字段可见、must 失败=不阻断）并记录告警，SHALL NOT 崩溃（R08）。

#### Scenario: when 驱动显隐
- **WHEN** 字段带 `when`（如 `../class='sub-interface'`），用户改动被引用字段的值
- **THEN** 引擎 SHALL 实时重算该字段 `visible`；`visible=false` 的字段 SHALL 隐藏且 SHALL NOT 参与提交与校验

#### Scenario: must 阻断非法提交
- **WHEN** 字段带 `must`（如 `(../suppress>../reuse)` 或 `(../interval) mod 10 = 0`）且当前表单违反该约束
- **THEN** 引擎 SHALL 返回违例，前端 SHALL 阻止提交并行内提示（message 取 YANG `description` 或生成的通用提示）

#### Scenario: 表达式语法错误降级
- **WHEN** `when`/`must` 表达式无法被求值器解析
- **THEN** SHALL 降级（可见 / 不阻断）并记录告警，页面 SHALL NOT 崩溃（R08）

### Requirement: FE-08 choice/case 渲染

`FieldRenderer` SHALL 将 `type:"choice"` 的字段渲染为互斥切换控件（任一 case 含多字段→`el-tabs`，所有 case 均为单叶→`el-radio-group`），分支内子字段按 `cases[].fields` 递归渲染。切换到某 case 时 SHALL 清空其它非激活 case 的数据（YANG choice 互斥语义），提交 payload SHALL 只含激活 case 的字段且保持其扁平 path。

#### Scenario: choice 渲染为切换控件
- **WHEN** schema 含 `type:"choice"` 节点（如 IFM `bandwidth-type` 的 mbps/kbps 两 case）
- **THEN** SHALL 渲染为 Tabs/RadioGroup，可切换不同 case 的配置块

#### Scenario: 切换 case 清空非激活分支
- **WHEN** 用户从 case A 切到 case B
- **THEN** SHALL 清空 case A 字段值，提交时 SHALL 只携带 case B 字段（扁平 path）

### Requirement: FE-09 leaf-list 与 pattern 校验

`FieldRenderer` SHALL 支持 `type:"leaf-list"`（可增删的多值输入行，成员复用叶渲染），并 SHALL 对带 `pattern` 的 string 字段绑定正则校验；非法正则 SHALL 降级为不校验并告警（R08），SHALL NOT 崩溃。

#### Scenario: leaf-list 增删多值
- **WHEN** 字段为 `type:"leaf-list"`
- **THEN** SHALL 渲染可增删的多值输入，提交为数组

#### Scenario: pattern 校验
- **WHEN** string 字段带 `pattern`（如 IFM `number` 的接口编号正则）
- **THEN** SHALL 以该正则校验输入，不匹配时行内报错、阻止提交
