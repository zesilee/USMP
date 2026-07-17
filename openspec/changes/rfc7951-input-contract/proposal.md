# rfc7951-input-contract — 配置写入收敛为 RFC7951 单一契约（一步到位）

## Why

config-api 写路径现存双路径：generic（driver 注册表 → `ygot.Unmarshal` RFC7951）优先、失败落 685 行手写 legacy 转换器（仅覆盖 vlan/ifm/system 的 camelCase/复数键/整数枚举旧形状）。实测（2026-07-17 探针）：模块控制台 list 主流量已走 generic；仍踩 legacy 的只剩 form-tab 扁平载荷与旧形状测试；而**无 legacy 转换器的模块（bgp/ni/xpl…）的 form-tab 载荷会静默退化为 raw map 入库**（类型化 diff/编码链不认 map）——这是正确性隐患而不止是债。用户拍板：USMP 未现网运行，无演进包袱，**架构一步到位**：单一 RFC7951 输入契约，删光兼容路径。

## What Changes

- **输入契约收敛（BREAKING，无现网消费者）**：`POST /api/v1/config/:ip/*path` 的 body 契约 = **以 path 为根的 RFC7951 子树**（YANG 真名 kebab、枚举名字符串）。服务端按「描述符锚点相对包裹 + 根级 Unmarshal」单一路径解码（spike 已验证可行）；解不出/未注册路径 **显式 400**，SHALL NOT 静默存 raw map。
- **driver 注册表声明式化推进**：Descriptor 新增编码锚点路径字段（`NewStruct` 容器的规范配置路径），供包裹计算；谓词声明式化方向的第一块数据。
- **删除 legacy**：`convertToTypedStruct` + `convertMapToHuawei{Ifm,Vlan,System}`（~685 行，task 3.6 收口）；yang-api 硬编码 alias 假 schema 分支（task 2.5 收口）；孤儿 `DeviceConfigPage.vue`（354 行）+ 死类型 `yang-schema.ts`（1047 行）。
- **测试形状对齐**：~20 处后端测试的旧形状（复数键/camelCase）改 RFC7951；新增包裹/锚点 B1 矩阵与 form-tab 端到端 B2。

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `config-api`: BR-05 输入契约明确为 RFC7951 子树；BR-06 由「path 关键字路由到手写转换函数、其余回退原始 map」改为「driver 注册表锚点包裹 + ygot 根级解码，未注册/不可解显式 400」。
- `device-driver-registry`: 新增编码锚点路径契约（DR-05）。
- `frontend`: FE-03 配置下发主链路的载体由已退役的 `DeviceConfigPage` 改述为通用模块控制台（ModuleFormTab/ModuleListTab + useConfigSubmit），行为契约（下发→回读→轮询对账→降级）不变。

## Impact

- **后端**：`config_codec.go`（wrap+单一路径）、`config_handler.go`（删 685 行转换器）、`yang_handler.go`（删 alias 分支）、`pkg/yang-runtime/driver`（+锚点字段）、`internal/drivers/huawei.go`（登记锚点）。
- **前端**：删 `DeviceConfigPage.vue`、`types/yang-schema.ts`；现行控制台流零改动（list 已 RFC7951；form-tab 载荷=锚点相对子树，由服务端包裹正确解码）。
- **正确性收益**：form-tab 对全部模块（含 bgp/ni 等）从「可能 raw-map 静默入库」变为类型化解码或显式 400。
- **体积**：删除 ~2140 行 + 改动 ~500 行；分 3 PR（立项/实现/纯删除+收尾，按体积门禁拆）。
- **不受影响**：GET 读路径（本就 EmitJSON RFC7951）、DELETE 行删除通道（parseDeleteTarget 独立）、意图控制器写路径。
