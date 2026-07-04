# netconf-simulator — 行为契约（反向还原）

> 反向还原自 `backend/simulator/`，忠实 as-built。详见 `design.md`。测试专用。

## 能力概述

为 YANG 业务提供可编程模拟设备对端，支撑端到端集成测试（T02）。存在 netconfsim(协议级) 与 netsim(数据级) 两个模拟器。

## 行为契约

### NS-01 NETCONF 会话
- **Given** 集成测试启动 `netconfsim.NewSimulator()`
- **When** 客户端经 SSH(随机端口, admin/admin) 连接 netconf subsystem
- **Then** 发 NETCONF 1.0 hello，进入 RPC 分发

### NS-02 数据存操作
- **Given** 收到 RPC
- **When** 分发
- **Then** get-config→返回 running/candidate XML；edit-config→写 candidate；commit→candidate 转 running；discard-changes→丢弃 candidate；其它→`<ok/>`

### NS-03 故障注入
- **Given** `ScenarioConfig{ErrorOnRPC, RejectAuth}`
- **When** 对应 RPC/认证发生
- **Then** 返回错误/拒绝，供测试覆盖设备离线、下发失败等降级路径（R08）

### NS-04 断言助手
- **Given** 下发后需校验
- **When** `Datastore.Extract{VLANs,Interfaces,HuaweiVLANs,...}`
- **Then** 从 running XML 解析出结构供断言

### NS-05 前端 E2E 后端（netsim）
- **Given** 前端 Playwright 套件
- **When** `cmd/test-server` 启动
- **Then** 内存 netsim 经 Gin REST 直供 VLAN CRUD

## 契约缺口（详见 design.md §6）

- 两个模拟器概念重叠；netconfsim 仅测试可见（import `testing`），非独立可部署网元。

## 关联
- `design.md`、`device-protocol/spec.md`（被测对端）、`netconf-sim-integration-test` 技能、`backend/deploy/manifests/netconf-simulator`。
