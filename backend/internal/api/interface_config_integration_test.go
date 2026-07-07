package api

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/internal/cache"
	ifmctl "github.com/leezesi/usmp/backend/internal/controller/ifm"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
	testsupport "github.com/leezesi/usmp/backend/simulator/netconfsim/testsupport"
	"github.com/stretchr/testify/assert"
)

// IFM 枚举字符串（"up"）应映射到 ygot 枚举值（PortStatus_up=2）。
func TestEnumInt_IfmPortStatus(t *testing.T) {
	n, ok := enumInt("up", "E_HuaweiIfm_PortStatus")
	if !ok || huawei.E_HuaweiIfm_PortStatus(n) != huawei.HuaweiIfm_PortStatus_up {
		t.Fatalf("enumInt(up) = %d ok=%v, want PortStatus_up", n, ok)
	}
}

// 端到端：前端形状的接口配置 map（枚举字符串 admin-status:"up"）→ convertMapToHuaweiIfm
// → 对账 → NETCONF → 模拟网元。验证接口配置枚举字符串一路正确落到设备。
func TestInterfaceConfig_Integration_EnumStringToDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	sim := netsim.NewSimulator()
	if err := sim.Start(); err != nil {
		t.Fatalf("start sim: %v", err)
	}
	defer sim.Stop()

	c := cache.NewTTLLRUCache(100, 30*time.Second, 1*time.Minute)
	cs := manager.NewInMemoryConfigStore(c)
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	raw := map[string]interface{}{
		"interface": []interface{}{
			map[string]interface{}{
				"name":         "GigabitEthernet0/0/1",
				"description":  "uplink",
				"admin-status": "up",
				"mtu":          float64(1500),
			},
		},
	}

	typed, err := convertMapToHuaweiIfm(raw)
	assert.NoError(t, err)

	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/ifm:ifm/ifm:interfaces"
	assert.NoError(t, cs.Set(deviceID, path, typed))

	r := ifmctl.New(cs, pool, nil)
	result := r.Reconcile(context.Background(), reconcile.Request{DeviceID: deviceID, Path: path})
	if result.Error != nil {
		t.Fatalf("reconcile: %v", result.Error)
	}

	testsupport.AssertHuaweiInterfaceExists(t, sim, "GigabitEthernet0/0/1")
	testsupport.AssertHuaweiInterfaceDescription(t, sim, "GigabitEthernet0/0/1", "uplink")
	// admin-status "up" → 2（HuaweiIfm_PortStatus_up）
	testsupport.AssertHuaweiInterfaceAdminStatus(t, sim, "GigabitEthernet0/0/1", 2)
	testsupport.AssertHuaweiInterfaceMtu(t, sim, "GigabitEthernet0/0/1", 1500)
}

// TestInterfaceConfig_Integration_NoCredentialsInDeviceID 复现「生产接线」下的下发：
// HTTP API 触发对账时 DeviceID 是 URL 里的裸设备标识，**不含凭据**——而其它集成测试
// 都用 "user:pass@ip:port" 把 admin/admin 塞进了 DeviceID，恰好绕过了缺凭据的缺陷。
// 本用例用无凭据的 "ip:port"（reconciler 的 ip:port 分支，Username/Password 为空），
// 若 NETCONF 客户端不对空凭据兜底，SSH 只会提供 "none" 认证被模拟器拒绝
// （"attempted methods [none]"，即界面「新增接口」下发失败的真实报错）。
// 断言配置仍真正落到模拟器，锁住凭据兜底这条生产链路。
func TestInterfaceConfig_Integration_NoCredentialsInDeviceID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	sim := netsim.NewSimulator()
	if err := sim.Start(); err != nil {
		t.Fatalf("start sim: %v", err)
	}
	defer sim.Stop()

	c := cache.NewTTLLRUCache(100, 30*time.Second, 1*time.Minute)
	cs := manager.NewInMemoryConfigStore(c)
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	raw := map[string]interface{}{
		"interface": []interface{}{
			map[string]interface{}{"name": "GigabitEthernet0/0/2"},
		},
	}
	typed, err := convertMapToHuaweiIfm(raw)
	assert.NoError(t, err)

	// 关键：无 "user:pass@"，只有 ip:port —— 正是 API 触发对账时产生的设备标识形状。
	deviceID := fmt.Sprintf("%s:%d", sim.Addr(), sim.Port())
	path := "/ifm:ifm/ifm:interfaces"
	assert.NoError(t, cs.Set(deviceID, path, typed))

	r := ifmctl.New(cs, pool, nil)
	result := r.Reconcile(context.Background(), reconcile.Request{DeviceID: deviceID, Path: path})
	if result.Error != nil {
		t.Fatalf("无凭据设备标识的对账仍须认证成功并下发（回归 ssh none-auth 缺陷）: %v", result.Error)
	}

	testsupport.AssertHuaweiInterfaceExists(t, sim, "GigabitEthernet0/0/2")
}
