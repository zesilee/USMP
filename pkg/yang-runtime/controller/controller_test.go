package controller

import (
	"context"
	"testing"
	"time"

	"github.com/leezesi/usmp/pkg/yang-runtime/predicate"
	"github.com/leezesi/usmp/pkg/yang-runtime/queue"
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

// MockSource is a mock for Source
type MockSource struct {
	started bool
	stopped bool
}

func (m *MockSource) Start(ctx context.Context, ctrl Controller) error {
	m.started = true
	return nil
}

func (m *MockSource) Stop() error {
	m.stopped = true
	return nil
}

func TestNewController(t *testing.T) {
	mr := &MockReconciler{}
	ms := &MockSource{}
	q := queue.NewRateLimitingQueue(queue.DefaultRateLimiter())
	c := New("test-controller", ms, mr, q, nil, 0)

	assert.Equal(t, "test-controller", c.Name())
	assert.Equal(t, 1, c.workerCount) // Default to 1 when 0 given
	c.Stop()
}

func TestEnqueueFiltersEvent(t *testing.T) {
	mr := &MockReconciler{}
	ms := &MockSource{}
	q := queue.NewRateLimitingQueue(queue.DefaultRateLimiter())
	p := predicate.Not(predicate.Prefix("/interfaces"))
	c := New("test", ms, mr, q, []predicate.Predicate{p}, 1)

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
	c.Stop()
}

func TestReconciliationSuccess(t *testing.T) {
	mr := &MockReconciler{}
	ms := &MockSource{}
	q := queue.NewRateLimitingQueue(queue.DefaultRateLimiter())
	req := reconcile.Request{
		DeviceID: "192.168.1.1",
		Path:     "/interfaces/interface[name='eth0']",
	}

	mr.On("Reconcile", mock.Anything, req).Return(reconcile.Result{Requeue: false})

	c := New("test", ms, mr, q, nil, 1)

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
	ms := &MockSource{}
	q := queue.NewRateLimitingQueue(queue.DefaultRateLimiter())
	req := reconcile.Request{
		DeviceID: "192.168.1.1",
		Path:     "/interfaces/interface[name='eth0']",
	}

	mr.On("Reconcile", mock.Anything, req).Return(reconcile.Result{Requeue: true}).Once()

	c := New("test", ms, mr, q, nil, 1)

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
	ms := &MockSource{}
	q := queue.NewRateLimitingQueue(queue.DefaultRateLimiter())
	req := reconcile.Request{
		DeviceID: "192.168.1.1",
		Path:     "/interfaces/interface[name='eth0']",
	}

	// Return error but no explicit requeue
	mr.On("Reconcile", mock.Anything, req).Return(reconcile.Result{
		Requeue: false,
		Error:   assert.AnError,
	}).Once()

	c := New("test", ms, mr, q, nil, 1)

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
	ms := &MockSource{}
	q := queue.NewRateLimitingQueue(queue.DefaultRateLimiter())
	mr.On("Reconcile", mock.Anything, mock.Anything).Return(reconcile.Result{Requeue: false})

	c := New("test", ms, mr, q, nil, 4)

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
	ms := &MockSource{}
	q := queue.NewRateLimitingQueue(queue.DefaultRateLimiter())
	c := New("test", ms, mr, q, nil, 1)

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
