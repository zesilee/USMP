package intent

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	uns "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/leezesi/usmp/backend/internal/generated/business"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
)

// BIO-05/BIO-06（矩阵 B1/B2）—— 删除生命周期与收缩差集：finalizer 拦截删除、
// 认领清理、部分失败保留重试、devices 收缩孤儿清理、desired scrub。

type fakeCleaner struct {
	calls    [][]Claim
	failures map[string]error
}

func (f *fakeCleaner) Cleanup(ctx context.Context, claims []Claim) map[string]error {
	f.calls = append(f.calls, append([]Claim{}, claims...))
	return f.failures
}

// 首次成功 reconcile 会挂 finalizer。
func TestReconcileAddsFinalizer(t *testing.T) {
	cl := newFakeClient(t, newCR(1, validSpec()))
	r := NewReconciler(cl).WithPush(&fakePusher{results: syncedResults()}, &fakeCleaner{}, newStore(), nil)
	if res := r.Reconcile(context.Background(), reconcile.Request{DeviceID: "default/biz-100", Path: IntentPath}); res.Error != nil {
		t.Fatal(res.Error)
	}
	if !hasFinalizer(getCR(t, cl)) {
		t.Fatal("finalizer not added on first reconcile")
	}
}

// 删除：清理全部认领 → finalizer 摘除 → CR 消失（矩阵 B1）。
func TestReconcileDeleteCleansAndReleases(t *testing.T) {
	cr := newCR(1, validSpec())
	cr.SetFinalizers([]string{Finalizer})
	now := metav1.Now()
	cr.SetDeletionTimestamp(&now)
	_ = uns.SetNestedSlice(cr.Object, []interface{}{
		map[string]interface{}{"device": "10.0.0.1", "module": "vlan", "path": VlanPath + "/vlan[id=100]"},
		map[string]interface{}{"device": "10.0.0.2", "module": "ifm", "path": IfmPath + "/interface[name=GE0/0/3]"},
	}, "status", "claims")

	cl := newFakeClient(t, cr)
	cleaner := &fakeCleaner{}
	cs := newStore()
	// desired 里预置意图条目，删除必须先 scrub。
	id := uint16(100)
	_ = cs.Set("10.0.0.1", VlanPath, &huawei.HuaweiVlan_Vlan_Vlans{Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{100: {Id: &id}}})
	r := NewReconciler(cl).WithPush(&fakePusher{results: syncedResults()}, cleaner, cs, nil)

	res := r.Reconcile(context.Background(), reconcile.Request{DeviceID: "default/biz-100", Path: IntentPath})
	if res.Error != nil {
		t.Fatal(res.Error)
	}
	if len(cleaner.calls) != 1 || len(cleaner.calls[0]) != 2 {
		t.Fatalf("cleaner calls = %+v, want one call with both claims", cleaner.calls)
	}
	if v, _ := cs.Get("10.0.0.1", VlanPath); v != nil {
		if vl, ok := v.(*huawei.HuaweiVlan_Vlan_Vlans); ok && vl.Vlan[100] != nil {
			t.Error("desired vlan 100 not scrubbed on delete")
		}
	}
	// finalizer 摘除后 fake client 完成删除：CR 应消失。
	u := &uns.Unstructured{}
	u.SetGroupVersionKind(GVK)
	if err := cl.Get(context.Background(), crKey("default/biz-100"), u); err == nil {
		t.Fatalf("CR should be gone after finalizer release, still present: finalizers=%v", u.GetFinalizers())
	}
}

// 删除部分失败：finalizer 保留、deviceStates failed、退避重试（矩阵 B1 负路径）。
func TestReconcileDeletePartialFailureRetains(t *testing.T) {
	cr := newCR(1, validSpec())
	cr.SetFinalizers([]string{Finalizer})
	now := metav1.Now()
	cr.SetDeletionTimestamp(&now)
	_ = uns.SetNestedSlice(cr.Object, []interface{}{
		map[string]interface{}{"device": "10.0.0.2", "module": "vlan", "path": VlanPath + "/vlan[id=100]"},
	}, "status", "claims")

	cl := newFakeClient(t, cr)
	cleaner := &fakeCleaner{failures: map[string]error{"10.0.0.2": errors.New("device offline")}}
	r := NewReconciler(cl).WithPush(&fakePusher{results: syncedResults()}, cleaner, newStore(), nil)

	res := r.Reconcile(context.Background(), reconcile.Request{DeviceID: "default/biz-100", Path: IntentPath})
	if res.Error != nil {
		t.Fatal(res.Error)
	}
	if res.RequeueAfter <= 0 {
		t.Error("partial cleanup failure should requeue")
	}
	u := getCR(t, cl)
	if !hasFinalizer(u) {
		t.Fatal("finalizer must be retained while cleanup fails")
	}
	states, _, _ := uns.NestedSlice(u.Object, "status", "deviceStates")
	if len(states) != 1 || states[0].(map[string]interface{})["phase"] != PhaseFailed {
		t.Fatalf("deviceStates = %v, want failed 10.0.0.2", states)
	}
}

// 收缩差集：devices [A,B]→[A]，B 的认领被清理、desired 被 scrub、认领推进（矩阵 B2）。
func TestReconcileShrinkCleansOrphans(t *testing.T) {
	cl := newFakeClient(t, newCR(1, validSpec()))
	cleaner := &fakeCleaner{}
	cs := newStore()
	pusher := &fakePusher{results: syncedResults()}
	r := NewReconciler(cl).WithPush(pusher, cleaner, cs, nil)

	ctx := context.Background()
	req := reconcile.Request{DeviceID: "default/biz-100", Path: IntentPath}
	if res := r.Reconcile(ctx, req); res.Error != nil {
		t.Fatal(res.Error)
	}
	if len(cleaner.calls) != 0 {
		t.Fatalf("no cleanup expected on first reconcile, got %+v", cleaner.calls)
	}

	// spec 收缩到仅 devA + generation 递进。
	u := getCR(t, cl)
	spec := map[string]interface{}{
		"vlan-id": int64(100),
		"name":    "office",
		"devices": []interface{}{
			map[string]interface{}{"ip": "10.0.0.1", "access-ports": []interface{}{"GE0/0/1"}},
		},
	}
	_ = uns.SetNestedMap(u.Object, spec, "spec")
	u.SetGeneration(2)
	if err := cl.Update(ctx, u); err != nil {
		t.Fatal(err)
	}
	pusher.results = map[string]TxResult{"10.0.0.1": {Device: "10.0.0.1"}}

	if res := r.Reconcile(ctx, req); res.Error != nil {
		t.Fatal(res.Error)
	}
	if len(cleaner.calls) != 1 {
		t.Fatalf("shrink should trigger exactly one cleanup, got %d", len(cleaner.calls))
	}
	for _, c := range cleaner.calls[0] {
		if c.Device != "10.0.0.2" {
			t.Errorf("orphan cleanup should target only removed device, got %+v", c)
		}
	}
	claims, _, _ := uns.NestedSlice(getCR(t, cl).Object, "status", "claims")
	for _, c := range claims {
		if c.(map[string]interface{})["device"] == "10.0.0.2" {
			t.Errorf("claims should no longer contain removed device: %v", claims)
		}
	}
}

// cleanupChanges：vlan→keyed delete、ifm→l2-attribute remove（不删接口条目，PR#145）。
func TestCleanupChangesShapes(t *testing.T) {
	changes, err := cleanupChanges([]Claim{
		{Device: "10.0.0.1", Module: "vlan", Path: VlanPath + "/vlan[id=100]"},
		{Device: "10.0.0.1", Module: "ifm", Path: IfmPath + "/interface[name=GE0/0/1]"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 {
		t.Fatalf("changes = %d, want 2", len(changes))
	}
	if changes[0].Type.String() != "DELETE" {
		t.Errorf("vlan cleanup should be a DELETE change, got %v", changes[0].Type)
	}
	raw, _ := changes[1].NewValue.(string)
	if !strings.Contains(raw, `nc:operation="remove"`) || !strings.Contains(raw, "l2-attribute") || strings.Contains(raw, "operation=\"delete\"") {
		t.Errorf("ifm cleanup should remove l2-attribute subtree, got %s", raw)
	}
	if !strings.Contains(raw, "GE0/0/1") {
		t.Errorf("ifm cleanup missing port name: %s", raw)
	}

	if _, err := cleanupChanges([]Claim{{Device: "d", Module: "vlan", Path: "no-key"}}); err == nil {
		t.Error("unparsable claim path should error")
	}
}

// removeClaimsFromDesired：vlan 条目移除、ifm 仅剥 L2 链保留手工字段。
func TestRemoveClaimsFromDesired(t *testing.T) {
	cs := newStore()
	id100, id300 := uint16(100), uint16(300)
	_ = cs.Set("10.0.0.1", VlanPath, &huawei.HuaweiVlan_Vlan_Vlans{Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
		100: {Id: &id100}, 300: {Id: &id300},
	}})
	mtu := uint32(1500)
	port := "GE0/0/1"
	pvid := uint16(100)
	entry := ifmPortEntry(port, &huawei.HuaweiIfm_Ifm_Interfaces_Interface_Ethernet_MainInterface_L2Attribute{Pvid: &pvid})
	entry.Mtu = &mtu
	_ = cs.Set("10.0.0.1", IfmPath, &huawei.HuaweiIfm_Ifm_Interfaces{Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{port: entry}})

	removeClaimsFromDesired(cs, []Claim{
		{Device: "10.0.0.1", Module: "vlan", Path: VlanPath + "/vlan[id=100]"},
		{Device: "10.0.0.1", Module: "ifm", Path: IfmPath + "/interface[name=GE0/0/1]"},
	})

	v, _ := cs.Get("10.0.0.1", VlanPath)
	vl := v.(*huawei.HuaweiVlan_Vlan_Vlans)
	if vl.Vlan[100] != nil || vl.Vlan[300] == nil {
		t.Errorf("vlan scrub wrong: %+v", vl.Vlan)
	}
	i, _ := cs.Get("10.0.0.1", IfmPath)
	ifm := i.(*huawei.HuaweiIfm_Ifm_Interfaces)
	got := ifm.Interface[port]
	if got == nil || got.Mtu == nil || *got.Mtu != 1500 {
		t.Errorf("manual mtu must survive scrub: %+v", got)
	}
	if got != nil && got.Ethernet != nil {
		t.Errorf("intent L2 chain must be stripped: %+v", got.Ethernet)
	}
}

// 集成（矩阵 B1 设备面）：push 后 Cleanup 全部认领——sim 上 vlan 消失、L2 链清空。
func TestCleanupClaims_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	simA, simB := startTxSim(t, "127.0.0.1", nil), startTxSim(t, "127.0.0.2", nil)
	tx := txStack(t, simA, simB)

	id := uint16(100)
	name := "tx-test"
	spec := &business.UsmpBusinessVlan_BusinessVlanService{
		VlanId: &id, Name: &name,
		Devices: map[string]*business.UsmpBusinessVlan_BusinessVlanService_Devices{
			devA: {Ip: s(devA), AccessPorts: []string{"GE0/0/1"}},
			devB: {Ip: s(devB), TrunkPorts: []string{"GE0/0/3"}},
		},
	}
	frags, claims, err := ExpandBusinessVlan(spec)
	if err != nil {
		t.Fatal(err)
	}
	for dev, res := range tx.Push(context.Background(), frags) {
		if res.Err != nil {
			t.Fatalf("push %s: %v", dev, res.Err)
		}
	}
	if _, ok := simA.RunningHuaweiVLANs()[100]; !ok {
		t.Fatal("precondition: vlan not pushed")
	}

	failures := tx.Cleanup(context.Background(), claims)
	if len(failures) != 0 {
		t.Fatalf("cleanup failures: %v", failures)
	}
	if _, ok := simA.RunningHuaweiVLANs()[100]; ok {
		t.Fatal("simA vlan 100 should be deleted by cleanup")
	}
	if _, ok := simB.RunningHuaweiVLANs()[100]; ok {
		t.Fatal("simB vlan 100 should be deleted by cleanup")
	}
	if iface := simA.RunningHuaweiInterfaces()["GE0/0/1"]; iface != nil && iface.L2.Pvid == 100 {
		t.Fatalf("simA GE0/0/1 L2 chain should be removed, got %+v", iface.L2)
	}
	// 幂等：二次清理无 data-missing 错误（operation=remove）。
	if failures := tx.Cleanup(context.Background(), claims); len(failures) != 0 {
		t.Fatalf("second cleanup should be idempotent, got %v", failures)
	}
}

var _ = time.Now // keep import symmetry with sibling test files
