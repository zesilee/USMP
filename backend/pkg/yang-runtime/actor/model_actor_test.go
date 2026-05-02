package actor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
)

func TestNewModelActor(t *testing.T) {
	translator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]()
	clientPool := &mockClientPool{}

	actor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]("actor-1", "device-1", clientPool, translator)

	assert.NotNil(t, actor)
	assert.Equal(t, "actor-1", actor.actorID)
	assert.Equal(t, "device-1", actor.deviceID)
}

func TestActorStartStop(t *testing.T) {
	translator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]()
	clientPool := &mockClientPool{}

	actor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]("actor-1", "device-1", clientPool, translator)

	err := actor.Start()
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	status := actor.Status()
	assert.Equal(t, StatusReady, status.Status)

	err = actor.Stop()
	require.NoError(t, err)

	_, err = actor.Send(&TranslateCmd{})
	assert.Error(t, err)
}

func TestHandleTranslate(t *testing.T) {
	translator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]()
	clientPool := &mockClientPool{}

	actor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]("actor-1", "device-1", clientPool, translator)
	err := actor.Start()
	require.NoError(t, err)
	defer actor.Stop()

	desc := "Test VLAN"
	name := "Vlan100"
	payload := map[string]interface{}{
		"Description": &desc,
		"Name":        &name,
	}

	cmd := &TranslateCmd{
		BaseMessage: NewBaseMessage("test-translate-1", MsgTranslate),
		Path:        "",
		Payload:     payload,
		Operation:   OperationMerge,
	}

	promise, err := actor.Send(cmd)
	require.NoError(t, err)

	result := <-promise
	assert.True(t, result.Success)
	assert.NoError(t, result.Error)
	assert.Equal(t, int64(1), result.Version)
	assert.NotEmpty(t, result.Checksum)
}

func TestHandleValidate(t *testing.T) {
	translator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]()
	clientPool := &mockClientPool{}

	actor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]("actor-1", "device-1", clientPool, translator)
	err := actor.Start()
	require.NoError(t, err)
	defer actor.Stop()

	name := "Vlan100"
	payload := map[string]interface{}{
		"Name": &name,
	}
	translateCmd := &TranslateCmd{
		BaseMessage: NewBaseMessage("test-translate-1", MsgTranslate),
		Payload:     payload,
		Operation:   OperationMerge,
	}

	promise, _ := actor.Send(translateCmd)
	<-promise

	validateCmd := &ValidateCmd{
		BaseMessage: NewBaseMessage("test-validate-1", MsgValidate),
	}

	promise, err = actor.Send(validateCmd)
	require.NoError(t, err)

	result := <-promise
	assert.True(t, result.Success)
	assert.NoError(t, result.Error)
}

func TestHandleRollback(t *testing.T) {
	translator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]()
	clientPool := &mockClientPool{}

	actor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]("actor-1", "device-1", clientPool, translator)
	err := actor.Start()
	require.NoError(t, err)
	defer actor.Stop()

	name1 := "version1"
	cmd1 := &TranslateCmd{
		BaseMessage: NewBaseMessage("t1", MsgTranslate),
		Payload:     map[string]interface{}{"Name": &name1},
		Operation:   OperationMerge,
	}
	promise, _ := actor.Send(cmd1)
	result1 := <-promise
	assert.Equal(t, int64(1), result1.Version)
	version1Checksum := result1.Checksum

	name2 := "version2"
	cmd2 := &TranslateCmd{
		BaseMessage: NewBaseMessage("t2", MsgTranslate),
		Payload:     map[string]interface{}{"Name": &name2},
		Operation:   OperationMerge,
	}
	promise, _ = actor.Send(cmd2)
	result2 := <-promise
	assert.Equal(t, int64(2), result2.Version)

	rollbackCmd := &RollbackCmd{
		BaseMessage:   NewBaseMessage("rollback-1", MsgRollback),
		TargetVersion: 1,
	}

	promise, err = actor.Send(rollbackCmd)
	require.NoError(t, err)

	rollbackResult := <-promise
	assert.True(t, rollbackResult.Success)
	assert.Equal(t, int64(3), rollbackResult.Version)
	assert.Equal(t, version1Checksum, rollbackResult.Checksum)
}

func TestHandleStatusQuery(t *testing.T) {
	translator := NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]()
	clientPool := &mockClientPool{}

	actor := NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans_Vlan]("actor-1", "device-1", clientPool, translator)
	err := actor.Start()
	require.NoError(t, err)
	defer actor.Stop()

	cmd1 := &StatusQueryCmd{
		BaseMessage:    NewBaseMessage("status-1", MsgStatusQuery),
		IncludeDetails: false,
	}

	promise, _ := actor.Send(cmd1)
	result1 := <-promise
	assert.True(t, result1.Success)
	assert.Equal(t, "actor-1", result1.Data["actor_id"])
	assert.Equal(t, "device-1", result1.Data["device_id"])

	cmd2 := &StatusQueryCmd{
		BaseMessage:    NewBaseMessage("status-2", MsgStatusQuery),
		IncludeDetails: true,
	}

	promise, _ = actor.Send(cmd2)
	result2 := <-promise
	assert.True(t, result2.Success)
	assert.Contains(t, result2.Data, "history_count")
	assert.Contains(t, result2.Data, "last_activity")
}

type mockClientPool struct{}

func (p *mockClientPool) Get(info client.DeviceConnectionInfo) (client.Client, error) {
	return &mockClient{}, nil
}

func (p *mockClientPool) Release(ip string) {}

func (p *mockClientPool) CloseAll() error { return nil }

func (p *mockClientPool) Stats() client.PoolStats {
	return client.PoolStats{}
}

type mockClient struct{}

func (c *mockClient) Get(ctx context.Context, path string, opts ...client.GetOption) (*client.GetResult, error) {
	return &client.GetResult{
		Path:      path,
		Data:      &huawei.HuaweiVlan_Vlan_Vlans_Vlan{},
		Timestamp: time.Now(),
	}, nil
}

func (c *mockClient) Set(ctx context.Context, changes []client.Change, opts ...client.SetOption) (*client.SetResult, error) {
	return &client.SetResult{
		Success:   true,
		Message:   "OK",
		Timestamp: time.Now(),
	}, nil
}

func (c *mockClient) Subscribe(ctx context.Context, path string, handler func(client.Notification)) error {
	return nil
}

func (c *mockClient) Close() error { return nil }

func (c *mockClient) IsConnected() bool { return true }

func (c *mockClient) DiscardCandidate(ctx context.Context) error { return nil }
