---
name: netconf-sim-integration-test
description: 基于 NETCONF 模拟网元为新增业务生成端到端集成测试用例，要求测试通过才能提交
---

# NETCONF 模拟网元集成测试技能

## 激活时机
1. 新增 YANG 模块 Controller/Reconciler 业务功能时（T02）
2. 要求添加集成测试时
3. 完成业务功能实现后，必须添加集成测试才能提交代码

## 核心职责
基于 `backend/simulator/netconfsim` 模拟网元，为新增 YANG 模块生成端到端集成测试，覆盖：
1. **正常流程** — 配置创建、修改、删除全流程（下发→回读→收敛）
2. **异常场景** — 设备离线、NETCONF rpc-error、commit 失败、超时
3. **结果验证** — 必须断言模拟网元上的最终配置，不能只测不报错

> 新增 YANG 模型接入设备配置时，本技能只是执行层；用例矩阵以 `yang-config-test-design` 技能为准（T02b）。

## 一、位置与命名
集成测试放在对应业务包的 `*_integration_test.go`，开头必须 `testing.Short()` 跳过（T03）：
```
backend/internal/api/vlan_hardening_integration_test.go     ← 现成范例
backend/internal/controller/<module>/reconciler_integration_test.go
```

## 二、测试用例必须覆盖
| 场景 | 必须 |
|------|------|
| 新增配置全流程（下发→模拟器断言） | ✅ |
| 修改已有配置 | ✅ |
| 删除配置（DELETE 命令通道语义，见 docs/memory/config-delete-semantics.md） | ✅ |
| NETCONF rpc-error / commit 失败 | ✅ |
| 空配置查询 | ✅ |

## 三、真实 API（以仓库现有代码为准，勿凭记忆编造）
```go
import (
    "github.com/leezesi/usmp/backend/simulator/netconfsim"
    "github.com/leezesi/usmp/backend/simulator/netconfsim/testsupport"
)

sim := netconfsim.NewSimulator()   // 随机端口，defer sim.Stop()
sc := netconfsim.NewScenarioConfig() // 注入 rpc 错误等异常场景
// 断言用 testsupport 包（华为口径）：
testsupport.AssertHuaweiVlanExists(t, sim, 100)
testsupport.AssertHuaweiVlanName(t, sim, 10, "vlan-ten")
testsupport.AssertHuaweiVlanMemberPort(t, sim, 100, "GigabitEthernet0/0/1", ...)
```
- 模块 import 前缀是 **`github.com/leezesi/usmp/backend/...`**（module 名含 /backend）。
- 新模块缺断言 helper 时，在 `testsupport` 内按 `AssertHuawei<Module><Field>` 命名补齐，勿在测试里裸解析 XML。
- 写新用例前先读同目录现有 `*_integration_test.go` 抄真实接线方式（连接池、handler、changeset）。

## 四、强制规则
1. 所有新增 YANG 模块业务必须提供至少一个正常流程 + 一个异常场景集成测试
2. 测试全绿才能提交（R15/W02）
3. 每个集成测试只测一个场景，保持简洁
4. 测试注入异常时用优雅关闭，别强杀 sim（历史竞态坑见 docs/memory/scrapligo-concurrency-pitfalls.md）
