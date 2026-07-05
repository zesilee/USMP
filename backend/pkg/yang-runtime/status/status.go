// Package status tracks the most recent reconciliation outcome per
// device+path so it can be surfaced (e.g. via the API) without persisting
// anything to a database (R03). It is an in-memory, concurrency-safe store
// (R09) and degrades to "unknown" rather than erroring when a device+path
// has never been reconciled (R08).
package status

import (
	"sync"
	"time"
)

// Outcome is the coarse reconciliation state of a device+path.
type Outcome string

const (
	// OutcomeUnknown means the device+path has never been reconciled.
	OutcomeUnknown Outcome = "unknown"
	// OutcomeConverged means the last run found desired == actual (no changes).
	OutcomeConverged Outcome = "converged"
	// OutcomeDrifted means the last run detected drift and applied changes.
	OutcomeDrifted Outcome = "drifted"
	// OutcomeReconciling means a run is pending/requeued (in progress).
	OutcomeReconciling Outcome = "reconciling"
	// OutcomeError means the last run failed (device unreachable / apply failed).
	OutcomeError Outcome = "error"
)

// Status is the recorded outcome of the most recent reconciliation for a
// single device+path.
type Status struct {
	DeviceID  string    `json:"device_id"`
	Path      string    `json:"path"`
	Outcome   Outcome   `json:"outcome"`
	DiffCount int       `json:"diff_count"`
	LastRun   time.Time `json:"last_run"`
	LastError string    `json:"last_error,omitempty"`
}

// Recorder is the write side of the store. The controller records the outcome
// after each reconcile. Kept minimal so the controller depends only on this.
type Recorder interface {
	Record(deviceID, path string, outcome Outcome, diffCount int, err error)
}

// RecorderSetter is implemented by controllers that can accept a Recorder.
// The Manager injects the shared store via this optional interface, so
// controllers that do not implement it simply do not record (degradation).
type RecorderSetter interface {
	SetStatusRecorder(Recorder)
}

// Store is an in-memory, concurrency-safe reconcile-status store.
// The zero value is not usable; construct with NewStore.
type Store struct {
	mu sync.RWMutex
	m  map[string]Status
}

// NewStore creates an empty Store.
func NewStore() *Store {
	return &Store{m: make(map[string]Status)}
}

func key(deviceID, path string) string { return deviceID + "|" + path }

// Record stores the outcome of a reconciliation for deviceID+path, stamping
// the current time. A nil err clears LastError.
func (s *Store) Record(deviceID, path string, outcome Outcome, diffCount int, err error) {
	st := Status{
		DeviceID:  deviceID,
		Path:      path,
		Outcome:   outcome,
		DiffCount: diffCount,
		LastRun:   time.Now(),
	}
	if err != nil {
		st.LastError = err.Error()
	}
	s.mu.Lock()
	s.m[key(deviceID, path)] = st
	s.mu.Unlock()
}

// Get returns the recorded status for deviceID+path. found is false when the
// pair has never been reconciled; callers should treat that as OutcomeUnknown.
func (s *Store) Get(deviceID, path string) (st Status, found bool) {
	s.mu.RLock()
	st, found = s.m[key(deviceID, path)]
	s.mu.RUnlock()
	return
}

// ListByDevice returns all recorded statuses for a single device.
func (s *Store) ListByDevice(deviceID string) []Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Status, 0)
	for _, st := range s.m {
		if st.DeviceID == deviceID {
			out = append(out, st)
		}
	}
	return out
}

// Snapshot returns a copy of every recorded status.
func (s *Store) Snapshot() []Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Status, 0, len(s.m))
	for _, st := range s.m {
		out = append(out, st)
	}
	return out
}
