# netconf-simulator — delta（replace-gin-with-beego）

## MODIFIED Requirements

### Requirement: NS-05 前端 E2E 后端（内存 REST 桩）

前端 Playwright 套件的后端 SHALL 由 `cmd/test-server` 承担：进程内 `vlanStore`（内存 VLAN 存储）经进程内 REST 桩（与主服务同一 Web 框架，现为 beego/v2）直供 VLAN CRUD，SHALL NOT 经 NETCONF，命名 SHALL 诚实标识其为内存桩而非 NETCONF 模拟。

#### Scenario: 前端 E2E 经 REST 桩操作 VLAN
- **WHEN** 前端 Playwright 套件启动 `cmd/test-server` 并发起 VLAN CRUD 请求
- **THEN** 进程内 `vlanStore` SHALL 经 REST 桩直接响应，不经 NETCONF
