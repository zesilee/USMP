## ADDED Requirements

### Requirement: BN-01 公网 BGP 基础邻居配置面接入

系统 SHALL 将公网 BGP peer（`/ni:network-instance/instances/instance[_public_]/bgp/base-process/peers/peer`，key=`address`）下**全部 config-true 标量**（含 mandatory `remote-as`）及基础子容器（`timer`/`graceful-restart`/`bfd-parameter`）的 config-true 标量接入模型驱动读改下发闭环，字段无遗漏、完备性由 schema 驱动用例保证。系统 SHALL 复用 network-instance 描述符与 reconciler（不新增描述符/控制器），经 XC-06 per-node namespace 使 huawei-bgp 子树编码带正确 namespace。

#### Scenario: 下发公网 peer 配置并回读收敛
- **WHEN** 下发含 `instance[_public_]/bgp/base-process/peers/peer[address]`（remote-as + 若干标量 + timer/graceful-restart 子容器）的 desired `/ni:network-instance`
- **THEN** 经容器根整根收敛编码为单条 `<network-instance>…<bgp xmlns="urn:huawei:yang:huawei-bgp">…<peers><peer>…` 下发，回读后 desired↔actual 收敛（二次 reconcile 无新增 change）

#### Scenario: 全属性可配（peer config-true 标量）
- **WHEN** schema 驱动用例枚举 peer 子树每个 config-true 标量 leaf 并赋值→编码→解码
- **THEN** 整体 DeepEqual 且 config-true 标量计数与 schema 一致（漏/多都失败；策略/状态字段按 config 继承与排除清单不计）

#### Scenario: peer 下发 XML 携带正确 per-node namespace
- **WHEN** 编码含 peer 的 network-instance
- **THEN** `<bgp>` 元素 SHALL 带 `xmlns="urn:huawei:yang:huawei-bgp"`，`<peers>`/`<peer>`/`<address>`/`<remote-as>` 继承之而不重复发；ni 原生 `<name>` 不另发 xmlns

### Requirement: BN-02 地址族条目（afs/af，枚举 key）

系统 SHALL 支持在 peer 下建立地址族条目 `afs/af`（key=`af-type`，枚举 `E_HuaweiBgp_AfType`），验证 list-under-list 与枚举 key 的编解码往返。

#### Scenario: af-type 枚举 key 编解码往返
- **WHEN** 对含一条或多条 `af[af-type]` 的 peer 编码再解码
- **THEN** 往返 DeepEqual，af-type 枚举 key 值一致、list-under-list 结构正确

### Requirement: BN-03 完备测试矩阵（yang-config-test-design / T02b）

系统 SHALL 产出并通过基础邻居完备测试矩阵：全属性可配、端到端到设备（含 namespace 真值）、并发-race、边界、嵌套、幂等、负路径，缺层视为未完成禁止合并。

#### Scenario: 并发下发无数据竞态
- **WHEN** 多协程并发对含 peers 的 network-instance 触发 reconcile（`go test -race`）
- **THEN** 无数据竞态、无 panic，收敛一致

#### Scenario: 边界（description 域约束由 ygot 强制）
- **WHEN** peer `description` 越界（>255 或含 `?`）
- **THEN** ΛValidate 拦截；合法值通过

#### Scenario: remote-as mandatory 事实登记（ygot 不强制）
- **WHEN** peer 无 `remote-as`（YANG 声明 `mandatory true`）
- **THEN** ygot ΛValidate **不强制** list-entry mandatory leaf（实测），本平台 SHALL 由设备侧/API 层兜底该约束；本 change 用测试锁定此事实，防误判 ygot 会拦

#### Scenario: 幂等
- **WHEN** 对同一含 peers 的 desired 连续两次 reconcile
- **THEN** 第二次无新增 change

### Requirement: BN-04 分期边界与删除语义

系统 SHALL 明确本 change 仅接公网 peer 基础邻居面；af 内策略属性（route-policy/route-filter/ACL/tunnel-policy）、peer-groups、dynamic-peer-prefixes、egress-engineer、fake-as-parameter、config-false 状态、per-VPN（非 `_public_`）peers 均排除。peer/af 删除沿用声明式 subset 语义（天然不删）+ DELETE 命令通道推迟债。

#### Scenario: 策略属性与状态字段不被误下发（负路径，防越序）
- **WHEN** 构造 peer 配置但未 set 策略子容器/config-false 状态字段
- **THEN** 下发报文 SHALL NOT 出现 import/export route-policy、route-filter、ACL、tunnel-policy、`*-state` 等（越序属 2b，门控未集成模型）

#### Scenario: 声明式删除为 subset（平台契约）
- **WHEN** 从 desired 移除某 peer 后对账
- **THEN** 声明式通道不下发删除（Changes==0、设备保留、无永久漂移）；设备侧删除须走 DELETE 命令通道（推迟债）
