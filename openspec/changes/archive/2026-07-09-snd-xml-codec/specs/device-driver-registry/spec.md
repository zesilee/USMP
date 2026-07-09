# device-driver-registry — delta（snd-xml-codec）

## MODIFIED Requirements

### Requirement: DR-01 编译期驱动描述符注册表

系统 SHALL 提供进程内驱动描述符注册表（`pkg/yang-runtime/driver`）：描述符携带 `Vendor`、`Module`、路由谓词与 `ControllerToken`（路由目标控制器名）、XML 解码与 RFC7951 编码闭包，以及可选的 XML 编解码数据（根元素名、模块 namespace、SchemaTree 入口、GoStruct 类型标识——供通用 XML 编解码引擎消费，见 yang-xml-codec XC-01/02/03）；SHALL 经 `init()` 编译期注册（无运行时插件加载），`Register`/`Lookup` 并发访问 SHALL 无数据竞态（R09）。查找未命中 SHALL 返回 `ok=false` 供调用方降级（R08），SHALL NOT panic。

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

### Requirement: DR-03 配置编解码查表化

`/config` 读写链路的 per-model 编解码分发 SHALL 经驱动描述符注册表查表获得编解码闭包，SHALL NOT 以散落的路径字符串匹配硬编码；未命中路径 SHALL 保持既有降级行为（R08）。描述符的 XML 解码 SHALL 由通用编解码引擎按描述符数据驱动（yang-xml-codec XC-02），SHALL NOT 指向 per-model 手写解析器。

#### Scenario: 既有编解码行为等价
- **WHEN** 对 ifm/vlan/system 路径执行配置读（decode）与写（encode）
- **THEN** 编解码结果 SHALL 与改造前逐字节一致（存量 B2/B3 套件全绿）

#### Scenario: 回读解码经通用引擎
- **WHEN** vlan/ifm 模块的 XML 回读解码
- **THEN** SHALL 经描述符数据驱动的通用解码器完成，字段覆盖与下发编码对称（XC-02）
