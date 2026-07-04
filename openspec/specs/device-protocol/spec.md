# device-protocol — 行为契约（反向还原）

> 反向还原自 `backend/pkg/yang-runtime/client/`，忠实 as-built。详见 `design.md`。

## 能力概述

统一设备接入层（C5）：连接池 + NETCONF/gNMI 协议客户端，向上暴露 Get/Set/Subscribe/DiscardCandidate。

## 行为契约

### DP-01 连接复用
- **Given** 已为某设备 IP 建立连接
- **When** 再次 `ClientPool.Get(info)` 且 `IsConnected()`
- **Then** 复用现有 client；否则经 factory 新建（双检锁）

### DP-02 协议选择
- **Given** `DeviceConnectionInfo.Protocol`
- **When** factory 创建 client
- **Then** NETCONF→NETCONF、GNMI→gNMI、AUTO→按端口(0/830→NETCONF, 9339→gNMI, 默认 NETCONF)。**实践中 AUTO 恒落 NETCONF**

### DP-03 NETCONF 读
- **Given** 需要 actual 配置
- **When** `Client.Get(path, WithDatastore("running"))`
- **Then** get-config(running) + XPath filter，返回结果供反序列化为 ygot

### DP-04 NETCONF 写（含 2PC）
- **Given** `[]Change` + `WithCommit(true)`
- **When** `Client.Set`
- **Then** 逐 change edit-config(candidate) → commit(running)；失败检查 `<rpc-error>`；`DiscardCandidate` 可回滚 candidate

### DP-05 惰性重连
- **Given** 连接断开（`!IsConnected()`）
- **When** 下次 Get/Set
- **Then** 单次 reconnect（无退避、无重试计数），失败上抛由上层 requeue

## 契约缺口（详见 design.md §5）

- gNMI Get/Set 为空壳（发空请求）；NETCONF `Subscribe` 未实现；`Release` no-op；`CloseAll` 吞错。

## 关联
- `design.md`、`yang-controller-runtime/spec.md`（消费方）、`netconf-simulator/spec.md`（对端）、`netconf-switch-protocol` 技能。
