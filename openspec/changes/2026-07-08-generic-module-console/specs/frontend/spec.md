# frontend — generic-module-console delta

## ADDED Requirements

### Requirement: FE-10 通用模块控制台（Tab 由模块根派生）

前端 SHALL 提供通用模块控制台页（路由 `/module/:module`，零 per-module props）：
右侧内容区 SHALL 渲染面包屑（配置/厂商/模块/激活 Tab）与一级 Tab；Tab 集合 SHALL 由
nested schema 模块根的顶层子节点自动派生——list→列表 Tab、group/choice→表单 Tab、
散落根叶子聚合为「基本属性」表单 Tab。SHALL NOT 针对任一具体 YANG 模块硬编码
Tab/列/字段。Tab 切换 SHALL 保留各 Tab 的表单与搜索状态。

#### Scenario: huawei-ifm 派生

- **WHEN** 打开 `/module/huawei-ifm`
- **THEN** Tab 集合 SHALL 含 `global`（表单）、`damp`（表单）、`auto-recovery-times`（列表或表单）、
  `interfaces`（列表）等根子节点，无任何硬编码模块名

#### Scenario: schema 加载失败降级

- **WHEN** schema API 失败
- **THEN** 页面 SHALL 展示错误提示且不崩（R08），设备选择仍可用

### Requirement: FE-11 模型驱动列表 Tab（列派生/高级搜索/分页/操作门禁）

列表 Tab SHALL：
- 按分层启发式（key→operationExclude∋update 的 identity 叶→带 when 的条件叶→enum→其余标量）
  从 list 子叶派生表格列并封顶，enum 列渲染 Tag、值 up/down 类渲染状态点（值驱动）；
- 对带 `when` 的列按行数据求值：不满足 SHALL 显示 `-`，求值失败 SHALL 降级正常渲染（R08）；
- 工具栏 SHALL 含新增按钮与「高级搜索」折叠面板，搜索字段集 SHALL 仅取 `supportFilter=true`
  的叶（enum→下拉、其余→文本），支持查询/重置，客户端过滤；
- 表格底部 SHALL 分页（总数/页码/每页条数）；
- 操作列 SHALL 受 `operationExclude` 门禁：list 级含 update/delete 时隐藏对应按钮；
  编辑抽屉中叶级含 update 的字段 SHALL 禁用（新增态可填）。

#### Scenario: 高级搜索过滤

- **WHEN** 数据含 3 条 main-interface 与 2 条 sub-interface，按 class=sub-interface 查询
- **THEN** 表格 SHALL 仅显 2 行；重置后 SHALL 还原 5 行

#### Scenario: 行级 when 单元格

- **WHEN** 行 class=main-interface 且 parent-name 列的 when 为 `../class='sub-interface'`
- **THEN** 该行 parent-name 单元格 SHALL 显示 `-`；sub-interface 行 SHALL 显示其父接口名

#### Scenario: 编辑态 identity 字段禁用

- **WHEN** 编辑一条既有记录且某叶 `operationExclude` 含 `update`
- **THEN** 该字段 SHALL 禁用；新增抽屉中同字段 SHALL 可编辑

### Requirement: FE-12 presence 容器渲染与门禁

`presence=true` 的 group SHALL 渲染为开关：关闭时对应键 SHALL NOT 进入 payload
（YANG presence 语义）；容器 `must` 依赖不满足时开关 SHALL 禁用并强制关闭，
must 求值失败 SHALL 降级为可用（R08）。

#### Scenario: 条件互斥开关

- **WHEN** `ipv4-ignore-primary-sub=true`
- **THEN** `ipv4-conflict-enable` 开关 SHALL 禁用且为关；置 false 后 SHALL 恢复可用

### Requirement: FE-13 模型驱动业务导航与路由迁移

左侧业务配置菜单 SHALL 由 `/yang/modules` 返回的模块列表驱动生成（指向 `/module/:name`），
加载失败 SHALL 回退既有硬编码项（R08）；旧路由 `/config/interface`、`/config/vlan`
SHALL 重定向到对应 `/module/:module`。

#### Scenario: 菜单生成与回退

- **WHEN** `/yang/modules` 返回含 `huawei-ifm` 的列表
- **THEN** 菜单 SHALL 含指向 `/module/huawei-ifm` 的项
- **WHEN** 该 API 失败
- **THEN** 菜单 SHALL 显示回退项且不崩
