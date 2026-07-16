package intent

import (
	"context"
	"sort"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Condition types on the intent CR status (BIC-04).
const (
	CondValidated = "Validated"
	CondConverged = "Converged"
)

// Device phases in status.deviceStates (BIC-04).
const (
	PhasePending = "pending"
	PhaseSynced  = "synced"
	PhaseFailed  = "failed"
)

// setCondition upserts a condition by type; lastTransitionTime moves only when
// the status value changes (K8s condition convention).
func setCondition(u *unstructured.Unstructured, condType, status, reason, message string, now time.Time) {
	conds, _, _ := unstructured.NestedSlice(u.Object, "status", "conditions")
	next := map[string]interface{}{
		"type":               condType,
		"status":             status,
		"reason":             reason,
		"message":            message,
		"lastTransitionTime": now.UTC().Format(time.RFC3339),
	}
	replaced := false
	for i, c := range conds {
		m, ok := c.(map[string]interface{})
		if !ok || m["type"] != condType {
			continue
		}
		if m["status"] == status {
			next["lastTransitionTime"] = m["lastTransitionTime"]
		}
		conds[i] = next
		replaced = true
	}
	if !replaced {
		conds = append(conds, next)
	}
	_ = unstructured.SetNestedSlice(u.Object, conds, "status", "conditions")
}

// claimsToStatus renders claims for the status subresource.
func claimsToStatus(claims []Claim) []interface{} {
	out := make([]interface{}, 0, len(claims))
	for _, c := range claims {
		out = append(out, map[string]interface{}{
			"device": c.Device,
			"module": c.Module,
			"path":   c.Path,
		})
	}
	return out
}

// deviceStatesToStatus renders per-device phases for the status subresource.
func deviceStatesToStatus(states map[string]deviceState, now time.Time) []interface{} {
	devices := make([]string, 0, len(states))
	for d := range states {
		devices = append(devices, d)
	}
	sort.Strings(devices)
	out := make([]interface{}, 0, len(devices))
	for _, d := range devices {
		s := states[d]
		out = append(out, map[string]interface{}{
			"device":         d,
			"phase":          s.phase,
			"reason":         s.reason,
			"lastTransition": now.UTC().Format(time.RFC3339),
		})
	}
	return out
}

type deviceState struct {
	phase  string
	reason string
}

// updateStatusWithRetry re-reads the CR and applies mutate under optimistic
// concurrency, retrying on conflict (status 由控制器独占写入，BIC-04).
func updateStatusWithRetry(ctx context.Context, c client.Client, key types.NamespacedName, mutate func(*unstructured.Unstructured)) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(GVK)
		if err := c.Get(ctx, key, u); err != nil {
			return err
		}
		// 归一化：status 键可能以显式 nil 存在（fake client 的普通 Update 会写入
		// "status": nil），SetNestedField 遇非 map 值会静默失败——先立空 map。
		if s, ok := u.Object["status"]; !ok || s == nil {
			u.Object["status"] = map[string]interface{}{}
		}
		mutate(u)
		// 原样返回错误：RetryOnConflict 依赖 apierrors.IsConflict 判定。
		return c.Status().Update(ctx, u)
	})
}
