# netconf-simulator — 差异 / 补全清单（反向还原）

> 经 `refactor-netconf-simulator` 重构，以下差异均已消除。

## spec 与代码差异（已解决）

- [x] **D10 两个模拟器概念重叠**：删除 `netsim`，`test-server` 改内存 REST 桩；保留唯一的结构化 `netconfsim`
- [x] **netconfsim 仅测试可见**：core 去 `testing` 依赖，`Assert*` 迁入 `testsupport`；新增独立二进制 `cmd/netconf-simulator`
- [x] **RPC 分发靠字符串匹配**：`server.go` 改用 `classifyRPC`（`encoding/xml` 结构化解码）

## 改进建议（已落地）

- [x] 收敛两个模拟器为单一（netsim 删除，命名与用途边界清晰）
- [x] 将 netconfsim 与 `testing` 解耦，可作为独立容器化模拟网元（`cmd/netconf-simulator`）
- [x] RPC 分发改为结构化 NETCONF 解析，消除字符串匹配脆弱性
- [x] 结构化数据存 `treeDatastore`（通用 XML 树）+ edit-config operation 语义 + get-config subtree filter
