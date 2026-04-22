---
name: netconf-sim-integration-test
description: 基于 NETCONF 模拟网元为新增业务生成端到端集成测试用例，要求测试通过才能提交
---

# NETCONF 模拟网元集成测试技能

## 激活时机
1. **当新增 YANG 模块 Controller/Reconciler 业务功能时**，自动激活本技能
2. **当要求添加集成测试**时，自动激活本技能
3. 完成业务功能实现后，必须添加集成测试才能提交代码

## 核心职责
基于 `test/netconf-simulator` NETCONF 模拟服务器，为新增的 YANG 模块 Controller/Reconciler 生成完整的端到端集成测试用例，覆盖：
1. **正常流程** - 配置创建、修改、删除全流程
2. **异常场景** - 设备离线、NETCONF错误、超时等场景
3. **结果验证** - 验证配置最终正确下发到模拟网元

## 技能要求

### 一、集成测试位置
集成测试用例放在对应业务包的 `*_integration_test.go` 文件中，例如：
```
internal/controller/vlan/reconciler_integration_test.go
```

### 二、测试用例必须覆盖
| 测试场景 | 必须覆盖 |
|---------|---------|
| 新增VLAN配置全流程 | ✅ |
| 修改已有VLAN配置 | ✅ |
| 删除VLAN配置 | ✅ |
| 设备认证失败场景 | ✅ |
| NETCONF commit失败场景 | ✅ |
| 空配置查询 | ✅ |

### 三、使用模拟服务器步骤
1. **启动模拟器**：使用 `netsim.NewSimulator()` 启动，自动分配随机端口避免端口冲突
2. **设置初始状态**：根据测试场景设置模拟器初始配置
3. **创建真实客户端**：使用项目 `internal/netconf` 客户端连接到模拟器
4. **执行被测逻辑**：调用 Reconciler 完成协调
5. **验证结果**：使用模拟器 API 断言模拟器上的最终配置是否正确

### 四、测试代码模板
```go
package vlan

import (
	"testing"
	"context"

	"github.com/stretchr/testify/assert"
	"github.com/leezesi/usmp/test/netconf-simulator"
	"github.com/leezesi/usmp/internal/generated/openconfig"
	"github.com/leezesi/usmp/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/pkg/yang-runtime/reconcile"
)

func TestReconciler_Integration_CreateVLAN(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. 启动NETCONF模拟服务器
	sim := netsim.NewSimulator()
	err := sim.Start()
	assert.NoError(t, err)
	defer sim.Stop()

	// 2. 添加设备到客户端池
	pool := client.NewPool()
	err = pool.Add(client.DeviceConnectionInfo{
		IP:       sim.Addr(),
		Port:     sim.Port(),
		Username: sim.Username(),
		Password: sim.Password(),
		Protocol: client.ProtocolNETCONF,
	})
	assert.NoError(t, err)

	// 3. 设置期望状态 (desired)
	desired := &openconfig.Device{}
	// ... 设置期望配置 ...

	// 4. 执行协调
	r := NewReconciler(...)
	result := r.Reconcile(context.Background(), reconcile.Request{DeviceID: "test-device", Desired: desired})
	assert.NoError(t, result.Error)

	// 5. 验证模拟设备上的实际配置
	sim.AssertVlanExists(t, 100)
	sim.AssertVlanName(t, 100, "MyVLAN")
}

func TestReconciler_Integration_CommitError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. 启动模拟服务器
	sim := netsim.NewSimulator()
	err := sim.Start()
	assert.NoError(t, err)
	defer sim.Stop()

	// 2. 设置错误场景 - commit 总是失败
	sc := netsim.NewScenarioConfig()
	sc.ErrorOnRPC = map[string]error{
		"commit": fmt.Errorf("commit failed: device busy"),
	}
	sim.SetScenario(sc)

	// 3. 执行协调，验证错误被正确处理
	// ...
}
```

### 五、强制规则
1. **必须添加集成测试**：所有新增 YANG 模块业务，必须提供至少一个完整端到端集成测试用例
2. **必须通过测试**：所有集成测试必须执行成功才能提交代码
3. **使用 `-short` 跳过**：集成测试必须加上 `if testing.Short() { t.Skip() }`，方便日常单元测试快速运行
4. **单次迭代一个完整场景**：每个集成测试只测试一个场景，保持测试简洁
5. **必须验证最终状态**：测试必须断言模拟网元上的最终配置，不能只测试不报错

### 六、可用断言API
| API | 作用 |
|-----|------|
| `sim.AssertVlanExists(t, id)` | 断言VLAN存在 |
| `sim.AssertVlanName(t, id, name)` | 断言VLAN名称匹配 |
| `sim.AssertVlanCount(t, count)` | 断言VLAN数量匹配 |
| `sim.GetRunningConfig()` | 获取完整运行配置进行自定义断言 |

## 依赖
- `test/netconf-simulator` - NETCONF模拟服务器（已实现）
- `github.com/stretchr/testify` - 断言库（已有）

## 交付标准
新增 YANG 模块业务完成后，必须在对应目录生成 `*_integration_test.go`，包含：
1. 至少一个正常流程集成测试
2. 至少一个异常场景集成测试
3. 所有测试在 `go test -run Integration` 下执行成功
