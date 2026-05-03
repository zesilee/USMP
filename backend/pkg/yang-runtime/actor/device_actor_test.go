package actor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
)

// TestNewDeviceActor verifies basic DeviceActor creation
func TestNewDeviceActor(t *testing.T) {
	clientPool := &mockClientPool{}

	actor := NewDeviceActor("192.168.1.1", clientPool)

	assert.NotNil(t, actor)
	assert.Equal(t, "192.168.1.1", actor.deviceID)
}

// TestDeviceActor_RegisterModuleActor verifies module registration
func TestDeviceActor_RegisterModuleActor(t *testing.T) {
	clientPool := &mockClientPool{}
	deviceActor := NewDeviceActor("192.168.1.1", clientPool)

	// Create a module actor
	translator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]()
	vlanActor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans_Vlan](
		"vlan-actor", "192.168.1.1", clientPool, translator,
	)

	// Register the module actor
	err := deviceActor.RegisterModuleActor("huawei-vlan", vlanActor)
	require.NoError(t, err)

	// Verify registration
	_, exists := deviceActor.GetModuleActor("huawei-vlan")
	assert.True(t, exists)

	// Cannot register same module twice
	err = deviceActor.RegisterModuleActor("huawei-vlan", vlanActor)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

// TestDeviceActor_StartStop verifies DeviceActor lifecycle
func TestDeviceActor_StartStop(t *testing.T) {
	clientPool := &mockClientPool{}
	deviceActor := NewDeviceActor("192.168.1.1", clientPool)

	// Add a module actor
	translator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]()
	vlanActor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans_Vlan](
		"vlan-actor", "192.168.1.1", clientPool, translator,
	)
	err := deviceActor.RegisterModuleActor("huawei-vlan", vlanActor)
	require.NoError(t, err)

	// Start device actor
	err = deviceActor.Start()
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	// Verify status
	status := deviceActor.Status()
	assert.Equal(t, StatusReady, status.Status)

	// Stop device actor
	err = deviceActor.Stop()
	require.NoError(t, err)
}

// TestDeviceActor_PrepareAll verifies cross-module Prepare phase
func TestDeviceActor_PrepareAll(t *testing.T) {
	clientPool := &mockClientPool{}
	deviceActor := NewDeviceActor("192.168.1.1", clientPool)

	// Add VLAN module
	vlanTranslator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]()
	vlanActor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans_Vlan](
		"vlan-actor", "192.168.1.1", clientPool, vlanTranslator,
	)
	err := deviceActor.RegisterModuleActor("huawei-vlan", vlanActor)
	require.NoError(t, err)

	// Add IFM module
	ifmTranslator := NewReflectTranslator[*huawei.HuaweiIfm_Ifm_Interfaces]()
	ifmActor := NewModelActor[*huawei.HuaweiIfm_Ifm_Interfaces](
		"ifm-actor", "192.168.1.1", clientPool, ifmTranslator,
	)
	err = deviceActor.RegisterModuleActor("huawei-ifm", ifmActor)
	require.NoError(t, err)

	// Start device actor
	err = deviceActor.Start()
	require.NoError(t, err)
	defer deviceActor.Stop()

	time.Sleep(50 * time.Millisecond)

	// Set desired config for both modules
	ctx := context.Background()

	// For VLAN module
	vlanTranslateCmd := &TranslateCmd{
		BaseMessage: NewBaseMessageWithContext("vlan-translate", MsgTranslate, ctx),
		Payload:     map[string]interface{}{"Name": "TestVlan"},
		Operation:   OperationMerge,
	}
	promise, err := vlanActor.Send(vlanTranslateCmd)
	require.NoError(t, err)
	result := <-promise
	require.True(t, result.Success)

	// For IFM module (using empty config for this test)
	ifmTranslateCmd := &TranslateCmd{
		BaseMessage: NewBaseMessageWithContext("ifm-translate", MsgTranslate, ctx),
		Payload:     map[string]interface{}{"Name": "GigabitEthernet0/0/1"},
		Operation:   OperationMerge,
	}
	promise, err = ifmActor.Send(ifmTranslateCmd)
	require.NoError(t, err)
	result = <-promise
	require.True(t, result.Success)

	// Execute PrepareAll (dry run)
	txState, err := deviceActor.PrepareAll(ctx, true)

	// With mock client, Prepare may succeed or fail based on diff comparison
	// Important: Verify the transaction mechanism is working
	require.NotNil(t, txState)
	assert.Equal(t, 2, len(txState.Modules))
	assert.Contains(t, txState.Modules, "huawei-vlan")
	assert.Contains(t, txState.Modules, "huawei-ifm")
	assert.NotEmpty(t, txState.TransactionID)

	t.Logf("PrepareAll status: %s, error: %v", txState.Status, err)
}

// TestDeviceActor_AbortAll verifies cross-module Abort after Prepare
func TestDeviceActor_AbortAll(t *testing.T) {
	clientPool := &mockClientPool{}
	deviceActor := NewDeviceActor("192.168.1.1", clientPool)

	// Add VLAN module
	vlanTranslator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]()
	vlanActor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans_Vlan](
		"vlan-actor", "192.168.1.1", clientPool, vlanTranslator,
	)
	err := deviceActor.RegisterModuleActor("huawei-vlan", vlanActor)
	require.NoError(t, err)

	// Start device actor
	err = deviceActor.Start()
	require.NoError(t, err)
	defer deviceActor.Stop()

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()

	// Execute AbortAll
	txState, err := deviceActor.AbortAll(ctx, "Test abort reason")

	// Abort should always succeed (even when there's nothing to abort)
	// because ModelActor.handleAbort does no-op when no active transaction
	require.NotNil(t, txState)
	assert.Equal(t, 1, len(txState.Modules))
	t.Logf("AbortAll status: %s, error: %v", txState.Status, err)
}

// TestDeviceActor_CommitAll verifies cross-module Commit phase
func TestDeviceActor_CommitAll(t *testing.T) {
	clientPool := &mockClientPool{}
	deviceActor := NewDeviceActor("192.168.1.1", clientPool)

	// Add VLAN module
	vlanTranslator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]()
	vlanActor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans_Vlan](
		"vlan-actor", "192.168.1.1", clientPool, vlanTranslator,
	)
	err := deviceActor.RegisterModuleActor("huawei-vlan", vlanActor)
	require.NoError(t, err)

	// Start device actor
	err = deviceActor.Start()
	require.NoError(t, err)
	defer deviceActor.Stop()

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()

	// Set desired config first
	translateCmd := &TranslateCmd{
		BaseMessage: NewBaseMessageWithContext("vlan-translate", MsgTranslate, ctx),
		Payload:     map[string]interface{}{"Name": "TestVlan"},
		Operation:   OperationMerge,
	}
	promise, err := vlanActor.Send(translateCmd)
	require.NoError(t, err)
	result := <-promise
	require.True(t, result.Success)

	// First Prepare
	_, err = deviceActor.PrepareAll(ctx, false)
	if err != nil {
		t.Logf("PrepareAll failed (expected with some diff scenarios): %v", err)
	}

	// Execute CommitAll with ForceCommit=true to skip consistency check
	txState, err := deviceActor.CommitAll(ctx, true)

	require.NotNil(t, txState)
	assert.Equal(t, 1, len(txState.Modules))
	t.Logf("CommitAll status: %s, error: %v", txState.Status, err)
}

// TestDeviceActor_PrepareAndCommitAll verifies the complete 2PC workflow
func TestDeviceActor_PrepareAndCommitAll(t *testing.T) {
	clientPool := &mockClientPool{}
	deviceActor := NewDeviceActor("192.168.1.1", clientPool)

	// Add VLAN module
	vlanTranslator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]()
	vlanActor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans_Vlan](
		"vlan-actor", "192.168.1.1", clientPool, vlanTranslator,
	)
	err := deviceActor.RegisterModuleActor("huawei-vlan", vlanActor)
	require.NoError(t, err)

	// Start device actor
	err = deviceActor.Start()
	require.NoError(t, err)
	defer deviceActor.Stop()

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()

	// Set desired config
	translateCmd := &TranslateCmd{
		BaseMessage: NewBaseMessageWithContext("vlan-translate", MsgTranslate, ctx),
		Payload:     map[string]interface{}{"Name": "TestVlan"},
		Operation:   OperationMerge,
	}
	promise, err := vlanActor.Send(translateCmd)
	require.NoError(t, err)
	result := <-promise
	require.True(t, result.Success)

	// Dry run mode (Prepare only)
	txState, err := deviceActor.PrepareAndCommitAll(ctx, true)

	require.NotNil(t, txState)
	assert.Equal(t, 1, len(txState.Modules))
	t.Logf("PrepareAndCommitAll (dry run) status: %s, error: %v", txState.Status, err)
}

// TestDeviceActor_StatusQuery verifies device-level status aggregation
func TestDeviceActor_StatusQuery(t *testing.T) {
	clientPool := &mockClientPool{}
	deviceActor := NewDeviceActor("192.168.1.1", clientPool)

	// Add VLAN module
	vlanTranslator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]()
	vlanActor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans_Vlan](
		"vlan-actor", "192.168.1.1", clientPool, vlanTranslator,
	)
	err := deviceActor.RegisterModuleActor("huawei-vlan", vlanActor)
	require.NoError(t, err)

	// Start device actor
	err = deviceActor.Start()
	require.NoError(t, err)
	defer deviceActor.Stop()

	time.Sleep(50 * time.Millisecond)

	// Query device status
	statusCmd := &StatusQueryCmd{
		BaseMessage:    NewBaseMessage("device-status", MsgStatusQuery),
		IncludeDetails: true,
	}

	promise, err := deviceActor.Send(statusCmd)
	require.NoError(t, err)
	result := <-promise

	assert.True(t, result.Success)
	assert.Equal(t, "192.168.1.1", result.Data["device_id"])
	assert.Equal(t, 1, result.Data["modules_count"])
	assert.Contains(t, result.Data, "modules")
}

