# snd-vendor-registry — SND 声明式化第一步：厂商注册表 + Vendor 贯穿（P5-1）

## Why

P5（异构多设备 SND 驱动）explore 审计（2026-07-09，代码级核对 @main，结论固化于 `openspec/tasks/optimize-frontend-nce-insights.md` P5 节）实证三个根因，使「加一个新厂商设备」在现架构下无处下手：

1. **vendor 概念在生产链路不存在**：`DeviceConnectionInfo`（`client/client.go:91`）只有 IP/端口/凭据/协议，无 Vendor 字段；协议按端口判（`pool.go:125`）；唯一「厂商识别」是只读 schema 路径的 namespace 猜测（`field_gen.go:186`），不流入下发链路。
2. **translator 是半成品注册表**：`factory.go:21` `once.Do` 硬编码只注册 Huawei；生产调用点（crdsource 两处）写死 `VendorHuawei` 常量——注册表 API 形同虚设。
3. **路径分发硬编码散布**：`manager.go:325-342` 路径→控制器路由、`config_codec.go:28/71` 编解码表均用 `strings.Contains` 写死 system/vlan/ifm 三前缀，加新模块/厂商必改多处。

用户已拍板（2026-07-09）：**不需要运行时可插拔，纯 Go 编译期**——③编译期注册表起步→①声明式数据驱动终态。本 change 是③的落地件与①的公共前置：只做「注册表骨架 + vendor 数据通路」，**不碰任何 XML 字节序列化**（netconf.go 手写 builder 是后续独立 change，需 golden test 先行），行为严格等价、低回归风险。

## What Changes

- **S1 Vendor 贯穿设备连接层**：`DeviceConnectionInfo` 增 `Vendor` 字段；`device.Store` 透传；`devices-api` 注册入口接受可选 `vendor`（缺省 `huawei`，行为不变）并写入 store；swagger/契约再生成。
- **S2 translator 编译期自注册**：`factory.go` 的 `once.Do` 硬注册改为 huawei 实现文件 `init()` 自注册（③教科书形态）；crdsource 调用点从写死 `VendorHuawei` 改为按设备 Vendor 解析（查 device.Store，查不到缺省 huawei 降级 R08）；翻译逻辑零改动。
- **S3 驱动描述符注册表（新包）**：新增 `(vendor, module-prefix) → driver descriptor` 编译期注册表；`manager.go:325` 路径→控制器路由与 `config_codec.go` 编解码表改为查表（描述符先只承载 controller-name 与 codec 闭包，为①的声明式描述预留承载点）；`strings.Contains` 硬编码收敛至注册表数据。

**范围外（后续 change）**：netconf.go per-model XML builder/parser 声明式化（最大块、需 golden test）；ygot 生成管线参数化（Makefile 化、多厂商 -path）；plugin 钩子的厂商校验落地；Cisco/H3C 真实驱动实现。

## Capabilities

### New Capabilities

- `device-driver-registry`：DR-01 编译期驱动描述符注册表、DR-02 路径→控制器路由查表化、DR-03 配置编解码查表化。

### Modified Capabilities

- `devices-api`：BR-03/BR-04 注册入口增可选 vendor（缺省 huawei、未知厂商 400）。
- `device-store`：DS-01 连接信息携带 Vendor、DS-03 注册写入 Vendor。
- `translation-engine`：迁 OpenSpec 标准格式（借机，spec-format-standard 军规）+ TE-01 编译期自注册、TE-02 按设备 Vendor 解析驱动。

## Impact

- **后端**：`pkg/yang-runtime/client/client.go`（Vendor 字段）、`pkg/yang-runtime/device/store.go`、`internal/api/device_handler.go`（注册入口+swagger）、`pkg/translator/factory.go`+`huawei.go`（init 自注册）、`internal/crdsource/businessvlan.go`/`businessinterface.go`（按 Vendor 解析）、新包 `pkg/yang-runtime/driver`（描述符注册表）、`pkg/yang-runtime/manager/manager.go`（路由查表）、`internal/api/config_codec.go`（编解码查表）。
- **前端**：仅 `api.gen.ts` 契约再生成（注册接口新可选字段），无 UI 改动。
- **测试层**（§5.6，涉协议编解码与 Reconciler 链路 → B1+B2）：B1（Vendor 贯穿/注册表并发 race/描述符查表边界+未知厂商负例，表格驱动）、B3（注册 API 带/缺省/非法 vendor 契约）、**B2 回归**（存量 netconfsim 集成套件全绿证行为等价，无新增用例诉求）。
- **门禁**：`make gen-contract`；覆盖率棘轮 后端 58.3 / 前端 74/71/67/74；PR ≤1000 行（预估 ~600-800 含测试）。
- 不涉及：数据库（R03）、YANG 模型变更（无 yang-config-test-design 触发）、Stack A Actor 路径（不复活，[[dual-stack-migration]]）。
