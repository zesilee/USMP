package intent

import (
	"context"
	"reflect"
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
)

// BIO-07（矩阵 B3）—— 归属索引：聚合/替换/移除/前缀匹配/可重建。

func TestOwnershipIndexOwnersMatching(t *testing.T) {
	ix := NewOwnershipIndex()
	ix.Replace("default/biz-100", []Claim{
		{Device: "10.0.0.1", Module: "vlan", Path: VlanPath + "/vlan[id=100]"},
		{Device: "10.0.0.1", Module: "ifm", Path: IfmPath + "/interface[name=GE0/0/1]"},
	})
	ix.Replace("default/biz-200", []Claim{
		{Device: "10.0.0.2", Module: "vlan", Path: VlanPath + "/vlan[id=200]"},
	})

	// 模块级写命中条目级认领（config-api SetConfig 的形态）。
	if got := ix.Owners("10.0.0.1", VlanPath); !reflect.DeepEqual(got, []string{"default/biz-100"}) {
		t.Errorf("module-level owners = %v", got)
	}
	// 条目级精确命中。
	if got := ix.Owners("10.0.0.1", VlanPath+"/vlan[id=100]"); len(got) != 1 {
		t.Errorf("entry-level owners = %v", got)
	}
	// 设备隔离。
	if got := ix.Owners("10.0.0.2", VlanPath); !reflect.DeepEqual(got, []string{"default/biz-200"}) {
		t.Errorf("device isolation broken: %v", got)
	}
	// 未认领路径零命中。
	if got := ix.Owners("10.0.0.1", "/system:system"); len(got) != 0 {
		t.Errorf("unclaimed path owners = %v", got)
	}

	// 替换与移除。
	ix.Replace("default/biz-100", []Claim{{Device: "10.0.0.9", Module: "vlan", Path: VlanPath + "/vlan[id=100]"}})
	if got := ix.Owners("10.0.0.1", VlanPath); len(got) != 0 {
		t.Errorf("Replace should drop old claims: %v", got)
	}
	ix.Remove("default/biz-200")
	if got := ix.Owners("10.0.0.2", VlanPath); len(got) != 0 {
		t.Errorf("Remove should drop intent: %v", got)
	}
}

// Reconciler 成功路径更新索引；删除路径移除（可重建性=重放 reconcile 即重建）。
func TestReconcilerMaintainsOwnershipIndex(t *testing.T) {
	cl := newFakeClient(t, newCR(1, validSpec()))
	ix := NewOwnershipIndex()
	r := NewReconciler(cl).
		WithPush(&fakePusher{results: syncedResults()}, &fakeCleaner{}, newStore(), nil).
		WithOwnership(ix)

	ctx := context.Background()
	req := reconcile.Request{DeviceID: "default/biz-100", Path: IntentPath}
	if res := r.Reconcile(ctx, req); res.Error != nil {
		t.Fatal(res.Error)
	}
	if got := ix.Owners("10.0.0.1", VlanPath); len(got) != 1 || got[0] != "default/biz-100" {
		t.Fatalf("index not populated after successful reconcile: %v", got)
	}

	// 删除：清理成功后索引移除。
	u := getCR(t, cl)
	if err := cl.Delete(ctx, u); err != nil {
		t.Fatal(err)
	}
	if res := r.Reconcile(ctx, req); res.Error != nil {
		t.Fatal(res.Error)
	}
	if got := ix.Owners("10.0.0.1", VlanPath); len(got) != 0 {
		t.Fatalf("index not cleared after delete lifecycle: %v", got)
	}
}
