package api

import (
	"context"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/internal/cache"
	vlanctl "github.com/leezesi/usmp/backend/internal/controller/vlan"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
	testsupport "github.com/leezesi/usmp/backend/simulator/netconfsim/testsupport"
	"github.com/stretchr/testify/assert"
)

const vlanPath = "/vlan:vlan/vlan:vlans"

func newVlanSimStack(t *testing.T) (*netsim.Simulator, *manager.InMemoryConfigStore, client.ClientPool, device.Store, string) {
	t.Helper()
	sim := netsim.NewSimulator()
	if err := sim.Start(); err != nil {
		t.Fatalf("start sim: %v", err)
	}
	c := cache.NewTTLLRUCache(100, 30*time.Second, 1*time.Minute)
	cs := manager.NewInMemoryConfigStore(c)
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	// Register the sim in a DeviceStore (source of truth); DeviceID carries no creds.
	deviceID := "sim"
	ds := device.NewStore()
	ds.Put(deviceID, client.DeviceConnectionInfo{
		IP: sim.Addr(), Port: sim.Port(), Username: sim.Username(), Password: sim.Password(), Protocol: client.ProtocolNETCONF,
	})
	return sim, cs, pool, ds, deviceID
}

// applyVlan：走真实 API 转换路径下发一批 VLAN（前端形状 map）。
func applyVlan(t *testing.T, cs *manager.InMemoryConfigStore, pool client.ClientPool, ds device.Store, deviceID string, vlansPayload []interface{}) {
	t.Helper()
	typed, err := convertConfig(vlanPath, map[string]interface{}{"vlan": vlansPayload})
	assert.NoError(t, err)
	// 走与 SetConfig 处理器一致的合并存储逻辑（带锁，防覆盖抹除 + 并发丢更新）
	assert.NoError(t, storeConfigMerged(cs, deviceID, vlanPath, typed))
	res := vlanctl.New(cs, pool, ds).Reconcile(context.Background(), reconcile.Request{DeviceID: deviceID, Path: vlanPath})
	if res.Error != nil {
		t.Fatalf("reconcile: %v", res.Error)
	}
}

// P0：端口成员（VLAN 最核心功能）应端到端落到设备。此前只有 XML 生成 + map 转换单测，
// 设备侧无任何断言。此测试下发含 access/trunk 两个端口的 VLAN，断言设备运行配置正确。
func TestVlanConfig_Integration_MemberPortsToDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim, cs, pool, ds, deviceID := newVlanSimStack(t)
	defer sim.Stop()
	defer pool.CloseAll()

	applyVlan(t, cs, pool, ds, deviceID, []interface{}{
		map[string]interface{}{
			"id":   float64(100),
			"name": "with-ports",
			"member-ports": map[string]interface{}{
				"member-port": []interface{}{
					map[string]interface{}{"interface-name": "GigabitEthernet0/0/1", "access-type": "access"},
					map[string]interface{}{"interface-name": "GigabitEthernet0/0/2", "access-type": "trunk"},
				},
			},
		},
	})

	testsupport.AssertHuaweiVlanExists(t, sim, 100)
	testsupport.AssertHuaweiVlanMemberPort(t, sim, 100, "GigabitEthernet0/0/1",
		int(huawei.HuaweiVlan_AccessType_access))
	testsupport.AssertHuaweiVlanMemberPort(t, sim, 100, "GigabitEthernet0/0/2",
		int(huawei.HuaweiVlan_AccessType_trunk))
}

// P0：分两次下发不同 VLAN（真实前端行为——每次只发单个 VLAN），先前的 VLAN 不应被抹除。
// 若 edit-config 为 replace 语义或 ConfigStore 覆盖导致对账删除，会在真机造成配置丢失事故。
func TestVlanConfig_Integration_MergePreservesOtherVLANs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim, cs, pool, ds, deviceID := newVlanSimStack(t)
	defer sim.Stop()
	defer pool.CloseAll()

	// 先配 VLAN 10
	applyVlan(t, cs, pool, ds, deviceID, []interface{}{
		map[string]interface{}{"id": float64(10), "name": "vlan-ten"},
	})
	testsupport.AssertHuaweiVlanExists(t, sim, 10)

	// 再单独配 VLAN 20（模拟用户第二次新增）
	applyVlan(t, cs, pool, ds, deviceID, []interface{}{
		map[string]interface{}{"id": float64(20), "name": "vlan-twenty"},
	})

	// 关键断言：VLAN 20 已配置，且 VLAN 10 仍在（未被第二次下发抹除）
	testsupport.AssertHuaweiVlanExists(t, sim, 20)
	testsupport.AssertHuaweiVlanExists(t, sim, 10)
	testsupport.AssertHuaweiVlanName(t, sim, 10, "vlan-ten")
}
