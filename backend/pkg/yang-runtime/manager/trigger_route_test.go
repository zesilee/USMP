package manager

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	_ "github.com/leezesi/usmp/backend/internal/drivers"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/controller"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/predicate"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
)

type recordingReconciler struct {
	hit chan reconcile.Request
}

func (r *recordingReconciler) Reconcile(_ context.Context, req reconcile.Request) reconcile.Result {
	select {
	case r.hit <- req:
	default:
	}
	return reconcile.Result{}
}

// TestTriggerReconcileExactControllerName（full-yang-onboarding 回归）：
// token→控制器名匹配必须精确（vendor-token 全名）。全量 57 控制器下子串匹配
// 必撞：routing⊂routing-policy、multicast⊂l3-multicast、ifm⊂ifm-trunk——
// 注册序靠前的错误控制器会吞掉事件。
func TestTriggerReconcileExactControllerName(t *testing.T) {
	m := New()
	mk := func(name string) *recordingReconciler {
		rr := &recordingReconciler{hit: make(chan reconcile.Request, 1)}
		c := controller.ControllerManagedBy(name).WithReconciler(rr).WithPredicate(predicate.Always()).Build()
		m.AddController(c)
		return rr
	}
	// 故意先注册包含子串的近名控制器
	policy := mk("huawei-routing-policy")
	routing := mk("huawei-routing")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	assert.NoError(t, m.Start(ctx))
	defer m.Stop()

	ok := m.TriggerReconcile("dev1", "/routing:routing/static-routes")
	assert.True(t, ok, "routing 路径应命中控制器")

	select {
	case <-routing.hit:
		// 正确控制器收到事件
	case <-policy.hit:
		t.Fatal("huawei-routing-policy 被子串误中（应精确匹配 huawei-routing）")
	case <-time.After(3 * time.Second):
		t.Fatal("3s 内无任何控制器收到事件")
	}
}
