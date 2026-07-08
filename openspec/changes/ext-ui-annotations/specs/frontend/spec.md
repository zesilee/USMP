# frontend delta — ext-ui-annotations

## ADDED Requirements

### Requirement: FE-14 state 子树只读降级

通用模块控制台 SHALL 将 `readonly=true` 的字段降级为只读呈现而非可编辑控件：
整棵 readonly 子树派生的 Tab SHALL 渲染只读视图（可查看、不可编辑）；混合容器内的
readonly 叶 SHALL 呈现禁用态。readonly 字段 SHALL NOT 进入 diff/下发 payload/校验门禁。

#### Scenario: 只读 Tab 降级

- **WHEN** 模块根下某容器整棵为 readonly（如 ifm `remote-interfaces`）
- **THEN** 其 Tab SHALL 以只读视图呈现且 SHALL NOT 提供编辑/下发入口

#### Scenario: 混合容器内只读叶

- **WHEN** 可编辑容器内存在 readonly 叶
- **THEN** 该叶 SHALL 渲染禁用态且 SHALL NOT 参与 payload 与校验

#### Scenario: 只读 list 呈现（边界）

- **WHEN** readonly 子树含 list 节点
- **THEN** SHALL 以只读表格呈现行数据，SHALL NOT 渲染增删改操作列

### Requirement: FE-15 动态缺省占位与单位后缀

字段渲染器对 `dynamicDefault=true` 的字段 SHALL 呈现「系统自动分配」占位语义：
空值 SHALL NOT 触发必填校验、SHALL NOT 视为待下发变更；对携带 `units` 的字段
SHALL 在输入控件展示单位后缀。

#### Scenario: 动态缺省占位

- **WHEN** 字段 `dynamicDefault=true` 且用户未填写
- **THEN** 输入框 SHALL 展示系统自动分配占位提示
- **AND** 空值 SHALL NOT 计入 diff/payload，SHALL NOT 报必填错误

#### Scenario: 用户显式覆写动态缺省（边界）

- **WHEN** 用户对 `dynamicDefault` 字段输入了显式值
- **THEN** 该值 SHALL 正常进入校验与下发 payload

#### Scenario: 单位后缀

- **WHEN** 字段携带 `units: "bit/s"`
- **THEN** 输入控件 SHALL 展示 `bit/s` 后缀

## MODIFIED Requirements

### Requirement: FE-13 模型驱动业务导航与路由迁移

左侧业务配置菜单 SHALL 由 `/yang/modules` 返回的模块列表驱动生成（指向 `/module/:name`），
加载失败 SHALL 回退既有硬编码项（R08）；旧路由 `/config/interface`、`/config/vlan`
SHALL 重定向到对应 `/module/:module`。模块项携带 `category` 时菜单 SHALL 按 category
分组展示；无 `category` 的模块 SHALL 归入默认分组，分组渲染 SHALL NOT 因缺失 category 失败（R08）。

#### Scenario: 菜单生成与回退

- **WHEN** `/yang/modules` 返回含 `ifm`（模块根名）的列表
- **THEN** 菜单 SHALL 含指向 `/module/ifm` 的项
- **WHEN** 该 API 失败
- **THEN** 菜单 SHALL 显示回退项且不崩

#### Scenario: 任务域分组

- **WHEN** 模块列表含带 `category` 与不带 `category` 的模块
- **THEN** 菜单 SHALL 按 category 分组展示带值模块，无值模块 SHALL 归入默认分组且渲染不失败
