# device-driver-registry — delta (retire-ygot-runtime)

## MODIFIED Requirements

### Requirement: DR-01 编译期驱动描述符注册表

系统 SHALL 提供进程内驱动描述符注册表（`pkg/yang-runtime/driver`）：描述符携带 `Vendor`、`Module`、路由谓词与 `ControllerToken`（路由目标控制器名）、XML 解码与 RFC7951 编码闭包，以及可选的 XML 编解码数据（模块 namespace + Schema IR 入口，供通用 XML 编解码引擎消费，见 yang-xml-codec XC-01/02/03；根元素名与 Object 类型自 IR/`NewStruct` 派生）；描述符签名 SHALL 以自研 `Object` 接口族（yang-native-runtime YN-01）表述，SHALL NOT 引用 `ygot.GoStruct`/`ytypes.UnmarshalOpt`；RFC7951 编解码闭包 SHALL 由生成的 per-type JSON 方法承载（YN-02）。注册体验 SHALL 保持「每模块一条描述符/表行」不变。SHALL 经 `init()` 编译期注册（无运行时插件加载），`Register`/`Lookup` 并发访问 SHALL 无数据竞态（R09）。查找未命中 SHALL 返回 `ok=false` 供调用方降级（R08），SHALL NOT panic。

#### Scenario: 注册并按路径查得
- **WHEN** 已注册 huawei/`vlan:` 描述符，`Lookup("huawei", "/vlan:vlans/...")`
- **THEN** SHALL 返回该描述符，`ok=true`

#### Scenario: 按 Object 类型查得编解码数据
- **WHEN** 以已注册模块的 Object 类型（含其内层 list map 形态）查找编码描述符
- **THEN** SHALL 返回携带 XML 编解码数据的描述符，`ok=true`

#### Scenario: 未命中降级（负路径）
- **WHEN** `Lookup` 一个无描述符覆盖的路径
- **THEN** SHALL 返回 `ok=false`，调用方保持既有 fallback 行为

#### Scenario: 并发注册与查询
- **WHEN** 多协程并发 `Register`/`Lookup`
- **THEN** SHALL 无数据竞态（`-race` 通过，R09）

#### Scenario: 类型置换零行为漂移
- **WHEN** 描述符签名从 ygot 类型切换为 Object 接口族后运行存量 B2/B3 套件
- **THEN** 路由/编解码行为 SHALL 与切换前完全一致（YN-06 对拍通道兜底）
