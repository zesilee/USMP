# netconf-simulator — 模拟网元架构设计（反向还原）

> **权威性**：测试专用设施（支撑 T02 集成测试）。经 `refactor-netconf-simulator` 重构后为**单一、结构化、协议保真、测试/部署解耦**的 NETCONF 模拟网元（迁移债 D10 已消除）。
> **还原基准**：重构后 `main`，代码根 `backend/simulator/netconfsim/`。

## 1. 职责

为「新增 YANG 业务的端到端集成测试」提供可编程的模拟设备对端（T02：新增 YANG 模块必须有 NETCONF 模拟网元集成测试），并可作为独立二进制容器化部署。

## 2. 单一模拟器（D10 已解决）

重构前存在两个职责重叠的模拟器（`netconfsim` 真 SSH + `netsim` 裸内存），现已统一：

- 保留唯一的 `simulator/netconfsim`（协议级，真实 SSH + 结构化 XML 数据树）。
- 删除 `simulator/netsim`；其唯一使用方 `cmd/test-server` 改用诚实命名的内存 VLAN REST 桩（`cmd/test-server/fixture.go`，与 NETCONF 无关，仅服务前端 Playwright E2E 的 REST 接口）。

## 3. `netconfsim` 组件

- `Simulator`（`simulator.go`）：起 SSH server（`server.go`），接受 `netconf` subsystem，发 NETCONF hello；**hello 广告 `base:1.0` + `:candidate` + `:writable-running`**（`buildHello`，保持 1.0 EOM framing）。
- **结构化 RPC 分发**（`classifyRPC`）：用 `encoding/xml` 解码 `<rpc>` 判定操作类型，替代旧的 `strings.Contains` 子串匹配：
  - `get-config` → 返回 running/candidate 树序列化 XML（可套 subtree filter）
  - `edit-config` → 写 candidate（或 running+commit）
  - `commit` → candidate→running；`discard-changes` → 丢弃 candidate；其它 → `<ok/>`
- **结构化数据存** `treeDatastore`（`tree_datastore.go` + `datatree.go`）：running/candidate 均为通用 XML 数据树 `*dataNode`（模型无关，huawei/openconfig 统一处理）。提供：
  - `SetCandidate`/`Commit`/`DiscardCandidate`/`GetRunning`/`GetCandidate`（整树替换 + 深拷贝隔离）
  - `EditConfig`：RFC6241 `merge/replace/create/delete/remove` operation 增量语义（`editconfig.go`）
  - `GetConfigFiltered`：RFC6241 §6 subtree filter（`filter.go`）
- **断言查通用树**：`RunningHuaweiVLANs`/`...Full`/`RunningHuaweiInterfaces`/`RunningHuaweiSystem`/`RunningOCInterfaces`（`query.go`），供 `testsupport` 结构化断言，取代旧 blob `Datastore.Extract*` 的 XML string-parsing。
- **故障注入** `ScenarioConfig`（`scenarios.go`）：`ErrorOnRPC` map + `RejectAuth`，供测试构造设备离线/下发失败等异常路径（呼应 §9 异常处理）。
- **测试/部署解耦**：core 包无 `testing` 依赖；`Assert*` 助手位于子包 `netconfsim/testsupport`；独立二进制 `cmd/netconf-simulator`（flag `-addr/-port`，`SetListen`）可构建容器化部署。
- 内置 YANG 模型 `netconfsim/yang/`（openconfig-interfaces/vlan/types/extensions）。

## 4. test-server（前端 E2E 后端）

`cmd/test-server`（Gin :8080，`gin.TestMode`）以进程内 `vlanStore`（`fixture.go`，并发安全内存 VLAN 存储）为后端，路由 `/api/v1/devices`、`/config/:ip/vlans[...]` CRUD，服务前端 Playwright 套件。**该服务不经 NETCONF，为诚实命名的 REST 桩**。

## 5. 数据流（集成测试）

```
*_integration_test.go
  → netconfsim.NewSimulator() 起随机端口 SSH
  → 被测 reconciler/actor 经真实 NETCONF 客户端连接
  → edit-config/commit 落到结构化 treeDatastore
  → testsupport.Assert*（查通用树）校验 running 结果
  [异常测试] ScenarioConfig.ErrorOnRPC / RejectAuth 触发降级路径
```

## 6. as-built 状态（重构后）

| 项 | 状态 |
|------|------|
| 单一模拟器（D10） | ✅ 已解决：删除 netsim，test-server 改内存 REST 桩 |
| 结构化数据存 | ✅ `treeDatastore` 通用 XML 树，消除 1068 行 string-parsing |
| 协议保真 | ✅ capability 广告 + `encoding/xml` 结构化 RPC 分发 |
| 测试/部署解耦 | ✅ core 无 `testing` 依赖；独立二进制 `cmd/netconf-simulator` |

## 7. 关联
- `netconf-sim-integration-test` 技能；`device-protocol/design.md`（被测客户端对端）；`backend/deploy/manifests/netconf-simulator`（Kind 部署清单，容器化模拟器）。
- 重构记录：`openspec/changes/archive/refactor-netconf-simulator/`（proposal/design/tasks）。
