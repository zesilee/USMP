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
	r := New(cs, pool)
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
	sim.AssertHuaweiVlanExists(t, 100)
	sim.AssertHuaweiVlanName(t, 100, "TestVLAN")
	sim.AssertHuaweiVlanCount(t, 1)
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
	r := New(cs, pool)
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
	sim.AssertHuaweiVlanExists(t, 100)
	sim.AssertHuaweiVlanName(t, 100, "NewName")
	sim.AssertHuaweiVlanCount(t, 1)
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
	r := New(cs, pool)
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
	sim.AssertHuaweiVlanExists(t, 100)
	sim.AssertHuaweiVlanCount(t, 1)
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
	r := New(cs, pool)
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
	sim.AssertHuaweiVlanCount(t, 0)
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
	r := New(cs, pool)
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
	r := New(cs, pool)
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
