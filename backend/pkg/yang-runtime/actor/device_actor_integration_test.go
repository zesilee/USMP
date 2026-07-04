package actor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
	testsupport "github.com/leezesi/usmp/backend/simulator/netconfsim/testsupport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeviceActor_Integration_SingleModule tests DeviceActor with a single module
func TestDeviceActor_Integration_SingleModule(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Start NETCONF simulator
	sim := netsim.NewSimulator()
	err := sim.Start()
	require.NoError(t, err)
	defer sim.Stop()

	// 2. Create device actor
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	deviceActor := NewDeviceActor(deviceID, pool)

	// 3. Register VLAN module actor
	vlanTranslator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans]()
	vlanActor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans](
		"vlan-actor",
		deviceID,
		pool,
		vlanTranslator,
	)

	err = deviceActor.RegisterModuleActor("huawei-vlan", vlanActor)
	require.NoError(t, err)

	// 4. Start device actor
	err = deviceActor.Start()
	require.NoError(t, err)
	defer deviceActor.Stop()

	time.Sleep(100 * time.Millisecond)

	// 5. Send translate command to set desired state
	translateCmd := &TranslateCmd{
		BaseMessage: NewBaseMessage("translate-vlan", MsgTranslate),
		Payload: map[string]interface{}{
			"Id":   1000,
			"Name": "DeviceActorTestVLAN",
			"Type": huawei.HuaweiVlan_VlanType_common,
		},
		Operation: OperationMerge,
	}

	promise, err := vlanActor.Send(translateCmd)
	require.NoError(t, err)
	result := <-promise
	require.True(t, result.Success, "translate failed: %v", result.Error)

	// 6. Execute PrepareAll (cross-module prepare)
	ctx := context.Background()
	txState, err := deviceActor.PrepareAll(ctx, false)
	require.NoError(t, err, "PrepareAll failed: %v", err)
	require.NotNil(t, txState)

	assert.Equal(t, "prepared", txState.Status)
	assert.Contains(t, txState.Modules, "huawei-vlan")

	// 7. Execute CommitAll
	txState, err = deviceActor.CommitAll(ctx, false)
	require.NoError(t, err, "CommitAll failed: %v", err)
	require.NotNil(t, txState)

	assert.Equal(t, "committed", txState.Status)

	// 8. Verify VLAN exists in simulator
	testsupport.AssertHuaweiVlanExists(t, sim, 1000)
	testsupport.AssertHuaweiVlanName(t, sim, 1000, "DeviceActorTestVLAN")
}

// TestDeviceActor_Integration_DryRun tests DeviceActor dry run mode
func TestDeviceActor_Integration_DryRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Start NETCONF simulator
	sim := netsim.NewSimulator()
	err := sim.Start()
	require.NoError(t, err)
	defer sim.Stop()

	// 2. Create device actor
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	deviceActor := NewDeviceActor(deviceID, pool)

	// 3. Register VLAN module actor
	vlanTranslator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans]()
	vlanActor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans](
		"vlan-actor",
		deviceID,
		pool,
		vlanTranslator,
	)

	err = deviceActor.RegisterModuleActor("huawei-vlan", vlanActor)
	require.NoError(t, err)

	err = deviceActor.Start()
	require.NoError(t, err)
	defer deviceActor.Stop()

	time.Sleep(100 * time.Millisecond)

	// 4. Set desired state
	translateCmd := &TranslateCmd{
		BaseMessage: NewBaseMessage("translate-vlan", MsgTranslate),
		Payload: map[string]interface{}{
			"Id":   2000,
			"Name": "DryRunTestVLAN",
			"Type": huawei.HuaweiVlan_VlanType_common,
		},
		Operation: OperationMerge,
	}

	promise, err := vlanActor.Send(translateCmd)
	require.NoError(t, err)
	result := <-promise
	require.True(t, result.Success)

	// 5. Dry run Prepare
	ctx := context.Background()
	txState, err := deviceActor.PrepareAll(ctx, true)
	require.NoError(t, err)
	require.NotNil(t, txState)

	assert.Equal(t, "prepared", txState.Status)

	// 6. Verify VLAN should NOT exist in running config (dry run mode)
	vlans := sim.RunningHuaweiVLANs()
	assert.Equal(t, 0, len(vlans), "dry run should not modify running config")
}

// TestDeviceActor_Integration_Abort tests AbortAll after Prepare
func TestDeviceActor_Integration_Abort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Start NETCONF simulator
	sim := netsim.NewSimulator()
	err := sim.Start()
	require.NoError(t, err)
	defer sim.Stop()

	// 2. Create device actor
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	deviceActor := NewDeviceActor(deviceID, pool)

	// 3. Register VLAN module actor
	vlanTranslator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans]()
	vlanActor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans](
		"vlan-actor",
		deviceID,
		pool,
		vlanTranslator,
	)

	err = deviceActor.RegisterModuleActor("huawei-vlan", vlanActor)
	require.NoError(t, err)

	err = deviceActor.Start()
	require.NoError(t, err)
	defer deviceActor.Stop()

	time.Sleep(100 * time.Millisecond)

	// 4. Set desired state
	translateCmd := &TranslateCmd{
		BaseMessage: NewBaseMessage("translate-vlan", MsgTranslate),
		Payload: map[string]interface{}{
			"Id":   3000,
			"Name": "ToBeAbortedVLAN",
			"Type": huawei.HuaweiVlan_VlanType_common,
		},
		Operation: OperationMerge,
	}

	promise, err := vlanActor.Send(translateCmd)
	require.NoError(t, err)
	result := <-promise
	require.True(t, result.Success)

	// 5. Prepare
	ctx := context.Background()
	txState, err := deviceActor.PrepareAll(ctx, false)
	require.NoError(t, err)
	require.NotNil(t, txState)

	// 6. Abort the transaction
	txState, err = deviceActor.AbortAll(ctx, "Abort for integration test")
	require.NoError(t, err, "AbortAll failed: %v", err)
	require.NotNil(t, txState)

	assert.Equal(t, "aborted", txState.Status)

	// 7. Verify VLAN should NOT exist after abort
	vlans := sim.RunningHuaweiVLANs()
	assert.Equal(t, 0, len(vlans), "aborted changes should not be in running config")
}

// TestDeviceActor_Integration_PrepareAndCommitAll tests the complete
// PrepareAndCommitAll convenience function
func TestDeviceActor_Integration_PrepareAndCommitAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Start NETCONF simulator
	sim := netsim.NewSimulator()
	err := sim.Start()
	require.NoError(t, err)
	defer sim.Stop()

	// 2. Create device actor
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	deviceActor := NewDeviceActor(deviceID, pool)

	// 3. Register VLAN module actor
	vlanTranslator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans]()
	vlanActor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans](
		"vlan-actor",
		deviceID,
		pool,
		vlanTranslator,
	)

	err = deviceActor.RegisterModuleActor("huawei-vlan", vlanActor)
	require.NoError(t, err)

	err = deviceActor.Start()
	require.NoError(t, err)
	defer deviceActor.Stop()

	time.Sleep(100 * time.Millisecond)

	// 4. Set desired state
	translateCmd := &TranslateCmd{
		BaseMessage: NewBaseMessage("translate-vlan", MsgTranslate),
		Payload: map[string]interface{}{
			"Id":   4000,
			"Name": "FullFlowVLAN",
			"Type": huawei.HuaweiVlan_VlanType_common,
		},
		Operation: OperationMerge,
	}

	promise, err := vlanActor.Send(translateCmd)
	require.NoError(t, err)
	result := <-promise
	require.True(t, result.Success)

	// 5. Execute full PrepareAndCommitAll
	ctx := context.Background()
	txState, err := deviceActor.PrepareAndCommitAll(ctx, false)
	require.NoError(t, err, "PrepareAndCommitAll failed: %v", err)
	require.NotNil(t, txState)

	assert.Equal(t, "committed", txState.Status)

	// 6. Verify the VLAN exists
	testsupport.AssertHuaweiVlanExists(t, sim, 4000)
	testsupport.AssertHuaweiVlanName(t, sim, 4000, "FullFlowVLAN")
}

// TestDeviceActor_Integration_ApplyAll tests ApplyAll for direct config push
func TestDeviceActor_Integration_ApplyAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Start NETCONF simulator
	sim := netsim.NewSimulator()
	err := sim.Start()
	require.NoError(t, err)
	defer sim.Stop()

	// 2. Create device actor
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())
	deviceActor := NewDeviceActor(deviceID, pool)

	// 3. Register VLAN module actor
	vlanTranslator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans]()
	vlanActor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans](
		"vlan-actor",
		deviceID,
		pool,
		vlanTranslator,
	)

	err = deviceActor.RegisterModuleActor("huawei-vlan", vlanActor)
	require.NoError(t, err)

	err = deviceActor.Start()
	require.NoError(t, err)
	defer deviceActor.Stop()

	time.Sleep(100 * time.Millisecond)

	// 4. Set desired state
	translateCmd := &TranslateCmd{
		BaseMessage: NewBaseMessage("translate-vlan", MsgTranslate),
		Payload: map[string]interface{}{
			"Id":   500,
			"Name": "ApplyAllVLAN",
			"Type": huawei.HuaweiVlan_VlanType_common,
		},
		Operation: OperationMerge,
	}

	promise, err := vlanActor.Send(translateCmd)
	require.NoError(t, err)
	result := <-promise
	require.True(t, result.Success, "translate failed: %v", result.Error)

	// 5. Execute ApplyAll directly
	ctx := context.Background()
	results, err := deviceActor.ApplyAll(ctx)
	require.NoError(t, err, "ApplyAll failed: %v", err)

	// Verify at least one result was returned
	assert.GreaterOrEqual(t, len(results), 1, "expected at least one apply result")

	// 6. Verify VLAN exists
	testsupport.AssertHuaweiVlanExists(t, sim, 500)
	testsupport.AssertHuaweiVlanName(t, sim, 500, "ApplyAllVLAN")
}
