package interfaces

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/internal/cache"
	"github.com/leezesi/usmp/backend/internal/generated/openconfig"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
	testsupport "github.com/leezesi/usmp/backend/simulator/netconfsim/testsupport"
	"github.com/stretchr/testify/assert"
)

// TestReconciler_Integration_CreateInterface tests the full flow for creating interface configuration
func TestReconciler_Integration_CreateInterface(t *testing.T) {
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

	// 4. Set desired interface configuration
	desired := &openconfig.OpenconfigInterfaces_Interfaces{
		Interface: map[string]*openconfig.OpenconfigInterfaces_Interfaces_Interface{
			"GigabitEthernet0/0": {
				Name: stringPtr("GigabitEthernet0/0"),
				Config: &openconfig.OpenconfigInterfaces_Interfaces_Interface_Config{
					Name:        stringPtr("GigabitEthernet0/0"),
					Type:        openconfig.OpenconfigInterfaces_Interfaces_Interface_Config_Type_PHYSICAL,
					Enabled:     boolPtr(true),
					Mtu:         uint16Ptr(1500),
					Description: stringPtr("Uplink to core router"),
				},
			},
		},
	}

	// deviceID format "user:pass@host:port" includes credentials and port for integration testing
	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/interfaces"
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

	// 6. Verify result - reconciliation should succeed without requeue
	assert.NoError(t, result.Error)
	assert.False(t, result.Requeue)

	// 7. Verify config was actually applied - read back from simulator
	testsupport.AssertInterfaceExists(t, sim, "GigabitEthernet0/0")
	testsupport.AssertInterfaceEnabled(t, sim, "GigabitEthernet0/0", true)
	testsupport.AssertInterfaceMtu(t, sim, "GigabitEthernet0/0", 1500)
}

// TestReconciler_Integration_ModifyInterface tests modifying an existing interface configuration
func TestReconciler_Integration_ModifyInterface(t *testing.T) {
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

	// 4. Set desired configuration - modified description and MTU
	desired := &openconfig.OpenconfigInterfaces_Interfaces{
		Interface: map[string]*openconfig.OpenconfigInterfaces_Interfaces_Interface{
			"GigabitEthernet0/0": {
				Name: stringPtr("GigabitEthernet0/0"),
				Config: &openconfig.OpenconfigInterfaces_Interfaces_Interface_Config{
					Name:        stringPtr("GigabitEthernet0/0"),
					Type:        openconfig.OpenconfigInterfaces_Interfaces_Interface_Config_Type_PHYSICAL,
					Enabled:     boolPtr(true),
					Mtu:         uint16Ptr(9000), // Jumbo frames
					Description: stringPtr("Updated description - uplink to core"),
				},
			},
		},
	}

	// deviceID format "user:pass@host:port" includes credentials and port for integration testing
	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/interfaces"
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

	// 6. Verify result
	assert.NoError(t, result.Error)
	assert.False(t, result.Requeue)

	// 7. Verify config was updated in simulator
	testsupport.AssertInterfaceExists(t, sim, "GigabitEthernet0/0")
	testsupport.AssertInterfaceMtu(t, sim, "GigabitEthernet0/0", 9000)
}

// TestReconciler_Integration_DisableInterface tests disabling an interface
func TestReconciler_Integration_DisableInterface(t *testing.T) {
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

	// 4. Set desired configuration - disable the interface
	desired := &openconfig.OpenconfigInterfaces_Interfaces{
		Interface: map[string]*openconfig.OpenconfigInterfaces_Interfaces_Interface{
			"GigabitEthernet0/1": {
				Name: stringPtr("GigabitEthernet0/1"),
				Config: &openconfig.OpenconfigInterfaces_Interfaces_Interface_Config{
					Name:        stringPtr("GigabitEthernet0/1"),
					Type:        openconfig.OpenconfigInterfaces_Interfaces_Interface_Config_Type_PHYSICAL,
					Enabled:     boolPtr(false),
					Description: stringPtr("Backup interface - disabled"),
				},
			},
		},
	}

	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/interfaces"
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

	// 6. Verify result
	assert.NoError(t, result.Error)
	assert.False(t, result.Requeue)

	// 7. Verify interface was disabled in simulator
	testsupport.AssertInterfaceExists(t, sim, "GigabitEthernet0/1")
	testsupport.AssertInterfaceEnabled(t, sim, "GigabitEthernet0/1", false)
}

// TestReconciler_Integration_CommitError tests handling of commit failures
func TestReconciler_Integration_CommitError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Start NETCONF simulator
	sim := netsim.NewSimulator()
	err := sim.Start()
	assert.NoError(t, err)
	defer sim.Stop()

	// 2. Setup error scenario - commit will fail
	sc := netsim.NewScenarioConfig()
	sc.ErrorOnRPC = map[string]error{
		"commit": fmt.Errorf("commit failed: device busy"),
	}
	sim.SetScenario(sc)

	// 3. Create in-memory cache and config store
	c := cache.NewTTLLRUCache(100, 30*time.Second, 1*time.Minute)
	cs := manager.NewInMemoryConfigStore(c)

	// 4. Create client pool
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	// 5. Set desired configuration
	desired := &openconfig.OpenconfigInterfaces_Interfaces{
		Interface: map[string]*openconfig.OpenconfigInterfaces_Interfaces_Interface{
			"GigabitEthernet0/0": {
				Name: stringPtr("GigabitEthernet0/0"),
				Config: &openconfig.OpenconfigInterfaces_Interfaces_Interface_Config{
					Name:        stringPtr("GigabitEthernet0/0"),
					Type:        openconfig.OpenconfigInterfaces_Interfaces_Interface_Config_Type_PHYSICAL,
					Enabled:     boolPtr(true),
					Mtu:         uint16Ptr(1500),
					Description: stringPtr("Test interface"),
				},
			},
		},
	}

	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/interfaces"
	err = cs.Set(deviceID, path, desired)
	assert.NoError(t, err)

	// 6. Execute reconciliation - should fail with commit error
	r := New(cs, pool)
	req := reconcile.Request{
		DeviceID: deviceID,
		Path:     path,
	}

	ctx := context.Background()
	result := r.Reconcile(ctx, req)

	// 7. Verify error was properly handled
	assert.True(t, result.Requeue, "should requeue on commit failure")
	assert.Error(t, result.Error, "should return an error")
}

// Helper functions for pointer types
func stringPtr(s string) *string {
	return &s
}

func uint16Ptr(v uint16) *uint16 {
	return &v
}

func boolPtr(b bool) *bool {
	return &b
}
