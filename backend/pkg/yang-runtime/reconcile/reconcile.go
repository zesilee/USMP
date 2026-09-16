package reconcile

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"
)

// SyncTracker is an optional ConfigStore extension exposing the desired
// lifecycle (YR-09): an entry is *pending* from the moment it is written until
// a reconcile confirms the device matches it; only then does its TTL start.
// gen is the store's write generation, so a confirmation read against an
// older write cannot mark a newer one as synced. Stores that do not implement
// it get the pre-lifecycle behaviour (test doubles, third-party stores).
type SyncTracker interface {
	// Track reports the write generation, when the entry became pending and
	// whether it is still pending. ok is false when nothing is stored.
	Track(deviceID, path string) (gen uint64, pendingSince time.Time, pending, ok bool)
	// MarkSynced starts the TTL of the entry if it is still at gen.
	MarkSynced(deviceID, path string, gen uint64) bool
	// Abandon deletes the entry if it is still at gen.
	Abandon(deviceID, path string, gen uint64) bool
}

// ErrDesiredAbandoned is wrapped into the terminal error returned when a
// pending desired could not be delivered within AbandonAfter (YR-09).
var ErrDesiredAbandoned = errors.New("desired 已放弃（超过放弃上限仍未送达设备）")

// DefaultAbandonAfter is how long a pending desired may keep failing before it
// is abandoned (see USMP_DESIRED_ABANDON_AFTER).
const DefaultAbandonAfter = 30 * time.Minute

// abandonAfterEnv is the env var overriding DefaultAbandonAfter (Go duration).
const abandonAfterEnv = "USMP_DESIRED_ABANDON_AFTER"

// abandonAfterNanos is 0 until first use; resolved lazily so tests and main can
// re-resolve after changing the environment (SetAbandonAfter(0)).
var abandonAfterNanos int64

// AbandonAfter returns the effective abandon limit.
func AbandonAfter() time.Duration {
	if n := atomic.LoadInt64(&abandonAfterNanos); n > 0 {
		return time.Duration(n)
	}
	d := parseAbandonAfter(os.Getenv(abandonAfterEnv))
	atomic.StoreInt64(&abandonAfterNanos, int64(d))
	return d
}

// SetAbandonAfter overrides the abandon limit; 0 resets to env/default.
func SetAbandonAfter(d time.Duration) {
	atomic.StoreInt64(&abandonAfterNanos, int64(d))
}

// parseAbandonAfter parses the env value; empty → default, invalid or
// non-positive → default with a warning (R08: never fail startup over a knob).
func parseAbandonAfter(raw string) time.Duration {
	if raw == "" {
		return DefaultAbandonAfter
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		log.Printf("reconcile: %s=%q 非法，回退默认 %s", abandonAfterEnv, raw, DefaultAbandonAfter)
		return DefaultAbandonAfter
	}
	return d
}

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
	configStore  ConfigStore
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
		configStore:  cs,
		deviceClient: dc,
		diffEngine:   de,
	}
}

// Reconcile implements the Reconciler interface
func (g *GenericReconciler) Reconcile(ctx context.Context, req Request) Result {
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

	// Nothing declared for this path → no-op. This is deliberately NOT turned
	// into a deletion: BIO-05 puts deletes on the DELETE command channel, and
	// 「声明式通道不承载删除」is the contract, not an oversight.
	//
	// The diff engine does have a desiredNil → DeleteChange branch, which serves
	// that command channel. Do not "fix" this early return to reach it: every
	// expired or failed desired read would then be translated into wiping live
	// device config. Locked by reconcile_desired_absent_test.go.
	if desired == nil {
		return Result{NoDesired: true}
	}

	// Lifecycle snapshot (YR-09): the generation read here is what MarkSynced /
	// Abandon are guarded with, so a user write landing mid-reconcile can never
	// be marked synced (and expire) on the strength of this run.
	lc := g.trackLifecycle(req)

	actual, err := g.deviceClient.Get(ctx, req.DeviceID)
	if err != nil {
		return g.fail(req, lc, err)
	}

	changes, err := g.diffEngine.Diff(desired, actual, req.Path)
	if err != nil {
		return g.fail(req, lc, err)
	}

	if len(changes) == 0 {
		// Device matches intent: delivery is confirmed, the TTL clock starts now.
		// Only pending entries are marked so a periodic converged read never
		// extends an already-synced entry (口径 A: 送到就撒手).
		if lc.tracker != nil && lc.pending {
			lc.tracker.MarkSynced(req.DeviceID, req.Path, lc.gen)
		}
		return Result{}
	}

	if err := g.deviceClient.Set(ctx, req.DeviceID, changes); err != nil {
		return g.fail(req, lc, err)
	}

	// All changes applied successfully; report the drift that was corrected.
	// Not marked synced yet: YR-05 requeues a re-verify, whose zero-change run
	// is the confirmation.
	return Result{Changes: len(changes)}
}

// lifecycle is the per-run snapshot of the desired entry's sync state.
type lifecycle struct {
	tracker SyncTracker
	gen     uint64
	since   time.Time
	pending bool
}

func (g *GenericReconciler) trackLifecycle(req Request) lifecycle {
	tracker, ok := g.configStore.(SyncTracker)
	if !ok {
		return lifecycle{}
	}
	gen, since, pending, found := tracker.Track(req.DeviceID, req.Path)
	if !found {
		return lifecycle{}
	}
	return lifecycle{tracker: tracker, gen: gen, since: since, pending: pending}
}

// fail turns a device/diff error into a Result. A pending desired that has
// been failing longer than AbandonAfter is abandoned: deleted from the store
// and reported as a terminal error so the controller records it and stops
// retrying — never silently "converged" (YR-09). Abandon is generation-guarded:
// if the user rewrote the entry mid-run the newer value keeps its own clock.
func (g *GenericReconciler) fail(req Request, lc lifecycle, cause error) Result {
	if lc.tracker != nil && lc.pending {
		if waited := time.Since(lc.since); waited > AbandonAfter() {
			if lc.tracker.Abandon(req.DeviceID, req.Path, lc.gen) {
				return Result{
					Terminal: true,
					Error: &ReconcileError{
						DeviceID: req.DeviceID,
						Path:     req.Path,
						Err:      fmt.Errorf("%w：等待 %s，最后错误: %v", ErrDesiredAbandoned, waited.Round(time.Second), cause),
					},
				}
			}
		}
	}
	return Result{
		Requeue: true,
		Error: &ReconcileError{
			DeviceID: req.DeviceID,
			Path:     req.Path,
			Err:      cause,
		},
	}
}
