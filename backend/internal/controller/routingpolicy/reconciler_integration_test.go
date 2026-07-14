package routingpolicy

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/openconfig/ygot/ygot"
	"github.com/stretchr/testify/assert"

	"github.com/leezesi/usmp/backend/internal/cache"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
)

func simStore(sim *netsim.Simulator) (device.Store, string) {
	ds := device.NewStore()
	id := "sim"
	ds.Put(id, client.DeviceConnectionInfo{
		IP: sim.Addr(), Port: sim.Port(), Username: sim.Username(), Password: sim.Password(), Protocol: client.ProtocolNETCONF,
	})
	return ds, id
}

// rtpDesired：policy-definitions/policy-definition(name+address-family-mismatch-deny)
// 标量边界。BGP 2b import/export route-policy leafref 引用的目标实例。
func rtpDesired() *huawei.HuaweiRoutingPolicy_RoutingPolicy {
	rp := &huawei.HuaweiRoutingPolicy_RoutingPolicy{
		PolicyDefinitions: &huawei.HuaweiRoutingPolicy_RoutingPolicy_PolicyDefinitions{},
	}
	pd, _ := rp.PolicyDefinitions.NewPolicyDefinition("RP-A")
	pd.AddressFamilyMismatchDeny = ygot.Bool(true)
	return rp
}

// TestReconciler_Integration_RoutingPolicyConvergesAndReadable：rtp 意图（嵌套
// policy-definition list）下发→回读→二次对账必须收敛（Changes==0，RTP-04/XC-05）。
func TestReconciler_Integration_RoutingPolicyConvergesAndReadable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim := netsim.NewSimulator()
	if err := sim.Start(); err != nil {
		t.Fatalf("start simulator: %v", err)
	}
	defer sim.Stop()

	c := cache.NewTTLLRUCache(100, 30*time.Second, 1*time.Minute)
	cs := manager.NewInMemoryConfigStore(c)
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	ds, deviceID := simStore(sim)
	if err := cs.Set(deviceID, RoutingPolicyPath, rtpDesired()); err != nil {
		t.Fatalf("config store set: %v", err)
	}

	r := New(cs, pool, ds)
	req := reconcile.Request{DeviceID: deviceID, Path: RoutingPolicyPath}
	ctx := context.Background()

	// 第一轮：设备无 rtp 配置 → 检测漂移并下发
	first := r.Reconcile(ctx, req)
	if first.Error != nil {
		t.Fatalf("first reconcile: %v", first.Error)
	}
	assert.Greater(t, first.Changes, 0, "首轮应检测到漂移并下发 rtp 配置")

	// 回读设备：policy-definition 真正落盘（含嵌套 list）
	dc := &deviceClient{clientPool: pool, resolver: ds}
	got, err := dc.Get(ctx, deviceID)
	if err != nil {
		t.Fatalf("readback: %v", err)
	}
	rp, ok := got.(*huawei.HuaweiRoutingPolicy_RoutingPolicy)
	if !ok {
		t.Fatalf("unexpected readback type %T", got)
	}
	if rp.PolicyDefinitions == nil || rp.PolicyDefinitions.PolicyDefinition["RP-A"] == nil {
		t.Fatalf("回读缺 policy-definition 实例: %#v", rp.PolicyDefinitions)
	}
	if pd := rp.PolicyDefinitions.PolicyDefinition["RP-A"]; pd.AddressFamilyMismatchDeny == nil || !*pd.AddressFamilyMismatchDeny {
		t.Fatalf("回读缺 policy-definition 标量: %#v", pd)
	}

	// 第二轮：desired==actual → 必须收敛
	second := r.Reconcile(ctx, req)
	if second.Error != nil {
		t.Fatalf("second reconcile: %v", second.Error)
	}
	assert.Equal(t, 0, second.Changes, "二轮必须收敛（Changes==0），否则容器根一直漂移")
}

// TestReconciler_Integration_RoutingPolicyEditConfigFailure：设备 edit-config 拒绝时对账
// 须明确报错、不 panic（R08/§9）。
func TestReconciler_Integration_RoutingPolicyEditConfigFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim := netsim.NewSimulator()
	if err := sim.Start(); err != nil {
		t.Fatalf("start simulator: %v", err)
	}
	defer sim.Stop()
	sc := netsim.NewScenarioConfig()
	sc.ErrorOnRPC["edit-config"] = fmt.Errorf("device rejected edit-config")
	sim.SetScenario(sc)

	c := cache.NewTTLLRUCache(100, 30*time.Second, 1*time.Minute)
	cs := manager.NewInMemoryConfigStore(c)
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	ds, deviceID := simStore(sim)
	if err := cs.Set(deviceID, RoutingPolicyPath, rtpDesired()); err != nil {
		t.Fatalf("config store set: %v", err)
	}
	r := New(cs, pool, ds)
	result := r.Reconcile(context.Background(), reconcile.Request{DeviceID: deviceID, Path: RoutingPolicyPath})
	assert.Error(t, result.Error, "edit-config 被拒时对账须报错（诚实透出，非静默成功）")
}
