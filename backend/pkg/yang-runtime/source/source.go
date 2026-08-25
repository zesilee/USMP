package source

import (
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/controller"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/predicate"
)

// BaseSource provides the common base implementation shared by all sources.
type BaseSource struct {
	controller controller.Controller
}

// EnqueueEvent enqueues an event to the controller, or drops it if no
// controller is attached yet.
func (b *BaseSource) EnqueueEvent(evt predicate.Event) {
	if b.controller != nil {
		b.controller.Enqueue(evt)
	}
}
