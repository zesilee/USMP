package vlan

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/openconfig"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockClient is a mock implementation of client.Client
type MockClient struct {
	mock.Mock
}

func (m *MockClient) Get(ctx context.Context, path string, opts ...client.GetOption) (*client.GetResult, error) {
	args := m.Called(ctx, path, opts)
	if res, ok := args.Get(0).(*client.GetResult); ok {
		return res, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockClient) Set(ctx context.Context, changes []client.Change, opts ...client.SetOption) (*client.SetResult, error) {
	args := m.Called(ctx, changes, opts)
	if res, ok := args.Get(0).(*client.SetResult); ok {
		return res, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockClient) Subscribe(ctx context.Context, path string, handler func(client.Notification)) error {
	args := m.Called(ctx, path, handler)
	return args.Error(0)
}

func (m *MockClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockClient) IsConnected() bool {
	args := m.Called()
	return args.Bool(0)
}

// MockClientPool is a mock implementation of client.ClientPool
type MockClientPool struct {
	mock.Mock
}

func (m *MockClientPool) Get(info client.DeviceConnectionInfo) (client.Client, error) {
	args := m.Called(info)
	if c, ok := args.Get(0).(client.Client); ok {
		return c, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockClientPool) Release(ip string) {
	m.Called(ip)
}

func (m *MockClientPool) CloseAll() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockClientPool) Stats() client.PoolStats {
	args := m.Called()
	return args.Get(0).(client.PoolStats)
}

func TestDeviceClient_Get_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	deviceID := "192.168.1.1"

	// Create test JSON response with one VLAN
	deviceRoot := &openconfig.Device{
		Vlans: &openconfig.OpenconfigVlan_Vlans{
			Vlan: map[uint16]*openconfig.OpenconfigVlan_Vlans_Vlan{
				100: {
					VlanId: ptrUint16(100),
					Config: &openconfig.OpenconfigVlan_Vlans_Vlan_Config{
						Name: ptrString("VLAN100"),
					},
				},
			},
		},
	}

	jsonBytes, err := json.Marshal(deviceRoot)
	assert.NoError(t, err)

	mockClient := new(MockClient)
	mockClient.On("Get", ctx, "/vlans", mock.Anything).Return(&client.GetResult{
		Data: jsonBytes,
	}, nil)

	mockPool := new(MockClientPool)
	mockPool.On("Get", client.DeviceConnectionInfo{IP: deviceID}).Return(mockClient, nil)

	dc := &deviceClient{
		clientPool: mockPool,
	}

	// Act
	result, err := dc.Get(ctx, deviceID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)

	vlans, ok := result.(*openconfig.OpenconfigVlan_Vlans)
	assert.True(t, ok)
	assert.NotNil(t, vlans)
	assert.Contains(t, vlans.Vlan, uint16(100))
	vlan := vlans.Vlan[100]
	assert.NotNil(t, vlan)
	assert.Equal(t, uint16(100), *vlan.VlanId)
	assert.Equal(t, "VLAN100", *vlan.Config.Name)

	mockPool.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

func TestDeviceClient_Get_ClientPoolGetError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	deviceID := "192.168.1.1"

	mockPool := new(MockClientPool)
	mockPool.On("Get", client.DeviceConnectionInfo{IP: deviceID}).Return(nil, assert.AnError)

	dc := &deviceClient{
		clientPool: mockPool,
	}

	// Act
	result, err := dc.Get(ctx, deviceID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, assert.AnError, err)

	mockPool.AssertExpectations(t)
}

func TestDeviceClient_Get_ClientGetError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	deviceID := "192.168.1.1"

	mockClient := new(MockClient)
	mockClient.On("Get", ctx, "/vlans", mock.Anything).Return(nil, assert.AnError)

	mockPool := new(MockClientPool)
	mockPool.On("Get", client.DeviceConnectionInfo{IP: deviceID}).Return(mockClient, nil)

	dc := &deviceClient{
		clientPool: mockPool,
	}

	// Act
	result, err := dc.Get(ctx, deviceID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, assert.AnError, err)

	mockPool.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

func TestDeviceClient_Get_InvalidJSON(t *testing.T) {
	// Arrange
	ctx := context.Background()
	deviceID := "192.168.1.1"

	invalidJSON := []byte(`{"invalid": json}`)

	mockClient := new(MockClient)
	mockClient.On("Get", ctx, "/vlans", mock.Anything).Return(&client.GetResult{
		Data: invalidJSON,
	}, nil)

	mockPool := new(MockClientPool)
	mockPool.On("Get", client.DeviceConnectionInfo{IP: deviceID}).Return(mockClient, nil)

	dc := &deviceClient{
		clientPool: mockPool,
	}

	// Act
	result, err := dc.Get(ctx, deviceID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)

	mockPool.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

func TestDeviceClient_Get_AlreadyUnmarshaledDeviceRoot(t *testing.T) {
	// Arrange
	ctx := context.Background()
	deviceID := "192.168.1.1"

	deviceRoot := &openconfig.Device{
		Vlans: &openconfig.OpenconfigVlan_Vlans{
			Vlan: map[uint16]*openconfig.OpenconfigVlan_Vlans_Vlan{
				200: {
					VlanId: ptrUint16(200),
				},
			},
		},
	}

	mockClient := new(MockClient)
	mockClient.On("Get", ctx, "/vlans", mock.Anything).Return(&client.GetResult{
		Data: deviceRoot,
	}, nil)

	mockPool := new(MockClientPool)
	mockPool.On("Get", client.DeviceConnectionInfo{IP: deviceID}).Return(mockClient, nil)

	dc := &deviceClient{
		clientPool: mockPool,
	}

	// Act
	result, err := dc.Get(ctx, deviceID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)

	vlans, ok := result.(*openconfig.OpenconfigVlan_Vlans)
	assert.True(t, ok)
	assert.Contains(t, vlans.Vlan, uint16(200))

	mockPool.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

func TestDeviceClient_Set_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	deviceID := "192.168.1.1"
	changes := []reconcile.Change{
		{
			Path:         "/vlans/vlan[100]",
			Type:         "ADD",
			DesiredValue: map[string]interface{}{"name": "VLAN100"},
		},
	}

	mockClient := new(MockClient)
	mockClient.On("Set", ctx, mock.AnythingOfType("[]client.Change"), mock.Anything).Return(&client.SetResult{
		Success: true,
	}, nil)

	mockPool := new(MockClientPool)
	mockPool.On("Get", client.DeviceConnectionInfo{IP: deviceID}).Return(mockClient, nil)

	dc := &deviceClient{
		clientPool: mockPool,
	}

	// Act
	err := dc.Set(ctx, deviceID, changes)

	// Assert
	assert.NoError(t, err)

	mockPool.AssertExpectations(t)
	mockClient.AssertExpectations(t)

	// Check that changes were converted correctly
	mockClient.AssertNumberOfCalls(t, "Set", 1)
	call := mockClient.Calls[0]
	clientChanges := call.Arguments[1].([]client.Change)
	assert.Len(t, clientChanges, 1)
	assert.Equal(t, client.AddChange, clientChanges[0].Type)
	assert.Equal(t, "/vlans/vlan[100]", clientChanges[0].Path)
}

func TestDeviceClient_Set_ClientPoolGetError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	deviceID := "192.168.1.1"
	changes := []reconcile.Change{}

	mockPool := new(MockClientPool)
	mockPool.On("Get", client.DeviceConnectionInfo{IP: deviceID}).Return(nil, assert.AnError)

	dc := &deviceClient{
		clientPool: mockPool,
	}

	// Act
	err := dc.Set(ctx, deviceID, changes)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)

	mockPool.AssertExpectations(t)
}

func TestVlanReconciler_FullReconcile(t *testing.T) {
	// Arrange
	ctx := context.Background()
	deviceID := "192.168.1.1"
	req := reconcile.Request{
		DeviceID: deviceID,
		Path:     "/vlans",
	}

	// desired VLAN configuration
	desired := &openconfig.OpenconfigVlan_Vlans{
		Vlan: map[uint16]*openconfig.OpenconfigVlan_Vlans_Vlan{
			100: {
				VlanId: ptrUint16(100),
				Config: &openconfig.OpenconfigVlan_Vlans_Vlan_Config{
					Name:   ptrString("VLAN100"),
					Status: openconfig.OpenconfigVlan_Vlans_Vlan_Config_Status_ACTIVE,
				},
			},
		},
	}

	mockCS := new(mockConfigStore)
	mockCS.On("Get", deviceID, "/vlans").Return(desired, nil)

	// actual is empty on device
	actual := &openconfig.OpenconfigVlan_Vlans{
		Vlan: map[uint16]*openconfig.OpenconfigVlan_Vlans_Vlan{},
	}
	deviceRoot := &openconfig.Device{Vlans: actual}
	jsonActual, err := json.Marshal(deviceRoot)
	assert.NoError(t, err)

	mockClient := new(MockClient)
	mockClient.On("Get", ctx, "/vlans", mock.Anything).Return(&client.GetResult{
		Data: jsonActual,
	}, nil)
	mockClient.On("Set", ctx, mock.Anything, mock.Anything).Return(&client.SetResult{
		Success: true,
	}, nil)

	mockPool := new(MockClientPool)
	mockPool.On("Get", client.DeviceConnectionInfo{IP: deviceID}).Return(mockClient, nil)

	r := New(mockCS, mockPool)

	// Act
	result := r.Reconcile(ctx, req)

	// Assert
	assert.False(t, result.Requeue)
	assert.Nil(t, result.Error)
	mockCS.AssertExpectations(t)
	mockPool.AssertExpectations(t)
	mockClient.AssertExpectations(t)
	// One Set call should be made with one ADD change
	mockClient.AssertNumberOfCalls(t, "Set", 1)
}

func TestVlanReconciler_ConfigStoreGetError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	deviceID := "192.168.1.1"
	req := reconcile.Request{
		DeviceID: deviceID,
		Path:     "/vlans",
	}

	mockCS := new(mockConfigStore)
	mockCS.On("Get", deviceID, "/vlans").Return(nil, assert.AnError)

	mockPool := new(MockClientPool)

	r := New(mockCS, mockPool)

	// Act
	result := r.Reconcile(ctx, req)

	// Assert
	assert.True(t, result.Requeue)
	assert.Error(t, result.Error)
}

func TestVlanReconciler_NoDiff(t *testing.T) {
	// Arrange
	ctx := context.Background()
	deviceID := "192.168.1.1"
	req := reconcile.Request{
		DeviceID: deviceID,
		Path:     "/vlans",
	}

	desired := &openconfig.OpenconfigVlan_Vlans{
		Vlan: map[uint16]*openconfig.OpenconfigVlan_Vlans_Vlan{
			100: {
				VlanId: ptrUint16(100),
				Config: &openconfig.OpenconfigVlan_Vlans_Vlan_Config{
					Name:   ptrString("VLAN100"),
					Status: openconfig.OpenconfigVlan_Vlans_Vlan_Config_Status_ACTIVE,
				},
			},
		},
	}

	// desired and actual are identical
	deviceRoot := &openconfig.Device{Vlans: desired}
	jsonBytes, err := json.Marshal(deviceRoot)
	assert.NoError(t, err)

	mockCS := new(mockConfigStore)
	mockCS.On("Get", deviceID, "/vlans").Return(desired, nil)

	mockClient := new(MockClient)
	mockClient.On("Get", ctx, "/vlans", mock.Anything).Return(&client.GetResult{
		Data: jsonBytes,
	}, nil)

	mockPool := new(MockClientPool)
	mockPool.On("Get", client.DeviceConnectionInfo{IP: deviceID}).Return(mockClient, nil)

	r := New(mockCS, mockPool)

	// Act
	result := r.Reconcile(ctx, req)

	// Assert
	assert.False(t, result.Requeue)
	assert.Nil(t, result.Error)
	// No changes, so no Set call
	mockClient.AssertNumberOfCalls(t, "Set", 0)
}

// mockConfigStore is a mock implementation of reconcile.ConfigStore
type mockConfigStore struct {
	mock.Mock
}

func (m *mockConfigStore) Get(deviceID, path string) (interface{}, error) {
	args := m.Called(deviceID, path)
	return args.Get(0), args.Error(1)
}

func (m *mockConfigStore) Set(deviceID, path string, value interface{}) error {
	args := m.Called(deviceID, path, value)
	return args.Error(0)
}

func (m *mockConfigStore) Delete(deviceID, path string) error {
	args := m.Called(deviceID, path)
	return args.Error(0)
}

func (m *mockConfigStore) List(deviceID string) ([]string, error) {
	args := m.Called(deviceID)
	if res, ok := args.Get(0).([]string); ok {
		return res, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockConfigStore) ListDevices() ([]string, error) {
	args := m.Called()
	if res, ok := args.Get(0).([]string); ok {
		return res, args.Error(1)
	}
	return nil, args.Error(0)
}

func ptrUint16(v uint16) *uint16 {
	return &v
}

func ptrString(s string) *string {
	return &s
}
