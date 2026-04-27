package manager

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

// MockReconciler is a mock for testing
type MockReconciler struct {
	mock.Mock
}

func (m *MockReconciler) Reconcile(ctx context.Context, req reconcile.Request) reconcile.Result {
	args := m.Called(ctx, req)
	return args.Get(0).(reconcile.Result)
}

func TestNewManager(t *testing.T) {
	m := New()
	assert.NotNil(t, m)
	assert.NotNil(t, m.GetSchema())
	assert.NotNil(t, m.GetClientPool())
	assert.NotNil(t, m.GetPluginManager())
	assert.False(t, m.started)
}

func TestNewManagerWithOptions(t *testing.T) {
	m := New(
		WithDebug(),
		WithDefaultTimeout(30*time.Second),
	)
	assert.True(t, m.options.EnableDebug)
	assert.Equal(t, 30*time.Second, m.options.DefaultTimeout)
}

func TestAddController(t *testing.T) {
	m := New()
	mr := &MockReconciler{}
	ctrl := controller.ControllerManagedBy("test").WithReconciler(mr).Build()
	m.AddController(ctrl)
	assert.Len(t, m.Controllers(), 1)
}

func TestAddPlugin(t *testing.T) {
	m := New()
	// This would need a concrete implementation, just test it doesn't panic
	// Actual testing of plugin addition is covered in plugin package
	assert.NotPanics(t, func() {
		// Adding a plugin would correctly route it based on interface
		m.AddPlugin(nil)
	})
}

func TestStartStop(t *testing.T) {
	m := New()
	mr := &MockReconciler{}
	mr.On("Reconcile", mock.Anything, mock.Anything).Return(reconcile.Result{Requeue: false})
	ctrl := controller.ControllerManagedBy("test-controller").WithReconciler(mr).WithPredicate(predicate.Always()).Build()
	m.AddController(ctrl)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := m.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, m.started)

	err = m.Stop()
	assert.NoError(t, err)
	assert.False(t, m.started)
}

func TestStartStopIdempotent(t *testing.T) {
	m := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Starting multiple times should not error
	err := m.Start(ctx)
	assert.NoError(t, err)
	err = m.Start(ctx)
	assert.NoError(t, err)

	// Stopping multiple times should not error
	err = m.Stop()
	assert.NoError(t, err)
	err = m.Stop()
	assert.NoError(t, err)
}

func TestGetSchema(t *testing.T) {
	m := New()
	s := m.GetSchema()
	assert.NotNil(t, s)
	assert.Empty(t, s.Modules())
}

func TestGetClientPool(t *testing.T) {
	m := New()
	p := m.GetClientPool()
	assert.NotNil(t, p)
	stats := p.Stats()
	assert.Equal(t, 0, stats.ActiveConnections)
}
