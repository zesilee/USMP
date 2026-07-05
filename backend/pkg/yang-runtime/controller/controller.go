package controller

import (
	"context"
	"sync"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/predicate"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/queue"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/status"
)

// Source is the interface for event sources
// An event source generates events that trigger reconciliation
type Source interface {
	// Start starts the event source and sends events to the controller
	Start(ctx context.Context, ctrl Controller) error
	// Stop stops the event source
	Stop() error
}

// Controller is the interface that all controllers must implement
// A controller watches events from a Source, filters them through Predicates,
// and enqueues them for reconciliation via the Reconciler.
type Controller interface {
	// Start starts the controller worker goroutines
	Start(ctx context.Context) error
	// Stop stops the controller and waits for workers to finish
	Stop() error
	// Enqueue adds an event to the work queue
	Enqueue(evt predicate.Event)
	// Name returns the name of the controller
	Name() string
}

// DefaultController is the default implementation of Controller
type DefaultController struct {
	// name is the controller name
	name string
	// source is the event source that generates reconciliation events
	source Source
	// reconciler is the reconciler to use
	reconciler reconcile.Reconciler
	// queue is the work queue
	queue queue.RateLimitingInterface
	// predicates are the predicates to filter events
	predicates []predicate.Predicate
	// workerCount is the number of worker goroutines
	workerCount int
	// wg is the wait group for workers
	wg sync.WaitGroup
	// started indicates whether the controller has been started
	started bool
	// stopChan is used to signal workers to stop
	stopChan chan struct{}
	// recorder records the reconcile outcome per device+path after each run.
	// Optional: nil means outcomes are not recorded (degradation, R08).
	recorder status.Recorder
}

// SetStatusRecorder injects the reconcile-status recorder. Implements
// status.RecorderSetter so the Manager can wire the shared store at
// AddController time without changing existing call sites.
func (c *DefaultController) SetStatusRecorder(r status.Recorder) {
	c.recorder = r
}

// New creates a new DefaultController
func New(
	name string,
	s Source,
	r reconcile.Reconciler,
	q queue.RateLimitingInterface,
	predicates []predicate.Predicate,
	workerCount int,
) *DefaultController {
	if workerCount <= 0 {
		workerCount = 1
	}

	return &DefaultController{
		name:        name,
		source:      s,
		reconciler:  r,
		queue:       q,
		predicates:  predicates,
		workerCount: workerCount,
		stopChan:    make(chan struct{}),
	}
}

// Name implements Controller interface
func (c *DefaultController) Name() string {
	return c.name
}

// Start implements Controller interface
func (c *DefaultController) Start(ctx context.Context) error {
	if c.started {
		return nil
	}

	c.started = true

	// Start the event source if present
	if c.source != nil {
		if err := c.source.Start(ctx, c); err != nil {
			return err
		}
	}

	// Start worker goroutines
	for i := 0; i < c.workerCount; i++ {
		c.wg.Add(1)
		go c.worker(ctx)
	}

	return nil
}

// Stop implements Controller interface
func (c *DefaultController) Stop() error {
	if !c.started {
		return nil
	}

	c.queue.ShutDown()
	close(c.stopChan)
	c.wg.Wait()
	c.started = false

	return nil
}

// Enqueue implements Controller interface
func (c *DefaultController) Enqueue(evt predicate.Event) {
	// Apply all predicates
	for _, p := range c.predicates {
		var ok bool
		switch evt.Type {
		case predicate.CreateEvent:
			ok = p.Create(evt)
		case predicate.UpdateEvent:
			ok = p.Update(evt)
		case predicate.DeleteEvent:
			ok = p.Delete(evt)
		case predicate.GenericEvent:
			ok = p.Generic(evt)
		}
		if !ok {
			return
		}
	}

	// Convert event to request and add to queue
	req := evt.ForRequest()
	c.queue.Add(req)
}

func (c *DefaultController) worker(ctx context.Context) {
	defer c.wg.Done()

	for {
		// Use a goroutine to get the next item so we can respond to stop immediately
		resultChan := make(chan struct {
			obj      interface{}
			shutdown bool
		}, 1)

		go func() {
			obj, shutdown := c.queue.Get()
			resultChan <- struct {
				obj      interface{}
				shutdown bool
			}{obj, shutdown}
		}()

		select {
		case <-c.stopChan:
			return
		case result := <-resultChan:
			if result.shutdown {
				return
			}
			// Process the request
			c.process(ctx, result.obj.(reconcile.Request))
			c.queue.Done(result.obj)
		}
	}
}

func (c *DefaultController) process(ctx context.Context, req reconcile.Request) {
	result := c.reconciler.Reconcile(ctx, req)

	c.recordOutcome(req, result)

	// Handle requeuing
	switch {
	case result.Requeue:
		if result.RequeueAfter > 0 {
			c.queue.AddAfter(req, result.RequeueAfter)
		} else {
			c.queue.AddRateLimited(req)
		}
	case result.Error != nil:
		// If there's an error but Requeue isn't explicitly set, still requeue with rate limiting
		c.queue.AddRateLimited(req)
	default:
		// Success - forget the entry for rate limiting
		c.queue.Forget(req)
	}
}

// recordOutcome maps a reconcile Result to a coarse outcome and records it.
// No-op when no recorder is set (R08 degradation).
func (c *DefaultController) recordOutcome(req reconcile.Request, result reconcile.Result) {
	if c.recorder == nil {
		return
	}
	var (
		outcome status.Outcome
		diff    int
	)
	switch {
	case result.Error != nil:
		outcome = status.OutcomeError
	case result.Requeue:
		outcome = status.OutcomeReconciling
	case result.Changes > 0:
		outcome = status.OutcomeDrifted
		diff = result.Changes
	default:
		outcome = status.OutcomeConverged
	}
	c.recorder.Record(req.DeviceID, req.Path, outcome, diff, result.Error)
}

// ControllerOptions contains options for creating a controller
type ControllerOptions struct {
	// WorkerCount is the number of worker goroutines
	WorkerCount int
}

// DefaultWorkerCount is the default number of worker goroutines
const DefaultWorkerCount = 1
