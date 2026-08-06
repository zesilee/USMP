## Why

真机回读中，`/devm:devm/devm:physical-entitys`、`/fib:fib/fib:route-statistics` 等路径解码失败（`multi-key lists unsupported`），整树降级原始透传，前端表格零行可渲染。扫描生成代码发现全部 68 个模块共有 **360 个多键（复合键）YANG list**（ARP 记录、BGP 路由表、BFD 会话、器件清单等，绝大多数为 config-false 状态/统计表）——这是通用解码引擎的系统性缺口，任何含多键列表的页面都会空表，不是个别模块问题。

## What Changes

- `pkg/yang-runtime/xmlcodec` 解码引擎 `entryKey` 支持**多键列表**：ygot 为多键列表生成带 `path:` tag 的复合 key struct（如 `..._PhysicalEntity_Key{Class, Position, SerialNumber}`），引擎经 `ΛListKeyMap()` 取键值、按 path tag 填充 key struct 作为 map key——不再对 `len(km) > 1` 报错。
- 根级列表与嵌套列表（`decodeField` Map 分支）同机制生效。
- key 叶缺失时的宽容语义扩展到多键：从条目自身键字段复制可得值构造 key struct（缺失字段为零值），条目不丢弃。
- **不在本次范围**：删除通道（`delete.go` XC-03）的多键支持维持「明确不支持错误」——多键列表绝大多数为只读状态表，删除支持由真实需求驱动另开 change。

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `yang-xml-codec`: XC-02 通用 XML 解码——条目 key 获取从「仅单键，多键报错」扩展为「单键与多键（复合 key struct）均支持」；宽容合成 key 语义对多键列表同样成立。XC-03 删除编码的多键不支持契约**不变**。

## Impact

- 代码：`backend/pkg/yang-runtime/xmlcodec/decode.go`（`entryKey` 及其两个调用点语义扩展），无 API/前端改动——解码成功后经既有 `EmitJSON RFC7951 → peelToPath` 链路自然透出行。
- 用户可感知：devm 物理器件、fib 路由统计等含多键列表的页面从空表变为正常渲染数据行。
- 测试：B1 表格驱动（多键解码正常/嵌套/缺键宽容/负路径 + race）+ B2 模拟网元集成（多键列表回读端到端）；delete 通道行为零回归。
