package intent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	uns "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
)

// 矩阵 A7 / BIC-02/BIC-03、BIO-05 —— 真 apiserver（envtest）集成：
//  1. crdgen 生成的 CRD OpenAPI 在 apiserver 侧写入时拒绝字段级违规
//     （kubectl/GitOps 直写是受支持接入方式，写入时失败而非收敛时）；
//  2. finalizer 生命周期走真实 apiserver 语义（deletionTimestamp 拦截、
//     摘 finalizer 后对象真正消失）——fake client 无法完整仿真的层次。
//
// 依赖 envtest 二进制（setup-envtest）：KUBEBUILDER_ASSETS 未设且默认目录
// 不存在时跳过（本地未安装不阻塞；CI 由 compliance 工作流安装）。

func envtestAssets(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("KUBEBUILDER_ASSETS"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	matches, _ := filepath.Glob(filepath.Join(home, ".local/share/kubebuilder-envtest/k8s/*"))
	if len(matches) > 0 {
		return matches[0]
	}
	t.Skip("envtest binaries not installed (run: go run sigs.k8s.io/controller-runtime/tools/setup-envtest@release-0.19 use 1.31.0)")
	return ""
}

func startEnvtest(t *testing.T) client.Client {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping envtest integration in short mode")
	}
	env := &envtest.Environment{
		BinaryAssetsDirectory: envtestAssets(t),
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "deploy", "crds")},
	}
	cfg, err := env.Start()
	if err != nil {
		t.Fatalf("start envtest: %v", err)
	}
	t.Cleanup(func() { _ = env.Stop() })
	cl, err := client.New(cfg, client.Options{})
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	return cl
}

func envCR(name string, spec map[string]interface{}) *uns.Unstructured {
	u := &uns.Unstructured{}
	u.SetGroupVersionKind(GVK)
	u.SetNamespace("default")
	u.SetName(name)
	_ = uns.SetNestedMap(u.Object, spec, "spec")
	return u
}

// A7：apiserver 依据 crdgen 生成的 OpenAPI 在写入时拒绝字段级违规。
func TestEnvtestApiserverRejectsFieldViolations_Integration(t *testing.T) {
	cl := startEnvtest(t)
	ctx := context.Background()

	cases := []struct {
		name    string
		spec    map[string]interface{}
		errPart string
	}{
		{
			name:    "vlan-id 越界 4095",
			spec:    map[string]interface{}{"vlan-id": int64(4095), "devices": []interface{}{map[string]interface{}{"ip": "10.0.0.1"}}},
			errPart: "4094",
		},
		{
			name:    "vlan-id 越界 0",
			spec:    map[string]interface{}{"vlan-id": int64(0), "devices": []interface{}{map[string]interface{}{"ip": "10.0.0.1"}}},
			errPart: "greater than or equal to 1",
		},
		{
			name:    "缺 required vlan-id",
			spec:    map[string]interface{}{"devices": []interface{}{map[string]interface{}{"ip": "10.0.0.1"}}},
			errPart: "vlan-id",
		},
		{
			name:    "ip 违反 pattern",
			spec:    map[string]interface{}{"vlan-id": int64(100), "devices": []interface{}{map[string]interface{}{"ip": "not-an-ip"}}},
			errPart: "should match",
		},
		{
			name:    "name 违反 pattern（超长非法字符）",
			spec:    map[string]interface{}{"vlan-id": int64(100), "name": "空 格!", "devices": []interface{}{map[string]interface{}{"ip": "10.0.0.1"}}},
			errPart: "should match",
		},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := cl.Create(ctx, envCR("bad-"+string(rune('a'+i)), tc.spec))
			if err == nil {
				t.Fatalf("apiserver should reject %s at write time (A7)", tc.name)
			}
			if !apierrors.IsInvalid(err) {
				t.Fatalf("want Invalid error, got %v", err)
			}
			if tc.errPart != "" && !strings.Contains(err.Error(), tc.errPart) {
				t.Errorf("error should mention %q, got %v", tc.errPart, err)
			}
		})
	}

	// 合法 CR 放行（含全字段）。
	ok := envCR("good", map[string]interface{}{
		"vlan-id": int64(100), "name": "office",
		"devices": []interface{}{map[string]interface{}{
			"ip": "10.0.0.1", "access-ports": []interface{}{"GE0/0/1"}, "trunk-ports": []interface{}{"GE0/0/2"},
		}},
	})
	if err := cl.Create(ctx, ok); err != nil {
		t.Fatalf("valid CR should be accepted: %v", err)
	}
}

// BIO-05/BIC-04：真 apiserver 生命周期——finalizer 拦截删除、status 子资源回写、
// 摘除后对象真正消失（fake client 覆盖不了的 deletionTimestamp/GC 语义）。
func TestEnvtestFinalizerLifecycle_Integration(t *testing.T) {
	cl := startEnvtest(t)
	ctx := context.Background()

	cr := envCR("biz-100", map[string]interface{}{
		"vlan-id": int64(100), "name": "office",
		"devices": []interface{}{
			map[string]interface{}{"ip": "10.0.0.1", "access-ports": []interface{}{"GE0/0/1"}},
			map[string]interface{}{"ip": "10.0.0.2", "trunk-ports": []interface{}{"GE0/0/2"}},
		},
	})
	if err := cl.Create(ctx, cr); err != nil {
		t.Fatal(err)
	}

	cleaner := &fakeCleaner{}
	r := NewReconciler(cl).
		WithPush(&fakePusher{results: syncedResults()}, cleaner, newStore(), nil).
		WithOwnership(NewOwnershipIndex())
	req := reconcile.Request{DeviceID: "default/biz-100", Path: IntentPath}
	if res := r.Reconcile(ctx, req); res.Error != nil {
		t.Fatalf("reconcile: %v", res.Error)
	}

	key := types.NamespacedName{Namespace: "default", Name: "biz-100"}
	u := &uns.Unstructured{}
	u.SetGroupVersionKind(GVK)
	if err := cl.Get(ctx, key, u); err != nil {
		t.Fatal(err)
	}
	if !hasFinalizer(u) {
		t.Fatal("finalizer not persisted on real apiserver")
	}
	if c := condition(u, CondConverged); c == nil || c["status"] != "True" {
		t.Fatalf("Converged = %v, want True via status subresource", c)
	}
	if gen, _, _ := uns.NestedInt64(u.Object, "status", "observedGeneration"); gen != u.GetGeneration() {
		t.Fatalf("observedGeneration %d != generation %d", gen, u.GetGeneration())
	}

	// 删除：finalizer 拦截 → 对象带 deletionTimestamp 存活（真 apiserver 语义）。
	if err := cl.Delete(ctx, u); err != nil {
		t.Fatal(err)
	}
	if err := cl.Get(ctx, key, u); err != nil {
		t.Fatalf("CR should survive deletion until finalizer released: %v", err)
	}
	if u.GetDeletionTimestamp() == nil {
		t.Fatal("deletionTimestamp not set")
	}

	// 清理 reconcile：认领清理 → 摘 finalizer → apiserver 真正删除对象。
	if res := r.Reconcile(ctx, req); res.Error != nil {
		t.Fatalf("delete reconcile: %v", res.Error)
	}
	if len(cleaner.calls) != 1 || len(cleaner.calls[0]) == 0 {
		t.Fatalf("cleaner should run once with claims, got %+v", cleaner.calls)
	}
	if err := cl.Get(ctx, key, u); !apierrors.IsNotFound(err) {
		t.Fatalf("CR should be gone after finalizer release, got %v", err)
	}
}
