# netconf-simulator — 行为契约（反向还原）

> 反向还原自 `backend/simulator/netconfsim/`，忠实 as-built（经 `refactor-netconf-simulator` 重构）。详见 `design.md`。测试专用。

## 能力概述

为 YANG 业务提供可编程模拟设备对端，支撑端到端集成测试（T02）。经重构为**单一、结构化、协议保真、测试/部署解耦**的 NETCONF 模拟网元；前端 E2E 后端由诚实命名的内存 REST 桩承担（非 NETCONF）。

## 行为契约

### NS-01 NETCONF 会话
- **Given** 集成测试启动 `netconfsim.NewSimulator()`
- **When** 客户端经 SSH(随机端口, admin/admin) 连接 netconf subsystem
- **Then** 发 NETCONF hello，广告 `base:1.0` + `:candidate` + `:writable-running`，进入 RPC 分发

### NS-02 数据存操作
- **Given** 收到 RPC（`classifyRPC` 结构化解码判定类型）
- **When** 分发
- **Then** get-config→返回 running/candidate 树序列化 XML（可套 subtree filter）；edit-config→按 operation 写 candidate；commit→candidate 转 running；discard-changes→丢弃 candidate；其它→`<ok/>`

### NS-03 故障注入
- **Given** `ScenarioConfig{ErrorOnRPC, RejectAuth}`
- **When** 对应 RPC/认证发生
- **Then** 返回错误/拒绝，供测试覆盖设备离线、下发失败等降级路径（R08）

### NS-04 断言助手（查通用树）
- **Given** 下发后需校验
- **When** `Simulator.Running{HuaweiVLANs,HuaweiVLANsFull,HuaweiInterfaces,HuaweiSystem,OCInterfaces}`（`testsupport.Assert*` 调用）
- **Then** 在通用 XML 数据树上结构化查询出属性供断言（不再做 XML string-parsing）

### NS-05 前端 E2E 后端（内存 REST 桩）
- **Given** 前端 Playwright 套件
- **When** `cmd/test-server` 启动
- **Then** 进程内 `vlanStore`（内存 VLAN 存储）经 Gin REST 直供 VLAN CRUD（不经 NETCONF）

### NS-06 独立可部署二进制
- **Given** 需容器化部署模拟网元
- **When** 构建 `cmd/netconf-simulator`（flag `-addr/-port`）
- **Then** 产出无 `testing` 依赖的独立可执行文件，绑定端口、优雅退出

## 契约状态（详见 design.md §6）

- 迁移债 D10（两个模拟器并存）已消除；netconfsim 已与 `testing` 解耦并可独立部署。

## 关联
- `design.md`、`device-protocol/spec.md`（被测对端）、`netconf-sim-integration-test` 技能、`backend/deploy/manifests/netconf-simulator`。
