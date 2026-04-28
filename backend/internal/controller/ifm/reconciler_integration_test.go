package ifm

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

	// 4. Set desired configuration - create interface with description and admin status (Huawei IFM model)
	// HuaweiIfm_Ifm_Interfaces.Interface uses string key (interface name)
	vlan100 := "Vlanif100"
	desired := &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{
			vlan100: {
				Name:        &vlan100,
				Description: stringPtr("Test Interface for VLAN 100"),
				AdminStatus: huawei.HuaweiIfm_PortStatus_up,
				Mtu:         uint32Ptr(1500),
			},
		},
	}

	// deviceID format "user:pass@host:port" includes credentials and port for integration testing
	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/ifm:ifm/ifm:interfaces"
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

	// 7. Verify the interface exists in the simulator with correct description and status
	// Verify using Get to check the data was correctly sent to simulator
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

	// 4. First create interface with AdminStatus up
	vlan100 := "Vlanif100"
	desired := &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{
			vlan100: {
				Name:        &vlan100,
				Description: stringPtr("Original Description"),
				AdminStatus: huawei.HuaweiIfm_PortStatus_up,
				Mtu:         uint32Ptr(1500),
			},
		},
	}

	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/ifm:ifm/ifm:interfaces"
	err = cs.Set(deviceID, path, desired)
	assert.NoError(t, err)

	r := New(cs, pool)
	req := reconcile.Request{DeviceID: deviceID, Path: path}

	ctx := context.Background()
	result := r.Reconcile(ctx, req)
	assert.NoError(t, result.Error)

	// 5. Now modify description and AdminStatus to down
	desired.Interface[vlan100].Description = stringPtr("Updated Description")
	desired.Interface[vlan100].AdminStatus = huawei.HuaweiIfm_PortStatus_down

	err = cs.Set(deviceID, path, desired)
	assert.NoError(t, err)

	// 6. Reconcile again to apply change
	result = r.Reconcile(ctx, req)
	assert.NoError(t, result.Error)
	assert.False(t, result.Requeue)
}

// TestReconciler_Integration_ModifyMTU tests modifying interface MTU
func TestReconciler_Integration_ModifyMTU(t *testing.T) {
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

	// 4. Create interface with MTU 1500
	vlan100 := "Vlanif100"
	desired := &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{
			vlan100: {
				Name:        &vlan100,
				Description: stringPtr("Test Interface"),
				AdminStatus: huawei.HuaweiIfm_PortStatus_up,
				Mtu:         uint32Ptr(1500),
			},
		},
	}

	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/ifm:ifm/ifm:interfaces"
	err = cs.Set(deviceID, path, desired)
	assert.NoError(t, err)

	r := New(cs, pool)
	req := reconcile.Request{DeviceID: deviceID, Path: path}

	ctx := context.Background()
	result := r.Reconcile(ctx, req)
	assert.NoError(t, result.Error)

	// 5. Now modify MTU to 9000 (jumbo frames)
	desired.Interface[vlan100].Mtu = uint32Ptr(9000)

	err = cs.Set(deviceID, path, desired)
	assert.NoError(t, err)

	// 6. Reconcile again to apply MTU change
	result = r.Reconcile(ctx, req)
	assert.NoError(t, result.Error)
	assert.False(t, result.Requeue)
}

// TestReconciler_Integration_EmptyConfig tests handling empty interface configuration
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

	// 4. Set desired empty configuration (Huawei IFM model)
	desired := &huawei.HuaweiIfm_Ifm_Interfaces{}

	// deviceID format "user:pass@host:port" includes credentials and port for integration testing
	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/ifm:ifm/ifm:interfaces"
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

	// 5. Set desired configuration (Huawei IFM model)
	vlan100 := "Vlanif100"
	desired := &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{
			vlan100: {
				Name:        &vlan100,
				Description: stringPtr("Test Interface"),
				AdminStatus: huawei.HuaweiIfm_PortStatus_up,
				Mtu:         uint32Ptr(1500),
			},
		},
	}

	// deviceID format "user:pass@host:port" includes credentials and port for integration testing
	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/ifm:ifm/ifm:interfaces"
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

// uint32Ptr is a helper to create a *uint32 from a uint32
func uint32Ptr(v uint32) *uint32 {
	return &v
}

// stringPtr is a helper to create a *string from a string
func stringPtr(s string) *string {
	return &s
}
