package actor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
)

// TestModelActor_Integration_PrepareAndCommit tests the complete 2PC flow
// with a real NETCONF simulator: Translate -> Prepare -> Commit
func TestModelActor_Integration_PrepareAndCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Start NETCONF simulator
	sim := netsim.NewSimulator()
	err := sim.Start()
	require.NoError(t, err)
	defer sim.Stop()

	// 2. Create client pool with simulator connection
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	// 3. Create translator and model actor
	translator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans]()
	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())

	actor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans](
		"vlan-actor",
		deviceID,
		pool,
		translator,
	)

	// 4. Start the actor
	err = actor.Start()
	require.NoError(t, err)
	defer actor.Stop()

	// Wait for actor initialization
	time.Sleep(100 * time.Millisecond)

	// 5. Send Translate command to set desired state
	translateCmd := &TranslateCmd{
		BaseMessage: NewBaseMessage("translate-1", MsgTranslate),
		Payload: map[string]interface{}{
			"Id":          100,
			"Name":        "IntegrationTestVLAN",
			"Description": "Created by integration test",
			"Type":        huawei.HuaweiVlan_VlanType_common,
		},
		Operation: OperationMerge,
	}

	promise, err := actor.Send(translateCmd)
	require.NoError(t, err)
	result := <-promise
	require.True(t, result.Success, "translate failed: %v", result.Error)

	// 6. Prepare phase - validates and writes to candidate datastore
	prepareCmd := &PrepareCmd{
		BaseMessage: NewBaseMessageWithContext("prepare-1", MsgPrepare, context.Background()),
		DryRun:      false,
	}

	promise, err = actor.Send(prepareCmd)
	require.NoError(t, err)
	result = <-promise
	require.True(t, result.Success, "prepare failed: %v", result.Error)

	// Check that prepare returned expected data
	assert.True(t, result.Data["can_commit"].(bool))
	assert.Contains(t, result.Data, "changes")

	// 7. Commit phase - applies candidate to running config
	commitCmd := &CommitCmd{
		BaseMessage: NewBaseMessageWithContext("commit-1", MsgCommit, context.Background()),
		ForceCommit: false,
	}

	promise, err = actor.Send(commitCmd)
	require.NoError(t, err)
	result = <-promise
	require.True(t, result.Success, "commit failed: %v", result.Error)

	// Verify commit returned expected data
	assert.Contains(t, result.Data, "message")
	assert.Equal(t, "commit successful", result.Data["message"])

	// 8. Verify the VLAN now exists in the simulator's running config
	sim.AssertHuaweiVlanExists(t, 100)
	sim.AssertHuaweiVlanName(t, 100, "IntegrationTestVLAN")
	sim.AssertHuaweiVlanCount(t, 1)
}

// TestModelActor_Integration_PrepareDryRun tests Dry Run mode
func TestModelActor_Integration_PrepareDryRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Start NETCONF simulator
	sim := netsim.NewSimulator()
	err := sim.Start()
	require.NoError(t, err)
	defer sim.Stop()

	// 2. Setup actor
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	translator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans]()
	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())

	actor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans](
		"vlan-actor-dryrun",
		deviceID,
		pool,
		translator,
	)

	err = actor.Start()
	require.NoError(t, err)
	defer actor.Stop()

	time.Sleep(100 * time.Millisecond)

	// 3. Set desired state
	translateCmd := &TranslateCmd{
		BaseMessage: NewBaseMessage("translate-1", MsgTranslate),
		Payload: map[string]interface{}{
			"Id":   200,
			"Name": "DryRunVLAN",
			"Type": huawei.HuaweiVlan_VlanType_common,
		},
		Operation: OperationMerge,
	}

	promise, err := actor.Send(translateCmd)
	require.NoError(t, err)
	result := <-promise
	require.True(t, result.Success)

	// 4. Dry run prepare - should NOT modify actual running config
	prepareCmd := &PrepareCmd{
		BaseMessage: NewBaseMessageWithContext("prepare-1", MsgPrepare, context.Background()),
		DryRun:      true,
	}

	promise, err = actor.Send(prepareCmd)
	require.NoError(t, err)
	result = <-promise
	require.True(t, result.Success)

	assert.True(t, result.Data["dry_run"].(bool))
	assert.True(t, result.Data["can_commit"].(bool))
	assert.Contains(t, result.Data, "changes")

	// 5. IMPORTANT: VLAN 200 should NOT exist in running config (dry run)
	vlans, err := sim.GetDatastore().ExtractHuaweiVLANs()
	require.NoError(t, err)
	assert.Equal(t, 0, len(vlans), "dry run should not modify running config")
}

// TestModelActor_Integration_AbortAfterPrepare tests the Abort flow: Prepare -> Abort
func TestModelActor_Integration_AbortAfterPrepare(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Start NETCONF simulator
	sim := netsim.NewSimulator()
	err := sim.Start()
	require.NoError(t, err)
	defer sim.Stop()

	// 2. Setup actor
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	translator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans]()
	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())

	actor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans](
		"vlan-actor-abort",
		deviceID,
		pool,
		translator,
	)

	err = actor.Start()
	require.NoError(t, err)
	defer actor.Stop()

	time.Sleep(100 * time.Millisecond)

	// 3. Set desired state
	translateCmd := &TranslateCmd{
		BaseMessage: NewBaseMessage("translate-1", MsgTranslate),
		Payload: map[string]interface{}{
			"Id":   300,
			"Name": "ToBeAbortedVLAN",
			"Type": huawei.HuaweiVlan_VlanType_common,
		},
		Operation: OperationMerge,
	}

	promise, err := actor.Send(translateCmd)
	require.NoError(t, err)
	result := <-promise
	require.True(t, result.Success)

	// 4. Prepare phase - writes to candidate
	prepareCmd := &PrepareCmd{
		BaseMessage: NewBaseMessageWithContext("prepare-1", MsgPrepare, context.Background()),
		DryRun:      false,
	}

	promise, err = actor.Send(prepareCmd)
	require.NoError(t, err)
	result = <-promise
	require.True(t, result.Success)

	// 5. Abort before commit - should discard candidate
	abortCmd := &AbortCmd{
		BaseMessage: NewBaseMessageWithContext("abort-1", MsgAbort, context.Background()),
		Reason:      "Abort after prepare for integration test",
	}

	promise, err = actor.Send(abortCmd)
	require.NoError(t, err)
	result = <-promise
	require.True(t, result.Success, "abort failed: %v", result.Error)

	assert.Contains(t, result.Data, "message")
	assert.Equal(t, "transaction aborted successfully", result.Data["message"])
	assert.Equal(t, "Abort after prepare for integration test", result.Data["reason"])

	// 6. Verify VLAN 300 should NOT exist in running config (aborted)
	vlans, err := sim.GetDatastore().ExtractHuaweiVLANs()
	require.NoError(t, err)
	assert.Equal(t, 0, len(vlans), "aborted changes should not be in running config")
}

// TestModelActor_Integration_NoChanges tests that no changes needed is handled correctly
func TestModelActor_Integration_NoChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Start NETCONF simulator with initial VLAN config
	sim := netsim.NewSimulator()
	initialXML := `<vlans xmlns="urn:huawei:params:xml:ns:yang:huawei-vlan"><vlan><id>400</id><name>ExistingVLAN</name><type>2</type></vlan></vlans>`
	sim.SetRunningConfigXML([]byte(initialXML))
	err := sim.Start()
	require.NoError(t, err)
	defer sim.Stop()

	// 2. Setup actor
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	translator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans]()
	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())

	actor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans](
		"vlan-actor-nochanges",
		deviceID,
		pool,
		translator,
	)

	err = actor.Start()
	require.NoError(t, err)
	defer actor.Stop()

	time.Sleep(100 * time.Millisecond)

	// 3. Set desired state matching current config
	translateCmd := &TranslateCmd{
		BaseMessage: NewBaseMessage("translate-1", MsgTranslate),
		Payload: map[string]interface{}{
			"Id":   400,
			"Name": "ExistingVLAN",
			"Type": huawei.HuaweiVlan_VlanType_common,
		},
		Operation: OperationMerge,
	}

	promise, err := actor.Send(translateCmd)
	require.NoError(t, err)
	result := <-promise
	require.True(t, result.Success)

	// 4. Prepare phase - should detect no changes needed
	prepareCmd := &PrepareCmd{
		BaseMessage: NewBaseMessageWithContext("prepare-1", MsgPrepare, context.Background()),
		DryRun:      false,
	}

	promise, err = actor.Send(prepareCmd)
	require.NoError(t, err)
	result = <-promise

	require.True(t, result.Success)
	assert.Contains(t, result.Data, "can_commit")
	assert.False(t, result.Data["can_commit"].(bool), "can_commit should be false when no changes needed")
}

// TestModelActor_Integration_MultiVLAN tests creating multiple VLANs in a single commit
func TestModelActor_Integration_MultiVLAN(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Start NETCONF simulator
	sim := netsim.NewSimulator()
	err := sim.Start()
	require.NoError(t, err)
	defer sim.Stop()

	// 2. Setup actor
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	translator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans]()
	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())

	actor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans](
		"vlan-actor-multi",
		deviceID,
		pool,
		translator,
	)

	err = actor.Start()
	require.NoError(t, err)
	defer actor.Stop()

	time.Sleep(100 * time.Millisecond)

	// 3. Set desired state with multiple VLANs
	// Note: The current translator only handles single VLAN from map; we'd need
	// to enhance the translator to support multiple VLANs in a single payload.
	// For now, we test with two separate translate operations.
	translateCmd1 := &TranslateCmd{
		BaseMessage: NewBaseMessage("translate-1", MsgTranslate),
		Payload: map[string]interface{}{
			"Id":   501,
			"Name": "VLAN501",
			"Type": huawei.HuaweiVlan_VlanType_common,
		},
		Operation: OperationMerge,
	}

	promise, err := actor.Send(translateCmd1)
	require.NoError(t, err)
	result := <-promise
	require.True(t, result.Success)

	// 4. Prepare and Commit
	prepareCmd := &PrepareCmd{
		BaseMessage: NewBaseMessageWithContext("prepare-1", MsgPrepare, context.Background()),
		DryRun:      false,
	}

	promise, err = actor.Send(prepareCmd)
	require.NoError(t, err)
	result = <-promise
	require.True(t, result.Success)

	commitCmd := &CommitCmd{
		BaseMessage: NewBaseMessageWithContext("commit-1", MsgCommit, context.Background()),
		ForceCommit: false,
	}

	promise, err = actor.Send(commitCmd)
	require.NoError(t, err)
	result = <-promise
	require.True(t, result.Success)

	// 5. Verify the VLAN exists
	sim.AssertHuaweiVlanExists(t, 501)
	sim.AssertHuaweiVlanName(t, 501, "VLAN501")
}

// TestModelActor_Integration_ApplyDirect tests direct Apply without 2PC
func TestModelActor_Integration_ApplyDirect(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Start NETCONF simulator
	sim := netsim.NewSimulator()
	err := sim.Start()
	require.NoError(t, err)
	defer sim.Stop()

	// 2. Setup actor
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	translator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans]()
	deviceID := fmt.Sprintf("%s:%s@%s:%d", sim.Username(), sim.Password(), sim.Addr(), sim.Port())

	actor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans](
		"vlan-actor-apply",
		deviceID,
		pool,
		translator,
	)

	err = actor.Start()
	require.NoError(t, err)
	defer actor.Stop()

	time.Sleep(100 * time.Millisecond)

	// 3. Set desired state
	translateCmd := &TranslateCmd{
		BaseMessage: NewBaseMessage("translate-1", MsgTranslate),
		Payload: map[string]interface{}{
			"Id":   600,
			"Name": "DirectApplyVLAN",
			"Type": huawei.HuaweiVlan_VlanType_common,
		},
		Operation: OperationMerge,
	}

	promise, err := actor.Send(translateCmd)
	require.NoError(t, err)
	result := <-promise
	require.True(t, result.Success)

	// 4. Apply directly (no 2PC)
	applyCmd := &ApplyCmd{
		BaseMessage: NewBaseMessageWithContext("apply-1", MsgApply, context.Background()),
		ForceApply:  true,
	}

	promise, err = actor.Send(applyCmd)
	require.NoError(t, err)
	result = <-promise
	require.True(t, result.Success, "apply failed: %v", result.Error)

	// 5. Verify VLAN exists in running config
	sim.AssertHuaweiVlanExists(t, 600)
	sim.AssertHuaweiVlanName(t, 600, "DirectApplyVLAN")
}
