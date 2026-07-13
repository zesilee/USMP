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

type peerT = huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance_Bgp_BaseProcess_Peers_Peer

// niWithPublicPeers 构造 instance[_public_] + bgp/base-process/peers（公网基础邻居）。
func niWithPublicPeers(peers ...*peerT) *huawei.HuaweiNetworkInstance_NetworkInstance {
	m := map[string]*peerT{}
	for _, p := range peers {
		m[*p.Address] = p
	}
	return &huawei.HuaweiNetworkInstance_NetworkInstance{
		Instances: &huawei.HuaweiNetworkInstance_NetworkInstance_Instances{
			Instance: map[string]*huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance{
				"_public_": {
					Name: ygot.String("_public_"),
					Bgp: &huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance_Bgp{
						BaseProcess: &huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance_Bgp_BaseProcess{
							Peers: &huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance_Bgp_BaseProcess_Peers{Peer: m},
						},
					},
				},
			},
		},
	}
}

func peer(addr, remoteAs, desc string) *peerT {
	return &peerT{Address: ygot.String(addr), RemoteAs: ygot.String(remoteAs), Description: ygot.String(desc)}
}

// TestReconciler_Integration_BgpNeighborConverges：公网 BGP 基础邻居（peers under
// instance[_public_]/bgp/base-process）下发→回读→二次收敛（BN-01/BN-03）。锁死深层嵌套
// （instance/bgp/base-process/peers/peer）+ per-node namespace 端到端在模拟网元跑通。
func TestReconciler_Integration_BgpNeighborConverges(t *testing.T) {
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

	desired := niWithPublicPeers(peer("10.0.0.1", "100", "peerA"), peer("10.0.0.2", "200", "peerB"))
	if err := cs.Set(deviceID, NetworkInstancePath, desired); err != nil {
		t.Fatalf("config store set: %v", err)
	}

	first := r.Reconcile(ctx, req)
	if first.Error != nil {
		t.Fatalf("first reconcile: %v", first.Error)
	}
	assert.Greater(t, first.Changes, 0, "首轮应下发 peers")

	// 回读：peers 真正落盘（深层嵌套幸存）
	dc := &deviceClient{clientPool: pool, resolver: ds}
	got, err := dc.Get(ctx, deviceID)
	if err != nil {
		t.Fatalf("readback: %v", err)
	}
	ni := got.(*huawei.HuaweiNetworkInstance_NetworkInstance)
	pa := ni.Instances.Instance["_public_"].Bgp.BaseProcess.Peers.Peer["10.0.0.1"]
	if pa == nil || pa.RemoteAs == nil || *pa.RemoteAs != "100" || pa.Description == nil || *pa.Description != "peerA" {
		t.Fatalf("回读缺 peer 10.0.0.1 字段: %#v", pa)
	}
	if ni.Instances.Instance["_public_"].Bgp.BaseProcess.Peers.Peer["10.0.0.2"] == nil {
		t.Fatalf("回读缺 peer 10.0.0.2")
	}

	// 二次收敛
	second := r.Reconcile(ctx, req)
	if second.Error != nil {
		t.Fatalf("second reconcile: %v", second.Error)
	}
	assert.Equal(t, 0, second.Changes, "二轮必须收敛（Changes==0），否则 peers 永久漂移")
}
