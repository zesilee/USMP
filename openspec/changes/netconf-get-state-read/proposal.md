# netconf-get-state-read

## Why

YANG 模型中 `config false` 的状态字段（如接口 `dynamic` 容器的 oper-status/mac-address/bandwidth、VLAN `status`）在前端只读控件中恒为空：后端读路径只发 `<get-config>`，按 RFC6241 语义它只返回配置数据——对真机和模拟网元都读不到状态数据。前端只读渲染管道（FieldDef.Readonly → disabled 控件）与通用解码（xmlcodec 不过滤 config-false）早已就绪，缺的只是「读态」这一环。

## What Changes

- 模拟网元 netconfsim 实现 NETCONF `<get>` RPC（RFC6241 §7.7）：返回 running 配置树与状态数据 overlay 树的合并结果，支持 subtree filter；`<get-config>` 行为不变（仅返回配置树）。
- 模拟网元新增状态数据 overlay 数据存：`SetStateDataXML` 注入、独立于 running 树（edit-config 不触碰）、附带 `DemoStateSeed` 演示种子（IFM 接口 `dynamic` + VLAN `status`，设备侧数字枚举形态）。
- device-protocol NETCONF 客户端新增 `WithStateData()` GetOption：置位时发 `<get>`（scrapligo `Driver.Get`），缺省仍发 `<get-config>`；断线重试/幂等语义与现有读一致。
- config-api `GET /config` 读路径改用 `WithStateData()`：返回数据含 config=false 状态子树，缓存/TTL/降级语义不变。
- 对账（Reconciler）读路径**不变**：仍走 `<get-config>`，diff 只比配置数据（desired⊆actual 子集比对本就忽略多余叶，此处显式保持语义纯净）。
- standalone 模拟网元二进制（cmd/netconf-simulator）启动时注入 `DemoStateSeed`，staging E2E 可断言只读字段有值。

## Capabilities

### New Capabilities

（无——全部为既有能力的 Requirement 扩展）

### Modified Capabilities

- `netconf-simulator`: 新增 Requirement NS-08「`<get>` RPC 与状态数据 overlay」——`<get>` 返回配置+状态合并树（支持 subtree filter），状态树独立注入、不受 edit-config 影响；未注入状态时 `<get>` 退化为与 `<get-config>` 等值。
- `device-protocol`: 新增 Requirement DP-09「NETCONF `<get>` 状态读」——`WithStateData()` 置位时发 `<get>` 携 subtree filter；缺省行为（DP-03 get-config）不变。
- `config-api`: 修改 Requirement BR-01「配置读取」——`GET /config` 回读数据 SHALL 包含设备返回的 config=false 状态子树（有则带出，无则不构造）；缓存键/TTL/新鲜度语义不变。

## Impact

- `backend/simulator/netconfsim/`：server.go（RPC 分发 + handleGet）、tree_datastore.go 或新 state 存、seed.go（DemoStateSeed）、cmd/netconf-simulator/main.go。
- `backend/pkg/yang-runtime/client/`：client.go（GetOption）、netconf.go（Get 按 option 选 RPC）。
- `backend/internal/api/config_handler.go`：fetchFromDevice 加 `WithStateData()`。
- 前端零改动（只读渲染管道已就绪）；staging E2E 冒烟可追加只读字段断言。
- 测试：B1（sim get/合并/filter/并发）、B2（集成：seed 状态→API 回读含状态）、B3（handler 含状态数据透传）；覆盖率棘轮不下降（T08）。
