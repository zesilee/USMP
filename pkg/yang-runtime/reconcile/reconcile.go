package reconcile

import (
	"context"
)

// Reconciler is the interface that must be implemented by all reconcilers
// A reconciler compares the desired configuration with the actual configuration
// on the device and applies necessary changes to align them.
type Reconciler interface {
	// Reconcile performs the reconciliation
	// It takes a Request and returns a Result indicating what should happen next
	Reconcile(ctx context.Context, req Request) Result
}

// ReconcilerFunc is a function type that implements Reconciler
type ReconcilerFunc func(ctx context.Context, req Request) Result

// Reconcile implements the Reconciler interface
func (f ReconcilerFunc) Reconcile(ctx context.Context, req Request) Result {
	return f(ctx, req)
}

// ConfigStore is the interface for accessing desired configuration state
// The reconciler reads the desired state from here and compares it with
// the actual state from the device.
type ConfigStore interface {
	// Get retrieves the desired configuration at the given path for a device
	Get(deviceID, path string) (interface{}, error)
	// Set stores the desired configuration at the given path for a device
	Set(deviceID, path string, value interface{}) error
	// Delete removes the desired configuration at the given path for a device
	Delete(deviceID, path string) error
	// List lists all paths that have desired configuration for a device
	List(deviceID string) ([]string, error)
	// ListDevices lists all devices that have desired configuration
	ListDevices() ([]string, error)
}

// DeviceClient is the interface for accessing the actual device configuration
// This is typically implemented by the client package.
type DeviceClient interface {
	// Get retrieves the actual configuration from the device at the given path
	Get(ctx context.Context, deviceID string) (interface{}, error)
	// Set applies configuration changes to the device
	Set(ctx context.Context, deviceID string, changes []Change) error
}

// GenericReconciler is a base implementation of Reconciler that handles the common
// reconciliation pattern: get desired, get actual, compute diff, apply changes.
type GenericReconciler struct {
	configStore ConfigStore
	deviceClient DeviceClient
	diffEngine   DiffEngine
}

// DiffEngine is the interface for computing the difference between desired and actual configuration
type DiffEngine interface {
	// Diff computes the difference between desired and actual configuration
	Diff(desired, actual interface{}, path string) ([]Change, error)
}

// NewGenericReconciler creates a new GenericReconciler
func NewGenericReconciler(
	cs ConfigStore,
	dc DeviceClient,
	de DiffEngine,
) *GenericReconciler {
	return &GenericReconciler{
		configStore: cs,
		deviceClient: dc,
		diffEngine:   de,
	}
}

// Reconcile implements the Reconciler interface
func (g *GenericReconciler) Reconcile(ctx context.Context, req Request) Result {
	// Get desired configuration from config store
	desired, err := g.configStore.Get(req.DeviceID, req.Path)
	if err != nil {
		return Result{
			Requeue: true,
			Error: &ReconcileError{
				DeviceID: req.DeviceID,
				Path:     req.Path,
				Err:      err,
			},
		}
	}

	// If desired is nil, it means the configuration should be deleted
	if desired == nil {
		// No changes needed if it's already gone
		return Result{
			Requeue: false,
			Error:   nil,
		}
	}

	// Get actual configuration from device
	actual, err := g.deviceClient.Get(ctx, req.DeviceID)
	if err != nil {
		return Result{
			Requeue: true,
			Error: &ReconcileError{
				DeviceID: req.DeviceID,
				Path:     req.Path,
				Err:      err,
			},
		}
	}

	// Compute diff
	changes, err := g.diffEngine.Diff(desired, actual, req.Path)
	if err != nil {
		return Result{
			Requeue: true,
			Error: &ReconcileError{
				DeviceID: req.DeviceID,
				Path:     req.Path,
				Err:      err,
			},
		}
	}

	// If no changes, we're done
	if len(changes) == 0 {
		return Result{
			Requeue: false,
			Error:   nil,
		}
	}

	// Apply changes to device
	if err := g.deviceClient.Set(ctx, req.DeviceID, changes); err != nil {
		return Result{
			Requeue: true,
			Error: &ReconcileError{
				DeviceID: req.DeviceID,
				Path:     req.Path,
				Err:      err,
			},
		}
	}

	// All changes applied successfully
	return Result{
		Requeue: false,
		Error:   nil,
	}
}
