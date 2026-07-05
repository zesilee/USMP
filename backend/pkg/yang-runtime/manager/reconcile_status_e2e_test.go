package manager_test

import (
	"context"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/controller"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/predicate"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/queue"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/status"
	"github.com/stretchr/testify/assert"
)

// stubReconciler returns a fixed Result, letting us drive a real controller.
type stubReconciler struct{ result reconcile.Result }

func (s *stubReconciler) Reconcile(ctx context.Context, req reconcile.Request) reconcile.Result {
	return s.result
}

// TestReconcileStatus_EndToEnd wires a real Manager + real Controller + queue
// and asserts a reconcile outcome flows all the way into the manager's
// queryable status store: AddController injection -> worker -> process ->
// recordOutcome -> Store. Proves the whole PR-B1 write path together.
//
// Covers the two settling outcomes (Requeue:false, deterministic). The
// requeue-based outcomes (error/reconciling) keep re-enqueuing, so they are
// covered deterministically at the recordOutcome unit level instead
// (controller.TestProcess_OutcomePriority) rather than raced here.
func TestReconcileStatus_EndToEnd(t *testing.T) {
	cases := []struct {
		name     string
		result   reconcile.Result
		wantOut  status.Outcome
		wantDiff int
	}{
		{"drifted", reconcile.Result{Requeue: false, Changes: 3}, status.OutcomeDrifted, 3},
		{"converged", reconcile.Result{Requeue: false, Changes: 0}, status.OutcomeConverged, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := manager.New()
			q := queue.NewRateLimitingQueue(queue.DefaultRateLimiter())
			rec := &stubReconciler{result: tc.result}
			c := controller.New("vlan", nil, rec, q, nil, 1)

			mgr.AddController(c) // wires controller to the manager's shared store

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			assert.NoError(t, c.Start(ctx))
			defer c.Stop()

			c.Enqueue(predicate.Event{
				DeviceID: "10.0.0.1",
				Path:     "/vlans",
				Type:     predicate.UpdateEvent,
			})

			var (
				st status.Status
				ok bool
			)
			for i := 0; i < 100; i++ {
				if st, ok = mgr.GetReconcileStatus().Get("10.0.0.1", "/vlans"); ok {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}

			assert.True(t, ok, "reconcile outcome should be recorded and queryable via the manager")
			assert.Equal(t, tc.wantOut, st.Outcome)
			assert.Equal(t, tc.wantDiff, st.DiffCount)
		})
	}
}
