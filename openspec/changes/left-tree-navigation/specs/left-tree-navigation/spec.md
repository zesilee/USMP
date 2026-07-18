# left-tree-navigation — SND 左树导航（ADDED）

## ADDED Requirements

### Requirement: LT-01 构建期左树生成

系统 SHALL 在构建期从 `snd/webui/template/left-tree.json` 生成左树 Go 结构（`tools/lefttreegen` → 生成物入库，R04 regen-and-diff 守护）：保留分组层级与 zh-cn/en-us 双语名；每个叶子 SHALL 携带 `sourceModule`（xpath 模块名）与该模块顶层数据容器名集合（goyang 构建期解析；解析失败的模块容器集为空并留日志，SHALL NOT 阻断生成，R08）。运行期 SHALL NOT 读取 snd 目录任何文件。

#### Scenario: 生成物结构完整
- **WHEN** 执行 go:generate 左树生成
- **THEN** 生成物 SHALL 含 14 个顶层分组、65 个叶子及其双语名与根容器映射，重复执行字节一致

#### Scenario: 模块解析失败降级（负路径）
- **WHEN** 某叶子模块 goyang 解析失败
- **THEN** 该叶 rootContainers SHALL 为空（运行期恒不可用），生成 SHALL 成功且留日志

### Requirement: LT-02 左树查询接口

`GET /api/v1/yang/left-tree` SHALL 返回完整左树；每个叶子 SHALL 附 `available`（其根容器与当前 schema 树已加载模块有交集）与 `module`（首个**已加载**根容器名，供前端路由 `/module/<module>`；不可用叶省略）。带 `device=<id>` 时 SHALL 对 available 叶叠加 `supported`（CN-02 协商子集含其根容器）；能力协商不可得 SHALL 省略 `supported` 字段（unknown ≠ 不支持），未注册设备 SHALL 信封 404。

#### Scenario: 已接入叶可路由
- **WHEN** 请求左树且 vlan 模块已加载
- **THEN** xpath `/huawei-vlan` 叶 SHALL `available:true` 且 `module:"vlan"`

#### Scenario: 未接入叶占位
- **WHEN** 某叶模块无已加载根容器
- **THEN** 该叶 SHALL `available:false` 且 SHALL 仍出现在树中（全树+占位拍板）

#### Scenario: 设备能力叠加
- **WHEN** 带 `device=` 且该设备 hello 能力仅含 huawei-vlan
- **THEN** `/huawei-vlan` 叶 SHALL `supported:true`，其他 available 叶 SHALL `supported:false`；协商不可得时 SHALL 全树省略 supported

### Requirement: LT-03 前端左树渲染与降级

前端原生配置导航 SHALL 以左树接口渲染分组树（≤3 层；本期文案取 zh-cn，en-us 字段保留供 i18n）：available 叶 SHALL 为可点菜单项路由 `/module/<module>`；不可用叶 SHALL 渲染禁用态与「未接入」提示，SHALL NOT 可点。左树接口失败 SHALL 回退既有 category 分组导航（R08 导航不消失）。叶子 SHALL 携带 `data-test="lefttree-leaf-<sourceModule>"` 锚点（F2/E2E 共用）。

#### Scenario: 分组树渲染
- **WHEN** 左树加载成功
- **THEN** SHALL 渲染 14 个顶层分组，已接入叶可点、未接入叶禁用带提示

#### Scenario: 接口失败降级（负路径）
- **WHEN** left-tree 请求失败
- **THEN** SHALL 回退 category 分组导航，SHALL NOT 空白/崩溃
