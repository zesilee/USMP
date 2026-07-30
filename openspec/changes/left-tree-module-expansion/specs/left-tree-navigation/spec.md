# left-tree-navigation — 左树模块级展开 delta

## MODIFIED Requirements

### Requirement: LT-01 构建期左树生成

系统 SHALL 在构建期从 `snd/webui/template/left-tree.json` 生成左树 Go 结构（`tools/lefttreegen` → 生成物入库，R04 regen-and-diff 守护）：保留分组层级与 zh-cn/en-us 双语名；每个叶子 SHALL 携带 `sourceModule`（xpath 模块名）与该模块顶层数据容器名集合（goyang 构建期解析；解析失败的模块容器集为空并留日志，SHALL NOT 阻断生成，R08）。运行期 SHALL NOT 读取 snd 目录任何文件。

每个可解析叶子 SHALL 另携带模块级 `children`：每个顶层数据容器一个 `kind=container` 节点、该模块每个 rpc 一个 `kind=rpc` 节点（rpc 与 container 平级平铺，SHALL NOT 引入额外分组层）。rpc 节点 SHALL 携带 `highRisk`，分类 SHALL 与 rpcgen 共用同一分类器（口径 SHALL NOT 复制漂移）。children 节点的 zh/en 标签 SHALL 构建期自 snd res 烘焙（container 键 `/<sourceModule>:<root>`、rpc 键 `/<sourceModule>:<rpcName>` 的 name）；res 缺键/缺文件 SHALL 回退原始节点名并留日志，SHALL NOT 阻断生成（R08）。

#### Scenario: 生成物结构完整
- **WHEN** 执行 go:generate 左树生成
- **THEN** 生成物 SHALL 含 14 个顶层分组、65 个叶子及其双语名与根容器映射，重复执行字节一致

#### Scenario: 模块解析失败降级（负路径）
- **WHEN** 某叶子模块 goyang 解析失败
- **THEN** 该叶 rootContainers 与 children SHALL 为空（运行期恒不可用），生成 SHALL 成功且留日志

#### Scenario: 叶子 children 烘焙
- **WHEN** 生成后查看 huawei-ifm 叶
- **THEN** children SHALL 含 1 个 container 节点（name=ifm，zh=通用接口，en=Common Interface）与 10 个 rpc 节点（含 name=reset-if-counters-by-name，zh=按接口名清除统计；restart-if SHALL `highRisk:true`）

#### Scenario: res 缺键回退（负路径）
- **WHEN** 某容器/rpc 在双语 res 均无对应键
- **THEN** 该节点 zh/en SHALL 回退原始 YANG 节点名，生成 SHALL 成功且留日志

### Requirement: LT-02 左树查询接口

`GET /api/v1/yang/left-tree` SHALL 返回完整左树；每个叶子 SHALL 附 `available`（其根容器与当前 schema 树已加载模块有交集）与 `module`（首个**已加载**根容器名，供前端路由 `/module/<module>`；不可用叶省略）。带 `device=<id>` 时 SHALL 对 available 叶叠加 `supported`（CN-02 协商子集含其根容器）；能力协商不可得 SHALL 省略 `supported` 字段（unknown ≠ 不支持），未注册设备 SHALL 信封 404。

available 叶 SHALL 透出模块级 `children`：**已加载**根容器的 `kind=container` 节点与该模块 rpc 的 `kind=rpc` 节点（携 `highRisk`，仅 true 时出现），节点含 name 与 zh/en 标签；未加载根容器的 container 节点 SHALL NOT 透出。不可用叶 SHALL NOT 透出 children。分组与模块叶自身 SHALL NOT 携带 `kind`（既有契约字段形状向后兼容）。

#### Scenario: 已接入叶可路由
- **WHEN** 请求左树且 vlan 模块已加载
- **THEN** xpath `/huawei-vlan` 叶 SHALL `available:true` 且 `module:"vlan"`

#### Scenario: 未接入叶占位
- **WHEN** 某叶模块无已加载根容器
- **THEN** 该叶 SHALL `available:false` 且 SHALL 仍出现在树中（全树+占位拍板），SHALL NOT 携带 children

#### Scenario: 设备能力叠加
- **WHEN** 带 `device=` 且该设备 hello 能力仅含 huawei-vlan
- **THEN** `/huawei-vlan` 叶 SHALL `supported:true`，其他 available 叶 SHALL `supported:false`；协商不可得时 SHALL 全树省略 supported

#### Scenario: 模块叶 children 透出
- **WHEN** 请求左树且 ifm 模块已加载
- **THEN** `/huawei-ifm` 叶 children SHALL 含 kind=container 的 ifm 节点与 10 个 kind=rpc 节点，`restart-if` SHALL `highRisk:true`，温和 rpc SHALL 省略 highRisk

### Requirement: LT-03 前端左树渲染与降级

前端原生配置导航 SHALL 以左树接口渲染分组树；available 模块叶 SHALL 为可展开分组（默认折叠），其下平铺 children：`kind=container` 节点 SHALL 可点路由 `/module/<module>`，`kind=rpc` 节点 SHALL 可点路由 `/module/<module>/rpc/<rpcName>`，`highRisk` rpc 节点 SHALL 呈现警示视觉标识（真实图标/规范占位符，R12）。不可用叶 SHALL 渲染禁用态与「未接入」提示，SHALL NOT 可点、SHALL NOT 可展开。标签 SHALL 按当前语言取 zh/en，缺失回退另一语言再回退 name，SHALL NOT 空标签（R08）。左树接口失败 SHALL 回退既有 category 分组导航（R08 导航不消失）。模块叶 SHALL 保留 `data-test="lefttree-leaf-<sourceModule>"` 锚点；container 子节点 SHALL 携带 `data-test="lefttree-node-<module>"`、rpc 子节点 SHALL 携带 `data-test="lefttree-rpc-<module>-<rpcName>"`（F2/E2E 共用）。各层缩进 SHALL 随深度单调递增。

#### Scenario: 模块叶展开呈现 container 与 rpc 平级

- **WHEN** 左树加载成功并展开 huawei-ifm 叶
- **THEN** SHALL 平铺呈现「通用接口」container 节点与 10 个 rpc 节点（如「按接口名清除统计」），点击 container SHALL 路由 `/module/ifm`，点击 rpc SHALL 路由 `/module/ifm/rpc/reset-if-counters-by-name`

#### Scenario: 高危 rpc 树节点警示

- **WHEN** 展开含高危 rpc 的模块叶（huawei-ifm 的 restart-if）
- **THEN** 该 rpc 节点 SHALL 呈现警示视觉标识，温和 rpc 节点 SHALL NOT 呈现

#### Scenario: 接口失败降级（负路径）

- **WHEN** left-tree 请求失败
- **THEN** SHALL 回退 category 分组导航，SHALL NOT 空白/崩溃

#### Scenario: 未接入叶不可展开（负路径）

- **WHEN** 渲染 `available:false` 的叶
- **THEN** SHALL 禁用态占位，SHALL NOT 展开出 children
