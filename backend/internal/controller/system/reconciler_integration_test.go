package system

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

// TestReconciler_Integration_SystemBasics tests basic system configuration
func TestReconciler_Integration_SystemBasics(t *testing.T) {
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

	// 4. Set desired system configuration
	desired := &huawei.HuaweiSystem_System{
		SystemInfo: &huawei.HuaweiSystem_System_SystemInfo{
			SysName:     stringPtr("TestRouter"),
			SysContact:  stringPtr("admin@example.com"),
			SysLocation: stringPtr("Beijing, China"),
		},
	}

	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/system:system"
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

	// 7. Verify the configuration exists in the simulator
	testsupport.AssertHuaweiSystem(t, sim, "TestRouter", "admin@example.com", "Beijing, China")
}

// TestReconciler_Integration_SystemModify tests modifying existing system configuration
func TestReconciler_Integration_SystemModify(t *testing.T) {
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

	// 4. First set initial config
	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	path := "/system:system"

	initial := &huawei.HuaweiSystem_System{
		SystemInfo: &huawei.HuaweiSystem_System_SystemInfo{
			SysName: stringPtr("OldRouter"),
		},
	}

	err = cs.Set(deviceID, path, initial)
	assert.NoError(t, err)

	r := New(cs, pool)
	req := reconcile.Request{DeviceID: deviceID, Path: path}

	ctx := context.Background()
	result := r.Reconcile(ctx, req)
	assert.NoError(t, result.Error)

	// 5. Now modify system configuration
	updated := &huawei.HuaweiSystem_System{
		SystemInfo: &huawei.HuaweiSystem_System_SystemInfo{
			SysName:     stringPtr("NewRouter"),
			SysContact:  stringPtr("support@example.com"),
			SysLocation: stringPtr("Shanghai, China"),
		},
	}

	err = cs.Set(deviceID, path, updated)
	assert.NoError(t, err)

	// 6. Reconcile again to apply change
	result = r.Reconcile(ctx, req)
	assert.NoError(t, result.Error)
	assert.False(t, result.Requeue)

	// 7. Verify updated configuration
	testsupport.AssertHuaweiSystemName(t, sim, "NewRouter")
	testsupport.AssertHuaweiSystem(t, sim, "", "support@example.com", "Shanghai, China")
}

// stringPtr is a helper to create a *string from a string
func stringPtr(s string) *string {
	return &s
}
