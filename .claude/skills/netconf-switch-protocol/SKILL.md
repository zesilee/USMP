---
name: netconf-switch-protocol
description: SSH 830 端口 NETCONF 对接（RFC6241）：get-config/edit-config/RPC 一律经自研 netconfcore 引擎与 C5 ClientPool，断线重连异常容错由框架承担
---

# NETCONF 设备对接技能

## 一、激活时机
涉及「NETCONF」「设备连接」「配置获取/下发」「RPC 请求」「断线重连」时激活。

## 二、硬约束
- **唯一引擎 = 自研 netconfcore**，入口 `backend/pkg/yang-runtime/client/netconf.go`。
  **NC-01：禁止回引 scrapligo**——守护测试 `backend/pkg/yang-runtime/client/no_scrapligo_guard_test.go` 拦截，无运行时回退。
- 仅 NETCONF (SSH 830)；gNMI (9339/9340) 为规划能力，工厂返回显式未实现错误。禁止 Telnet/SNMP（R02）。
- 运行配置不持久化：读取走 TTL+LRU 缓存（Key=设备IP+YANG路径，TTL 30s，下发后主动失效，§8/R03）。

## 三、开发口径
1. **连接一律走 C5 ClientPool**（§4）：获取/复用/断线重连/超时重试都由连接池承担，业务代码禁止自建连接、禁止自写重连循环。
2. **写操作串行化**：edit-config 在写事务内经 opMu 串行；`ExecuteRPC` 有副作用**不重试**、结果不入缓存。
3. **回读契约**：响应=以请求路径为根的子树（peelToPath）；改回读形状/解码前必读 `docs/memory/readback-subtree-peel.md`。
4. **能力协商**：查询设备能力必须 Peek 不拨号；节点级能力被动学习见 `docs/memory/device-node-capability.md`。
5. **联调**：模拟网元 `backend/simulator/netconfsim`（进程版 `backend/cmd/netconf-simulator`），集成测试见 `netconf-sim-integration-test` 技能。

## 四、必读文档
- 封帧兼容坑与引擎沿革：`docs/memory/netconf-selfdev.md`
- 真机验证手册与状态：`docs/netconf-core-field-validation.md`（真机结论回来前，勿用于生产设备变更）
