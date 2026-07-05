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

	r := ifmctl.New(cs, pool)
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
