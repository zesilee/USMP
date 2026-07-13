## ADDED Requirements

### Requirement: NI-01 network-instance 原生配置面接入

系统 SHALL 将 `huawei-network-instance` 的 `/ni:network-instance` 下**全部原生 config-true 字段**接入 Stack B 模型驱动配置闭环，SchemaTree 入口为 `HuaweiNetworkInstance_NetworkInstance`（容器根 + 嵌套 list），字段无遗漏：`global`{`cfg-router-id`, `as-notation-plain`, `route-distinguisher-auto-ip`} 与 `instances/instance`（key=`name`）{`name`, `description`}。字段完备性 SHALL 由 schema 驱动用例保证，不做人工挑选。

#### Scenario: 下发原生 network-instance 配置并回读收敛
- **WHEN** 用户下发含 `global` 标量与一条 `instance`（name+description）的 desired `/ni:network-instance`
- **THEN** 系统经容器根 xmlcodec 编码为单条 `<network-instance>…</network-instance>` edit-config 下发，回读解析后 desired↔actual 收敛（二次 reconcile 无新增 change）

#### Scenario: 全属性可配（本期原生字段）
- **WHEN** schema 驱动用例按 `module:"huawei-network-instance"` 标签枚举 `HuaweiNetworkInstance_NetworkInstance` 下每个 config-true 标量 leaf 并赋值→编码→解码
- **THEN** 整体 DeepEqual 且 config-true 原生标量计数恰为 **5**（global 3 + instance name/description 2；`sys-router-id`/`vrf-id` 为 config-false 不计）

#### Scenario: ygot 生成物零漂移（R04 门禁）
- **WHEN** `gen.conf` 的 `modules` 增加 `huawei-network-instance` 后执行 `make gen-yang`
- **THEN** 经 CG-02 确定性规范化后 `regen-and-diff` 零漂移，`go build ./...` 全通过，`HuaweiNetworkInstance_NetworkInstance` 入口可用

### Requirement: NI-02 命名空间显式登记

系统 SHALL 为 network-instance 描述符显式登记 namespace `urn:huawei:yang:huawei-network-instance`（取 8.20.10 YANG 声明的 module namespace），不依赖内嵌 schema 派生（实测返回空）。

#### Scenario: 编码报文携带正确 namespace
- **WHEN** 编码 `/ni:network-instance` 配置
- **THEN** 输出 XML 根节点携带 `xmlns="urn:huawei:yang:huawei-network-instance"`

### Requirement: NI-03 路由/编解码谓词锚定整棵子树，单描述符覆盖

系统 SHALL 注册**单条** `driver.Descriptor` 覆盖整棵 `/ni:network-instance` 子树，`MatchRoute`/`MatchDecode`/`MatchEncode` 谓词以 `HasPrefix("/ni:network-instance")` 锚定。因 augment 合并（peers=huawei-bgp、l3vpn 字段），该子树为单一 ygot 根，未来 peering 分期 SHALL 扩展同一描述符的驱动面而非另立描述符。

#### Scenario: network-instance 路径命中
- **WHEN** 路由路径为 `/ni:network-instance` 或其子路径
- **THEN** 命中 network-instance 描述符，编解码走通用引擎容器根模式（XC-05）

#### Scenario: 注册可达性
- **WHEN** 生产二进制或集成测试二进制经空白导入 `internal/drivers` 触发注册
- **THEN** network-instance 描述符可达，编码不落 `xml.Marshal` 兜底

#### Scenario: 本期只驱动原生字段（负路径，防越序）
- **WHEN** 本期 reconciler 构造 desired 时未 set augment 子字段（`Instance.Bgp` / `Afs` / `Parameter` / `TrafficStatisticEnable`）
- **THEN** 编码报文 SHALL NOT 出现这些 augment 节点（peers/AF/l3vpn 属于后续分期，未集成不下发）

### Requirement: NI-04 模拟网元端到端集成

系统 SHALL 通过 `simulator/netconfsim` 通用 tree datastore（按 local 名存取，预期无需专用方言）支撑 B2 端到端集成测试，覆盖嵌套 list（`instances/instance`）的编解码往返。

#### Scenario: 模拟网元接受并回读 network-instance 配置
- **WHEN** reconciler 向模拟网元下发含多条 instance 的 `/ni:network-instance` 配置
- **THEN** 模拟网元接受 edit-config，get-config 回读出等价配置，reconcile 收敛

#### Scenario: 重复下发幂等
- **WHEN** 对同一 desired 连续两次 reconcile
- **THEN** 第二次无新增 change（幂等）

### Requirement: NI-05 完备测试矩阵（yang-config-test-design / T02b）

系统 SHALL 产出并通过 network-instance 完备测试矩阵：全属性可配、端到端到设备、并发-race、边界、嵌套 list 增删改、幂等、负路径，缺层视为未完成禁止合并。

#### Scenario: 并发下发无数据竞态
- **WHEN** 多协程并发对不同 instance 触发 reconcile（`go test -race`）
- **THEN** 无数据竞态、无 panic，收敛结果一致

#### Scenario: 嵌套 list 增/改设备侧收敛
- **WHEN** 在既有配置上新增或修改 `instances/instance` 条目（含 description）
- **THEN** 编解码往返 DeepEqual，声明式对账下发后设备侧收敛（二次 reconcile Changes==0）

#### Scenario: 嵌套 list 编解码覆盖任意条目集（增删改数据级）
- **WHEN** 对含 0/1/N 条 instance 的配置编码→解码
- **THEN** 往返 DeepEqual，条目集精确一致（数据级增删改无丢失）

#### Scenario: 声明式通道删除为 subset 语义（平台契约，非本模块缺陷）
- **WHEN** 从 desired 移除某 `instance` 条目后再对账
- **THEN** 声明式通道按 subset 语义**不下发删除**（Changes==0、设备保留该条目、无永久漂移）——与 VLAN/IFM/BGP 一致；设备侧删除须走独立 DELETE 命令通道（本 MVP 未接，登记为 NI-06 推迟债）

#### Scenario: 边界与负路径
- **WHEN** 输入越界值（name 超 31 字符、description 超 242 字符或含 `?`、非法 ipv4）
- **THEN** 校验拦截或明确错误，下发失败时缓存不更新（保留原配置）

### Requirement: NI-06 分期边界与 `_public_` 不可删语义

系统 SHALL 明确本 change 范围仅原生 config-true 字段；config-false 回读态、BGP peers/AF、l3vpn augment 字段均排除。声明式对账通道按 subset 语义天然不删除（`_public_` 及任何 instance 都不会被声明式通道删除）；设备侧 instance 删除（含 `_public_` 不可删守卫）须走独立 DELETE 命令通道，本 MVP **不接**、登记为推迟债（沿用 BGP 容器根 node-delete 债与 `config-delete-semantics` 平台契约）。

#### Scenario: config-false 只读态不被误下发
- **WHEN** 构造 network-instance 配置
- **THEN** `sys-router-id`/`vrf-id`（config-false）SHALL NOT 出现在下发报文

#### Scenario: `_public_` 经声明式通道天然不被删除（负路径）
- **WHEN** desired 不含 `name='_public_'` 的 instance 后对账
- **THEN** 声明式 subset 语义使 `_public_` 保留、不下发任何删除（与设备侧「不可删」一致）；显式 DELETE 命令通道对 `_public_` 的守卫属推迟债范围

#### Scenario: 分期硬前置登记
- **WHEN** 规划 BGP 二期 peers（`Instance.Bgp`）
- **THEN** 其前置为本 change（network-instance）已交付，peers 扩展同一 `/ni:network-instance` 描述符驱动面
