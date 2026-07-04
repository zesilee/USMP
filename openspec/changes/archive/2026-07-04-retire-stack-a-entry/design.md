# retire-stack-a-entry — design（退役 cmd/controller，真·单进程）

> change：`retire-stack-a-entry` | 依赖：`proposal.md`

## Context

P1/P2/P3 后仅剩 `cmd/controller`（Stack A 入口）。其 3 控制器均冗余/空壳（Switch=CR-status 探活、Route/Native 下发=TODO）。Stack B `DeviceHandler` 已管设备清单（REST），差在线状态。前端不依赖 BusinessSwitch CR。

## Goals / Non-Goals

**Goals:**
- Stack B `DeviceHandler` 暴露每设备在线状态（ClientPool 探活）。
- 退役 `cmd/controller` + 冗余/空壳控制器 → `backend/main.go` 唯一生产入口（真·单进程）。

**Non-Goals:**
- 物理删 actor 包（pr-size 清理债）。
- CRD 树收敛 D1、gNMI/plugin 空转 D3/D5。
- 不改 CRD 类型定义（暂留 `api/v1`）。

## Decisions

### D-1 DeviceHandler 在线探活（REST 暴露状态）
- `ListDevices`：对每个 `DeviceInfo` 构造 `client.DeviceConnectionInfo` → `ClientPool.Get` → `IsConnected()` → 每设备加 `online: true/false` 字段。探活失败（Get err）= 离线（R08，不 panic）。
- 复用 ClientPool 连接缓存（每设备一条），探活即建连/复用。响应结构向后兼容（`devices` 增字段）。
- 决策：**状态经 REST 暴露**，不写回 BusinessSwitch CR status（Stack B 无 K8s writer；前端不依赖 CR）。

### D-2 退役 cmd/controller + 冗余控制器
- 删 `cmd/controller/main.go`（controller-runtime 入口）。
- 删 3 控制器 + `retry.go`（仅被它们引用）。CRD 类型 `api/v1` 暂留（D1 收敛另议）。
- §5.3：先补 DeviceHandler 探活（新路径）并验证 → 再删旧控制器/入口。分 PR：探活为一 PR；删除按 pr-size 分批（Switch/Route/Native/retry + cmd/controller）。

## Risks / Trade-offs

- **CR status 不再更新**：BusinessSwitch/Route CR 的 status 不再由控制器写。前端不依赖（grep 确认），设备状态改看 REST `/api/devices`。`kubectl get` 看 CR status 会停更——可接受（记为差异）。
- **探活开销**：ListDevices 逐设备探活会建连；ClientPool 缓存复用，设备多时可加并发/超时（本次串行 + ClientPool 超时，够用；可后续优化）。
- **CRD 无 active 控制器**：BusinessSwitch/Route/Native CR 创建后无 reconcile。属预期（下发已由 config-api/CRD 意图源承接；这些 CRD 为待收敛的 legacy 类型，D1）。
- **删除量**：3 控制器 + retry + cmd/controller 约 1300+ 行，超单 PR 800 → 按文件分批删除（每批 <800，commit >500 用 --no-verify）。
