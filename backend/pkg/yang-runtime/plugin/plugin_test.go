package plugin

import (
	"context"
	"errors"
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockValidationPlugin is a mock validation plugin
type MockValidationPlugin struct {
	mock.Mock
	name string
}

func (m *MockValidationPlugin) Name() string {
	return m.name
}

func (m *MockValidationPlugin) Validate(ctx context.Context, req reconcile.Request, change *Change) error {
	args := m.Called(ctx, req, change)
	return args.Error(0)
}

// MockMutationPlugin is a mock mutation plugin
type MockMutationPlugin struct {
	mock.Mock
	name string
}

func (m *MockMutationPlugin) Name() string {
	return m.name
}

func (m *MockMutationPlugin) Mutate(ctx context.Context, req reconcile.Request, change *Change) (*Change, error) {
	args := m.Called(ctx, req, change)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Change), args.Error(1)
}

// MockNotificationPlugin is a mock notification plugin
type MockNotificationPlugin struct {
	mock.Mock
	name string
}

func (m *MockNotificationPlugin) Name() string {
	return m.name
}

func (m *MockNotificationPlugin) OnSuccess(ctx context.Context, req reconcile.Request, change *Change) {
	m.Called(ctx, req, change)
}

func (m *MockNotificationPlugin) OnFailure(ctx context.Context, req reconcile.Request, change *Change, err error) {
	m.Called(ctx, req, change, err)
}

func TestNewManager(t *testing.T) {
	m := NewManager()
	assert.NotNil(t, m)
	assert.Empty(t, m.ValidationPlugins())
	assert.Empty(t, m.MutationPlugins())
}

func TestAddValidationPlugin(t *testing.T) {
	m := NewManager()
	p := &MockValidationPlugin{name: "test-validation"}
	m.AddValidationPlugin(p)

	plugins := m.ValidationPlugins()
	assert.Len(t, plugins, 1)
	assert.Equal(t, "test-validation", plugins[0].Name())
}

func TestValidateAllPass(t *testing.T) {
	m := NewManager()
	p1 := &MockValidationPlugin{}
	p2 := &MockValidationPlugin{}

	req := reconcile.Request{DeviceID: "192.168.1.1", Path: "/interfaces/interface"}
	change := &Change{
		Path:     "/interfaces/interface[name='eth0']/description",
		OldValue: "old",
		NewValue: "new",
		DeviceID: "192.168.1.1",
	}

	p1.On("Validate", mock.Anything, req, change).Return(nil)
	p2.On("Validate", mock.Anything, req, change).Return(nil)

	m.AddValidationPlugin(p1)
	m.AddValidationPlugin(p2)

	err := m.Validate(context.Background(), req, change)
	assert.NoError(t, err)
	p1.AssertExpectations(t)
	p2.AssertExpectations(t)
}

func TestValidateFirstFails(t *testing.T) {
	m := NewManager()
	p1 := &MockValidationPlugin{}
	p2 := &MockValidationPlugin{}

	req := reconcile.Request{DeviceID: "192.168.1.1", Path: "/interfaces/interface"}
	change := &Change{
		Path:     "/interfaces/interface[name='eth0']/description",
		OldValue: "old",
		NewValue: "new",
		DeviceID: "192.168.1.1",
	}

	expectedErr := errors.New("validation failed")
	p1.On("Validate", mock.Anything, req, change).Return(expectedErr)

	m.AddValidationPlugin(p1)
	m.AddValidationPlugin(p2)

	err := m.Validate(context.Background(), req, change)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	p1.AssertExpectations(t)
	p2.AssertNotCalled(t, "Validate", mock.Anything, mock.Anything, mock.Anything)
}

func TestMutate(t *testing.T) {
	m := NewManager()
	p := &MockMutationPlugin{}

	req := reconcile.Request{DeviceID: "192.168.1.1", Path: "/interfaces/interface"}
	change := &Change{
		Path:     "/interfaces/interface[name='eth0']/description",
		OldValue: "old",
		NewValue: "new",
		DeviceID: "192.168.1.1",
	}

	mutated := *change
	mutated.NewValue = "mutated"

	p.On("Mutate", mock.Anything, req, change).Return(&mutated, nil)

	m.AddMutationPlugin(p)

	result, err := m.Mutate(context.Background(), req, change)
	assert.NoError(t, err)
	assert.Equal(t, "mutated", result.NewValue)
	p.AssertExpectations(t)
}

func TestNotifySuccess(t *testing.T) {
	m := NewManager()
	p := &MockNotificationPlugin{name: "test-notify"}

	req := reconcile.Request{DeviceID: "192.168.1.1", Path: "/interfaces/interface"}
	change := &Change{
		Path:     "/interfaces/interface[name='eth0']/description",
		OldValue: "old",
		NewValue: "new",
		DeviceID: "192.168.1.1",
	}

	p.On("OnSuccess", mock.Anything, req, change).Return()

	m.AddNotificationPlugin(p)

	m.OnSuccess(context.Background(), req, change)
	p.AssertExpectations(t)
}

func TestNotifyFailure(t *testing.T) {
	m := NewManager()
	p := &MockNotificationPlugin{name: "test-notify"}

	req := reconcile.Request{DeviceID: "192.168.1.1", Path: "/interfaces/interface"}
	change := &Change{
		Path:     "/interfaces/interface[name='eth0']/description",
		OldValue: "old",
		NewValue: "new",
		DeviceID: "192.168.1.1",
	}
	err := errors.New("test error")

	p.On("OnFailure", mock.Anything, req, change, err).Return()

	m.AddNotificationPlugin(p)

	m.OnFailure(context.Background(), req, change, err)
	p.AssertExpectations(t)
}

func TestMultiplePluginsDifferentTypes(t *testing.T) {
	m := NewManager()

	vp := &MockValidationPlugin{name: "validation"}
	mp := &MockMutationPlugin{name: "mutation"}
	np := &MockNotificationPlugin{name: "notification"}

	m.AddValidationPlugin(vp)
	m.AddMutationPlugin(mp)
	m.AddNotificationPlugin(np)

	assert.Len(t, m.ValidationPlugins(), 1)
	assert.Len(t, m.MutationPlugins(), 1)
	assert.Equal(t, "validation", m.ValidationPlugins()[0].Name())
	assert.Equal(t, "mutation", m.MutationPlugins()[0].Name())
}
