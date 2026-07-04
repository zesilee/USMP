# netconf-simulator — 差异 / 补全清单（反向还原）

## spec 与代码差异

- [ ] **D10 两个模拟器概念重叠**：netconfsim(协议级 SSH) 与 netsim(数据级内存) 并存
- [ ] **netconfsim 仅测试可见**：`scenarios.go`/`simulator.go` 直接 import `testing`，非独立可部署网元
- [ ] **RPC 分发靠字符串匹配**：`server.go:156` 按子串识别 RPC，非完整 NETCONF 解析

## 改进建议

- [ ] 收敛两个模拟器为单一（或明确各自用途边界与命名）
- [ ] 将 netconfsim 与 `testing` 解耦，使其可作为独立容器化模拟网元（呼应 deploy manifests）
- [ ] RPC 分发改为结构化 NETCONF 解析，减少字符串匹配脆弱性
