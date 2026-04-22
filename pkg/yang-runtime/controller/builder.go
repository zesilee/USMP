package controller

import (
	"github.com/leezesi/usmp/pkg/yang-runtime/predicate"
	"github.com/leezesi/usmp/pkg/yang-runtime/queue"
	"github.com/leezesi/usmp/pkg/yang-runtime/reconcile"
)

// Builder builds a controller
type Builder struct {
	name        string
	reconciler  reconcile.Reconciler
	source      Source
	predicates  []predicate.Predicate
	workerCount int
	queue       queue.RateLimitingInterface
}

// ControllerManagedBy creates a new Builder for a controller with the given name
func ControllerManagedBy(name string) *Builder {
	return &Builder{
		name:        name,
		workerCount: DefaultWorkerCount,
	}
}

// WithReconciler sets the reconciler for the controller
func (b *Builder) WithReconciler(r reconcile.Reconciler) *Builder {
	b.reconciler = r
	return b
}

// WithSource sets the event source for the controller
func (b *Builder) WithSource(s Source) *Builder {
	b.source = s
	return b
}

// WithPredicate adds a predicate to the controller
func (b *Builder) WithPredicate(p predicate.Predicate) *Builder {
	b.predicates = append(b.predicates, p)
	return b
}

// WithPredicates adds multiple predicates to the controller
func (b *Builder) WithPredicates(ps []predicate.Predicate) *Builder {
	b.predicates = append(b.predicates, ps...)
	return b
}

// WithWorkerCount sets the number of worker goroutines
func (b *Builder) WithWorkerCount(n int) *Builder {
	b.workerCount = n
	return b
}

// WithQueue sets a custom work queue
func (b *Builder) WithQueue(q queue.RateLimitingInterface) *Builder {
	b.queue = q
	return b
}

// Build builds and returns the controller
func (b *Builder) Build() Controller {
	q := b.queue
	if q == nil {
		q = queue.NewRateLimitingQueue(queue.DefaultRateLimiter())
	}

	return New(b.name, b.source, b.reconciler, q, b.predicates, b.workerCount)
}
