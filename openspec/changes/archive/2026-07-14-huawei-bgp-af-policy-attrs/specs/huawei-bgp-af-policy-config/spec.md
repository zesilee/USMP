## ADDED Requirements

### Requirement: AFPOL-01 BGP AF import-filter-policy 策略属性接入

系统 SHALL 接入华为 BGP AF `import-filter-policy` 容器下引用已集成目标模型的可选 leafref 策略属性，路径 `/ni:network-instance/instances/instance[_public_]/bgp/base-process/afs/af[type]/ipv4-unicast/import-filter-policy`，提供读改下发闭环。本能力 SHALL **复用既有 network-instance 描述符与 reconciler**（2a 已覆盖整棵 ni 子树含 huawei-bgp augment），SHALL NOT 新增描述符/控制器/codegen。AF 策略属性 SHALL 经通用 xmlcodec 编码，`<bgp>` 子树 SHALL 带 `huawei-bgp` namespace（XC-06 per-node namespace，已登记）。

本波次范围 SHALL 覆盖 `import-filter-policy` 下引用**已集成**模型的 leafref 属性：

- `acl-name-or-num`（leafref → `/acl:acl/groups/group/identity`，acl 已集成）
- `filter-name`（leafref → `/xpl:xpl/route-filters/route-filter/name`，xpl 已集成）
- `filter-parameter`（xpl route-filter 参数名）

本波次 SHALL NOT 覆盖（注册为 follow-up，非简化遗漏）：`ipv4-prefix-filter`（leafref → rtp `ipv4-prefix-filters`，该 rtp 子树未集成——波次③ 仅接 `policy-definitions`）；深层嵌套的 `route-policy`（→ rtp policy-definition）与 `tunnel-policy`（→ tnlm）；`export-filter-policys`、ipv6/vpn 等其他地址族、peer 级策略属性。

#### Scenario: AF 策略属性经 ni 链路编码带 namespace
- **WHEN** 在 `af[ipv4-unicast]/ipv4-unicast/import-filter-policy` 填 `acl-name-or-num` 与 `filter-name` 并经 network-instance 描述符编码
- **THEN** 输出 XML SHALL 含 `<bgp xmlns="urn:huawei:yang:huawei-bgp">`、`<import-filter-policy>`、`<acl-name-or-num>...</acl-name-or-num>`、`<filter-name>...</filter-name>`，字段值真值正确

#### Scenario: 下发 AF 策略属性并回读收敛
- **WHEN** 经 ni reconciler 向模拟网元下发含 AF import-filter-policy 策略属性的 `_public_` 实例
- **THEN** 回读 running config 后 desired↔actual 收敛（Changes==0，无持续漂移）

#### Scenario: 零新描述符（复用 ni 链路）
- **WHEN** 以 AF 策略属性路径触发编解码分发
- **THEN** SHALL 由既有 network-instance 描述符处理（`ControllerToken="network-instance"`），SHALL NOT 新增 acl/xpl 描述符于此路径

### Requirement: AFPOL-02 leafref 编排依赖登记

AF 策略属性 SHALL 以目标实例名（string）承载 leafref 引用。真机 `require-instance`（默认真）要求被引实例先存在——系统 SHALL 在编排语义上要求目标模型（acl group / xpl route-filter）**先由其对应 reconciler 配置**，再下发引用它的 BGP AF 属性。本能力 SHALL 交付属性下发能力并登记该编排依赖，SHALL NOT 改变跨模型事务模型（各模型 reconciler 独立）。

#### Scenario: 引用已配置目标合法
- **WHEN** acl group `G1` / xpl route-filter `RF1` 已由各自 reconciler 配置，随后下发 AF `acl-name-or-num=G1`/`filter-name=RF1`
- **THEN** 引用 SHALL 合法（目标实例存在），配置 SHALL 收敛

#### Scenario: 编排依赖登记（非静默）
- **WHEN** 规划 AF 策略属性下发
- **THEN** 系统文档/spec SHALL 明确"目标实例须先配置"的编排前置（不隐含跨模型自动事务）

### Requirement: AFPOL-03 完备测试矩阵（yang-config-test-design / T02b）

本 change 作为 BGP AF 策略属性接入 SHALL 触发 `yang-config-test-design`，产出并通过完备测试矩阵。矩阵 SHALL 覆盖：编码真值（属性值 + huawei-bgp namespace）、B2 端到端（经 ni reconciler 下发→回读→收敛）、幂等、并发-race、负路径、leafref 编排依赖登记。

#### Scenario: 幂等
- **WHEN** 对同一份含 AF 策略属性的配置连续下发两次
- **THEN** 第二次 SHALL 判定为 no-op（无 diff）

#### Scenario: 并发安全（race）
- **WHEN** 在 `-race` 下并发编码含 AF 策略属性的 ni 树
- **THEN** SHALL 无数据竞态、无 panic（R09）

#### Scenario: 负路径不 panic
- **WHEN** 以缺失/空 AF 策略属性容器编码
- **THEN** SHALL 正常跳过空字段、SHALL NOT panic（R08）
