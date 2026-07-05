package reconcile

import (
	"fmt"
	"time"
)

// Request contains the information needed for a reconciliation
// It carries the device ID and the path that triggered the reconciliation
type Request struct {
	// DeviceID is the unique identifier of the device (typically IP address)
	DeviceID string
	// Path is the YANG path that triggered the reconciliation
	Path string
}

// Result contains the result of a reconciliation
type Result struct {
	// Requeue indicates whether the request should be requeued after this reconciliation
	Requeue bool
	// RequeueAfter indicates how long to wait before requeuing
	RequeueAfter time.Duration
	// Error is the error that occurred during reconciliation
	Error error
	// Changes is how many changes this reconcile applied to align actual with
	// desired. 0 means already in sync (converged); >0 means drift was detected
	// and corrected (drifted). Lets the controller distinguish converged from
	// drifted when recording reconcile status.
	Changes int
}

// ConfigResult contains the result of comparing desired and actual configuration
type ConfigResult struct {
	// ChangesDetected indicates whether any changes were detected
	ChangesDetected bool
	// Changes is the list of changes to apply
	Changes []Change
}

// Change represents a configuration change that needs to be applied
type Change struct {
	// Path is the YANG path to the changed node
	Path string
	// Type is the type of change (add/delete/modify)
	Type string
	// DesiredValue is the desired value after change
	DesiredValue interface{}
	// ActualValue is the current actual value before change
	ActualValue interface{}
}

// ReconcileError wraps an error that occurred during reconciliation with additional context
type ReconcileError struct {
	// DeviceID is the device where the error occurred
	DeviceID string
	// Path is the path where the error occurred
	Path string
	// Err is the underlying error
	Err error
}

// Error implements the error interface
func (e *ReconcileError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("reconciliation failed for device %q path %q: %v", e.DeviceID, e.Path, e.Err)
	}
	return fmt.Sprintf("reconciliation failed for device %q: %v", e.DeviceID, e.Err)
}

// Unwrap returns the underlying error
func (e *ReconcileError) Unwrap() error {
	return e.Err
}
