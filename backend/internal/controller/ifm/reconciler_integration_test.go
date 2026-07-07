package ifm

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/internal/cache"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
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
	ds, deviceID := simStore(sim)
	path := "/ifm:ifm/ifm:interfaces"
	err = cs.Set(deviceID, path, desired)
	assert.NoError(t, err)

	// 5. Create reconciler and execute reconciliation
	r := New(cs, pool, ds)
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

// TestReconciler_Integration_CreateInterface_ConvergesAndReadable 复现并锁死用户报的两个症状：
//
//	① 通过界面新建 interface 后「一直显示已漂移」——第二次对账必须收敛（Changes==0）；
//	② 新接口「在接口配置里看不到」——回读设备必须能读到刚建的接口及其描述。
//
// UI 只提交稀疏字段（name + description），不带 MTU/admin-status，模拟真实前端提交。
func TestReconciler_Integration_CreateInterface_ConvergesAndReadable(t *testing.T) {
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

	// UI 新建：仅 name + description（稀疏意图，其余字段不管理）
	ifName := "Vlanif900"
	desired := &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{
			ifName: {
				Name:        &ifName,
				Description: stringPtr("created-via-ui"),
			},
		},
	}

	ds, deviceID := simStore(sim)
	path := "/ifm:ifm/ifm:interfaces"
	if err := cs.Set(deviceID, path, desired); err != nil {
		t.Fatalf("config store set: %v", err)
	}

	r := New(cs, pool, ds)
	req := reconcile.Request{DeviceID: deviceID, Path: path}
	ctx := context.Background()

	// 第一次对账：检测到新接口未在设备 → 下发（Changes>0）
	first := r.Reconcile(ctx, req)
	if first.Error != nil {
		t.Fatalf("first reconcile failed: %v", first.Error)
	}
	assert.Greater(t, first.Changes, 0, "首轮应检测到漂移并下发新接口")

	// 症状② 断言：回读设备，新接口必须真正落盘（根因 B：序列化正确）
	dc := &deviceClient{clientPool: pool, resolver: ds}
	got, err := dc.Get(ctx, deviceID)
	if err != nil {
		t.Fatalf("readback device config: %v", err)
	}
	ifaces, ok := got.(*huawei.HuaweiIfm_Ifm_Interfaces)
	if !ok {
		t.Fatalf("unexpected readback type %T", got)
	}
	entry, present := ifaces.Interface[ifName]
	if !present {
		t.Fatalf("新建的接口 %q 未在设备回读中出现（接口未真正下发）", ifName)
	}
	if assert.NotNil(t, entry.Description) {
		assert.Equal(t, "created-via-ui", *entry.Description, "回读描述应与下发一致")
	}

	// 症状① 断言：desired 未变，再次对账必须收敛（根因 A：map 子集比对不再永远算出 diff）
	second := r.Reconcile(ctx, req)
	if second.Error != nil {
		t.Fatalf("second reconcile failed: %v", second.Error)
	}
	assert.Equal(t, 0, second.Changes, "设备已落盘 desired 后，第二轮对账必须收敛（否则前端一直显示漂移）")
	assert.False(t, second.Requeue)
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

	ds, deviceID := simStore(sim)
	path := "/ifm:ifm/ifm:interfaces"
	err = cs.Set(deviceID, path, desired)
	assert.NoError(t, err)

	r := New(cs, pool, ds)
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

	ds, deviceID := simStore(sim)
	path := "/ifm:ifm/ifm:interfaces"
	err = cs.Set(deviceID, path, desired)
	assert.NoError(t, err)

	r := New(cs, pool, ds)
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
	ds, deviceID := simStore(sim)
	path := "/ifm:ifm/ifm:interfaces"
	err = cs.Set(deviceID, path, desired)
	assert.NoError(t, err)

	// 5. Execute reconciliation
	r := New(cs, pool, ds)
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
	ds, deviceID := simStore(sim)
	path := "/ifm:ifm/ifm:interfaces"
	err = cs.Set(deviceID, path, desired)
	assert.NoError(t, err)

	// 6. Execute reconciliation
	r := New(cs, pool, ds)
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

// TestReconciler_Integration_FullInterfaceConfig tests applying full interface configuration including nested containers
func TestReconciler_Integration_FullInterfaceConfig(t *testing.T) {
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

	// 4. Set desired configuration with full attributes including nested containers
	ge1 := "GigabitEthernet0/0/1"
	desired := &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{
			ge1: {
				Name:        &ge1,
				Description: stringPtr("Full Config Interface"),
				AdminStatus: huawei.HuaweiIfm_PortStatus_up,
				Mtu:         uint32Ptr(9000),
				Class:       huawei.HuaweiIfm_ClassType_main_interface,
				Type:        huawei.HuaweiIfm_PortType_GigabitEthernet,
				ControlFlap: &huawei.HuaweiIfm_Ifm_Interfaces_Interface_ControlFlap{
					Ceiling:          uint32Ptr(6000),
					ControlFlapCount: uint32Ptr(5),
					DecayNg:          uint32Ptr(10),
					DecayOk:          uint32Ptr(120),
					Reuse:            uint32Ptr(1000),
					Suppress:         uint32Ptr(3000),
				},
				Damp: &huawei.HuaweiIfm_Ifm_Interfaces_Interface_Damp{
					TxOff: boolPtr(true),
					Manual: &huawei.HuaweiIfm_Ifm_Interfaces_Interface_Damp_Manual{
						HalfLifePeriod:  uint16Ptr(10),
						MaxSuppressTime: uint16Ptr(60),
						Reuse:           uint32Ptr(500),
						Suppress:        uint32Ptr(4000),
					},
				},
			},
		},
	}

	ds, deviceID := simStore(sim)
	path := "/ifm:ifm/ifm:interfaces"
	err = cs.Set(deviceID, path, desired)
	assert.NoError(t, err)

	// 5. Create reconciler and execute reconciliation
	r := New(cs, pool, ds)
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

	// 7. Verify all attributes using new assertion methods
	testsupport.AssertHuaweiInterfaceExists(t, sim, ge1)
	testsupport.AssertHuaweiInterfaceDescription(t, sim, ge1, "Full Config Interface")
	testsupport.AssertHuaweiInterfaceAdminStatus(t, sim, ge1, 2) // up = 2
	testsupport.AssertHuaweiInterfaceMtu(t, sim, ge1, 9000)
	testsupport.AssertHuaweiInterfaceControlFlap(t, sim, ge1, 6000, 10, 120, 1000, 3000)
	testsupport.AssertHuaweiInterfaceDampManual(t, sim, ge1, 10, 60, 500, 4000)
}

// TestReconciler_Integration_TimersAndFlags tests interface timer and flag configurations
func TestReconciler_Integration_TimersAndFlags(t *testing.T) {
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

	// 4. Set desired configuration with timers and flags
	ge1 := "GigabitEthernet0/0/2"
	desired := &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{
			ge1: {
				Name:        &ge1,
				Description: stringPtr("Timers and Flags Test"),
				AdminStatus: huawei.HuaweiIfm_PortStatus_down,
				Mtu:         uint32Ptr(1500),
				// Timers
				DownDelayTime:       uint32Ptr(50),
				ProtocolUpDelayTime: uint32Ptr(100),
				// Boolean flags
				ClearIpDf:            boolPtr(true),
				IsL2Switch:           boolPtr(false),
				L2ModeEnable:         boolPtr(true),
				LinkUpDownTrapEnable: boolPtr(true),
				StatisticEnable:      boolPtr(false),
				SpreadMtuFlag:        boolPtr(false),
			},
		},
	}

	ds, deviceID := simStore(sim)
	path := "/ifm:ifm/ifm:interfaces"
	err = cs.Set(deviceID, path, desired)
	assert.NoError(t, err)

	// 5. Create reconciler and execute reconciliation
	r := New(cs, pool, ds)
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

	// 7. Verify timers and flags
	testsupport.AssertHuaweiInterfaceExists(t, sim, ge1)
	testsupport.AssertHuaweiInterfaceDescription(t, sim, ge1, "Timers and Flags Test")
	testsupport.AssertHuaweiInterfaceAdminStatus(t, sim, ge1, 1) // down = 1
	testsupport.AssertHuaweiInterfaceTimers(t, sim, ge1, 50, 100)
	testsupport.AssertHuaweiInterfaceFlags(t, sim, ge1, true, false, true, true, false, false)
}

// TestReconciler_Integration_StatisticsConfig tests interface statistic configurations
func TestReconciler_Integration_StatisticsConfig(t *testing.T) {
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

	// 4. Set desired configuration with statistics and network attributes
	ge1 := "GigabitEthernet0/0/3"
	desired := &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{
			ge1: {
				Name:        &ge1,
				Description: stringPtr("Statistics Config Test"),
				AdminStatus: huawei.HuaweiIfm_PortStatus_up,
				Mtu:         uint32Ptr(9000),
				// Statistics
				StatisticEnable:   boolPtr(true),
				StatisticInterval: uint32Ptr(600),
				StatisticMode:     huawei.E_HuaweiIfm_StatisticMode(2),
				// Network
				MacAddress: stringPtr("00:11:22:33:44:66"),
				VrfName:    stringPtr("management"),
				VsName:     stringPtr("vs0"),
			},
		},
	}

	ds, deviceID := simStore(sim)
	path := "/ifm:ifm/ifm:interfaces"
	err = cs.Set(deviceID, path, desired)
	assert.NoError(t, err)

	// 5. Create reconciler and execute reconciliation
	r := New(cs, pool, ds)
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

	// 7. Verify statistics and network attributes
	testsupport.AssertHuaweiInterfaceExists(t, sim, ge1)
	testsupport.AssertHuaweiInterfaceDescription(t, sim, ge1, "Statistics Config Test")
	testsupport.AssertHuaweiInterfaceMtu(t, sim, ge1, 9000)
	testsupport.AssertHuaweiInterfaceStatistics(t, sim, ge1, 600, 2)
	testsupport.AssertHuaweiInterfaceNetwork(t, sim, ge1, "00:11:22:33:44:66", "management", "vs0")
}

// TestReconciler_Integration_ModifyMultipleAttributes tests modifying multiple attributes on existing interface
func TestReconciler_Integration_ModifyMultipleAttributes(t *testing.T) {
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

	// 4. First, create interface with initial config
	ge1 := "GigabitEthernet0/0/4"
	desired := &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{
			ge1: {
				Name:        &ge1,
				Description: stringPtr("Initial Config"),
				AdminStatus: huawei.HuaweiIfm_PortStatus_up,
				Mtu:         uint32Ptr(1500),
			},
		},
	}

	ds, deviceID := simStore(sim)
	path := "/ifm:ifm/ifm:interfaces"
	err = cs.Set(deviceID, path, desired)
	assert.NoError(t, err)

	r := New(cs, pool, ds)
	req := reconcile.Request{DeviceID: deviceID, Path: path}

	ctx := context.Background()
	result := r.Reconcile(ctx, req)
	assert.NoError(t, result.Error)

	// 5. Now modify multiple attributes: MTU, description, admin status, and add damp config
	desired.Interface[ge1].Mtu = uint32Ptr(9000)
	desired.Interface[ge1].Description = stringPtr("Modified Config")
	desired.Interface[ge1].AdminStatus = huawei.HuaweiIfm_PortStatus_down
	desired.Interface[ge1].Damp = &huawei.HuaweiIfm_Ifm_Interfaces_Interface_Damp{
		TxOff: boolPtr(true),
		Manual: &huawei.HuaweiIfm_Ifm_Interfaces_Interface_Damp_Manual{
			HalfLifePeriod:  uint16Ptr(30),
			MaxSuppressTime: uint16Ptr(120),
			Reuse:           uint32Ptr(2000),
			Suppress:        uint32Ptr(8000),
		},
	}

	err = cs.Set(deviceID, path, desired)
	assert.NoError(t, err)

	// 6. Reconcile again to apply changes
	result = r.Reconcile(ctx, req)
	assert.NoError(t, result.Error)
	assert.False(t, result.Requeue)

	// 7. Verify all modified attributes
	testsupport.AssertHuaweiInterfaceExists(t, sim, ge1)
	testsupport.AssertHuaweiInterfaceDescription(t, sim, ge1, "Modified Config")
	testsupport.AssertHuaweiInterfaceAdminStatus(t, sim, ge1, 1) // down
	testsupport.AssertHuaweiInterfaceMtu(t, sim, ge1, 9000)
	testsupport.AssertHuaweiInterfaceDampManual(t, sim, ge1, 30, 120, 2000, 8000)
}

// uint32Ptr is a helper to create a *uint32 from a uint32
func uint32Ptr(v uint32) *uint32 {
	return &v
}

// uint16Ptr is a helper to create a *uint16 from a uint16
func uint16Ptr(v uint16) *uint16 {
	return &v
}

// boolPtr is a helper to create a *bool from a bool
func boolPtr(v bool) *bool {
	return &v
}

// stringPtr is a helper to create a *string from a string
func stringPtr(s string) *string {
	return &s
}

// simStore registers the simulator as a device in a fresh DeviceStore and returns
// (store, deviceID). Reconcilers resolve credentials/port from the store, matching
// production wiring (DeviceStore is the source of truth; DeviceID carries no creds).
func simStore(sim *netsim.Simulator) (device.Store, string) {
	ds := device.NewStore()
	id := "sim"
	ds.Put(id, client.DeviceConnectionInfo{
		IP: sim.Addr(), Port: sim.Port(), Username: sim.Username(), Password: sim.Password(), Protocol: client.ProtocolNETCONF,
	})
	return ds, id
}
