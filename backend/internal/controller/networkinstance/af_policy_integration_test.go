package networkinstance

import (
	"context"
	"testing"

	"github.com/openconfig/ygot/ygot"
	"github.com/stretchr/testify/assert"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
)

// niWithPublicAfPolicy 构造 _public_ 实例的 AF ipv4-unicast/import-filter-policy 策略属性
// （acl-name-or-num→acl、filter-name/filter-parameter→xpl）。BGP 2b 波次⑤ 目标面。
func niWithPublicAfPolicy() *huawei.HuaweiNetworkInstance_NetworkInstance {
	ni := &huawei.HuaweiNetworkInstance_NetworkInstance{
		Instances: &huawei.HuaweiNetworkInstance_NetworkInstance_Instances{
			Instance: map[string]*huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance{
				"_public_": {
					Name: ygot.String("_public_"),
					Bgp: &huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance_Bgp{
						BaseProcess: &huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance_Bgp_BaseProcess{
							Afs: &huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance_Bgp_BaseProcess_Afs{},
						},
					},
				},
			},
		},
	}
	afs := ni.Instances.Instance["_public_"].Bgp.BaseProcess.Afs
	af, _ := afs.NewAf(huawei.HuaweiBgp_AfTypeDeviations_ipv4uni)
	af.Ipv4Unicast = &huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance_Bgp_BaseProcess_Afs_Af_Ipv4Unicast{
		ImportFilterPolicy: &huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance_Bgp_BaseProcess_Afs_Af_Ipv4Unicast_ImportFilterPolicy{
			AclNameOrNum:     ygot.String("G1"),
			Ipv4PrefixFilter: ygot.String("PF1"),
		},
	}
	return ni
}

// TestReconciler_Integration_BgpAfPolicyConverges：经既有 ni reconciler 下发 AF
// import-filter-policy 策略属性（引用已集成的 acl/xpl 目标）→ 回读 → 二次收敛
// （Changes==0，AFPOL-01/03）。零新描述符/reconciler，复用 2a 链路 + XC-06 namespace。
func TestReconciler_Integration_BgpAfPolicyConverges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim := netsim.NewSimulator()
	if err := sim.Start(); err != nil {
		t.Fatalf("start simulator: %v", err)
	}
	defer sim.Stop()

	r, cs, ds, deviceID, pool := newHarness(t, sim)
	defer pool.CloseAll()
	ctx := context.Background()
	req := reconcile.Request{DeviceID: deviceID, Path: NetworkInstancePath}

	if err := cs.Set(deviceID, NetworkInstancePath, niWithPublicAfPolicy()); err != nil {
		t.Fatalf("config store set: %v", err)
	}

	first := r.Reconcile(ctx, req)
	if first.Error != nil {
		t.Fatalf("first reconcile: %v", first.Error)
	}
	assert.Greater(t, first.Changes, 0, "首轮应下发 AF 策略属性")

	// 回读：AF import-filter-policy 策略属性真正落盘
	dc := &deviceClient{clientPool: pool, resolver: ds}
	got, err := dc.Get(ctx, deviceID)
	if err != nil {
		t.Fatalf("readback: %v", err)
	}
	ni := got.(*huawei.HuaweiNetworkInstance_NetworkInstance)
	af := ni.Instances.Instance["_public_"].Bgp.BaseProcess.Afs.Af[huawei.HuaweiBgp_AfTypeDeviations_ipv4uni]
	if af == nil || af.Ipv4Unicast == nil || af.Ipv4Unicast.ImportFilterPolicy == nil {
		t.Fatalf("回读缺 AF import-filter-policy: %#v", af)
	}
	ifp := af.Ipv4Unicast.ImportFilterPolicy
	if ifp.AclNameOrNum == nil || *ifp.AclNameOrNum != "G1" || ifp.Ipv4PrefixFilter == nil || *ifp.Ipv4PrefixFilter != "PF1" {
		t.Fatalf("回读 AF 策略属性真值缺失: %#v", ifp)
	}

	// 二次收敛（幂等）
	second := r.Reconcile(ctx, req)
	if second.Error != nil {
		t.Fatalf("second reconcile: %v", second.Error)
	}
	assert.Equal(t, 0, second.Changes, "二轮必须收敛（Changes==0），否则 AF 策略属性永久漂移")
}
