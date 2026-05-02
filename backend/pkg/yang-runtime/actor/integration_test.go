package actor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActorManager_EndToEnd(t *testing.T) {
	// Setup
	clientPool := &mockClientPool{}
	configStore := &testConfigStore{
		data: make(map[string]map[string]interface{}),
	}
	manager := NewActorManager(clientPool, configStore)

	// Get device actor - should register all modules
	deviceActor := manager.GetDeviceActor("192.168.1.1")
	require.NotNil(t, deviceActor)
	assert.Equal(t, "192.168.1.1", deviceActor.deviceID)

	// Verify VLAN module is registered
	vlanActor, exists := deviceActor.GetModuleActor("vlans")
	assert.True(t, exists, "VLAN module should be auto-registered")
	assert.NotNil(t, vlanActor)

	// Start the actor
	err := deviceActor.Start()
	require.NoError(t, err)
	defer deviceActor.Stop()

	time.Sleep(50 * time.Millisecond) // Give goroutine time to start

	// Check status
	status := deviceActor.Status()
	assert.Equal(t, StatusReady, status.Status)
}

func TestActorReconciler_TranslateOnly(t *testing.T) {
	// This tests the translation flow without actual device communication
	clientPool := &mockClientPool{}
	configStore := &testConfigStore{
		data: make(map[string]map[string]interface{}),
	}
	manager := NewActorManager(clientPool, configStore)

	deviceActor := manager.GetDeviceActor("192.168.1.2")
	err := deviceActor.Start()
	require.NoError(t, err)
	defer deviceActor.Stop()

	time.Sleep(50 * time.Millisecond)

	moduleActor, exists := deviceActor.GetModuleActor("vlans")
	require.True(t, exists)

	// Send translate command
	cmd := &TranslateCmd{
		BaseMessage: NewBaseMessage("test", MsgTranslate),
		Path:        "/vlans",
		Payload: map[string]interface{}{
			"Name": "TestVLAN",
		},
		Operation: OperationMerge,
	}

	promise, err := moduleActor.Send(cmd)
	require.NoError(t, err)

	result := <-promise
	assert.True(t, result.Success, "Translate should succeed")
	assert.NoError(t, result.Error)
	assert.Positive(t, result.Version, "Should have version > 0")
}

func TestActorManager_MultipleDevices(t *testing.T) {
	clientPool := &mockClientPool{}
	configStore := &testConfigStore{
		data: make(map[string]map[string]interface{}),
	}
	manager := NewActorManager(clientPool, configStore)

	// Create multiple device actors
	devices := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}

	for _, deviceID := range devices {
		actor := manager.GetDeviceActor(deviceID)
		require.NotNil(t, actor)
		assert.Equal(t, deviceID, actor.deviceID)
	}

	// Verify all devices are tracked
	statusMap := manager.GetStatus()
	assert.Len(t, statusMap, len(devices))

	for _, deviceID := range devices {
		_, exists := statusMap[deviceID]
		assert.True(t, exists, "Device %s should be in status map", deviceID)
	}
}

// testConfigStore implements reconcile.ConfigStore for testing
type testConfigStore struct {
	data map[string]map[string]interface{}
}

func (s *testConfigStore) Get(deviceID, path string) (interface{}, error) {
	if deviceData, ok := s.data[deviceID]; ok {
		return deviceData[path], nil
	}
	return nil, nil
}

func (s *testConfigStore) Set(deviceID, path string, value interface{}) error {
	if _, ok := s.data[deviceID]; !ok {
		s.data[deviceID] = make(map[string]interface{})
	}
	s.data[deviceID][path] = value
	return nil
}

func (s *testConfigStore) Delete(deviceID, path string) error {
	if deviceData, ok := s.data[deviceID]; ok {
		delete(deviceData, path)
	}
	return nil
}

func (s *testConfigStore) List(deviceID string) ([]string, error) {
	if deviceData, ok := s.data[deviceID]; ok {
		paths := make([]string, 0, len(deviceData))
		for path := range deviceData {
			paths = append(paths, path)
		}
		return paths, nil
	}
	return []string{}, nil
}

func (s *testConfigStore) ListDevices() ([]string, error) {
	devices := make([]string, 0, len(s.data))
	for deviceID := range s.data {
		devices = append(devices, deviceID)
	}
	return devices, nil
}
