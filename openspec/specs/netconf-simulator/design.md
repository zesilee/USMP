# netconf-simulator — 模拟网元架构设计（反向还原）

> **权威性**：测试专用设施（支撑 T02 集成测试）。存在**两个无关的模拟器**（迁移债 D10）。
> **还原基准**：`main@b1cfbae`，代码根 `backend/simulator/`。

## 1. 职责

为「新增 YANG 业务的端到端集成测试」提供可编程的模拟设备对端（T02：新增 YANG 模块必须有 NETCONF 模拟网元集成测试）。

## 2. ⚠️ 两个并存的模拟器

| | `simulator/netconfsim` | `simulator/netsim` |
|---|---|---|
| 形态 | **真实 SSH 服务器**，讲 NETCONF | 内存 VLAN 模拟，REST 直供 |
| 传输 | SSH `127.0.0.1:0`(随机端口) + password(admin/admin) + 生成 RSA host key | 无网络，进程内 |
| 用途 | `*_integration_test.go`（actor、controller/{system,ifm,interfaces,vlan}、e2e） | `cmd/test-server/main.go`（前端 Playwright E2E 后端） |
| 概念 | 协议级仿真 | 数据级桩 |

二者无关并存——**`cmd/test-server` 跑的不是 netconfsim**。

## 3. `netconfsim` 组件

- `Simulator`（`simulator.go`）：起 SSH server（`server.go:33`），接受 `netconf` subsystem，发 NETCONF 1.0 hello（`server.go:85`），按字符串匹配分发 RPC（`server.go:156`）：
  - `get-config` → 返回 running/candidate 数据存 XML（`server.go:245`）
  - `edit-config` → 解析写 candidate（或 running+commit）（`server.go:260`）
  - `commit` → candidate→running（`server.go:336`）
  - `discard-changes` → 丢弃 candidate（`server.go:349`）
  - 其它 → `<ok/>`
- `Datastore`（`datastore.go`）：running/candidate XML 缓冲 + `SetCandidate`/`Commit`/`DiscardCandidate` + `Extract{VLANs,Interfaces,HuaweiVLANs,HuaweiInterfaces,HuaweiSystem}` 断言助手。
- **故障注入** `ScenarioConfig`（`scenarios.go:11`）：`ErrorOnRPC` map（get-config/edit-config/commit/discard-changes）+ `RejectAuth`，供测试构造设备离线/下发失败等异常路径（呼应 §9 异常处理）。
- 内置 YANG 模型 `netconfsim/yang/`（openconfig-interfaces/vlan/types/extensions）。

## 4. `netsim` 组件（供 test-server）

内存 VLAN 存储，暴露 `GetAllVLANs/AddVLAN/GetVLAN/DeleteVLAN`；`cmd/test-server/main.go`（Gin :8080，`gin.TestMode`）直接以其为后端，路由 `/api/v1/devices`、`/config/:ip/vlans[...]` CRUD，服务前端 Playwright 套件。

## 5. 数据流（集成测试）

```
*_integration_test.go
  → netconfsim.NewSimulator() 起随机端口 SSH
  → 被测 reconciler/actor 经真实 NETCONF 客户端连接
  → edit-config/commit 落到模拟 Datastore
  → Extract*/断言校验 running 结果
  [异常测试] ScenarioConfig.ErrorOnRPC / RejectAuth 触发降级路径
```

## 6. as-built 缺口

| 缺口 | 说明 |
|------|------|
| 两个模拟器概念重叠 | `netconfsim`(协议级) 与 `netsim`(数据级) 并存（迁移债 D10） |
| netconfsim 仅测试可见 | `scenarios.go`/`simulator.go` 直接 import `testing`，非独立可部署网元 |

## 7. 关联
- `netconf-sim-integration-test` 技能；`device-protocol/design.md`（被测客户端对端）；`backend/deploy/manifests/netconf-simulator`（Kind 部署清单，另有容器化模拟器）。
