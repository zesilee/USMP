package source

import (
	"github.com/leezesi/usmp/pkg/yang-runtime/controller"
	"github.com/leezesi/usmp/pkg/yang-runtime/predicate"
)

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
