package api

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/internal/cache"
	vlanctl "github.com/leezesi/usmp/backend/internal/controller/vlan"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
	testsupport "github.com/leezesi/usmp/backend/simulator/netconfsim/testsupport"
	"github.com/stretchr/testify/assert"
)

// 端到端：前端形状的配置 map（枚举用字符串名，如 admin-status:"up"）→ API 层
// convertMapToHuaweiVlan → ConfigStore → 对账 → NETCONF edit-config → 模拟网元运行配置。
// 验证「VLAN 配不了」修复后，枚举字符串一路正确落到设备（这是此前断裂处）。
func TestVlanConfig_Integration_EnumStringToDevice(t *testing.T) {
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

	// 前端提交的原始 map：枚举为字符串名、含描述。
	raw := map[string]interface{}{
		"vlans": []interface{}{
			map[string]interface{}{
				"id":           float64(100),
				"name":         "TestVLAN",
				"description":  "from-api-map",
				"admin-status": "up",
				"mac-learning": "enable",
			},
		},
	}

	typed, err := convertMapToHuaweiVlan(raw)
	assert.NoError(t, err)

	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/vlan:vlan/vlan:vlans"
	assert.NoError(t, cs.Set(deviceID, path, typed))

	r := vlanctl.New(cs, pool, nil)
	result := r.Reconcile(context.Background(), reconcile.Request{DeviceID: deviceID, Path: path})
	if result.Error != nil {
		t.Fatalf("reconcile: %v", result.Error)
	}

	// 断言设备运行配置：VLAN 100 存在，名称/描述正确，枚举字符串已落为设备侧枚举值。
	testsupport.AssertHuaweiVlanExists(t, sim, 100)
	testsupport.AssertHuaweiVlanName(t, sim, 100, "TestVLAN")
	testsupport.AssertHuaweiVlanDescription(t, sim, 100, "from-api-map")
	// admin-status "up" → 2（HuaweiVlan_AdminStatus_up）
	testsupport.AssertHuaweiVlanAdminStatus(t, sim, 100, 2)
	// mac-learning "enable" → 2（HuaweiVlan_EnableStatus_enable）
	testsupport.AssertHuaweiVlanMacLearning(t, sim, 100, 2)
}
