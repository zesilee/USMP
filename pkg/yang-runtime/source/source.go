package source

import (
	"context"

	"github.com/leezesi/usmp/pkg/yang-runtime/controller"
	"github.com/leezesi/usmp/pkg/yang-runtime/predicate"
)

// Source is the interface for event sources
// An event source generates events that trigger reconciliation
type Source interface {
	// Start starts the event source and sends events to the controller
	Start(ctx context.Context, ctrl controller.Controller) error
	// Stop stops the event source
	Stop() error
}

// BaseSource provides common base implementation for all sources
type BaseSource struct {
	controller controller.Controller
}

// EnqueueEvent enqueues an event to the controller
func (b *BaseSource) EnqueueEvent(evt predicate.Event) {
	if b.controller != nil {
		b.controller.Enqueue(evt)
	}
}
