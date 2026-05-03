package actor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
)

func TestTwoPC_DryRunPrepare(t *testing.T) {
	translator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]()
	clientPool := &mockClientPool{}

	actor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]("actor-1", "192.168.1.1", clientPool, translator)
	err := actor.Start()
	require.NoError(t, err)
	defer actor.Stop()

	time.Sleep(50 * time.Millisecond)

	// First translate to set desired state
	translateCmd := &TranslateCmd{
		BaseMessage: NewBaseMessage("translate-1", MsgTranslate),
		Payload: map[string]interface{}{
			"Name":        "TestVlan",
			"Description": "Test Description",
		},
		Operation: OperationMerge,
	}

	promise, err := actor.Send(translateCmd)
	require.NoError(t, err)
	result := <-promise
	require.True(t, result.Success)

	// Dry run prepare - computes diff but does not apply
	prepareCmd := &PrepareCmd{
		BaseMessage: NewBaseMessage("prepare-dryrun", MsgPrepare),
		DryRun:      true,
	}

	promise, err = actor.Send(prepareCmd)
	require.NoError(t, err)
	result = <-promise

	assert.True(t, result.Success)
	assert.NoError(t, result.Error)
	assert.Contains(t, result.Data, "dry_run")
	assert.Equal(t, true, result.Data["dry_run"])
	assert.Contains(t, result.Data, "can_commit")
}

func TestTwoPC_PrepareWithNoChanges(t *testing.T) {
	translator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]()
	clientPool := &mockClientPool{}

	actor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]("actor-2", "192.168.1.1", clientPool, translator)
	err := actor.Start()
	require.NoError(t, err)
	defer actor.Stop()

	time.Sleep(50 * time.Millisecond)

	// Set desired to match the current actual state (empty struct)
	// This should result in "no changes needed" during prepare
	prepareCmd := &PrepareCmd{
		BaseMessage: NewBaseMessage("prepare-nochange", MsgPrepare),
		DryRun:      true,
	}

	promise, err := actor.Send(prepareCmd)
	require.NoError(t, err)
	result := <-promise

	assert.True(t, result.Success)
	assert.Contains(t, result.Data, "can_commit")
	assert.Equal(t, false, result.Data["can_commit"])
}

func TestTwoPC_AbortTransaction(t *testing.T) {
	translator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]()
	clientPool := &mockClientPool{}

	actor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]("actor-3", "192.168.1.1", clientPool, translator)
	err := actor.Start()
	require.NoError(t, err)
	defer actor.Stop()

	time.Sleep(50 * time.Millisecond)

	// First translate to set desired state
	translateCmd := &TranslateCmd{
		BaseMessage: NewBaseMessage("translate-1", MsgTranslate),
		Payload: map[string]interface{}{
			"Name":        "TestVlan",
			"Description": "Test Description",
		},
		Operation: OperationMerge,
	}

	promise, err := actor.Send(translateCmd)
	require.NoError(t, err)
	result := <-promise
	require.True(t, result.Success)

	// Prepare (non-dryrun) to create an active transaction
	prepareCmd := &PrepareCmd{
		BaseMessage: NewBaseMessage("prepare-1", MsgPrepare),
		DryRun:      false,
	}

	promise, err = actor.Send(prepareCmd)
	require.NoError(t, err)
	result = <-promise
	require.True(t, result.Success)
	assert.True(t, result.Data["can_commit"].(bool))

	// Abort the active transaction
	abortCmd := &AbortCmd{
		BaseMessage: NewBaseMessageWithContext("abort-1", MsgAbort, context.Background()),
		Reason:      "User requested abort",
	}

	promise, err = actor.Send(abortCmd)
	require.NoError(t, err)
	result = <-promise

	assert.True(t, result.Success)
	assert.NoError(t, result.Error)
	assert.Contains(t, result.Data, "message")
	assert.Contains(t, result.Data, "reason")
	assert.Equal(t, "User requested abort", result.Data["reason"])
}

func TestTwoPC_MessageTypes(t *testing.T) {
	// Test that all 2PC message types implement Message interface
	var msg Message

	msg = &AbortCmd{
		BaseMessage: NewBaseMessage("test-abort", MsgAbort),
		Reason:      "test",
	}
	assert.Equal(t, MsgAbort, msg.Type())
	assert.Equal(t, "test-abort", msg.ID())
	assert.NotNil(t, msg.Context())

	msg = &PrepareCmd{
		BaseMessage: NewBaseMessage("test-prepare", MsgPrepare),
		DryRun:      true,
	}
	assert.Equal(t, MsgPrepare, msg.Type())
	assert.Equal(t, "test-prepare", msg.ID())

	msg = &CommitCmd{
		BaseMessage: NewBaseMessage("test-commit", MsgCommit),
		ForceCommit: true,
	}
	assert.Equal(t, MsgCommit, msg.Type())
	assert.Equal(t, "test-commit", msg.ID())
}

func TestTwoPC_MessageContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	msg := &PrepareCmd{
		BaseMessage: NewBaseMessageWithContext("test-ctx", MsgPrepare, ctx),
	}

	assert.Equal(t, ctx, msg.Context())
	assert.Equal(t, "test-ctx", msg.ID())
}

// TestTwoPC_CommitWithoutPrepare verifies that Commit fails if Prepare was not called first
func TestTwoPC_CommitWithoutPrepare(t *testing.T) {
	translator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]()
	clientPool := &mockClientPool{}

	actor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]("actor-commit-noprep", "192.168.1.1", clientPool, translator)
	err := actor.Start()
	require.NoError(t, err)
	defer actor.Stop()

	time.Sleep(50 * time.Millisecond)

	// Try to commit without calling Prepare first
	commitCmd := &CommitCmd{
		BaseMessage: NewBaseMessage("commit-1", MsgCommit),
		ForceCommit: false,
	}

	promise, err := actor.Send(commitCmd)
	require.NoError(t, err)
	result := <-promise

	// Should fail with "no active transaction" error
	assert.False(t, result.Success)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "no active transaction")
}

// TestTwoPC_PrepareThenCommit verifies the complete Prepare -> Commit workflow
// Note: mock client doesn't persist config, so we use ForceCommit to skip consistency verification
func TestTwoPC_PrepareThenCommit(t *testing.T) {
	translator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]()
	clientPool := &mockClientPool{}

	actor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]("actor-full-2pc", "192.168.1.1", clientPool, translator)
	err := actor.Start()
	require.NoError(t, err)
	defer actor.Stop()

	time.Sleep(50 * time.Millisecond)

	// First translate to set desired state
	translateCmd := &TranslateCmd{
		BaseMessage: NewBaseMessage("translate-1", MsgTranslate),
		Payload: map[string]interface{}{
			"Name":        "TestVlan",
			"Description": "Test Description",
		},
		Operation: OperationMerge,
	}

	promise, err := actor.Send(translateCmd)
	require.NoError(t, err)
	result := <-promise
	require.True(t, result.Success)

	// Prepare (non-dryrun) to create candidate config
	prepareCmd := &PrepareCmd{
		BaseMessage: NewBaseMessage("prepare-1", MsgPrepare),
		DryRun:      false,
	}

	promise, err = actor.Send(prepareCmd)
	require.NoError(t, err)
	result = <-promise
	require.True(t, result.Success)
	assert.True(t, result.Data["can_commit"].(bool))

	// Commit the prepared transaction with ForceCommit
	// (mock client doesn't persist config, so consistency check would fail)
	commitCmd := &CommitCmd{
		BaseMessage: NewBaseMessage("commit-1", MsgCommit),
		ForceCommit: true,
	}

	promise, err = actor.Send(commitCmd)
	require.NoError(t, err)
	result = <-promise

	// Commit should succeed even with consistency mismatch (ForceCommit bypasses failure)
	assert.True(t, result.Success)
	assert.NoError(t, result.Error)
	assert.Greater(t, result.Version, int64(0))
	assert.NotEmpty(t, result.Checksum)
	assert.Contains(t, result.Data, "message")
}

// TestTwoPC_SecondPrepareFails verifies that a second Prepare fails if first is still active
func TestTwoPC_SecondPrepareFails(t *testing.T) {
	translator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]()
	clientPool := &mockClientPool{}

	actor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]("actor-concurrent", "192.168.1.1", clientPool, translator)
	err := actor.Start()
	require.NoError(t, err)
	defer actor.Stop()

	time.Sleep(50 * time.Millisecond)

	// Translate desired state
	translateCmd := &TranslateCmd{
		BaseMessage: NewBaseMessage("translate-1", MsgTranslate),
		Payload:     map[string]interface{}{"Name": "TestVlan"},
		Operation:   OperationMerge,
	}

	promise, err := actor.Send(translateCmd)
	require.NoError(t, err)
	result := <-promise
	require.True(t, result.Success)

	// First Prepare (non-dryrun) creates active transaction
	prepareCmd1 := &PrepareCmd{
		BaseMessage: NewBaseMessage("prepare-1", MsgPrepare),
		DryRun:      false,
	}

	promise, err = actor.Send(prepareCmd1)
	require.NoError(t, err)
	result = <-promise
	require.True(t, result.Success)

	// Second Prepare should fail because transaction is already active
	prepareCmd2 := &PrepareCmd{
		BaseMessage: NewBaseMessage("prepare-2", MsgPrepare),
		DryRun:      false,
	}

	promise, err = actor.Send(prepareCmd2)
	require.NoError(t, err)
	result = <-promise

	// Second Prepare should fail with "transaction already active"
	assert.False(t, result.Success)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "transaction already active")
}

// TestTwoPC_AbortThenPrepare verifies that after Abort, a new Prepare can proceed
func TestTwoPC_AbortThenPrepare(t *testing.T) {
	translator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]()
	clientPool := &mockClientPool{}

	actor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]("actor-abort-then-prep", "192.168.1.1", clientPool, translator)
	err := actor.Start()
	require.NoError(t, err)
	defer actor.Stop()

	time.Sleep(50 * time.Millisecond)

	// Translate desired state
	translateCmd := &TranslateCmd{
		BaseMessage: NewBaseMessage("translate-1", MsgTranslate),
		Payload:     map[string]interface{}{"Name": "TestVlan"},
		Operation:   OperationMerge,
	}

	promise, err := actor.Send(translateCmd)
	require.NoError(t, err)
	result := <-promise
	require.True(t, result.Success)

	// First Prepare
	prepareCmd1 := &PrepareCmd{
		BaseMessage: NewBaseMessage("prepare-1", MsgPrepare),
		DryRun:      false,
	}

	promise, err = actor.Send(prepareCmd1)
	require.NoError(t, err)
	result = <-promise
	require.True(t, result.Success)

	// Abort the transaction
	abortCmd := &AbortCmd{
		BaseMessage: NewBaseMessage("abort-1", MsgAbort),
		Reason:      "Testing abort then prepare",
	}

	promise, err = actor.Send(abortCmd)
	require.NoError(t, err)
	result = <-promise
	require.True(t, result.Success)

	// Second Prepare should now succeed
	prepareCmd2 := &PrepareCmd{
		BaseMessage: NewBaseMessage("prepare-2", MsgPrepare),
		DryRun:      false,
	}

	promise, err = actor.Send(prepareCmd2)
	require.NoError(t, err)
	result = <-promise
	assert.True(t, result.Success)
	assert.True(t, result.Data["can_commit"].(bool))
}
