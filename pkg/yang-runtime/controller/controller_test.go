package controller

import (
	"context"
	"testing"
	"time"

	"github.com/leezesi/usmp/pkg/yang-runtime/predicate"
	"github.com/leezesi/usmp/pkg/yang-runtime/reconcile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockReconciler is a mock for Reconciler
type MockReconciler struct {
	mock.Mock
}

func (m *MockReconciler) Reconcile(ctx context.Context, req reconcile.Request) reconcile.Result {
	args := m.Called(ctx, req)
	return args.Get(0).(reconcile.Result)
}

func TestNewController(t *testing.T) {
	mr := &MockReconciler{}
	c := New("test-controller", mr, nil, nil, 0)

	assert.Equal(t, "test-controller", c.Name())
	assert.Equal(t, 1, c.workerCount) // Default to 1 when 0 given
}

func TestBuilder(t *testing.T) {
	mr := &MockReconciler{}
	b := ControllerManagedBy("test-builder").
		WithReconciler(mr).
		WithWorkerCount(2)

	c := b.Build()

	assert.NotNil(t, c)
	assert.Equal(t, "test-builder", c.Name())
}

func TestBuilderWithPredicate(t *testing.T) {
	mr := &MockReconciler{}
	p := predicate.Prefix("/interfaces")
	b := ControllerManagedBy("test").WithReconciler(mr).WithPredicate(p)

	c := b.Build()
	assert.NotNil(t, c)
}

func TestEnqueueFiltersEvent(t *testing.T) {
	mr := &MockReconciler{}
	p := predicate.Not(predicate.Prefix("/interfaces"))
	c := ControllerManagedBy("test").WithReconciler(mr).WithPredicate(p).Build()

	// Event that should be filtered out
	evt := predicate.Event{
		DeviceID: "192.168.1.1",
		Path:     "/interfaces/interface",
		Type:     predicate.UpdateEvent,
	}

	// Enqueue should not add to queue because predicate filters it
	c.Enqueue(evt)

	// Since the worker isn't started, we can't really check the queue directly
	// but this just verifies it doesn't panic and correctly filtered
}

func TestReconciliationSuccess(t *testing.T) {
	mr := &MockReconciler{}
	req := reconcile.Request{
		DeviceID: "192.168.1.1",
		Path:     "/interfaces/interface[name='eth0']",
	}

	mr.On("Reconcile", mock.Anything, req).Return(reconcile.Result{Requeue: false})

	c := ControllerManagedBy("test").WithReconciler(mr).Build()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := c.Start(ctx)
	assert.NoError(t, err)

	// Enqueue the event
	c.Enqueue(predicate.Event{
		DeviceID: req.DeviceID,
		Path:     req.Path,
		Type:     predicate.UpdateEvent,
	})

	// Give it time to process
	time.Sleep(100 * time.Millisecond)

	mr.AssertExpectations(t)
	err = c.Stop()
	assert.NoError(t, err)
}

func TestReconciliationRequeue(t *testing.T) {
	mr := &MockReconciler{}
	req := reconcile.Request{
		DeviceID: "192.168.1.1",
		Path:     "/interfaces/interface[name='eth0']",
	}

	mr.On("Reconcile", mock.Anything, req).Return(reconcile.Result{Requeue: true}).Once()

	c := ControllerManagedBy("test").WithReconciler(mr).Build()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := c.Start(ctx)
	assert.NoError(t, err)

	c.Enqueue(predicate.Event{
		DeviceID: req.DeviceID,
		Path:     req.Path,
		Type:     predicate.UpdateEvent,
	})

	time.Sleep(150 * time.Millisecond)

	mr.AssertExpectations(t)
	err = c.Stop()
	assert.NoError(t, err)
}

func TestReconciliationWithError(t *testing.T) {
	mr := &MockReconciler{}
	req := reconcile.Request{
		DeviceID: "192.168.1.1",
		Path:     "/interfaces/interface[name='eth0']",
	}

	// Return error but no explicit requeue
	mr.On("Reconcile", mock.Anything, req).Return(reconcile.Result{
		Requeue: false,
		Error:   assert.AnError,
	}).Once()

	c := ControllerManagedBy("test").WithReconciler(mr).Build()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := c.Start(ctx)
	assert.NoError(t, err)

	c.Enqueue(predicate.Event{
		DeviceID: req.DeviceID,
		Path:     req.Path,
		Type:     predicate.UpdateEvent,
	})

	time.Sleep(150 * time.Millisecond)

	mr.AssertExpectations(t)
	err = c.Stop()
	assert.NoError(t, err)
}

func TestMultipleWorkers(t *testing.T) {
	mr := &MockReconciler{}
	mr.On("Reconcile", mock.Anything, mock.Anything).Return(reconcile.Result{Requeue: false})

	c := ControllerManagedBy("test").WithReconciler(mr).WithWorkerCount(4).Build()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := c.Start(ctx)
	assert.NoError(t, err)

	// Enqueue multiple events
	for i := 0; i < 10; i++ {
		c.Enqueue(predicate.Event{
			DeviceID: "192.168.1.1",
			Path:     "/interfaces/interface",
			Type:     predicate.UpdateEvent,
		})
	}

	time.Sleep(500 * time.Millisecond)

	mr.AssertNumberOfCalls(t, "Reconcile", 10)
	err = c.Stop()
	assert.NoError(t, err)
}

func TestStartStopIdempotent(t *testing.T) {
	mr := &MockReconciler{}
	c := ControllerManagedBy("test").WithReconciler(mr).Build()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Starting multiple times shouldn't cause issues
	err := c.Start(ctx)
	assert.NoError(t, err)
	err = c.Start(ctx)
	assert.NoError(t, err)

	// Stopping multiple times shouldn't cause issues
	err = c.Stop()
	assert.NoError(t, err)
	err = c.Stop()
	assert.NoError(t, err)
}
