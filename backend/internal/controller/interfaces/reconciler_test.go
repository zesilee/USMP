package interfaces

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

func TestDeviceClient_Get_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	deviceID := "192.168.1.1"

	// Create test JSON response with one interface
	deviceRoot := &openconfig.Device{
		Interfaces: &openconfig.OpenconfigInterfaces_Interfaces{
			Interface: map[string]*openconfig.OpenconfigInterfaces_Interfaces_Interface{
				"GigabitEthernet0/0": {
					Name: ptrString("GigabitEthernet0/0"),
					Config: &openconfig.OpenconfigInterfaces_Interfaces_Interface_Config{
						Name:        ptrString("GigabitEthernet0/0"),
						Type:        openconfig.OpenconfigInterfaces_Interfaces_Interface_Config_Type_PHYSICAL,
						Enabled:     ptrBool(true),
						Mtu:         ptrUint16(1500),
						Description: ptrString("Uplink to router"),
					},
				},
			},
		},
	}

	jsonBytes, err := json.Marshal(deviceRoot)
	assert.NoError(t, err)

	mockClient := new(MockClient)
	mockClient.On("Get", ctx, "/interfaces", mock.Anything).Return(&client.GetResult{
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

	interfaces, ok := result.(*openconfig.OpenconfigInterfaces_Interfaces)
	assert.True(t, ok)
	assert.NotNil(t, interfaces)
	assert.Contains(t, interfaces.Interface, "GigabitEthernet0/0")
	iface := interfaces.Interface["GigabitEthernet0/0"]
	assert.NotNil(t, iface)
	assert.Equal(t, "GigabitEthernet0/0", *iface.Name)
	assert.Equal(t, "GigabitEthernet0/0", *iface.Config.Name)
	assert.Equal(t, uint16(1500), *iface.Config.Mtu)
	assert.Equal(t, true, *iface.Config.Enabled)
	assert.Equal(t, "Uplink to router", *iface.Config.Description)

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
	mockClient.On("Get", ctx, "/interfaces", mock.Anything).Return(nil, assert.AnError)

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

func TestDeviceClient_Get_EmptyInterfaces(t *testing.T) {
	// Arrange
	ctx := context.Background()
	deviceID := "192.168.1.1"

	// Empty interfaces
	deviceRoot := &openconfig.Device{
		Interfaces: &openconfig.OpenconfigInterfaces_Interfaces{
			Interface: map[string]*openconfig.OpenconfigInterfaces_Interfaces_Interface{},
		},
	}

	jsonBytes, err := json.Marshal(deviceRoot)
	assert.NoError(t, err)

	mockClient := new(MockClient)
	mockClient.On("Get", ctx, "/interfaces", mock.Anything).Return(&client.GetResult{
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

	interfaces, ok := result.(*openconfig.OpenconfigInterfaces_Interfaces)
	assert.True(t, ok)
	assert.NotNil(t, interfaces)
	assert.Empty(t, interfaces.Interface)

	mockPool.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

func TestDeviceClient_Set_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	deviceID := "192.168.1.1"
	changes := []reconcile.Change{
		{
			Path:         "/interfaces/interface[GigabitEthernet0/0]",
			Type:         "MODIFY",
			DesiredValue: map[string]interface{}{"enabled": true},
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
	assert.Equal(t, client.ModifyChange, clientChanges[0].Type)
	assert.Equal(t, "/interfaces/interface[GigabitEthernet0/0]", clientChanges[0].Path)
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

func TestInterfacesReconciler_FullReconcile(t *testing.T) {
	// Arrange
	ctx := context.Background()
	deviceID := "192.168.1.1"
	req := reconcile.Request{
		DeviceID: deviceID,
		Path:     "/interfaces",
	}

	// desired interface configuration
	desired := &openconfig.OpenconfigInterfaces_Interfaces{
		Interface: map[string]*openconfig.OpenconfigInterfaces_Interfaces_Interface{
			"GigabitEthernet0/0": {
				Name: ptrString("GigabitEthernet0/0"),
				Config: &openconfig.OpenconfigInterfaces_Interfaces_Interface_Config{
					Name:        ptrString("GigabitEthernet0/0"),
					Type:        openconfig.OpenconfigInterfaces_Interfaces_Interface_Config_Type_PHYSICAL,
					Enabled:     ptrBool(true),
					Mtu:         ptrUint16(1500),
					Description: ptrString("Uplink"),
				},
			},
		},
	}

	mockCS := new(mockConfigStore)
	mockCS.On("Get", deviceID, "/interfaces").Return(desired, nil)

	// actual is empty on device
	actual := &openconfig.OpenconfigInterfaces_Interfaces{
		Interface: map[string]*openconfig.OpenconfigInterfaces_Interfaces_Interface{},
	}
	deviceRoot := &openconfig.Device{Interfaces: actual}
	jsonActual, err := json.Marshal(deviceRoot)
	assert.NoError(t, err)

	mockClient := new(MockClient)
	mockClient.On("Get", ctx, "/interfaces", mock.Anything).Return(&client.GetResult{
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

func TestInterfacesReconciler_ConfigStoreGetError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	deviceID := "192.168.1.1"
	req := reconcile.Request{
		DeviceID: deviceID,
		Path:     "/interfaces",
	}

	mockCS := new(mockConfigStore)
	mockCS.On("Get", deviceID, "/interfaces").Return(nil, assert.AnError)

	mockPool := new(MockClientPool)

	r := New(mockCS, mockPool)

	// Act
	result := r.Reconcile(ctx, req)

	// Assert
	assert.True(t, result.Requeue)
	assert.Error(t, result.Error)
}

func TestInterfacesReconciler_NoDiff(t *testing.T) {
	// Arrange
	ctx := context.Background()
	deviceID := "192.168.1.1"
	req := reconcile.Request{
		DeviceID: deviceID,
		Path:     "/interfaces",
	}

	desired := &openconfig.OpenconfigInterfaces_Interfaces{
		Interface: map[string]*openconfig.OpenconfigInterfaces_Interfaces_Interface{
			"GigabitEthernet0/0": {
				Name: ptrString("GigabitEthernet0/0"),
				Config: &openconfig.OpenconfigInterfaces_Interfaces_Interface_Config{
					Name:        ptrString("GigabitEthernet0/0"),
					Type:        openconfig.OpenconfigInterfaces_Interfaces_Interface_Config_Type_PHYSICAL,
					Enabled:     ptrBool(true),
					Mtu:         ptrUint16(1500),
					Description: ptrString("Uplink"),
				},
			},
		},
	}

	// desired and actual are identical
	deviceRoot := &openconfig.Device{Interfaces: desired}
	jsonBytes, err := json.Marshal(deviceRoot)
	assert.NoError(t, err)

	mockCS := new(mockConfigStore)
	mockCS.On("Get", deviceID, "/interfaces").Return(desired, nil)

	mockClient := new(MockClient)
	mockClient.On("Get", ctx, "/interfaces", mock.Anything).Return(&client.GetResult{
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

func TestDeviceClient_Get_MultipleInterfaces(t *testing.T) {
	// Arrange
	ctx := context.Background()
	deviceID := "192.168.1.1"

	// Create test response with multiple interfaces
	deviceRoot := &openconfig.Device{
		Interfaces: &openconfig.OpenconfigInterfaces_Interfaces{
			Interface: map[string]*openconfig.OpenconfigInterfaces_Interfaces_Interface{
				"GigabitEthernet0/0": {
					Name: ptrString("GigabitEthernet0/0"),
					Config: &openconfig.OpenconfigInterfaces_Interfaces_Interface_Config{
						Name:        ptrString("GigabitEthernet0/0"),
						Type:        openconfig.OpenconfigInterfaces_Interfaces_Interface_Config_Type_PHYSICAL,
						Enabled:     ptrBool(true),
						Mtu:         ptrUint16(1500),
						Description: ptrString("Uplink to router"),
					},
				},
				"GigabitEthernet0/1": {
					Name: ptrString("GigabitEthernet0/1"),
					Config: &openconfig.OpenconfigInterfaces_Interfaces_Interface_Config{
						Name:        ptrString("GigabitEthernet0/1"),
						Type:        openconfig.OpenconfigInterfaces_Interfaces_Interface_Config_Type_PHYSICAL,
						Enabled:     ptrBool(false),
						Mtu:         ptrUint16(9000),
						Description: ptrString("Backup link"),
					},
				},
				"Loopback0": {
					Name: ptrString("Loopback0"),
					Config: &openconfig.OpenconfigInterfaces_Interfaces_Interface_Config{
						Name:        ptrString("Loopback0"),
						Type:        openconfig.OpenconfigInterfaces_Interfaces_Interface_Config_Type_LOOPBACK,
						Enabled:     ptrBool(true),
						Description: ptrString("Management loopback"),
					},
				},
			},
		},
	}

	jsonBytes, err := json.Marshal(deviceRoot)
	assert.NoError(t, err)

	mockClient := new(MockClient)
	mockClient.On("Get", ctx, "/interfaces", mock.Anything).Return(&client.GetResult{
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

	interfaces, ok := result.(*openconfig.OpenconfigInterfaces_Interfaces)
	assert.True(t, ok)
	assert.Len(t, interfaces.Interface, 3)
	assert.Contains(t, interfaces.Interface, "GigabitEthernet0/0")
	assert.Contains(t, interfaces.Interface, "GigabitEthernet0/1")
	assert.Contains(t, interfaces.Interface, "Loopback0")

	// Verify enabled status for Loopback
	loopback := interfaces.Interface["Loopback0"]
	assert.True(t, *loopback.Config.Enabled)
	assert.Equal(t, "Management loopback", *loopback.Config.Description)

	// Verify disabled status for GE0/1
	ge01 := interfaces.Interface["GigabitEthernet0/1"]
	assert.False(t, *ge01.Config.Enabled)
	assert.Equal(t, uint16(9000), *ge01.Config.Mtu)

	mockPool.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

func ptrString(s string) *string {
	return &s
}

func ptrUint16(v uint16) *uint16 {
	return &v
}

func ptrBool(b bool) *bool {
	return &b
}
