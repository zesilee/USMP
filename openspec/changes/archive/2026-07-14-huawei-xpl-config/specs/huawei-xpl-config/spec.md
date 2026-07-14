## ADDED Requirements

### Requirement: XPL-01 xpl route-filter 配置面接入

系统 SHALL 通过 Stack B 驱动注册表接入华为 `huawei-xpl` 模块顶层独立根容器 `/xpl:xpl` 下 BGP 引用的目标子树 `route-filters/route-filter`，提供读改下发闭环，SchemaTree 入口为容器根 `HuaweiXpl_Xpl`（容器根非 list 根，走通用引擎 plain-container，见 yang-xml-codec XC-05）。xpl 配置 SHALL 由通用 XML 编解码引擎（`pkg/yang-runtime/xmlcodec`）按 ygot 生成数据驱动编解码，SHALL NOT 手写 per-model XML 解析/序列化。

`huawei-xpl` 的 ygot 结构体已随一期 `huawei-bgp` 全闭包生成，本 change **SHALL NOT 修改 `gen.conf`、SHALL NOT 重新生成、SHALL NOT 手改 `generated/`**（R04）。

本波次范围 SHALL 覆盖 `route-filters/route-filter`（key=`name`）下全部 config-true leaf，无遗漏：

- `name`（`xpl-filter-name` 类型，key）
- `content`（string 1..16380，mandatory，XPL 策略正文）

系统 SHALL NOT 做字段挑选式覆盖，完备性 SHALL 由 schema 驱动的用例保证。

本波次 SHALL NOT 覆盖（注册为 follow-up，非简化遗漏）：xpl 的其他策略 list——`global`、`as-path-lists`、`community-lists`、`ext-community-rt-lists`、`ext-community-soo-lists`、`ipv4-prefix-lists`、`ipv6-prefix-lists`、`rd-lists`、`large-community-lists`、`route-flow-group-lists`、`interface-lists` 等。它们非 BGP `route-filter` leafref 目标（BGP 仅引用 `route-filters/route-filter`），各由自身消费者门控。config-false 只读态 SHALL NOT 作为下发目标。

#### Scenario: 下发 route-filter 并回读收敛
- **WHEN** 向模拟网元下发 `/xpl:xpl/route-filters/route-filter`（`name=<有效名>`、`content=<XPL 正文>`）
- **THEN** 系统 SHALL 编码为 `huawei-xpl` namespace 的 NETCONF edit-config 报文下发，回读 running config 后 desired↔actual 收敛（无持续漂移）

#### Scenario: 全属性可配（本波次字段）
- **WHEN** 依次配置 route-filter 的 name、content
- **THEN** 每个字段 SHALL 能成功编码下发并原值回读，无字段丢失或被静默丢弃

#### Scenario: 不触碰 generated（R04 门禁）
- **WHEN** 运行 R04 regen-and-diff 门禁
- **THEN** `gen.conf` 与 `generated/` SHALL 相对基线零改动（本 change 不涉 codegen）

### Requirement: XPL-02 命名空间显式登记

xpl 驱动描述符 SHALL 显式携带模块 XML namespace 常量 `urn:huawei:yang:huawei-xpl`，SHALL NOT 依赖内嵌 gzip schema 的 `Entry.Namespace()` 派生（实测返回空）。编码产出的 xpl 配置报文根元素 SHALL 归属该 namespace。

#### Scenario: 编码报文携带正确 namespace
- **WHEN** 编码一份 `/xpl:xpl/route-filters/route-filter` 配置
- **THEN** 输出 XML 根容器 SHALL 声明 `urn:huawei:yang:huawei-xpl` namespace（前缀不敏感）

### Requirement: XPL-03 路由/编解码谓词精确锚定

xpl 驱动描述符的 `MatchRoute`/`MatchDecode`/`MatchEncode` 谓词 SHALL 精确锚定根路径 `/xpl:xpl`，SHALL NOT 误命中其他模块。查找未命中的路径 SHALL 返回 `ok=false` 供调用方降级（R08），SHALL NOT panic。

#### Scenario: xpl 路径命中
- **WHEN** 以 `/xpl:xpl/...` 路径触发路由/编解码分发
- **THEN** SHALL 命中 xpl 描述符（`ControllerToken="xpl"`）

#### Scenario: 注册可达性
- **WHEN** xpl 集成测试所在二进制/独立测试包运行
- **THEN** 该二进制 SHALL 空白导入 `internal/drivers` 触发注册，`Lookup("huawei", "/xpl:xpl/...")` SHALL 返回 `ok=true`

### Requirement: XPL-04 模拟网元端到端集成

`simulator/netconfsim` SHALL 支持 xpl edit-config（整树替换语义，对齐既有 RFC edit-config 通道）与 get-config 回读，支撑 Reconciler↔设备端到端集成测试（B2，`*_integration_test.go`，`testing.Short()` 跳过）。集成测试 SHALL 覆盖下发→回读→收敛全链路。

#### Scenario: 模拟网元接受并回读 route-filter 配置
- **WHEN** 集成测试向 netconfsim 下发 route-filter 配置并随后 get-config
- **THEN** netconfsim SHALL 返回与下发等价的 running config，Reconciler 判定收敛

#### Scenario: 重复下发幂等
- **WHEN** 对同一份 xpl 配置连续下发两次
- **THEN** 第二次 SHALL 判定为 no-op（无 diff、无重复 edit-config 副作用）

### Requirement: XPL-05 完备测试矩阵（yang-config-test-design / T02b）

本 change 作为「新增 YANG 模型接入设备配置」SHALL 触发 `yang-config-test-design`，产出并通过完备测试矩阵，未覆盖视为未完成、禁止合并（T02b）。矩阵 SHALL 覆盖：全属性可配、端到端到设备（B2）、并发-race、边界、幂等、负路径、删除语义。

#### Scenario: 边界校验
- **WHEN** 配置 route-filter `content`（长度 1 / 16380）的下界/上界值
- **THEN** 合法值 SHALL 成功编解码往返；越界值 SHALL 由校验拦截（不静默下发非法配置）

#### Scenario: 并发安全（race）
- **WHEN** 在 `-race` 下并发触发 xpl 编解码/查找
- **THEN** SHALL 无数据竞态、无 panic（R09）

#### Scenario: 负路径不 panic
- **WHEN** 以缺失/畸形 xpl 路径或空配置触发编解码
- **THEN** SHALL 返回明确错误或降级，SHALL NOT panic（R08）

#### Scenario: 删除语义（声明式 subset + DELETE 通道推迟债）
- **WHEN** 从 desired 移除某 route-filter 条目后下发
- **THEN** 沿用平台声明式 subset 语义（天然不删），DELETE 命令通道作为推迟债显式登记（同 network-instance NI-06 / tunnel-management 语义）
