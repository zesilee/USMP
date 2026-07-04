# retire-stack-a-entry — tasks（退役 cmd/controller，真·单进程）

> 依赖：proposal/design/specs。TDD（R06）。单 commit ≤500、PR ≤800（删除超出按文件分批）。§5.3：先补探活验证 → 再删旧入口/控制器。

## 1. DeviceHandler 在线探活（新路径）

- [x] 1.1 先写测试：ClientPool 桩——设备连接成功→`online:true`；`Get` 报错→`online:false`（R08）；多设备互不影响
- [x] 1.2 实现：`ListDevices` 逐设备 `ClientPool.Get`+`IsConnected` → 响应加 `online` 字段（DeviceInfo 或包装）；失败降级离线
- [x] 1.3 `go build ./...` + api 包测试绿

## 2. 退役 cmd/controller + 冗余控制器（删除，分批）

- [x] 2.1 删 `controllers/nativedeviceconfig_controller.go`（空壳下发，场景①由 config-api 承接）
- [x] 2.2 删 `controllers/businessroute_controller.go`（下发 TODO 空壳）
- [x] 2.3 删 `controllers/businessswitch_controller.go` + `controllers/retry.go`（探活已由 DeviceHandler 承接；retry 仅其用）
- [x] 2.4 删 `cmd/controller/main.go`（Stack A 入口）；grep 确认无残留引用
- [x] 2.5 每批 `go build ./...` + `go test ./...` 绿

## 3. 收尾与验收

- [x] 3.1 全量 `go test ./...` + `go build ./...` 绿；`backend/main.go` 单进程可起（无 K8s 优雅降级）
- [x] 3.2 更新 `system-architecture` 主 spec：单一入口、Stack A 退役
- [x] 3.3 满足 R01（单 Stack B 权威）/R08（探活降级）/R06（TDD）
- [~] 3.4 暂缓（记债）：CRD 类型 api/v1→biz/v1 收敛（D1）；物理删 actor 包（pr-size）；gNMI/plugin 空转（D3/D5）
