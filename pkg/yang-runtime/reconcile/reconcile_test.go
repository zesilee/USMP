package reconcile

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockConfigStore is a mock for ConfigStore
type MockConfigStore struct {
	mock.Mock
}

func (m *MockConfigStore) Get(deviceID, path string) (interface{}, error) {
	args := m.Called(deviceID, path)
	return args.Get(0), args.Error(1)
}

func (m *MockConfigStore) Set(deviceID, path string, value interface{}) error {
	args := m.Called(deviceID, path, value)
	return args.Error(0)
}

func (m *MockConfigStore) Delete(deviceID, path string) error {
	args := m.Called(deviceID, path)
	return args.Error(0)
}

func (m *MockConfigStore) List(deviceID string) ([]string, error) {
	args := m.Called(deviceID)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockConfigStore) ListDevices() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

// MockDeviceClient is a mock for DeviceClient
type MockDeviceClient struct {
	mock.Mock
}

func (m *MockDeviceClient) Get(ctx context.Context, path string) (interface{}, error) {
	args := m.Called(ctx, path)
	return args.Get(0), args.Error(1)
}

func (m *MockDeviceClient) Set(ctx context.Context, changes []Change) error {
	args := m.Called(ctx, changes)
	return args.Error(0)
}

// MockDiffEngine is a mock for DiffEngine
type MockDiffEngine struct {
	mock.Mock
}

func (m *MockDiffEngine) Diff(desired, actual interface{}, path string) ([]Change, error) {
	args := m.Called(desired, actual, path)
	return args.Get(0).([]Change), args.Error(1)
}

func TestRequest(t *testing.T) {
	req := Request{
		DeviceID: "192.168.1.1",
		Path:     "/interfaces/interface[name='eth0']/description",
	}

	assert.Equal(t, "192.168.1.1", req.DeviceID)
	assert.Equal(t, "/interfaces/interface[name='eth0']/description", req.Path)
}

func TestReconcileError(t *testing.T) {
	underlying := errors.New("connection refused")
	err := &ReconcileError{
		DeviceID: "192.168.1.1",
		Path:     "/interfaces/interface[name='eth0']",
		Err:      underlying,
	}

	assert.Contains(t, err.Error(), "192.168.1.1")
	assert.Contains(t, err.Error(), "/interfaces/interface[name='eth0']")
	assert.Contains(t, err.Error(), "connection refused")
	assert.Equal(t, underlying, err.Unwrap())
}

func TestReconcilerFunc(t *testing.T) {
	called := false
	f := ReconcilerFunc(func(ctx context.Context, req Request) Result {
		called = true
		return Result{}
	})

	f.Reconcile(context.Background(), Request{})
	assert.True(t, called)
}

func TestGenericReconciler_NoChanges(t *testing.T) {
	cs := &MockConfigStore{}
	dc := &MockDeviceClient{}
	de := &MockDiffEngine{}

	desired := map[string]interface{}{"description": "test"}
	actual := map[string]interface{}{"description": "test"}

	cs.On("Get", "192.168.1.1", "/interfaces/interface[name='eth0']").Return(desired, nil)
	dc.On("Get", mock.Anything, "/interfaces/interface[name='eth0']").Return(actual, nil)
	de.On("Diff", desired, actual, "/interfaces/interface[name='eth0']").Return([]Change{}, nil)

	r := NewGenericReconciler(cs, dc, de)
	result := r.Reconcile(context.Background(), Request{
		DeviceID: "192.168.1.1",
		Path:     "/interfaces/interface[name='eth0']",
	})

	assert.False(t, result.Requeue)
	assert.Nil(t, result.Error)
	cs.AssertExpectations(t)
	dc.AssertExpectations(t)
	de.AssertExpectations(t)
}

func TestGenericReconciler_WithChanges(t *testing.T) {
	cs := &MockConfigStore{}
	dc := &MockDeviceClient{}
	de := &MockDiffEngine{}

	desired := map[string]interface{}{"description": "new"}
	actual := map[string]interface{}{"description": "old"}
	changes := []Change{
		{
			Path:         "/interfaces/interface[name='eth0']/description",
			Type:         "MODIFY",
			DesiredValue: "new",
			ActualValue:  "old",
		},
	}

	cs.On("Get", "192.168.1.1", "/interfaces/interface[name='eth0']").Return(desired, nil)
	dc.On("Get", mock.Anything, "/interfaces/interface[name='eth0']").Return(actual, nil)
	de.On("Diff", desired, actual, "/interfaces/interface[name='eth0']").Return(changes, nil)
	dc.On("Set", mock.Anything, changes).Return(nil)

	r := NewGenericReconciler(cs, dc, de)
	result := r.Reconcile(context.Background(), Request{
		DeviceID: "192.168.1.1",
		Path:     "/interfaces/interface[name='eth0']",
	})

	assert.False(t, result.Requeue)
	assert.Nil(t, result.Error)
	cs.AssertExpectations(t)
	dc.AssertExpectations(t)
	de.AssertExpectations(t)
}

func TestGenericReconciler_ConfigStoreError(t *testing.T) {
	cs := &MockConfigStore{}
	dc := &MockDeviceClient{}
	de := &MockDiffEngine{}

	expectedErr := errors.New("failed to read config store")
	cs.On("Get", "192.168.1.1", "/interfaces/interface[name='eth0']").Return(nil, expectedErr)

	r := NewGenericReconciler(cs, dc, de)
	result := r.Reconcile(context.Background(), Request{
		DeviceID: "192.168.1.1",
		Path:     "/interfaces/interface[name='eth0']",
	})

	assert.True(t, result.Requeue)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "reconciliation failed")
	cs.AssertExpectations(t)
}

func TestGenericReconciler_DeviceGetError(t *testing.T) {
	cs := &MockConfigStore{}
	dc := &MockDeviceClient{}
	de := &MockDiffEngine{}

	desired := map[string]interface{}{"description": "test"}
	expectedErr := errors.New("device connection failed")

	cs.On("Get", "192.168.1.1", "/interfaces/interface[name='eth0']").Return(desired, nil)
	dc.On("Get", mock.Anything, "/interfaces/interface[name='eth0']").Return(nil, expectedErr)

	r := NewGenericReconciler(cs, dc, de)
	result := r.Reconcile(context.Background(), Request{
		DeviceID: "192.168.1.1",
		Path:     "/interfaces/interface[name='eth0']",
	})

	assert.True(t, result.Requeue)
	assert.Error(t, result.Error)
	cs.AssertExpectations(t)
	dc.AssertExpectations(t)
}

func TestGenericReconciler_DiffError(t *testing.T) {
	cs := &MockConfigStore{}
	dc := &MockDeviceClient{}
	de := &MockDiffEngine{}

	desired := map[string]interface{}{"description": "test"}
	actual := map[string]interface{}{"description": "test"}
	expectedErr := errors.New("diff computation failed")

	cs.On("Get", "192.168.1.1", "/interfaces/interface[name='eth0']").Return(desired, nil)
	dc.On("Get", mock.Anything, "/interfaces/interface[name='eth0']").Return(actual, nil)
	de.On("Diff", desired, actual, "/interfaces/interface[name='eth0']").Return([]Change{}, expectedErr)

	r := NewGenericReconciler(cs, dc, de)
	result := r.Reconcile(context.Background(), Request{
		DeviceID: "192.168.1.1",
		Path:     "/interfaces/interface[name='eth0']",
	})

	assert.True(t, result.Requeue)
	assert.Error(t, result.Error)
	cs.AssertExpectations(t)
	dc.AssertExpectations(t)
	de.AssertExpectations(t)
}

func TestGenericReconciler_SetError(t *testing.T) {
	cs := &MockConfigStore{}
	dc := &MockDeviceClient{}
	de := &MockDiffEngine{}

	desired := map[string]interface{}{"description": "new"}
	actual := map[string]interface{}{"description": "old"}
	changes := []Change{
		{
			Path:         "/interfaces/interface[name='eth0']/description",
			Type:         "MODIFY",
			DesiredValue: "new",
			ActualValue:  "old",
		},
	}
	expectedErr := errors.New("failed to apply changes")

	cs.On("Get", "192.168.1.1", "/interfaces/interface[name='eth0']").Return(desired, nil)
	dc.On("Get", mock.Anything, "/interfaces/interface[name='eth0']").Return(actual, nil)
	de.On("Diff", desired, actual, "/interfaces/interface[name='eth0']").Return(changes, nil)
	dc.On("Set", mock.Anything, changes).Return(expectedErr)

	r := NewGenericReconciler(cs, dc, de)
	result := r.Reconcile(context.Background(), Request{
		DeviceID: "192.168.1.1",
		Path:     "/interfaces/interface[name='eth0']",
	})

	assert.True(t, result.Requeue)
	assert.Error(t, result.Error)
	cs.AssertExpectations(t)
	dc.AssertExpectations(t)
	de.AssertExpectations(t)
}
