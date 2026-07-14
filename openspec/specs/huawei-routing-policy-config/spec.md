# huawei-routing-policy-config — 华为 routing-policy route-policy 配置管理（Stack B）

## Purpose

华为 routing-policy `policy-definitions/policy-definition`（`/rtp:routing-policy` 容器根，name + address-family-mismatch-deny 标量边界）的模型驱动读写闭环——沿用 VLAN/IFM/BGP/tunnel-management/xpl 驱动注册表 playbook：一期全闭包已生成结构体（零 codegen）→ 描述符注册（显式 namespace + SchemaTree 入口 HuaweiRoutingPolicy_RoutingPolicy）→ 通用 xmlcodec 编解码（容器根 XC-05 + per-node ns XC-06）→ routingpolicy 容器根 reconciler → B2 端到端集成。是 BGP 二期-2b DAG 前置（依赖 tunnel-management #155）：使 policy-definition 实例按 name 存在，解除 BGP AF import/export route-policy leafref（require-instance=true）的越序阻塞。policy-definition 深层 nodes 子树（conditions/actions，~1490 行）与 rtp 其他 filter 显式推迟 follow-up。由 change `huawei-routing-policy-config`（2026-07-14）引入。

## Requirements

### Requirement: RTP-01 routing-policy route-policy 配置面接入（标量边界）

系统 SHALL 通过 Stack B 驱动注册表接入华为 `huawei-routing-policy` 模块顶层独立根容器 `/rtp:routing-policy` 下 BGP import/export route-policy 引用的目标子树 `policy-definitions/policy-definition` 的**标量边界**，提供读改下发闭环，SchemaTree 入口为容器根 `HuaweiRoutingPolicy_RoutingPolicy`（容器根非 list 根，走通用引擎 plain-container，见 yang-xml-codec XC-05）。rtp 配置 SHALL 由通用 XML 编解码引擎（`pkg/yang-runtime/xmlcodec`）按 ygot 生成数据驱动编解码，SHALL NOT 手写 per-model XML 解析/序列化。

`huawei-routing-policy` 的 ygot 结构体已随一期 `huawei-bgp` 全闭包生成，本 change **SHALL NOT 修改 `gen.conf`、SHALL NOT 重新生成、SHALL NOT 手改 `generated/`**（R04）。

本波次范围 SHALL 覆盖 `policy-definitions/policy-definition`（key=`name`）的标量边界全部 config-true leaf，无遗漏：

- `name`（key）
- `address-family-mismatch-deny`（boolean）

系统 SHALL NOT 做字段挑选式覆盖，标量边界完备性 SHALL 由 schema 驱动的用例保证。

本波次 SHALL NOT 覆盖（注册为 follow-up，非简化遗漏）：(a) `policy-definition` 下**深层 `nodes/node` 子树**（node 的 conditions/actions 匹配与动作子句、嵌套 list、choice，~1490 行），门控于通用引擎对深层嵌套/choice 往返的支持；(b) rtp 的**其他 filter/list**（community-filters、ext-community-*、ipv4/ipv6-prefix-filters、as-path-filters、rd-filters、large-community-* 等），各由其自身 BGP 消费属性门控；(c) config-false 只读态，SHALL NOT 作为下发目标。

#### Scenario: 下发 policy-definition 并回读收敛
- **WHEN** 向模拟网元下发 `/rtp:routing-policy/policy-definitions/policy-definition`（`name=<有效名>`、`address-family-mismatch-deny=<bool>`）
- **THEN** 系统 SHALL 编码为 `huawei-routing-policy` namespace 的 NETCONF edit-config 报文下发，回读 running config 后 desired↔actual 收敛（无持续漂移）

#### Scenario: 全属性可配（本波次字段）
- **WHEN** 依次配置 policy-definition 的 name、address-family-mismatch-deny
- **THEN** 每个字段 SHALL 能成功编码下发并原值回读，无字段丢失或被静默丢弃

#### Scenario: 不触碰 generated（R04 门禁）
- **WHEN** 运行 R04 regen-and-diff 门禁
- **THEN** `gen.conf` 与 `generated/` SHALL 相对基线零改动（本 change 不涉 codegen）

### Requirement: RTP-02 命名空间显式登记

routing-policy 驱动描述符 SHALL 显式携带模块 XML namespace 常量 `urn:huawei:yang:huawei-routing-policy`，SHALL NOT 依赖内嵌 gzip schema 的 `Entry.Namespace()` 派生（实测返回空）。编码产出的 rtp 配置报文根元素 SHALL 归属该 namespace。

#### Scenario: 编码报文携带正确 namespace
- **WHEN** 编码一份 `/rtp:routing-policy/policy-definitions/policy-definition` 配置
- **THEN** 输出 XML 根容器 SHALL 声明 `urn:huawei:yang:huawei-routing-policy` namespace（前缀不敏感）

### Requirement: RTP-03 路由/编解码谓词精确锚定

routing-policy 驱动描述符的 `MatchRoute`/`MatchDecode`/`MatchEncode` 谓词 SHALL 精确锚定根路径 `/rtp:routing-policy`，SHALL NOT 误命中其他模块。查找未命中的路径 SHALL 返回 `ok=false` 供调用方降级（R08），SHALL NOT panic。

#### Scenario: routing-policy 路径命中
- **WHEN** 以 `/rtp:routing-policy/...` 路径触发路由/编解码分发
- **THEN** SHALL 命中 routing-policy 描述符（`ControllerToken="routing-policy"`）

#### Scenario: 注册可达性
- **WHEN** routing-policy 集成测试所在二进制/独立测试包运行
- **THEN** 该二进制 SHALL 空白导入 `internal/drivers` 触发注册，`Lookup("huawei", "/rtp:routing-policy/...")` SHALL 返回 `ok=true`

### Requirement: RTP-04 模拟网元端到端集成

`simulator/netconfsim` SHALL 支持 routing-policy edit-config（整树替换语义，对齐既有 RFC edit-config 通道）与 get-config 回读，支撑 Reconciler↔设备端到端集成测试（B2，`*_integration_test.go`，`testing.Short()` 跳过）。集成测试 SHALL 覆盖下发→回读→收敛全链路。

#### Scenario: 模拟网元接受并回读 policy-definition 配置
- **WHEN** 集成测试向 netconfsim 下发 policy-definition 配置并随后 get-config
- **THEN** netconfsim SHALL 返回与下发等价的 running config，Reconciler 判定收敛

#### Scenario: 重复下发幂等
- **WHEN** 对同一份 rtp 配置连续下发两次
- **THEN** 第二次 SHALL 判定为 no-op（无 diff、无重复 edit-config 副作用）

### Requirement: RTP-05 完备测试矩阵（yang-config-test-design / T02b）

本 change 作为「新增 YANG 模型接入设备配置」SHALL 触发 `yang-config-test-design`，产出并通过完备测试矩阵，未覆盖视为未完成、禁止合并（T02b）。矩阵 SHALL 覆盖：全（标量边界）属性可配、端到端到设备（B2）、并发-race、边界、幂等、负路径、删除语义。

#### Scenario: schema 驱动形状锁定
- **WHEN** 反射枚举 policy-definition 直属 config-true 标量 leaf
- **THEN** SHALL 恰好为 2（name + address-family-mismatch-deny），深层 nodes 仍为推迟容器（模型漂移/越界接入会使断言失败触发复审）

#### Scenario: 并发安全（race）
- **WHEN** 在 `-race` 下并发触发 rtp 编解码/查找
- **THEN** SHALL 无数据竞态、无 panic（R09）

#### Scenario: 负路径不 panic
- **WHEN** 以缺失/畸形 rtp 路径或空配置触发编解码
- **THEN** SHALL 返回明确错误或降级，SHALL NOT panic（R08）

#### Scenario: 删除语义（声明式 subset + DELETE 通道推迟债）
- **WHEN** 从 desired 移除某 policy-definition 条目后下发
- **THEN** 沿用平台声明式 subset 语义（天然不删），DELETE 命令通道作为推迟债显式登记（同 tunnel-management / xpl 语义）
