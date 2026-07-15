package intent

import (
	"context"
	"log"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
)

// Reconciler is the C3 reconciler for BusinessVlanService intents (BIO-01):
// fetch CR → decode+validate (admission-by-watch backstop, BIC-03) → expand
// (BIO-02) → status writeback (BIC-04). The cross-device 2PC push (BIO-03)
// plugs in behind the expansion (tasks 波次 7).
type Reconciler struct {
	client client.Client
	now    func() time.Time

	// Push wiring (BIO-03/BIO-04) — nil pusher means "no push stage" (unit
	// tests of the validate/status skeleton; devices stay pending).
	pusher  Pusher
	cs      reconcile.ConfigStore
	trigger func(deviceID, path string) bool
}

// pushRetryBackoff is the requeue delay after a failed cross-device push.
const pushRetryBackoff = 30 * time.Second

// NewReconciler builds an intent reconciler over a controller-runtime client.
func NewReconciler(c client.Client) *Reconciler {
	return &Reconciler{client: c, now: time.Now}
}

// WithPush wires the cross-device transaction stage: pusher executes the 2PC,
// cs receives desired fragments after success, trigger enqueues native
// reconciliation (manager.TriggerReconcile).
func (r *Reconciler) WithPush(p Pusher, cs reconcile.ConfigStore, trigger func(deviceID, path string) bool) *Reconciler {
	r.pusher, r.cs, r.trigger = p, cs, trigger
	return r
}

// Reconcile implements reconcile.Reconciler. Request.DeviceID carries the CR
// key ("namespace/name") — intent reconciles are CR-scoped, not device-scoped.
func (r *Reconciler) Reconcile(ctx context.Context, req reconcile.Request) reconcile.Result {
	key := crKey(req.DeviceID)

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(GVK)
	if err := r.client.Get(ctx, key, u); err != nil {
		if apierrors.IsNotFound(err) {
			// CR 已消失：删除生命周期由 finalizer 展开处理（BIO-05）；这里无事可做。
			return reconcile.Result{}
		}
		return reconcile.Result{Error: err}
	}
	gen := u.GetGeneration()

	spec, err := DecodeSpec(u)
	var frags []Fragment
	var claims []Claim
	if err == nil {
		frags, claims, err = ExpandBusinessVlan(spec)
	}
	if err != nil {
		// 非法意图：status 呈现原因、零展开零下发（R08 不崩溃）；不 requeue——
		// 非法直到 spec 变更（generation 变化会再触发）。
		msg := err.Error()
		if statusErr := updateStatusWithRetry(ctx, r.client, key, func(u *unstructured.Unstructured) {
			_ = unstructured.SetNestedField(u.Object, gen, "status", "observedGeneration")
			_ = unstructured.SetNestedSlice(u.Object, []interface{}{}, "status", "claims")
			setCondition(u, CondValidated, "False", "InvalidSpec", msg, r.now())
			setCondition(u, CondConverged, "False", "NotValidated", "spec failed validation", r.now())
		}); statusErr != nil {
			return reconcile.Result{Error: statusErr}
		}
		log.Printf("intent: %s generation %d invalid: %v", key, gen, err)
		return reconcile.Result{}
	}

	// 无推送级（骨架/单测模式）：每设备 pending，等待接线。
	if r.pusher == nil {
		states := map[string]deviceState{}
		for _, f := range frags {
			if _, ok := states[f.Device]; !ok {
				states[f.Device] = deviceState{phase: PhasePending, reason: "awaiting push"}
			}
		}
		if statusErr := updateStatusWithRetry(ctx, r.client, key, func(u *unstructured.Unstructured) {
			_ = unstructured.SetNestedField(u.Object, gen, "status", "observedGeneration")
			_ = unstructured.SetNestedSlice(u.Object, claimsToStatus(claims), "status", "claims")
			_ = unstructured.SetNestedSlice(u.Object, deviceStatesToStatus(states, r.now()), "status", "deviceStates")
			setCondition(u, CondValidated, "True", "SpecValid", "", r.now())
			setCondition(u, CondConverged, "Unknown", "PushPending", "cross-device push not yet wired (BIO-03)", r.now())
		}); statusErr != nil {
			return reconcile.Result{Error: statusErr}
		}
		return reconcile.Result{}
	}

	// 幂等短路（A5）：同代且已收敛 → 不重推事务；仍重写 desired 并触发原生对账
	//（BIO-04 稳态：意图周期 resync 对冲 desired TTL 过期）。
	if observedGeneration(u) == gen && conditionTrue(u, CondConverged) {
		writeDesired(r.cs, r.trigger, frags)
		return reconcile.Result{}
	}

	// 跨设备 2PC（BIO-03）：全体成功才写 desired；任何失败不留 desired。
	results := r.pusher.Push(ctx, frags)
	allSynced := true
	states := map[string]deviceState{}
	for dev, res := range results {
		switch {
		case res.Err != nil:
			allSynced = false
			states[dev] = deviceState{phase: PhaseFailed, reason: res.Err.Error()}
		case res.NonTransactional:
			states[dev] = deviceState{phase: PhaseSynced, reason: "non-transactional push (:confirmed-commit unsupported)"}
		default:
			states[dev] = deviceState{phase: PhaseSynced, reason: ""}
		}
	}

	if allSynced {
		writeDesired(r.cs, r.trigger, frags)
	}

	if statusErr := updateStatusWithRetry(ctx, r.client, key, func(u *unstructured.Unstructured) {
		_ = unstructured.SetNestedField(u.Object, gen, "status", "observedGeneration")
		_ = unstructured.SetNestedSlice(u.Object, claimsToStatus(claims), "status", "claims")
		_ = unstructured.SetNestedSlice(u.Object, deviceStatesToStatus(states, r.now()), "status", "deviceStates")
		setCondition(u, CondValidated, "True", "SpecValid", "", r.now())
		if allSynced {
			setCondition(u, CondConverged, "True", "PushSucceeded", "", r.now())
		} else {
			setCondition(u, CondConverged, "False", "PushFailed", "one or more devices failed; retrying with backoff", r.now())
		}
	}); statusErr != nil {
		return reconcile.Result{Error: statusErr}
	}
	if !allSynced {
		return reconcile.Result{RequeueAfter: pushRetryBackoff}
	}
	return reconcile.Result{}
}

// observedGeneration reads status.observedGeneration (0 when unset).
func observedGeneration(u *unstructured.Unstructured) int64 {
	gen, _, _ := unstructured.NestedInt64(u.Object, "status", "observedGeneration")
	return gen
}

// conditionTrue reports whether the named condition is True in status.
func conditionTrue(u *unstructured.Unstructured, condType string) bool {
	conds, _, _ := unstructured.NestedSlice(u.Object, "status", "conditions")
	for _, c := range conds {
		if m, ok := c.(map[string]interface{}); ok && m["type"] == condType {
			return m["status"] == "True"
		}
	}
	return false
}

// crKey parses a queue DeviceID ("namespace/name" or bare "name") into a CR key.
func crKey(deviceID string) types.NamespacedName {
	if i := strings.IndexByte(deviceID, '/'); i >= 0 {
		return types.NamespacedName{Namespace: deviceID[:i], Name: deviceID[i+1:]}
	}
	return types.NamespacedName{Namespace: "default", Name: deviceID}
}
