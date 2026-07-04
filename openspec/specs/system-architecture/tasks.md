# system-architecture — 迁移债 / 补全清单（反向还原）

> 本清单为 as-built 与目标架构的差异 + 演进待办，非本次实施步骤。勾选 = 已在代码中解决。

## spec 与代码差异（迁移债）

- [ ] **D1 双 CRD 树抢注同 group**：`api/v1` 与 `api/biz/v1` 均注册 `biz.usmp.io/v1`，BusinessVlan schema 不兼容
- [ ] **D2 Actor 子系统 vs R01**：最大子系统被红线禁止但生产在用
- [ ] **D3 plugin 空转**：四类插件可注册但从不被调用
- [ ] **D4 schema 层空转**：`SchemeDir` 未设，运行时 schema 为空，diff 传 nil
- [ ] **D5 gNMI 空壳**：Get 发空 GetRequest，Set 发空 Path/Val
- [ ] **D6 NativeDeviceConfig 下发 = TODO**：`applyNativeConfig` 仅 time.Sleep
- [ ] **D7 ConfigStore.List/ListDevices = stub**：返回 nil,nil
- [ ] **D8 多厂商翻译仅 Huawei**：Cisco/H3C/Juniper 仅枚举占位
- [ ] **D9 前端双代动态表单**：旧 yang/ 静态路径未接路由
- [x] **D10 两个模拟器并存**：已消除——删除 netsim，收敛为单一结构化 netconfsim，test-server 改内存 REST 桩（refactor-netconf-simulator）

## 改进建议（演进，需专门 /opsx:propose）

- [ ] 确立单栈：生产入口从 cmd/controller/main.go(Stack A) 切到 backend/main.go(Stack B)
- [ ] 收敛 CRD 树：退役 api/v1，统一 api/biz/v1 + api/core/v1
- [ ] 偿还空转件：接通 plugin 钩子、加载 schema、补 gNMI 或明确弃用、实现 NativeDeviceConfig 下发
- [ ] 每条迁移遵循 §5.3「旧保留 + 新并行 + 双路径验证 → 切换 → 删除」
