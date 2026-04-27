package predicate

import (
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
)

// Predicate is an interface for filtering events before they are enqueued
type Predicate interface {
	// Create returns true if the event should be processed
	Create(evt Event) bool
	// Update returns true if the event should be processed
	Update(evt Event) bool
	// Delete returns true if the event should be processed
	Delete(evt Event) bool
	// Generic returns true if the generic event should be processed
	Generic(evt Event) bool
}

// Event represents an event that triggered reconciliation
type Event struct {
	// DeviceID is the device that the event relates to
	DeviceID string
	// Path is the YANG path that the event relates to
	Path string
	// Type is the event type (create, update, delete, generic)
	Type EventType
	// Metadata is additional metadata about the event
	Metadata map[string]interface{}
}

// EventType represents the type of event
type EventType int

const (
	// CreateEvent represents a new object being created
	CreateEvent EventType = iota
	// UpdateEvent represents an existing object being updated
	UpdateEvent
	// DeleteEvent represents an existing object being deleted
	DeleteEvent
	// GenericEvent represents a generic/reconciliation-triggered event
	GenericEvent
)

// Funcs implements Predicate with function values for each method
type Funcs struct {
	CreateFunc  func(evt Event) bool
	UpdateFunc  func(evt Event) bool
	DeleteFunc  func(evt Event) bool
	GenericFunc func(evt Event) bool
}

// Create calls CreateFunc if set, otherwise returns true
func (f Funcs) Create(evt Event) bool {
	if f.CreateFunc == nil {
		return true
	}
	return f.CreateFunc(evt)
}

// Update calls UpdateFunc if set, otherwise returns true
func (f Funcs) Update(evt Event) bool {
	if f.UpdateFunc == nil {
		return true
	}
	return f.UpdateFunc(evt)
}

// Delete calls DeleteFunc if set, otherwise returns true
func (f Funcs) Delete(evt Event) bool {
	if f.DeleteFunc == nil {
		return true
	}
	return f.DeleteFunc(evt)
}

// Generic calls GenericFunc if set, otherwise returns true
func (f Funcs) Generic(evt Event) bool {
	if f.GenericFunc == nil {
		return true
	}
	return f.GenericFunc(evt)
}

// And returns a predicate that returns true only if all predicates return true
func And(predicates ...Predicate) Predicate {
	return Funcs{
		CreateFunc: func(evt Event) bool {
			for _, p := range predicates {
				if !p.Create(evt) {
					return false
				}
			}
			return true
		},
		UpdateFunc: func(evt Event) bool {
			for _, p := range predicates {
				if !p.Update(evt) {
					return false
				}
			}
			return true
		},
		DeleteFunc: func(evt Event) bool {
			for _, p := range predicates {
				if !p.Delete(evt) {
					return false
				}
			}
			return true
		},
		GenericFunc: func(evt Event) bool {
			for _, p := range predicates {
				if !p.Generic(evt) {
					return false
				}
			}
			return true
		},
	}
}

// Or returns a predicate that returns true if any predicate returns true
func Or(predicates ...Predicate) Predicate {
	return Funcs{
		CreateFunc: func(evt Event) bool {
			for _, p := range predicates {
				if p.Create(evt) {
					return true
				}
			}
			return false
		},
		UpdateFunc: func(evt Event) bool {
			for _, p := range predicates {
				if p.Update(evt) {
					return true
				}
			}
			return false
		},
		DeleteFunc: func(evt Event) bool {
			for _, p := range predicates {
				if p.Delete(evt) {
					return true
				}
			}
			return false
		},
		GenericFunc: func(evt Event) bool {
			for _, p := range predicates {
				if p.Generic(evt) {
					return true
				}
			}
			return false
		},
	}
}

// Not returns a predicate that negates the result of another predicate
func Not(p Predicate) Predicate {
	return Funcs{
		CreateFunc: func(evt Event) bool {
			return !p.Create(evt)
		},
		UpdateFunc: func(evt Event) bool {
			return !p.Update(evt)
		},
		DeleteFunc: func(evt Event) bool {
			return !p.Delete(evt)
		},
		GenericFunc: func(evt Event) bool {
			return !p.Generic(evt)
		},
	}
}

// Always returns a predicate that always accepts events
func Always() Predicate {
	return Funcs{
		CreateFunc:  func(evt Event) bool { return true },
		UpdateFunc:  func(evt Event) bool { return true },
		DeleteFunc:  func(evt Event) bool { return true },
		GenericFunc: func(evt Event) bool { return true },
	}
}

// Never returns a predicate that never accepts events
func Never() Predicate {
	return Funcs{
		CreateFunc:  func(evt Event) bool { return false },
		UpdateFunc:  func(evt Event) bool { return false },
		DeleteFunc:  func(evt Event) bool { return false },
		GenericFunc: func(evt Event) bool { return false },
	}
}

// ForRequest converts an Event to a Request for reconciliation
func (e Event) ForRequest() reconcile.Request {
	return reconcile.Request{
		DeviceID: e.DeviceID,
		Path:     e.Path,
	}
}
