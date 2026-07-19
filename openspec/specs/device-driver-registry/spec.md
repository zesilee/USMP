# device-driver-registry — 设备驱动描述符注册表

## Purpose

编译期设备驱动描述符注册表（`pkg/yang-runtime/driver`）：每 (vendor, module) 一条描述符，收敛此前散落在 manager 路由与 config 编解码里的路径字符串硬编码，是 SND 声明式化（P5）的承载点——①声明式数据驱动终态的路径/模板/能力元数据在描述符 struct 上扩展，不另起注册机制。纯 Go 编译期（用户拍板，无运行时插件加载）。

## Requirements

### Requirement: DR-01 编译期驱动描述符注册表

系统 SHALL 提供进程内驱动描述符注册表（`pkg/yang-runtime/driver`）：描述符携带 `Vendor`、`Module`、路由谓词与 `ControllerToken`（路由目标控制器名）、XML 解码与 RFC7951 编码闭包，以及可选的 XML 编解码数据（模块 namespace + SchemaTree 入口，供通用 XML 编解码引擎消费，见 yang-xml-codec XC-01/02/03；根元素名与 GoStruct 类型自 SchemaTree/`NewStruct` 派生）；SHALL 经 `init()` 编译期注册（无运行时插件加载），`Register`/`Lookup` 并发访问 SHALL 无数据竞态（R09）。查找未命中 SHALL 返回 `ok=false` 供调用方降级（R08），SHALL NOT panic。

#### Scenario: 注册并按路径查得
- **WHEN** 已注册 huawei/`vlan:` 描述符，`Lookup("huawei", "/vlan:vlans/...")`
- **THEN** SHALL 返回该描述符，`ok=true`

#### Scenario: 按 GoStruct 类型查得编解码数据
- **WHEN** 以已注册模块的 GoStruct 类型（含其内层 list map 形态）查找编码描述符
- **THEN** SHALL 返回携带 XML 编解码数据的描述符，`ok=true`

#### Scenario: 未命中降级（负路径）
- **WHEN** `Lookup` 一个无描述符覆盖的路径
- **THEN** SHALL 返回 `ok=false`，调用方保持既有 fallback 行为

#### Scenario: 并发注册与查询
- **WHEN** 多协程并发 `Register`/`Lookup`
- **THEN** SHALL 无数据竞态（`-race` 通过，R09）

### Requirement: DR-02 路径→控制器路由查表化

Manager 的配置路径→控制器路由 SHALL 经驱动描述符注册表查表获得目标控制器名，SHALL NOT 在 Manager 内硬编码模块前缀字符串匹配；注册表未命中时 SHALL 保持既有未匹配路径行为（等价降级）。

#### Scenario: 既有三模块路由等价
- **WHEN** 分别以 system/vlan/ifm 前缀路径触发路由
- **THEN** SHALL 路由到与改造前完全一致的控制器（逐路径对拍）

### Requirement: DR-03 配置编解码查表化

`/config` 读写链路的 per-model 编解码分发 SHALL 经驱动描述符注册表查表获得编解码闭包，SHALL NOT 以散落的路径字符串匹配硬编码；未命中路径 SHALL 保持既有降级行为（R08）。描述符的 XML 解码 SHALL 由通用编解码引擎按描述符数据驱动（yang-xml-codec XC-02），SHALL NOT 指向 per-model 手写解析器。

#### Scenario: 既有编解码行为等价
- **WHEN** 对 ifm/vlan/system 路径执行配置读（decode）与写（encode）
- **THEN** 编解码结果 SHALL 与改造前逐字节一致（存量 B2/B3 套件全绿）

#### Scenario: 回读解码经通用引擎
- **WHEN** vlan/ifm 模块的 XML 回读解码
- **THEN** SHALL 经描述符数据驱动的通用解码器完成，字段覆盖与下发编码对称（XC-02）

### Requirement: DR-04 厂商支持性查询

注册表 SHALL 提供 `VendorSupported(vendor string) bool` 查询（含包级 facade）：存在任一描述符的 `Vendor` 与入参**大小写无关**匹配时返回 true，否则返回 false；SHALL NOT panic（R08）。查询与 `Register` 并发调用 SHALL 无数据竞态（R09）。该查询是「厂商是否有已注册驱动」的唯一事实源，供 devices-api BR-04 厂商门禁消费（取代已退役 translation-engine 注册表的 `IsVendorSupported`）。

#### Scenario: 已注册厂商大小写无关命中
- **WHEN** 注册表含 `Vendor="huawei"` 的描述符，查询 `VendorSupported("Huawei")`
- **THEN** SHALL 返回 true

#### Scenario: 未注册厂商（负路径）
- **WHEN** 查询 `VendorSupported("nokia")` 且无任何 nokia 描述符
- **THEN** SHALL 返回 false，SHALL NOT panic

#### Scenario: 与注册并发（race）
- **WHEN** `Register` 与 `VendorSupported` 被多 goroutine 并发调用（`-race`）
- **THEN** SHALL 无数据竞态告警

### Requirement: DR-05 编码锚点声明

编码描述符 SHALL 声明 `EncodeAnchor`——其 `NewStruct` 容器对应的规范配置路径（如 `/vlan:vlan/vlan:vlans`、`/system:system`）。config-api 写路径 SHALL 据此计算请求路径相对锚点的后缀段并机械包裹请求体（段名剥模块前缀）后根级解码；请求路径与锚点不构成前缀关系时调用方 SHALL 显式拒绝。该字段是 SND 谓词声明式化的第一块数据（后续 MatchEncode 可由锚点派生）。

#### Scenario: 锚点相对包裹解码
- **WHEN** system 描述符锚点为 `/system:system`，请求路径 `/system:system/system:system-info` 携扁平子树
- **THEN** 写路径 SHALL 包裹为 `{"system-info": <子树>}` 并成功解码进 `HuaweiSystem_System`

#### Scenario: path 即锚点零包裹
- **WHEN** 请求路径与描述符锚点相同
- **THEN** 请求体 SHALL 原样作为锚点相对 RFC7951 解码

#### Scenario: 非前缀路径拒绝（负路径）
- **WHEN** 请求路径不以该描述符锚点为前缀
- **THEN** 调用方 SHALL 显式拒绝（400），SHALL NOT 猜测包裹

### Requirement: DR-06 表驱动 plain-container 注册与根名路径约定

对「单容器根、走通用 XML 引擎」的模块（plain-container 形态），注册 SHALL 表驱动：
每模块一条数据行（模块名/根容器名/namespace/GoStruct 构造子），由统一注册循环派生
全部谓词与锚点，SHALL NOT 逐模块手写描述符代码块。运行时配置路径 SHALL 以
**根容器名**为前缀段（`/<root>:<root>/...`，与前端 `configPathFor` 派生口径一致），
描述符谓词与 `EncodeAnchor` SHALL 锚定该口径；SHALL NOT 使用 YANG prefix 作为
路由 token（YANG prefix 仅属 XML 编解码的 namespace 层）。表行 namespace SHALL
显式登记（内嵌 schema 的 `Entry.Namespace()` 为空、不可派生）且全表唯一。

#### Scenario: 表行注册可路由可编解码

- **WHEN** 表中登记模块 `ntp`（根 `ntp`，namespace `urn:huawei:yang:huawei-ntp`）
- **THEN** `EncoderFor("/ntp:ntp")`/`DecoderFor("/ntp:ntp")` SHALL 命中，
  `EncodeAnchor` SHALL 为 `/ntp:ntp`，XML 编码 SHALL 携带登记的 namespace

#### Scenario: 根名与 YANG prefix 不同的模块按根名路由（回归）

- **WHEN** 模块根容器名与 YANG prefix 不同（如 `tunnel-management`/`tnlm`、
  `routing-policy`/`rtp`、`network-instance`/`ni`），前端按根名派生路径
  `/tunnel-management:tunnel-management/...`
- **THEN** 描述符 SHALL 命中该路径（此前锚定 YANG prefix 导致控制台写链路不可达）

#### Scenario: 全表 namespace 唯一且 SchemaTree 入口存在

- **WHEN** 遍历注册表全部华为描述符
- **THEN** 每条 XML spec 的 namespace SHALL 非空且互不重复，Schema 闭包 SHALL 能
  解析出对应根容器 Entry；违反即单测失败（防表行漂移）
