# Design: generic-module-console

## 1. 后端：扩展元数据透出

### 1.1 采集源与匹配规则

实证（内嵌 gzip schema 解码）：华为扩展存活于 goyang `Entry.Exts`（`[]*yang.Statement`），
`Kind()` 形如 `ext:support-filter`、`NName()` 为参数（`"true"` / `"update|delete"`）；
presence 存活于 `Entry.Extra["presence"]`（既有 `IsPresence()` 已消费）；容器级 `when/must`
与叶级同样存活于 `Entry.Extra`（复用 `firstExtraExpr`/`allExtraExprs`）。

匹配按扩展关键字**本名**（去前缀取 `:` 后段）：`support-filter`、`operation-exclude`。
前缀（`ext:`/`hw-ext:`…）随模块 import 别名变化，不可依赖 → 厂商无关。

- `support-filter`：参数 `"true"`（大小写不敏感）→ true；缺失/其他 → false。
- `operation-exclude`：参数按 `|` 与 `,` 切分、trim、小写归一 → `[]string`（如 `["update","delete"]`）。
  语义：该叶 create 后不可 update/delete（identity 字段）；出现在 list/container 上则整节点排除对应操作。

### 1.2 接口与 DTO 变更（扩宽不破坏）

```
schema.LeafNode      += SupportFilter() bool / OperationExcludes() []string
schema.ContainerNode += WhenExpr() string / MustExprs() []string   // IsPresence() 已有
api.FieldDef         += SupportFilter bool `json:"supportFilter,omitempty"`
                        OperationExclude []string `json:"operationExclude,omitempty"`
                        Presence bool `json:"presence,omitempty"`
```

`nodeToNestedField`：ContainerNode 分支填 `Presence`/`When`/`Must`；ListNode 分支填
`OperationExclude`（list 级）；`leafToField` 填叶级三项。扁平 `collectFields` 同步生效
（经 `leafToField`）。

## 2. 前端：通用模块控制台

### 2.1 页面结构

```
/module/:module → ModuleConsolePage
  ├─ 面包屑：配置 / {vendor} / {module title} / {active tab}
  ├─ 设备选择（Tab 间共享）
  └─ el-tabs（一级 Tab）← 模块根顶层子节点派生：
       list        → ModuleListTab（表格页）
       group       → ModuleFormTab（表单页；其下嵌套 list 仍由 FieldRenderer 渲染）
       choice      → ModuleFormTab（choice 已由 FieldRenderer Tabs/Radio 支持）
       散落根叶子   → 聚合为「基本属性」表单 Tab（排最前）
```

Tab 切换用 `el-tabs` 惰性渲染 + keep-alive 语义（组件不销毁），保留各 Tab 表单/搜索态。

### 2.2 列派生（纯函数 `deriveColumns(listField, cap=9)`）

分层取列、层内保持 schema 顺序，封顶 cap：

1. key（`keyField`，schema list 第一个 required/key 叶）
2. identity：`operationExclude` 含 `update` 的叶（create-only 标识字段）
3. 条件列：带 `when` 的叶（渲染期行级求值）
4. enum 叶 → el-tag（`main-*`→蓝 / `sub-*`→绿 之类**不做语义映射**，用 tag 色板轮转；
   值 ∈ {up,true} → 绿点、{down,false} → 红/灰点：值驱动、非字段名驱动）
5. 其余标量叶（boolean→tag、number/string→文本）

`group`/`list`/`choice` 子节点不入列。列元数据 = `Field` 本身（含 when/options/type）。

### 2.3 行级 when 单元格

列带 `when` 时对**行数据**求值（复用 `evalPredicate`，上下文=该行，`../x` 解析为行内兄弟叶）：
满足→正常渲染；不满足→显示 `-`。求值异常→降级正常渲染（R08）。

### 2.4 高级搜索（`supportFilter` 驱动）

- 面板默认折叠；工具栏「高级搜索」链接切换。
- 字段集 = list 子叶中 `supportFilter===true` 者；enum→el-select（clearable）、其余→el-input。
- 客户端过滤（后端 GET 无服务端过滤参数；数据集=单设备单 list，规模可控）：
  `filterRows(rows, criteria)` 纯函数，空条件跳过、enum 全等、字符串子串（大小写不敏感）。
- 重置=清空条件并收起违例；查询即时生效（受控 computed）。

### 2.5 操作列门禁

- list 级 `operationExclude`：含 `update`→隐藏「编辑」；含 `delete`→隐藏「删除」；
  两者全隐或 list 只读（全叶 readonly）→ 操作列整列隐藏（只读视图）。
- 叶级 `operationExclude` 含 `update`：编辑抽屉中该字段 `disabled`（新增态可填）。
- 删除走既有 setConfig 链路（整 list 回写去掉该行）；无删除后端语义时如实报错。

### 2.6 presence 容器（FieldRenderer 新分支）

`field.type==='group' && field.presence`：
- 渲染 el-switch；开→payload 含 `{}`（或其子叶对象），关→键不存在（YANG presence 语义）。
- 子叶存在时：开关开启后展开子表单。
- 容器 `must`（如 `../ipv4-ignore-primary-sub='false'`）：不满足时**禁用开关并强制关**
  （对齐 UI 门禁语义：条件不满足即节点不可存在）；求值失败降级为可用（R08）。

### 2.7 表单 Tab 数据流

- GET：`getConfig(ip, modulePrefixedPath(tabNode))`，路径由 schema path 加模块前缀推导
  （与列表页 configPath 同构）；读失败→空表单 + 告警条。
- POST：`setConfig` 同路径；后端不支持的转换路径返回错误→原样行内透出（§9），不伪装成功。

### 2.8 导航与路由迁移（渐进，旧并行）

- `Sidebar` 业务菜单 `v-for` menu store `businessModules`（`/yang/modules` 全量，按 vendor 分组），
  指向 `/module/:name`；加载失败回退现硬编码三项（R08）。
- `/config/interface` → redirect `/module/huawei-ifm`；`/config/vlan` → `/module/huawei-vlan`
  （`DeviceConfigPage.vue` 保留一个发布周期，无路由引用后随后续 change 删除）。
- 列表 Tab 的 configPath/listKey/keyField 全部由 schema 派生：
  configPath = `{module-prefix}:{root}/{…}`（复用 modules API 返回的 path + 节点 path），
  keyField = list 首个 key 叶，listKey = list 名。

## 3. 模拟数据（种子）

netconfsim ifm 种子扩为 5 条：`200GE0/1/0`、`200GE0/1/1`、`200GE0/1/2`
（main-interface/200GE/up）+ `200GE0/1/0.1`、`200GE0/1/1.1`
（sub-interface/Vlanif… 按模型枚举取值/down，parent-name 指向对应主接口）。
F2 用同形 fixture；F4 断言种子行可见、class 过滤生效。

## 4. 测试映射（§5.6）

| 改动 | 层 |
|------|----|
| entry.go Exts/presence/容器 when-must 采集 | B1 `entry_ext_test.go`（真实 IFM 断言 + 合成双前缀/畸形参数） |
| FieldDef 透传 | B3 `field_gen_ext_test.go` |
| deriveColumns / filterRows / 行级 when | F1 `test/utils/moduleConsole.test.ts` |
| ModuleConsolePage Tab 派生/面包屑 | F2 |
| ModuleListTab 搜索/分页/动态单元格/操作门禁 | F2 |
| presence 开关 + must 门禁、statistic-interval mod10 | F2 |
| Sidebar 模型驱动菜单 | F2 |
| 控制台端到端冒烟 | F4 staging-smoke 增补 |

覆盖率棘轮（T08）：补测后按需上调前端 thresholds / 后端 baseline。
