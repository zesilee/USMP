package api

import (
	"context"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/internal/cache"
	ifmctl "github.com/leezesi/usmp/backend/internal/controller/ifm"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
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

	deviceID := "sim"
	ds := device.NewStore()
	ds.Put(deviceID, client.DeviceConnectionInfo{
		IP: sim.Addr(), Port: sim.Port(), Username: sim.Username(), Password: sim.Password(), Protocol: client.ProtocolNETCONF,
	})
	path := "/ifm:ifm/ifm:interfaces"
	assert.NoError(t, cs.Set(deviceID, path, typed))

	r := ifmctl.New(cs, pool, ds)
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

// TestInterfaceConfig_Integration_CredsFromDeviceStore 验证「生产接线」下的下发：
// HTTP API 触发对账时 DeviceID 是 URL 里的裸设备标识（不含凭据），reconciler 从共享
// DeviceStore 解析出凭据/端口建连。设备先注册进库，再以纯 DeviceID 触发对账；断言配置
// 真正落到模拟器。锁住「凭据经共享库、而非 DeviceID 字符串或 admin/admin 兜底」这条链路。
func TestInterfaceConfig_Integration_CredsFromDeviceStore(t *testing.T) {
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

	// 设备以纯 DeviceID 注册进共享库，连接信息（含凭据/端口）由库提供。
	deviceID := "sim-store"
	ds := device.NewStore()
	ds.Put(deviceID, client.DeviceConnectionInfo{
		IP: sim.Addr(), Port: sim.Port(), Username: sim.Username(), Password: sim.Password(), Protocol: client.ProtocolNETCONF,
	})

	path := "/ifm:ifm/ifm:interfaces"
	assert.NoError(t, cs.Set(deviceID, path, typed))

	r := ifmctl.New(cs, pool, ds)
	result := r.Reconcile(context.Background(), reconcile.Request{DeviceID: deviceID, Path: path})
	if result.Error != nil {
		t.Fatalf("对账应从 DeviceStore 解析凭据认证成功并下发: %v", result.Error)
	}

	testsupport.AssertHuaweiInterfaceExists(t, sim, "GigabitEthernet0/0/2")
}
