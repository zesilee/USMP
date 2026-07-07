package vlan

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/internal/cache"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
	testsupport "github.com/leezesi/usmp/backend/simulator/netconfsim/testsupport"
	"github.com/stretchr/testify/assert"
)

// TestReconciler_Integration_CreateVLAN tests the full flow for creating a new VLAN
func TestReconciler_Integration_CreateVLAN(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Start NETCONF simulator
	sim := netsim.NewSimulator()
	err := sim.Start()
	assert.NoError(t, err)
	defer sim.Stop()

	// 2. Create in-memory cache and config store
	c := cache.NewTTLLRUCache(100, 30*time.Second, 1*time.Minute)
	cs := manager.NewInMemoryConfigStore(c)

	// 3. Create client pool with default factory that connects to simulator
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	// 4. Set desired configuration - create VLAN 100 with name "TestVLAN" (Huawei model)
	desired := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
			100: {
				Id:   uint16Ptr(100),
				Name: stringPtr("TestVLAN"),
				Type: huawei.HuaweiVlan_VlanType_common,
			},
		},
	}

	// deviceID format "user:pass@host:port" includes credentials and port for integration testing
	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/vlan:vlan/vlan:vlans"
	err = cs.Set(deviceID, path, desired)
	assert.NoError(t, err)

	// 5. Create reconciler and execute reconciliation
	r := New(cs, pool, nil)
	req := reconcile.Request{
		DeviceID: deviceID,
		Path:     path,
	}

	ctx := context.Background()
	result := r.Reconcile(ctx, req)

	// 6. Verify result
	if result.Error != nil {
		t.Fatalf("reconciliation failed: %v", result.Error)
	}
	assert.NoError(t, result.Error)
	assert.False(t, result.Requeue)

	// 7. Verify the VLAN exists in the simulator with correct name (Huawei model)
	testsupport.AssertHuaweiVlanExists(t, sim, 100)
	testsupport.AssertHuaweiVlanName(t, sim, 100, "TestVLAN")
	testsupport.AssertHuaweiVlanCount(t, sim, 1)
}

// TestReconciler_Integration_CreateVLAN_ConvergesAndReadable 锁死 VLAN 侧「一直漂移」回归：
// 设备回读 XML 原先无法解析进 ygot map（actual 恒空）→ 每轮都算出 diff。修复后第二次对账必须收敛。
func TestReconciler_Integration_CreateVLAN_ConvergesAndReadable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	sim := netsim.NewSimulator()
	if err := sim.Start(); err != nil {
		t.Fatalf("start simulator: %v", err)
	}
	defer sim.Stop()

	c := cache.NewTTLLRUCache(100, 30*time.Second, 1*time.Minute)
	cs := manager.NewInMemoryConfigStore(c)
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	// UI 新建：稀疏意图（id + name）
	desired := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
			300: {Id: uint16Ptr(300), Name: stringPtr("v300")},
		},
	}

	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/vlan:vlan/vlan:vlans"
	if err := cs.Set(deviceID, path, desired); err != nil {
		t.Fatalf("config store set: %v", err)
	}

	r := New(cs, pool, nil)
	req := reconcile.Request{DeviceID: deviceID, Path: path}
	ctx := context.Background()

	first := r.Reconcile(ctx, req)
	if first.Error != nil {
		t.Fatalf("first reconcile failed: %v", first.Error)
	}
	assert.Greater(t, first.Changes, 0, "首轮应检测到漂移并下发新 VLAN")

	// 回读设备：VLAN 必须真正落盘且能解析进 ygot map（根因 C 修复）
	dc := &deviceClient{clientPool: pool}
	got, err := dc.Get(ctx, deviceID)
	if err != nil {
		t.Fatalf("readback device config: %v", err)
	}
	vlans, ok := got.(*huawei.HuaweiVlan_Vlan_Vlans)
	if !ok {
		t.Fatalf("unexpected readback type %T", got)
	}
	if _, present := vlans.Vlan[300]; !present {
		t.Fatalf("新建的 VLAN 300 未在设备回读中出现")
	}

	// 二次对账必须收敛（不再永久漂移）
	second := r.Reconcile(ctx, req)
	if second.Error != nil {
		t.Fatalf("second reconcile failed: %v", second.Error)
	}
	assert.Equal(t, 0, second.Changes, "设备已落盘 desired 后第二轮对账必须收敛")
	assert.False(t, second.Requeue)
}

// TestReconciler_Integration_ModifyVLAN tests modifying an existing VLAN configuration
func TestReconciler_Integration_ModifyVLAN(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Start NETCONF simulator
	sim := netsim.NewSimulator()
	err := sim.Start()
	assert.NoError(t, err)
	defer sim.Stop()

	// 2. Create initial config with existing VLAN using the same format as will be set by the reconciler
	// We create the config manually in the same XML format that Huawei model produces
	initialXML := `<config><HuaweiVlan_Vlan><Vlans><Vlan><Id>100</Id><Name>OldName</Name><Type>2</Type></Vlan></Vlans></HuaweiVlan_Vlan></config>`
	sim.SetRunningConfigXML([]byte(initialXML))

	// 3. Create in-memory cache and config store
	c := cache.NewTTLLRUCache(100, 30*time.Second, 1*time.Minute)
	cs := manager.NewInMemoryConfigStore(c)

	// 4. Create client pool
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	// 5. Set desired configuration - modified name
	desired := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
			100: {
				Id:   uint16Ptr(100),
				Name: stringPtr("NewName"),
				Type: huawei.HuaweiVlan_VlanType_common,
			},
		},
	}

	// deviceID format "user:pass@host:port" includes credentials and port for integration testing
	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/vlan:vlan/vlan:vlans"
	err = cs.Set(deviceID, path, desired)
	assert.NoError(t, err)

	// 6. Execute reconciliation
	r := New(cs, pool, nil)
	req := reconcile.Request{
		DeviceID: deviceID,
		Path:     path,
	}

	ctx := context.Background()
	result := r.Reconcile(ctx, req)

	// 7. Verify result
	assert.NoError(t, result.Error)
	assert.False(t, result.Requeue)

	// 8. Verify the name was updated (Huawei model)
	testsupport.AssertHuaweiVlanExists(t, sim, 100)
	testsupport.AssertHuaweiVlanName(t, sim, 100, "NewName")
	testsupport.AssertHuaweiVlanCount(t, sim, 1)
}

// TestReconciler_Integration_DeleteVLAN tests deleting a VLAN configuration
func TestReconciler_Integration_DeleteVLAN(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Start NETCONF simulator
	sim := netsim.NewSimulator()
	err := sim.Start()
	assert.NoError(t, err)
	defer sim.Stop()

	// 2. Set initial configuration with two VLANs (Huawei model)
	initial := &huawei.Device{
		Vlan: &huawei.HuaweiVlan_Vlan{
			Vlans: &huawei.HuaweiVlan_Vlan_Vlans{
				Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
					100: {
						Id:   uint16Ptr(100),
						Name: stringPtr("VLAN100"),
						Type: huawei.HuaweiVlan_VlanType_common,
					},
					200: {
						Id:   uint16Ptr(200),
						Name: stringPtr("VLAN200"),
						Type: huawei.HuaweiVlan_VlanType_common,
					},
				},
			},
		},
	}
	sim.SetRunningConfig(initial)

	// 3. Create in-memory cache and config store
	c := cache.NewTTLLRUCache(100, 30*time.Second, 1*time.Minute)
	cs := manager.NewInMemoryConfigStore(c)

	// 4. Create client pool
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	// 5. Set desired configuration - only VLAN 100 remains (VLAN 200 deleted)
	desired := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
			100: {
				Id:   uint16Ptr(100),
				Name: stringPtr("VLAN100"),
				Type: huawei.HuaweiVlan_VlanType_common,
			},
		},
	}

	// deviceID format "user:pass@host:port" includes credentials and port for integration testing
	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/vlan:vlan/vlan:vlans"
	err = cs.Set(deviceID, path, desired)
	assert.NoError(t, err)

	// 6. Execute reconciliation
	r := New(cs, pool, nil)
	req := reconcile.Request{
		DeviceID: deviceID,
		Path:     path,
	}

	ctx := context.Background()
	result := r.Reconcile(ctx, req)

	// 7. Verify result
	assert.NoError(t, result.Error)
	assert.False(t, result.Requeue)

	// 8. Verify only VLAN 100 remains, VLAN 200 is gone (Huawei model)
	testsupport.AssertHuaweiVlanExists(t, sim, 100)
	testsupport.AssertHuaweiVlanCount(t, sim, 1)
}

// TestReconciler_Integration_EmptyConfig tests handling of empty VLAN configuration
func TestReconciler_Integration_EmptyConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Start NETCONF simulator with empty initial config
	sim := netsim.NewSimulator()
	err := sim.Start()
	assert.NoError(t, err)
	defer sim.Stop()

	// 2. Create in-memory cache and config store
	c := cache.NewTTLLRUCache(100, 30*time.Second, 1*time.Minute)
	cs := manager.NewInMemoryConfigStore(c)

	// 3. Create client pool
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	// 4. Set desired empty configuration (Huawei model)
	desired := &huawei.HuaweiVlan_Vlan_Vlans{}

	// deviceID format "user:pass@host:port" includes credentials and port for integration testing
	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/vlan:vlan/vlan:vlans"
	err = cs.Set(deviceID, path, desired)
	assert.NoError(t, err)

	// 5. Execute reconciliation
	r := New(cs, pool, nil)
	req := reconcile.Request{
		DeviceID: deviceID,
		Path:     path,
	}

	ctx := context.Background()
	result := r.Reconcile(ctx, req)

	// 6. Verify result - no changes, no errors
	assert.NoError(t, result.Error)
	assert.False(t, result.Requeue)

	// 7. Verify still no VLANs (Huawei model)
	testsupport.AssertHuaweiVlanCount(t, sim, 0)
}

// TestReconciler_Integration_CommitFailure tests handling when NETCONF commit fails
func TestReconciler_Integration_CommitFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Start NETCONF simulator
	sim := netsim.NewSimulator()
	err := sim.Start()
	assert.NoError(t, err)
	defer sim.Stop()

	// 2. Configure commit error scenario
	sc := netsim.NewScenarioConfig()
	sc.ErrorOnRPC["commit"] = fmt.Errorf("device busy: commit rejected")
	sim.SetScenario(sc)

	// 3. Create in-memory cache and config store
	c := cache.NewTTLLRUCache(100, 30*time.Second, 1*time.Minute)
	cs := manager.NewInMemoryConfigStore(c)

	// 4. Create client pool
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	// 5. Set desired configuration (Huawei model)
	desired := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
			100: {
				Id:   uint16Ptr(100),
				Name: stringPtr("TestVLAN"),
				Type: huawei.HuaweiVlan_VlanType_common,
			},
		},
	}

	// deviceID format "user:pass@host:port" includes credentials and port for integration testing
	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/vlan:vlan/vlan:vlans"
	err = cs.Set(deviceID, path, desired)
	assert.NoError(t, err)

	// 6. Execute reconciliation
	r := New(cs, pool, nil)
	req := reconcile.Request{
		DeviceID: deviceID,
		Path:     path,
	}

	ctx := context.Background()
	result := r.Reconcile(ctx, req)

	// 7. Verify error is returned and request is requeued
	assert.Error(t, result.Error)
	assert.True(t, result.Requeue)
	assert.Contains(t, result.Error.Error(), "commit rejected")
}

// TestReconciler_Integration_AuthenticationFailure tests authentication failure handling
func TestReconciler_Integration_AuthenticationFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Start NETCONF simulator
	sim := netsim.NewSimulator()
	err := sim.Start()
	assert.NoError(t, err)
	defer sim.Stop()

	// 2. Configure authentication rejection scenario
	sc := netsim.NewScenarioConfig()
	sc.RejectAuth = true
	sim.SetScenario(sc)

	// 3. Create in-memory cache and config store
	c := cache.NewTTLLRUCache(100, 30*time.Second, 1*time.Minute)
	cs := manager.NewInMemoryConfigStore(c)

	// 4. Create client pool - the connection attempt happens during reconciliation
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	// 5. Set desired configuration (Huawei model)
	desired := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
			100: {
				Id:   uint16Ptr(100),
				Name: stringPtr("TestVLAN"),
				Type: huawei.HuaweiVlan_VlanType_common,
			},
		},
	}

	// Use wrong credentials in deviceID to test authentication failure
	deviceID := fmt.Sprintf("wrong:wrong@%s:%d", sim.Addr(), sim.Port())
	path := "/vlan:vlan/vlan:vlans"
	err = cs.Set(deviceID, path, desired)
	assert.NoError(t, err)

	// 6. Execute reconciliation - this is where authentication fails
	r := New(cs, pool, nil)
	req := reconcile.Request{
		DeviceID: deviceID,
		Path:     path,
	}

	ctx := context.Background()
	result := r.Reconcile(ctx, req)

	// 7. Verify error is returned and request is requeued
	assert.Error(t, result.Error)
	assert.True(t, result.Requeue)
}

// uint16Ptr is a helper to create a *uint16 from a uint16
func uint16Ptr(v uint16) *uint16 {
	return &v
}

// stringPtr is a helper to create a *string from a string
func stringPtr(s string) *string {
	return &s
}

// uint32Ptr is a helper to create a *uint32 from a uint32
func uint32Ptr(v uint32) *uint32 {
	return &v
}

// ============================================
// Full VLAN attribute coverage tests
// ============================================

// TestReconciler_Integration_FullVLANConfig tests all configurable VLAN attributes.
// This test verifies complete end-to-end coverage of all 14 config=true YANG properties.
func TestReconciler_Integration_FullVLANConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Start NETCONF simulator
	sim := netsim.NewSimulator()
	err := sim.Start()
	assert.NoError(t, err)
	defer sim.Stop()

	// 2. Create in-memory cache and config store
	c := cache.NewTTLLRUCache(100, 30*time.Second, 1*time.Minute)
	cs := manager.NewInMemoryConfigStore(c)

	// 3. Create client pool
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	// 4. Set desired configuration with ALL configurable fields
	desired := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
			100: {
				Id:                      uint16Ptr(100),
				Name:                    stringPtr("FullConfigVLAN"),
				Description:             stringPtr("VLAN with complete configuration"),
				Type:                    huawei.HuaweiVlan_VlanType_common,
				AdminStatus:             huawei.HuaweiVlan_AdminStatus_up,
				BroadcastDiscard:        huawei.HuaweiVlan_EnableStatus_enable,
				UnknownMulticastDiscard: huawei.HuaweiVlan_EnableStatus_disable,
				MacLearning:             huawei.HuaweiVlan_EnableStatus_enable,
				MacAgingTime:            uint32Ptr(300),
				StatisticEnable:         huawei.HuaweiVlan_EnableStatus_enable,
				StatisticDiscard:        huawei.HuaweiVlan_EnableStatus_disable,
				SuperVlan:               uint16Ptr(999),
				UnkownUnicastDiscard: &huawei.HuaweiVlan_Vlan_Vlans_Vlan_UnkownUnicastDiscard{
					Discard:           huawei.HuaweiVlan_EnableStatus_enable,
					MacLearningEnable: huawei.HuaweiVlan_EnableStatus_disable,
				},
				Suppression: &huawei.HuaweiVlan_Vlan_Vlans_Vlan_Suppression{
					Inbound:  huawei.HuaweiVlan_EnableStatus_enable,
					Outbound: huawei.HuaweiVlan_EnableStatus_enable,
				},
			},
		},
	}

	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/vlan:vlan/vlan:vlans"
	err = cs.Set(deviceID, path, desired)
	assert.NoError(t, err)

	// 5. Execute reconciliation
	r := New(cs, pool, nil)
	req := reconcile.Request{
		DeviceID: deviceID,
		Path:     path,
	}

	ctx := context.Background()
	result := r.Reconcile(ctx, req)

	// 6. Verify success
	assert.NoError(t, result.Error)
	assert.False(t, result.Requeue)

	// 7. Verify ALL fields were correctly applied
	// Basic fields
	testsupport.AssertHuaweiVlanExists(t, sim, 100)
	testsupport.AssertHuaweiVlanName(t, sim, 100, "FullConfigVLAN")
	testsupport.AssertHuaweiVlanDescription(t, sim, 100, "VLAN with complete configuration")
	testsupport.AssertHuaweiVlanType(t, sim, 100, int(huawei.HuaweiVlan_VlanType_common))
	testsupport.AssertHuaweiVlanAdminStatus(t, sim, 100, int(huawei.HuaweiVlan_AdminStatus_up))

	// Traffic control fields
	testsupport.AssertHuaweiVlanBroadcastDiscard(t, sim, 100, int(huawei.HuaweiVlan_EnableStatus_enable))
	testsupport.AssertHuaweiVlanUnknownMulticastDiscard(t, sim, 100, int(huawei.HuaweiVlan_EnableStatus_disable))

	// MAC learning fields
	testsupport.AssertHuaweiVlanMacLearning(t, sim, 100, int(huawei.HuaweiVlan_EnableStatus_enable))
	testsupport.AssertHuaweiVlanMacAgingTime(t, sim, 100, uint32(300))

	// Statistics fields
	testsupport.AssertHuaweiVlanStatisticEnable(t, sim, 100, int(huawei.HuaweiVlan_EnableStatus_enable))
	testsupport.AssertHuaweiVlanStatisticDiscard(t, sim, 100, int(huawei.HuaweiVlan_EnableStatus_disable))

	// VLAN association
	testsupport.AssertHuaweiVlanSuperVlan(t, sim, 100, uint16(999))

	// Nested container fields
	testsupport.AssertHuaweiVlanUnkownUnicastDiscard(t, sim, 100,
		int(huawei.HuaweiVlan_EnableStatus_enable),
		int(huawei.HuaweiVlan_EnableStatus_disable),
	)
	testsupport.AssertHuaweiVlanSuppression(t, sim, 100,
		int(huawei.HuaweiVlan_EnableStatus_enable),
		int(huawei.HuaweiVlan_EnableStatus_enable),
	)
}

// NOTE: Pure IP and IP:Port format tests are implemented as unit tests
// in reconciler_test.go (TestParseDeviceIDFormats) because the simulator
// requires explicit credentials for authentication. Production devices
// will use pre-configured credentials from device registry.

// ============================================
// Attribute-specific test cases
// ============================================

// TestReconciler_Integration_DescriptionOnly tests updating just the description field.
func TestReconciler_Integration_DescriptionOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	sim := netsim.NewSimulator()
	err := sim.Start()
	assert.NoError(t, err)
	defer sim.Stop()

	c := cache.NewTTLLRUCache(100, 30*time.Second, 1*time.Minute)
	cs := manager.NewInMemoryConfigStore(c)
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	desired := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
			400: {
				Id:          uint16Ptr(400),
				Name:        stringPtr("DescTest"),
				Description: stringPtr("Testing description field only"),
				Type:        huawei.HuaweiVlan_VlanType_common,
			},
		},
	}

	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/vlan:vlan/vlan:vlans"
	err = cs.Set(deviceID, path, desired)
	assert.NoError(t, err)

	r := New(cs, pool, nil)
	result := r.Reconcile(context.Background(), reconcile.Request{DeviceID: deviceID, Path: path})

	assert.NoError(t, result.Error)
	testsupport.AssertHuaweiVlanDescription(t, sim, 400, "Testing description field only")
}

// TestReconciler_Integration_AdminStatus tests admin status enable/disable.
func TestReconciler_Integration_AdminStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	sim := netsim.NewSimulator()
	err := sim.Start()
	assert.NoError(t, err)
	defer sim.Stop()

	c := cache.NewTTLLRUCache(100, 30*time.Second, 1*time.Minute)
	cs := manager.NewInMemoryConfigStore(c)
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	desired := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
			500: {
				Id:          uint16Ptr(500),
				Name:        stringPtr("AdminDownVLAN"),
				Type:        huawei.HuaweiVlan_VlanType_common,
				AdminStatus: huawei.HuaweiVlan_AdminStatus_down, // Administratively down
			},
		},
	}

	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/vlan:vlan/vlan:vlans"
	err = cs.Set(deviceID, path, desired)
	assert.NoError(t, err)

	r := New(cs, pool, nil)
	result := r.Reconcile(context.Background(), reconcile.Request{DeviceID: deviceID, Path: path})

	assert.NoError(t, result.Error)
	testsupport.AssertHuaweiVlanAdminStatus(t, sim, 500, int(huawei.HuaweiVlan_AdminStatus_down))
}

// TestReconciler_Integration_MacAgingTime tests MAC aging time configuration.
func TestReconciler_Integration_MacAgingTime(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	sim := netsim.NewSimulator()
	err := sim.Start()
	assert.NoError(t, err)
	defer sim.Stop()

	c := cache.NewTTLLRUCache(100, 30*time.Second, 1*time.Minute)
	cs := manager.NewInMemoryConfigStore(c)
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	desired := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
			600: {
				Id:           uint16Ptr(600),
				Name:         stringPtr("AgingTimeTest"),
				Type:         huawei.HuaweiVlan_VlanType_common,
				MacAgingTime: uint32Ptr(600), // 10 minutes
			},
		},
	}

	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/vlan:vlan/vlan:vlans"
	err = cs.Set(deviceID, path, desired)
	assert.NoError(t, err)

	r := New(cs, pool, nil)
	result := r.Reconcile(context.Background(), reconcile.Request{DeviceID: deviceID, Path: path})

	assert.NoError(t, result.Error)
	testsupport.AssertHuaweiVlanMacAgingTime(t, sim, 600, uint32(600))
}
