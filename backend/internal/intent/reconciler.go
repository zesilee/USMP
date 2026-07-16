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
}

// NewReconciler builds an intent reconciler over a controller-runtime client.
func NewReconciler(c client.Client) *Reconciler {
	return &Reconciler{client: c, now: time.Now}
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

	// 每设备初始 pending；跨设备 2PC 下发（波次 7）在此之后接管并翻转 phase。
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

// crKey parses a queue DeviceID ("namespace/name" or bare "name") into a CR key.
func crKey(deviceID string) types.NamespacedName {
	if i := strings.IndexByte(deviceID, '/'); i >= 0 {
		return types.NamespacedName{Namespace: deviceID[:i], Name: deviceID[i+1:]}
	}
	return types.NamespacedName{Namespace: "default", Name: deviceID}
}
