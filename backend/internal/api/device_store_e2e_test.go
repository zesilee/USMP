package api

import (
	"context"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/internal/cache"
	ifmctl "github.com/leezesi/usmp/backend/internal/controller/ifm"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
	testsupport "github.com/leezesi/usmp/backend/simulator/netconfsim/testsupport"
	"github.com/stretchr/testify/assert"
)

// TestDeviceStore_E2E_RegisterPushReadbackConverge is the capstone for the
// shared-device-store change: a device registered ONLY in the shared DeviceStore
// (single source of truth, DeviceID carries no credentials) drives the full
// chain — reconcile resolves credentials from the store and pushes; the config
// read path resolves from the SAME store and reads the interface back; a second
// reconcile converges. This reproduces the original "新增接口" symptoms end-to-end
// and proves #100 (SSH none-auth) and #101 (perpetual drift / list empty) are
// fixed via the store rather than the removed admin/admin fallback or DeviceID
// string parsing.
func TestDeviceStore_E2E_RegisterPushReadbackConverge(t *testing.T) {
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

	// 1. Register the device ONLY in the shared store (no credentials in DeviceID).
	ds := device.NewStore()
	deviceID := "e2e-device"
	ds.Put(deviceID, client.DeviceConnectionInfo{
		IP: sim.Addr(), Port: sim.Port(), Username: sim.Username(), Password: sim.Password(), Protocol: client.ProtocolNETCONF,
	})

	// 2. Submit a name-only interface (the reported case).
	typed, err := convertConfig("/ifm:ifm/ifm:interfaces", map[string]interface{}{
		"interface": []interface{}{map[string]interface{}{"name": "GigabitEthernet0/0/3"}},
	})
	assert.NoError(t, err)
	path := "/ifm:ifm/ifm:interfaces"
	assert.NoError(t, cs.Set(deviceID, path, typed))

	ctx := context.Background()
	req := reconcile.Request{DeviceID: deviceID, Path: path}

	// 3. Reconcile: resolves credentials from the store and pushes (no SSH none-auth).
	r := ifmctl.New(cs, pool, ds)
	first := r.Reconcile(ctx, req)
	if first.Error != nil {
		t.Fatalf("first reconcile (creds from store) must authenticate and push: %v", first.Error)
	}
	assert.Greater(t, first.Changes, 0, "首轮应下发新接口")
	testsupport.AssertHuaweiInterfaceExists(t, sim, "GigabitEthernet0/0/3")

	// 4. Config read path resolves from the SAME store and returns the interface
	//    as a listable RFC7951 map (not opaque XML bytes).
	h := NewConfigHandler(fakePoolManager{pool: pool, store: ds})
	got, err := h.fetchFromDevice(ctx, deviceID, path)
	assert.NoError(t, err)
	m, ok := got.(map[string]interface{})
	assert.True(t, ok, "回读应返回可列表化的结构")
	if ok {
		list, _ := m["interface"].([]interface{})
		assert.NotEmpty(t, list, "回读应包含刚下发的接口（列表可见）")
	}

	// 5. Second reconcile converges (no perpetual drift).
	second := r.Reconcile(ctx, req)
	if second.Error != nil {
		t.Fatalf("second reconcile: %v", second.Error)
	}
	assert.Equal(t, 0, second.Changes, "设备已落盘后二次对账必须收敛")
}
