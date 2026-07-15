package intent

import (
	"context"
	"strings"
	"testing"
	"time"

	uns "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/leezesi/usmp/backend/internal/cache"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
)

// BIO-03/BIO-04（矩阵 A3/A5/A10④）—— Reconciler 推送编排：事务成功才写 desired
// 并触发原生对账；失败不写 desired 且退避重试；同代已收敛短路不重推但重写 desired
//（对冲 TTL 过期）；意图 desired 并入既有 desired 不抹除手工配置。

type fakePusher struct {
	calls   int
	results map[string]TxResult
}

func (f *fakePusher) Push(ctx context.Context, frags []Fragment) map[string]TxResult {
	f.calls++
	return f.results
}

func newStore() reconcile.ConfigStore {
	return manager.NewInMemoryConfigStore(cache.NewTTLLRUCache(100, time.Minute, time.Minute))
}

func syncedResults() map[string]TxResult {
	return map[string]TxResult{
		"10.0.0.1": {Device: "10.0.0.1"},
		"10.0.0.2": {Device: "10.0.0.2"},
	}
}

// 事务全体成功：Converged=True、deviceStates synced、desired 写入、原生对账被触发。
func TestReconcilePushSuccessWritesDesiredAndTriggers(t *testing.T) {
	cl := newFakeClient(t, newCR(1, validSpec()))
	pusher := &fakePusher{results: syncedResults()}
	cs := newStore()
	var triggered []string
	r := NewReconciler(cl).WithPush(pusher, nil, cs, func(deviceID, path string) bool {
		triggered = append(triggered, deviceID+path)
		return true
	})

	res := r.Reconcile(context.Background(), reconcile.Request{DeviceID: "default/biz-100", Path: IntentPath})
	if res.Error != nil {
		t.Fatalf("reconcile: %v", res.Error)
	}

	u := getCR(t, cl)
	if c := condition(u, CondConverged); c == nil || c["status"] != "True" {
		t.Fatalf("Converged = %v, want True", c)
	}
	states, _, _ := uns.NestedSlice(u.Object, "status", "deviceStates")
	for _, s := range states {
		if m := s.(map[string]interface{}); m["phase"] != PhaseSynced {
			t.Errorf("device %v phase = %v, want synced", m["device"], m["phase"])
		}
	}
	if v, _ := cs.Get("10.0.0.1", VlanPath); v == nil {
		t.Error("desired vlan not written for 10.0.0.1")
	}
	if v, _ := cs.Get("10.0.0.2", IfmPath); v == nil {
		t.Error("desired ifm not written for 10.0.0.2")
	}
	if len(triggered) == 0 {
		t.Error("native reconcile not triggered after push")
	}
}

// 事务失败：desired 不写（防周期对账绕过事务）、Converged=False、退避重试。
func TestReconcilePushFailureNoDesiredWrite(t *testing.T) {
	cl := newFakeClient(t, newCR(1, validSpec()))
	pusher := &fakePusher{results: map[string]TxResult{
		"10.0.0.1": {Device: "10.0.0.1", Err: context.DeadlineExceeded},
		"10.0.0.2": {Device: "10.0.0.2", Err: context.DeadlineExceeded},
	}}
	cs := newStore()
	r := NewReconciler(cl).WithPush(pusher, nil, cs, func(string, string) bool { return true })

	res := r.Reconcile(context.Background(), reconcile.Request{DeviceID: "default/biz-100", Path: IntentPath})
	if res.Error != nil {
		t.Fatalf("push failure should surface via status+requeue, not queue error: %v", res.Error)
	}
	if res.RequeueAfter <= 0 {
		t.Error("push failure should requeue with backoff")
	}
	if v, _ := cs.Get("10.0.0.1", VlanPath); v != nil {
		t.Error("desired must NOT be written when transaction failed")
	}
	u := getCR(t, cl)
	if c := condition(u, CondConverged); c == nil || c["status"] != "False" {
		t.Fatalf("Converged = %v, want False", c)
	}
	states, _, _ := uns.NestedSlice(u.Object, "status", "deviceStates")
	for _, s := range states {
		if m := s.(map[string]interface{}); m["phase"] != PhaseFailed {
			t.Errorf("device %v phase = %v, want failed", m["device"], m["phase"])
		}
	}
}

// 幂等短路（A5）：同代已收敛的二次 reconcile 不再推事务，但重写 desired（BIO-04）。
func TestReconcileIdempotentShortCircuit(t *testing.T) {
	cl := newFakeClient(t, newCR(1, validSpec()))
	pusher := &fakePusher{results: syncedResults()}
	cs := newStore()
	r := NewReconciler(cl).WithPush(pusher, nil, cs, func(string, string) bool { return true })

	ctx := context.Background()
	req := reconcile.Request{DeviceID: "default/biz-100", Path: IntentPath}
	if res := r.Reconcile(ctx, req); res.Error != nil {
		t.Fatal(res.Error)
	}
	if pusher.calls != 1 {
		t.Fatalf("first reconcile should push once, got %d", pusher.calls)
	}

	// 模拟 desired 过期（TTL 掉了）。
	_ = cs.Delete("10.0.0.1", VlanPath)
	if res := r.Reconcile(ctx, req); res.Error != nil {
		t.Fatal(res.Error)
	}
	if pusher.calls != 1 {
		t.Errorf("second reconcile at same generation must not re-push (got %d calls)", pusher.calls)
	}
	if v, _ := cs.Get("10.0.0.1", VlanPath); v == nil {
		t.Error("short-circuit must still rewrite desired（对冲 TTL 过期，BIO-04）")
	}
}

// 非事务降级呈现（A10③）：某设备能力缺失走普通 commit——synced 但 reason 标注。
func TestReconcileNonTransactionalDeviceAnnotated(t *testing.T) {
	cl := newFakeClient(t, newCR(1, validSpec()))
	pusher := &fakePusher{results: map[string]TxResult{
		"10.0.0.1": {Device: "10.0.0.1"},
		"10.0.0.2": {Device: "10.0.0.2", NonTransactional: true},
	}}
	r := NewReconciler(cl).WithPush(pusher, nil, newStore(), func(string, string) bool { return true })

	if res := r.Reconcile(context.Background(), reconcile.Request{DeviceID: "default/biz-100", Path: IntentPath}); res.Error != nil {
		t.Fatal(res.Error)
	}
	u := getCR(t, cl)
	states, _, _ := uns.NestedSlice(u.Object, "status", "deviceStates")
	var found bool
	for _, s := range states {
		m := s.(map[string]interface{})
		if m["device"] == "10.0.0.2" {
			found = true
			if m["phase"] != PhaseSynced || !strings.Contains(m["reason"].(string), "non-transactional") {
				t.Errorf("10.0.0.2 state = %v, want synced + non-transactional reason", m)
			}
		}
	}
	if !found {
		t.Fatal("10.0.0.2 missing from deviceStates")
	}
}

// 合并防抹除（A3）：意图片段并入既有 desired——手工 vlan/接口字段不丢，同名条目意图字段覆盖。
func TestMergeFragmentPreservesExisting(t *testing.T) {
	id300, id100 := uint16(300), uint16(100)
	manual := "manual"
	existingVlan := &huawei.HuaweiVlan_Vlan_Vlans{Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
		300: {Id: &id300, Name: &manual, Description: &manual},
	}}
	fragName := "biz-100"
	frag := &huawei.HuaweiVlan_Vlan_Vlans{Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
		100: {Id: &id100, Name: &fragName},
	}}
	merged, ok := mergeFragment(existingVlan, frag).(*huawei.HuaweiVlan_Vlan_Vlans)
	if !ok {
		t.Fatal("merged vlan wrong type")
	}
	if merged.Vlan[300] == nil || merged.Vlan[300].Description == nil || *merged.Vlan[300].Description != "manual" {
		t.Errorf("manual vlan 300 lost in merge: %+v", merged.Vlan[300])
	}
	if merged.Vlan[100] == nil || *merged.Vlan[100].Name != "biz-100" {
		t.Errorf("intent vlan 100 missing in merge: %+v", merged.Vlan[100])
	}

	// ifm：同名接口保留既有 mtu，仅覆盖 l2 属性。
	mtu := uint32(1500)
	port := "GE0/0/1"
	pvid := uint16(100)
	existingIfm := &huawei.HuaweiIfm_Ifm_Interfaces{Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{
		port: {Name: &port, Mtu: &mtu},
	}}
	fragIfm := &huawei.HuaweiIfm_Ifm_Interfaces{Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{
		port: ifmPortEntry(port, &huawei.HuaweiIfm_Ifm_Interfaces_Interface_Ethernet_MainInterface_L2Attribute{
			LinkType: huawei.HuaweiEthernet_LinkType_access,
			Pvid:     &pvid,
		}),
	}}
	mergedIfm, ok := mergeFragment(existingIfm, fragIfm).(*huawei.HuaweiIfm_Ifm_Interfaces)
	if !ok {
		t.Fatal("merged ifm wrong type")
	}
	got := mergedIfm.Interface[port]
	if got == nil || got.Mtu == nil || *got.Mtu != 1500 {
		t.Errorf("manual mtu lost in ifm merge: %+v", got)
	}
	if got.Ethernet == nil || got.Ethernet.MainInterface == nil || got.Ethernet.MainInterface.L2Attribute == nil ||
		got.Ethernet.MainInterface.L2Attribute.Pvid == nil || *got.Ethernet.MainInterface.L2Attribute.Pvid != 100 {
		t.Errorf("intent l2 attribute missing in ifm merge: %+v", got)
	}
}
