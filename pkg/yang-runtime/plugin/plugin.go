package plugin

import (
	"context"

	"github.com/leezesi/usmp/pkg/yang-runtime/reconcile"
)

// Plugin is the base interface for all plugins
type Plugin interface {
	// Name returns the plugin name
	Name() string
}

// ValidationPlugin validates configuration changes before they are applied
// It can reject changes that don't meet validation criteria
type ValidationPlugin interface {
	Plugin
	// Validate validates the proposed change
	// Returns an error if validation fails
	Validate(ctx context.Context, req reconcile.Request, change *Change) error
}

// Change represents a configuration change being mutated or validated
type Change struct {
	// Path is the YANG path to the changed node
	Path string
	// OldValue is the current value
	OldValue interface{}
	// NewValue is the proposed new value
	NewValue interface{}
	// DeviceID is the device identifier
	DeviceID string
}

// MutationPlugin mutates configuration changes before they are applied
// It can modify the proposed change to add defaults or fix conflicts
type MutationPlugin interface {
	Plugin
	// Mutate mutates the proposed change
	// Returns the modified change
	Mutate(ctx context.Context, req reconcile.Request, change *Change) (*Change, error)
}

// NotificationPlugin receives notifications after configuration changes are applied
// It can be used for logging, metrics, webhooks, etc.
type NotificationPlugin interface {
	Plugin
	// OnSuccess is called when a change is successfully applied
	OnSuccess(ctx context.Context, req reconcile.Request, change *Change)
	// OnFailure is called when a change fails to apply
	OnFailure(ctx context.Context, req reconcile.Request, change *Change, err error)
}

// ReconciliationHookPlugin provides hooks before and after reconciliation
type ReconciliationHookPlugin interface {
	Plugin
	// PreReconcile is called before reconciliation starts
	PreReconcile(ctx context.Context, req reconcile.Request) error
	// PostReconcile is called after reconciliation completes
	PostReconcile(ctx context.Context, req reconcile.Request, result reconcile.Result) error
}
