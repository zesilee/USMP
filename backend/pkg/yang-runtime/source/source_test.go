package source

import (
	"context"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/controller"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/predicate"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockController is a mock controller
type MockController struct {
	mock.Mock
}

func (m *MockController) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockController) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockController) Stop() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockController) Enqueue(evt predicate.Event) {
	m.Called(evt)
}

func TestNewPeriodicSource(t *testing.T) {
	devices := []string{"192.168.1.1", "192.168.1.2"}
	s := NewPeriodicSource(100*time.Millisecond, devices, "/interfaces")
	assert.NotNil(t, s)
	assert.Equal(t, 100*time.Millisecond, s.interval)
	assert.Equal(t, devices, s.deviceIDs)
}

func TestPeriodicSourceStartStop(t *testing.T) {
	mc := &MockController{}
	mc.On("Name").Return("test-controller")
	mc.On("Start", mock.Anything).Return(nil)
	mc.On("Stop").Return(nil)
	mc.On("Enqueue", mock.Anything).Return()

	devices := []string{"192.168.1.1"}
	s := NewPeriodicSource(50*time.Millisecond, devices, "/interfaces")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := s.Start(ctx, mc)
	assert.NoError(t, err)

	// Wait for at least one tick
	time.Sleep(100 * time.Millisecond)

	s.Stop()
	mc.AssertCalled(t, "Enqueue", mock.Anything)
}

// fakeLister is a DeviceLister test double.
type fakeLister struct{ ids []string }

func (f fakeLister) List() []string { return f.ids }

// TestPeriodicSource_EnqueuesFromLister: with a dynamic lister the source must
// enqueue one event per listed device each tick (regression: the old nil static
// list made the periodic source a no-op → no out-of-band drift detection).
func TestPeriodicSource_EnqueuesFromLister(t *testing.T) {
	mc := &MockController{}
	mc.On("Name").Return("test-controller")
	mc.On("Start", mock.Anything).Return(nil)
	mc.On("Stop").Return(nil)
	mc.On("Enqueue", mock.Anything).Return()

	s := NewPeriodicSourceWithLister(30*time.Millisecond, fakeLister{ids: []string{"192.168.1.1"}}, "/ifm:ifm/ifm:interfaces")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	assert.NoError(t, s.Start(ctx, mc))
	time.Sleep(90 * time.Millisecond)
	s.Stop()

	mc.AssertCalled(t, "Enqueue", mock.Anything)
}

// TestPeriodicSource_EmptyListerNoEnqueue: an empty lister must enqueue nothing
// (no panic, no spurious events).
func TestPeriodicSource_EmptyListerNoEnqueue(t *testing.T) {
	mc := &MockController{}
	mc.On("Name").Return("test-controller")
	mc.On("Start", mock.Anything).Return(nil)
	mc.On("Stop").Return(nil)

	s := NewPeriodicSourceWithLister(30*time.Millisecond, fakeLister{ids: nil}, "/ifm:ifm/ifm:interfaces")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	assert.NoError(t, s.Start(ctx, mc))
	time.Sleep(90 * time.Millisecond)
	s.Stop()

	mc.AssertNotCalled(t, "Enqueue", mock.Anything)
}

func TestBaseSourceEnqueue(t *testing.T) {
	mc := &MockController{}
	called := false
	expectedEvt := predicate.Event{
		DeviceID: "192.168.1.1",
		Path:     "/interfaces/interface",
		Type:     predicate.UpdateEvent,
	}

	mc.On("Enqueue", expectedEvt).Run(func(args mock.Arguments) {
		called = true
	}).Return()

	bs := &BaseSource{controller: mc}
	bs.EnqueueEvent(expectedEvt)

	assert.True(t, called)
	mc.AssertCalled(t, "Enqueue", expectedEvt)
}
func TestControllerBuilderIntegration(t *testing.T) {
	// Test that we can build a controller and source works together
	mr := &MockReconciler{}
	mr.On("Reconcile", mock.Anything, mock.Anything).Return(reconcile.Result{Requeue: false})

	ctrl := controller.ControllerManagedBy("test-controller").
		WithReconciler(mr).
		Build()

	devices := []string{"192.168.1.1"}
	s := NewPeriodicSource(1*time.Second, devices, "/interfaces")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := s.Start(ctx, ctrl)
	assert.NoError(t, err)

	s.Stop()
}

type MockReconciler struct {
	mock.Mock
}

func (m *MockReconciler) Reconcile(ctx context.Context, req reconcile.Request) reconcile.Result {
	args := m.Called(ctx, req)
	return args.Get(0).(reconcile.Result)
}
