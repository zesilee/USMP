package intent

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
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
	cleaner Cleaner
	cs      reconcile.ConfigStore
	trigger func(deviceID, path string) bool
}

// pushRetryBackoff is the requeue delay after a failed cross-device push.
const pushRetryBackoff = 30 * time.Second

// Finalizer gates CR deletion until claimed device config is cleaned (BIO-05).
const Finalizer = "biz.usmp.io/cleanup"

// NewReconciler builds an intent reconciler over a controller-runtime client.
func NewReconciler(c client.Client) *Reconciler {
	return &Reconciler{client: c, now: time.Now}
}

// WithPush wires the cross-device transaction stage: pusher executes the 2PC,
// cleaner runs the DELETE command channel (finalizer 删除/收缩差集), cs
// receives desired fragments after success, trigger enqueues native
// reconciliation (manager.TriggerReconcile).
func (r *Reconciler) WithPush(p Pusher, cl Cleaner, cs reconcile.ConfigStore, trigger func(deviceID, path string) bool) *Reconciler {
	r.pusher, r.cleaner, r.cs, r.trigger = p, cl, cs, trigger
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
	// 删除生命周期（BIO-05）：finalizer 拦截，清理认领配置后放行。
	if u.GetDeletionTimestamp() != nil {
		return r.reconcileDelete(ctx, key, u)
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

	// finalizer 先行：一旦开始下发就必须能拦住删除（BIO-05）。
	if err := r.ensureFinalizer(ctx, key); err != nil {
		return reconcile.Result{Error: err}
	}

	// 幂等短路（A5）：同代且已收敛 → 不重推事务；仍重写 desired 并触发原生对账
	//（BIO-04 稳态：意图周期 resync 对冲 desired TTL 过期）。
	if observedGeneration(u) == gen && conditionTrue(u, CondConverged) {
		writeDesired(r.cs, r.trigger, frags)
		return reconcile.Result{}
	}

	prevClaims := claimsFromStatus(u)

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

	// 收缩差集（BIO-06）：上一代认领 − 本代认领 = 孤儿，先 scrub desired（防周期
	// 对账推回）再走 DELETE 命令通道。差集只依赖 CR status（多实例可重放）。
	cleanupFailures := map[string]error{}
	if allSynced {
		if removed := subtractClaims(prevClaims, claims); len(removed) > 0 {
			removeClaimsFromDesired(r.cs, removed)
			if r.cleaner != nil {
				cleanupFailures = r.cleaner.Cleanup(ctx, removed)
			}
		}
		writeDesired(r.cs, r.trigger, frags)
	}

	// 认领只有在推送+清理全成后才推进到本代；否则保留上一代（重放差集重试清理）。
	statusClaims := claims
	if !allSynced || len(cleanupFailures) > 0 {
		statusClaims = prevClaims
	}
	for dev, err := range cleanupFailures {
		states[dev] = deviceState{phase: PhaseFailed, reason: "orphan cleanup failed: " + err.Error()}
	}

	if statusErr := updateStatusWithRetry(ctx, r.client, key, func(u *unstructured.Unstructured) {
		_ = unstructured.SetNestedField(u.Object, gen, "status", "observedGeneration")
		_ = unstructured.SetNestedSlice(u.Object, claimsToStatus(statusClaims), "status", "claims")
		_ = unstructured.SetNestedSlice(u.Object, deviceStatesToStatus(states, r.now()), "status", "deviceStates")
		setCondition(u, CondValidated, "True", "SpecValid", "", r.now())
		switch {
		case allSynced && len(cleanupFailures) == 0:
			setCondition(u, CondConverged, "True", "PushSucceeded", "", r.now())
		case allSynced:
			setCondition(u, CondConverged, "False", "CleanupPending", "orphan cleanup failed on some devices; retrying", r.now())
		default:
			setCondition(u, CondConverged, "False", "PushFailed", "one or more devices failed; retrying with backoff", r.now())
		}
	}); statusErr != nil {
		return reconcile.Result{Error: statusErr}
	}
	if !allSynced || len(cleanupFailures) > 0 {
		return reconcile.Result{RequeueAfter: pushRetryBackoff}
	}
	return reconcile.Result{}
}

// reconcileDelete runs the deletion lifecycle (BIO-05): scrub desired, clean
// claimed device config through the DELETE command channel, then release the
// finalizer. Partial failure keeps the finalizer and retries with backoff.
func (r *Reconciler) reconcileDelete(ctx context.Context, key types.NamespacedName, u *unstructured.Unstructured) reconcile.Result {
	if !hasFinalizer(u) {
		return reconcile.Result{}
	}
	claims := claimsFromStatus(u)
	removeClaimsFromDesired(r.cs, claims)

	var failures map[string]error
	if r.cleaner != nil && len(claims) > 0 {
		failures = r.cleaner.Cleanup(ctx, claims)
	}
	if len(failures) > 0 {
		states := map[string]deviceState{}
		for dev, err := range failures {
			states[dev] = deviceState{phase: PhaseFailed, reason: "cleanup failed: " + err.Error()}
		}
		if statusErr := updateStatusWithRetry(ctx, r.client, key, func(u *unstructured.Unstructured) {
			_ = unstructured.SetNestedSlice(u.Object, deviceStatesToStatus(states, r.now()), "status", "deviceStates")
			setCondition(u, CondConverged, "False", "DeleteCleanupFailed", "device cleanup failed; finalizer retained, retrying", r.now())
		}); statusErr != nil {
			return reconcile.Result{Error: statusErr}
		}
		return reconcile.Result{RequeueAfter: pushRetryBackoff}
	}

	if err := r.mutateWithRetry(ctx, key, func(u *unstructured.Unstructured) {
		var kept []string
		for _, f := range u.GetFinalizers() {
			if f != Finalizer {
				kept = append(kept, f)
			}
		}
		u.SetFinalizers(kept)
	}); err != nil && !apierrors.IsNotFound(err) {
		return reconcile.Result{Error: err}
	}
	log.Printf("intent: %s cleaned up on all devices, finalizer released", key)
	return reconcile.Result{}
}

// ensureFinalizer adds the cleanup finalizer if absent.
func (r *Reconciler) ensureFinalizer(ctx context.Context, key types.NamespacedName) error {
	return r.mutateWithRetry(ctx, key, func(u *unstructured.Unstructured) {
		for _, f := range u.GetFinalizers() {
			if f == Finalizer {
				return
			}
		}
		u.SetFinalizers(append(u.GetFinalizers(), Finalizer))
	})
}

// mutateWithRetry applies mutate to the latest CR under optimistic concurrency.
func (r *Reconciler) mutateWithRetry(ctx context.Context, key types.NamespacedName, mutate func(*unstructured.Unstructured)) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(GVK)
		if err := r.client.Get(ctx, key, u); err != nil {
			return err
		}
		before := fmt.Sprintf("%v", u.GetFinalizers())
		mutate(u)
		if fmt.Sprintf("%v", u.GetFinalizers()) == before {
			return nil
		}
		return r.client.Update(ctx, u)
	})
}

// hasFinalizer reports whether the CR carries the cleanup finalizer.
func hasFinalizer(u *unstructured.Unstructured) bool {
	for _, f := range u.GetFinalizers() {
		if f == Finalizer {
			return true
		}
	}
	return false
}

// claimsFromStatus reads the persisted claims (previous expansion, BIO-06).
func claimsFromStatus(u *unstructured.Unstructured) []Claim {
	raw, _, _ := unstructured.NestedSlice(u.Object, "status", "claims")
	out := make([]Claim, 0, len(raw))
	for _, c := range raw {
		m, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		device, _ := m["device"].(string)
		module, _ := m["module"].(string)
		path, _ := m["path"].(string)
		if device == "" || path == "" {
			continue
		}
		out = append(out, Claim{Device: device, Module: module, Path: path})
	}
	return out
}

// subtractClaims returns the claims in prev that are absent from cur.
func subtractClaims(prev, cur []Claim) []Claim {
	seen := map[Claim]bool{}
	for _, c := range cur {
		seen[c] = true
	}
	var out []Claim
	for _, c := range prev {
		if !seen[c] {
			out = append(out, c)
		}
	}
	return out
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
