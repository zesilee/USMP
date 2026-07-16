package intent

import (
	"context"
	"strings"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	uns "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
)

// BIO-01/BIC-03/BIC-04 —— 意图 Reconciler：CR → 解码校验 → 展开 → status 回写
// （Validated/Converged 条件、observedGeneration、claims、deviceStates），
// 校验失败零展开零下发，status 写冲突自动重试。

type unstr = uns.Unstructured

func newCR(gen int64, spec map[string]interface{}) *unstr {
	u := &unstr{}
	u.SetGroupVersionKind(GVK)
	u.SetName("biz-100")
	u.SetNamespace("default")
	u.SetGeneration(gen)
	_ = uns.SetNestedMap(u.Object, spec, "spec")
	return u
}

func validSpec() map[string]interface{} {
	return map[string]interface{}{
		"vlan-id": int64(100),
		"name":    "office",
		"devices": []interface{}{
			map[string]interface{}{"ip": "10.0.0.1", "access-ports": []interface{}{"GE0/0/1"}},
			map[string]interface{}{"ip": "10.0.0.2", "trunk-ports": []interface{}{"GE0/0/2"}},
		},
	}
}

func newFakeClient(t *testing.T, objs ...client.Object) client.Client {
	t.Helper()
	sch := runtime.NewScheme()
	sch.AddKnownTypeWithName(GVK, &unstr{})
	sch.AddKnownTypeWithName(schema.GroupVersionKind{Group: GVK.Group, Version: GVK.Version, Kind: GVK.Kind + "List"}, &uns.UnstructuredList{})
	proto := &unstr{}
	proto.SetGroupVersionKind(GVK)
	return fake.NewClientBuilder().WithScheme(sch).WithObjects(objs...).WithStatusSubresource(proto).Build()
}

func getCR(t *testing.T, c client.Client) *unstr {
	t.Helper()
	u := &unstr{}
	u.SetGroupVersionKind(GVK)
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "default", Name: "biz-100"}, u); err != nil {
		t.Fatalf("get CR: %v", err)
	}
	return u
}

func condition(u *unstr, condType string) map[string]interface{} {
	conds, _, _ := uns.NestedSlice(u.Object, "status", "conditions")
	for _, c := range conds {
		m, ok := c.(map[string]interface{})
		if ok && m["type"] == condType {
			return m
		}
	}
	return nil
}

// 合法 CR：Validated=True、observedGeneration 对齐、claims 全量、每设备 pending。
func TestReconcileValidCRWritesStatus(t *testing.T) {
	cl := newFakeClient(t, newCR(1, validSpec()))
	r := NewReconciler(cl)

	res := r.Reconcile(context.Background(), reconcile.Request{DeviceID: "default/biz-100", Path: IntentPath})
	if res.Error != nil {
		t.Fatalf("reconcile: %v", res.Error)
	}

	u := getCR(t, cl)
	if gen, _, _ := uns.NestedInt64(u.Object, "status", "observedGeneration"); gen != 1 {
		t.Errorf("observedGeneration = %d, want 1", gen)
	}
	v := condition(u, "Validated")
	if v == nil || v["status"] != "True" {
		t.Fatalf("Validated condition = %v, want True", v)
	}
	if c := condition(u, "Converged"); c == nil || c["status"] == "True" {
		t.Fatalf("Converged condition = %v, want present and not True before push", c)
	}
	// 2 设备 vlan 条目 + 2 端口 = 4 条认领。
	claims, _, _ := uns.NestedSlice(u.Object, "status", "claims")
	if len(claims) != 4 {
		t.Errorf("claims len = %d, want 4: %v", len(claims), claims)
	}
	states, _, _ := uns.NestedSlice(u.Object, "status", "deviceStates")
	if len(states) != 2 {
		t.Fatalf("deviceStates len = %d, want 2: %v", len(states), states)
	}
	for _, s := range states {
		m := s.(map[string]interface{})
		if m["phase"] != "pending" {
			t.Errorf("deviceState %v phase = %v, want pending (wave 7 前不下发)", m["device"], m["phase"])
		}
	}
}

// 非法 payload（vlan-id 超 range，穿过 apiserver 直写场景）：Validated=False、零展开。
func TestReconcileInvalidPayloadNoExpansion(t *testing.T) {
	bad := validSpec()
	bad["vlan-id"] = int64(5000)
	cl := newFakeClient(t, newCR(1, bad))
	r := NewReconciler(cl)

	res := r.Reconcile(context.Background(), reconcile.Request{DeviceID: "default/biz-100", Path: IntentPath})
	if res.Error != nil {
		t.Fatalf("invalid payload should not error the queue (no requeue until spec changes): %v", res.Error)
	}

	u := getCR(t, cl)
	v := condition(u, "Validated")
	if v == nil || v["status"] != "False" {
		t.Fatalf("Validated = %v, want False", v)
	}
	if msg, _ := v["message"].(string); !strings.Contains(msg, "vlan-id") && !strings.Contains(msg, "5000") {
		t.Errorf("Validated message should mention offending field/value, got %q", msg)
	}
	if claims, _, _ := uns.NestedSlice(u.Object, "status", "claims"); len(claims) != 0 {
		t.Errorf("claims should be empty on invalid spec, got %v", claims)
	}
	if gen, _, _ := uns.NestedInt64(u.Object, "status", "observedGeneration"); gen != 1 {
		t.Errorf("observedGeneration = %d, want 1（非法 spec 也要对齐代际）", gen)
	}
}

// CR 不存在：无错误无 panic（删除生命周期由 finalizer 波次处理）。
func TestReconcileMissingCRNoop(t *testing.T) {
	cl := newFakeClient(t)
	r := NewReconciler(cl)
	res := r.Reconcile(context.Background(), reconcile.Request{DeviceID: "default/absent", Path: IntentPath})
	if res.Error != nil {
		t.Fatalf("missing CR should be a no-op, got %v", res.Error)
	}
}

// status 写冲突：RetryOnConflict 自动重试后成功。
func TestReconcileStatusConflictRetry(t *testing.T) {
	conflicted := false
	sch := runtime.NewScheme()
	sch.AddKnownTypeWithName(GVK, &unstr{})
	sch.AddKnownTypeWithName(schema.GroupVersionKind{Group: GVK.Group, Version: GVK.Version, Kind: GVK.Kind + "List"}, &uns.UnstructuredList{})
	proto := &unstr{}
	proto.SetGroupVersionKind(GVK)
	cl := fake.NewClientBuilder().WithScheme(sch).WithObjects(newCR(1, validSpec())).WithStatusSubresource(proto).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourceUpdate: func(ctx context.Context, c client.Client, sub string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				if !conflicted {
					conflicted = true
					return apierrors.NewConflict(schema.GroupResource{Group: GVK.Group, Resource: "businessvlanservices"}, obj.GetName(), nil)
				}
				return c.SubResource(sub).Update(ctx, obj, opts...)
			},
		}).Build()

	r := NewReconciler(cl)
	res := r.Reconcile(context.Background(), reconcile.Request{DeviceID: "default/biz-100", Path: IntentPath})
	if res.Error != nil {
		t.Fatalf("conflict should be retried, got %v", res.Error)
	}
	if !conflicted {
		t.Fatal("interceptor did not fire")
	}
	if v := condition(getCR(t, cl), "Validated"); v == nil || v["status"] != "True" {
		t.Fatal("status not written after conflict retry")
	}
}

// BIO-01 降级：无可用 kubeconfig 时 Register 返回 (nil,nil) 不报错，进程不受影响。
func TestRegisterDegradesWithoutKubeconfig(t *testing.T) {
	t.Setenv("KUBECONFIG", "/nonexistent/kubeconfig-for-degrade-test")
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("KUBERNETES_SERVICE_PORT", "")
	c, err := Register(nil)
	if err != nil {
		t.Fatalf("Register should degrade gracefully, got error %v", err)
	}
	if c != nil {
		t.Fatal("Register should return nil cache when no cluster is reachable")
	}
}
