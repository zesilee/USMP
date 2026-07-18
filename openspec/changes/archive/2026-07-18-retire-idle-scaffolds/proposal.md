# retire-idle-scaffolds — 空转脚手架清除：plugin 包与 gNMI 空壳（roadmap D3/D5 终局）

## Why

架构优化路线仅剩的两笔空转债（2026-07-18 审计实证）：

- **D3 plugin**：`pkg/yang-runtime/plugin`（427 行）在 Manager 构造时即创建、接口全暴露，但全仓**零 AddPlugin 调用**、reconcile 流程**从不执行任何钩子**（yang-controller-runtime spec 已注记「声明但从不执行」）——纯空转脚手架。
- **D5 gNMI**：`client/gnmi.go`（274 行）的 Get/Set **发空请求**（device-protocol spec 注记「空壳」）——注册 9339 端口设备会「连接成功」然后伪装读写，是比死代码更糟的**假成功路径**（R08 精神反面）；`source/gnmi_sub.go`（71 行）订阅源零生产消费且依赖该 client。

用户方向：架构一步到位、无现网包袱——空壳删除，gNMI 降级为**显式未实现错误**（诚实拒绝优于伪装成功），协议本身保留为平台规划能力。

## What Changes

- **删 `pkg/yang-runtime/plugin` 整包**；Manager 收口：删 `GetPluginManager`/`AddPlugin` 接口方法与 `pluginManager` 字段；CLAUDE.md §4 C1 职责去「插件管理」。
- **删 `client/gnmi.go` 与 `source/gnmi_sub.go`**；`ProtocolGNMI` 枚举保留，factory 对 `GNMI` 与 `AUTO`+9339 SHALL 返回显式「gNMI 尚未实现（规划能力）」错误；`go mod tidy` 掉 openconfig/gnmi 直依赖。
- **文档对齐**：CLAUDE.md §1 协议行标注 gNMI 为规划能力、§4 C4 事件源改述（周期轮询 / CRD watch / 文件变更）；前端 Settings 端口提示标注「规划」。
- **FileSource 保留**（通用框架能力、有测试、无死依赖）。

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `yang-controller-runtime`: YR-07 事件源改述（周期轮询 / CRD watch / 文件变更，gNMI 订阅移除）；Purpose 缺口注记去掉 plugin 条目。
- `device-protocol`: DP-02 协议选择改述（GNMI/AUTO-9339 → 显式未实现错误）；DP-06 缺口清单更新（gNMI 空壳条目移除，代之以「gNMI 为规划能力、显式错误」）。

## Impact

- **删除**：plugin 整包（427）、gnmi.go（274）、gnmi_sub.go（71）+ manager 接口/字段 + go.mod 依赖瘦身。
- **行为变化（无现网消费者）**：Protocol=gnmi 或端口 9339 的设备注册后建连从「假成功」变为显式错误（devices-api 探活如实显示离线）；Device CRD 的 protocol 枚举保留 `gnmi` 值（前向兼容声明字段）。
- **交付**：单 change 两 PR（①立项 ②实现+删除+sync/archive——体积估 ~900 行，多为删除）。
