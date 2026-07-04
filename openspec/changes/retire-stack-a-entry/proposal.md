## Why

架构优化 P1/P2/P3 已把两个配置面收编到 Stack B、Actor 退出生产。最后一块 Stack A 残留是 `cmd/controller`（controller-runtime 独立入口），它跑 3 个控制器，复查均为**冗余或空壳**：BusinessSwitch 仅做设备在线探活（写 CR status）、BusinessRoute 下发是 `TODO` 空壳、NativeDeviceConfig 下发是空壳（场景①已由 P1 config-api 承接）。而 Stack B 已有 `DeviceHandler` + `/api/devices` REST 管理设备清单——只差**每设备在线状态**（cmd/controller 唯一的真实功能）。前端不引用 BusinessSwitch CR。因此给 DeviceHandler 补在线探活后，`cmd/controller` 即全冗余，可退役，实现 P2「统一进程」的最后收官。

## What Changes

- **DeviceHandler 补在线状态探活**：`ListDevices` 对每个设备经 `ClientPool.Get` + `IsConnected()` 返回 `online/offline`（Stack B REST 直接暴露设备在线状态，替代 BusinessSwitch 控制器的 CR-status 探活）。
- **退役 `cmd/controller` 入口**：删除 `cmd/controller/main.go`（controller-runtime 独立进程）。
- **删除冗余/空壳控制器**：`controllers/businessswitch_controller.go`（探活已由 DeviceHandler 承接）、`controllers/businessroute_controller.go`（下发空壳）、`controllers/nativedeviceconfig_controller.go`（下发空壳，场景①由 config-api 承接）、`controllers/retry.go`（仅被上述控制器用）。
- **BREAKING（内部）**：BusinessSwitch/Route/NativeDeviceConfig 的 controller-runtime reconcile 退役；CRD 类型（`api/v1`）暂留（收敛留 D1）。设备在线状态改由 Stack B REST `/api/devices` 暴露（前端不依赖 CR status，无对外损失）。
- **结果**：`backend/main.go` 成为**唯一生产入口**（原生面 reconciler + CRD 意图源 + 北向 API + 设备注册/探活），Stack A 入口消失，真·单进程。

## Capabilities

### Modified Capabilities
- `config-api`: DeviceHandler `/api/devices` 增每设备在线状态（ClientPool 探活）；设备注册/状态成为 Stack B REST 的职责。
- `business-crd`: BusinessSwitch/Route/NativeDeviceConfig 的独立 controller-runtime 控制器退役（冗余/空壳）；CRD 类型暂留。
- `system-architecture`: 退役 `cmd/controller` Stack A 入口，`backend/main.go` 单进程。

## Impact

- **后端**：`internal/api/device_handler.go`（补探活）、删 `cmd/controller/main.go` + `controllers/{businessswitch,businessroute,nativedeviceconfig}_controller.go` + `controllers/retry.go`。
- **测试**：DeviceHandler 在线状态单测（ClientPool 桩：在线/离线）；全量 `go test ./...` + `go build ./...` 绿。
- **红线**：R01（单 Stack B 权威）、R08（探活失败=离线降级）、R06（TDD）。
- **不在范围**：物理删 `pkg/yang-runtime/actor`（pr-size 清理债）；CRD 树 api/v1→biz/v1 收敛（D1）；gNMI/plugin 空转（D3/D5）。
- **迁移策略（§5.3）**：先补 DeviceHandler 探活（新能力）→ 验证 → 再删冗余控制器 + `cmd/controller`（双路径已切换）。
