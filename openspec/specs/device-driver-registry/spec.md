# device-driver-registry — 设备驱动描述符注册表

## Purpose

编译期设备驱动描述符注册表（`pkg/yang-runtime/driver`）：每 (vendor, module) 一条描述符，收敛此前散落在 manager 路由与 config 编解码里的路径字符串硬编码，是 SND 声明式化（P5）的承载点——①声明式数据驱动终态的路径/模板/能力元数据在描述符 struct 上扩展，不另起注册机制。纯 Go 编译期（用户拍板，无运行时插件加载）。

## Requirements

### Requirement: DR-01 编译期驱动描述符注册表

系统 SHALL 提供进程内驱动描述符注册表（`pkg/yang-runtime/driver`）：描述符携带 `Vendor`、`Module`、路由谓词与 `ControllerToken`（路由目标控制器名）、XML 解码与 RFC7951 编码闭包；SHALL 经 `init()` 编译期注册（无运行时插件加载），`Register`/`Lookup` 并发访问 SHALL 无数据竞态（R09）。查找未命中 SHALL 返回 `ok=false` 供调用方降级（R08），SHALL NOT panic。

#### Scenario: 注册并按路径查得
- **WHEN** 已注册 huawei/`vlan:` 描述符，`Lookup("huawei", "/vlan:vlans/...")`
- **THEN** SHALL 返回该描述符，`ok=true`

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

`/config` 读写链路的 per-model 编解码分发 SHALL 经驱动描述符注册表查表获得编解码闭包，SHALL NOT 以散落的路径字符串匹配硬编码；未命中路径 SHALL 保持既有降级行为（R08）。

#### Scenario: 既有编解码行为等价
- **WHEN** 对 ifm/vlan/system 路径执行配置读（decode）与写（encode）
- **THEN** 编解码结果 SHALL 与改造前逐字节一致（存量 B2/B3 套件全绿）
