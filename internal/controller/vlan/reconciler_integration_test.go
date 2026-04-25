package vlan

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/leezesi/usmp/internal/cache"
	"github.com/leezesi/usmp/internal/generated/openconfig"
	"github.com/leezesi/usmp/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/pkg/yang-runtime/reconcile"
	netsim "github.com/leezesi/usmp/test/netconf-simulator"
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

	// 4. Set desired configuration - create VLAN 100 with name "TestVLAN"
	desired := &openconfig.OpenconfigVlan_Vlans{
		Vlan: map[uint16]*openconfig.OpenconfigVlan_Vlans_Vlan{
			100: {
				VlanId:   uint16Ptr(100),
				Config: &openconfig.OpenconfigVlan_Vlans_Vlan_Config{
					Name:   stringPtr("TestVLAN"),
					Status: openconfig.OpenconfigVlan_Vlans_Vlan_Config_Status_ACTIVE,
					VlanId: uint16Ptr(100),
				},
			},
		},
	}

	// deviceID format "user:pass@host:port" includes credentials and port for integration testing
	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/vlans"
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

	// 7. Verify the VLAN exists in the simulator with correct name
	sim.AssertVlanExists(t, 100)
	sim.AssertVlanName(t, 100, "TestVLAN")
	sim.AssertVlanCount(t, 1)
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

	// 2. Set initial configuration with existing VLAN
	initial := &openconfig.Device{
		Vlans: &openconfig.OpenconfigVlan_Vlans{
			Vlan: map[uint16]*openconfig.OpenconfigVlan_Vlans_Vlan{
				100: {
					VlanId:   uint16Ptr(100),
					Config: &openconfig.OpenconfigVlan_Vlans_Vlan_Config{
						Name:   stringPtr("OldName"),
						Status: openconfig.OpenconfigVlan_Vlans_Vlan_Config_Status_ACTIVE,
						VlanId: uint16Ptr(100),
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

	// 5. Set desired configuration - modified name
	desired := &openconfig.OpenconfigVlan_Vlans{
		Vlan: map[uint16]*openconfig.OpenconfigVlan_Vlans_Vlan{
			100: {
				VlanId:   uint16Ptr(100),
				Config: &openconfig.OpenconfigVlan_Vlans_Vlan_Config{
					Name:   stringPtr("NewName"),
					Status: openconfig.OpenconfigVlan_Vlans_Vlan_Config_Status_ACTIVE,
					VlanId: uint16Ptr(100),
				},
			},
		},
	}

	// deviceID format "user:pass@host:port" includes credentials and port for integration testing
	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/vlans"
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

	// 8. Verify the name was updated
	sim.AssertVlanExists(t, 100)
	sim.AssertVlanName(t, 100, "NewName")
	sim.AssertVlanCount(t, 1)
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

	// 2. Set initial configuration with two VLANs
	initial := &openconfig.Device{
		Vlans: &openconfig.OpenconfigVlan_Vlans{
			Vlan: map[uint16]*openconfig.OpenconfigVlan_Vlans_Vlan{
				100: {
					VlanId:   uint16Ptr(100),
					Config: &openconfig.OpenconfigVlan_Vlans_Vlan_Config{
						Name:   stringPtr("VLAN100"),
						Status: openconfig.OpenconfigVlan_Vlans_Vlan_Config_Status_ACTIVE,
						VlanId: uint16Ptr(100),
					},
				},
				200: {
					VlanId:   uint16Ptr(200),
					Config: &openconfig.OpenconfigVlan_Vlans_Vlan_Config{
						Name:   stringPtr("VLAN200"),
						Status: openconfig.OpenconfigVlan_Vlans_Vlan_Config_Status_ACTIVE,
						VlanId: uint16Ptr(200),
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
	desired := &openconfig.OpenconfigVlan_Vlans{
		Vlan: map[uint16]*openconfig.OpenconfigVlan_Vlans_Vlan{
			100: {
				VlanId:   uint16Ptr(100),
				Config: &openconfig.OpenconfigVlan_Vlans_Vlan_Config{
					Name:   stringPtr("VLAN100"),
					Status: openconfig.OpenconfigVlan_Vlans_Vlan_Config_Status_ACTIVE,
					VlanId: uint16Ptr(100),
				},
			},
		},
	}

	// deviceID format "user:pass@host:port" includes credentials and port for integration testing
	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/vlans"
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

	// 8. Verify only VLAN 100 remains, VLAN 200 is gone
	sim.AssertVlanExists(t, 100)
	sim.AssertVlanCount(t, 1)
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

	// 4. Set desired empty configuration
	desired := &openconfig.OpenconfigVlan_Vlans{}

	// deviceID format "user:pass@host:port" includes credentials and port for integration testing
	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/vlans"
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

	// 7. Verify still no VLANs
	sim.AssertVlanCount(t, 0)
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

	// 5. Set desired configuration
	desired := &openconfig.OpenconfigVlan_Vlans{
		Vlan: map[uint16]*openconfig.OpenconfigVlan_Vlans_Vlan{
			100: {
				VlanId:   uint16Ptr(100),
				Config: &openconfig.OpenconfigVlan_Vlans_Vlan_Config{
					Name:   stringPtr("TestVLAN"),
					Status: openconfig.OpenconfigVlan_Vlans_Vlan_Config_Status_ACTIVE,
					VlanId: uint16Ptr(100),
				},
			},
		},
	}

	// deviceID format "user:pass@host:port" includes credentials and port for integration testing
	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/vlans"
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

	// 5. Set desired configuration
	desired := &openconfig.OpenconfigVlan_Vlans{
		Vlan: map[uint16]*openconfig.OpenconfigVlan_Vlans_Vlan{
			100: {
				VlanId:   uint16Ptr(100),
				Config: &openconfig.OpenconfigVlan_Vlans_Vlan_Config{
					Name:   stringPtr("TestVLAN"),
					Status: openconfig.OpenconfigVlan_Vlans_Vlan_Config_Status_ACTIVE,
					VlanId: uint16Ptr(100),
				},
			},
		},
	}

	// Use wrong credentials in deviceID to test authentication failure
	deviceID := fmt.Sprintf("wrong:wrong@%s:%d", sim.Addr(), sim.Port())
	path := "/vlans"
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
