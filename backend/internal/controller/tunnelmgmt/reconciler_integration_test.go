package tunnelmgmt

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

// tunnelDesired：标量边界意图——tunnel-policy(name+description) 嵌套 list +
// tunnel-down-switch/enable 标量。BGP 2b tunnel-policy leafref 引用的目标实例。
func tunnelDesired() *huawei.HuaweiTunnelManagement_TunnelManagement {
	tm := &huawei.HuaweiTunnelManagement_TunnelManagement{
		TunnelDownSwitch: &huawei.HuaweiTunnelManagement_TunnelManagement_TunnelDownSwitch{
			Enable: ygot.Bool(true),
		},
		TunnelPolicys: &huawei.HuaweiTunnelManagement_TunnelManagement_TunnelPolicys{},
	}
	p, _ := tm.TunnelPolicys.NewTunnelPolicy("policy-a")
	p.Description = ygot.String("bind te tunnels")
	return tm
}

// TestReconciler_Integration_TunnelManagementConvergesAndReadable：tunnel-management
// 意图（嵌套 tunnel-policy list + tunnel-down-switch 标量）下发→回读→二次对账必须收敛
// （Changes==0，TNLM-04/XC-05）。锁死「容器根一直漂移」：若 diff/client/编解码路径有
// list 中心假设，第二轮会算出非零 changes（永久漂移）。
func TestReconciler_Integration_TunnelManagementConvergesAndReadable(t *testing.T) {
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
	if err := cs.Set(deviceID, TunnelManagementPath, tunnelDesired()); err != nil {
		t.Fatalf("config store set: %v", err)
	}

	r := New(cs, pool, ds)
	req := reconcile.Request{DeviceID: deviceID, Path: TunnelManagementPath}
	ctx := context.Background()

	// 第一轮：设备无 tunnel-management 配置 → 检测漂移并下发
	first := r.Reconcile(ctx, req)
	if first.Error != nil {
		t.Fatalf("first reconcile: %v", first.Error)
	}
	assert.Greater(t, first.Changes, 0, "首轮应检测到漂移并下发 tunnel-management 配置")

	// 回读设备：tunnel-policy 与 tunnel-down-switch 真正落盘（含嵌套 list）
	dc := &deviceClient{clientPool: pool, resolver: ds}
	got, err := dc.Get(ctx, deviceID)
	if err != nil {
		t.Fatalf("readback: %v", err)
	}
	tm, ok := got.(*huawei.HuaweiTunnelManagement_TunnelManagement)
	if !ok {
		t.Fatalf("unexpected readback type %T", got)
	}
	if tm.TunnelPolicys == nil || tm.TunnelPolicys.TunnelPolicy["policy-a"] == nil {
		t.Fatalf("回读缺 tunnel-policy 实例: %#v", tm.TunnelPolicys)
	}
	if p := tm.TunnelPolicys.TunnelPolicy["policy-a"]; p.Description == nil || *p.Description != "bind te tunnels" {
		t.Fatalf("回读缺 tunnel-policy description: %#v", p)
	}
	if tm.TunnelDownSwitch == nil || tm.TunnelDownSwitch.Enable == nil || !*tm.TunnelDownSwitch.Enable {
		t.Fatalf("回读缺 tunnel-down-switch/enable: %#v", tm.TunnelDownSwitch)
	}

	// 第二轮：desired==actual → 必须收敛（否则容器根永久漂移）
	second := r.Reconcile(ctx, req)
	if second.Error != nil {
		t.Fatalf("second reconcile: %v", second.Error)
	}
	assert.Equal(t, 0, second.Changes, "二轮必须收敛（Changes==0），否则容器根一直漂移")
}

// TestReconciler_Integration_TunnelManagementEditConfigFailure：设备 edit-config 拒绝
// 时对账须明确报错、不 panic（R08/§9：下发失败保留原配置、缓存不更新）。
func TestReconciler_Integration_TunnelManagementEditConfigFailure(t *testing.T) {
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
	if err := cs.Set(deviceID, TunnelManagementPath, tunnelDesired()); err != nil {
		t.Fatalf("config store set: %v", err)
	}
	r := New(cs, pool, ds)
	result := r.Reconcile(context.Background(), reconcile.Request{DeviceID: deviceID, Path: TunnelManagementPath})
	assert.Error(t, result.Error, "edit-config 被拒时对账须报错（诚实透出，非静默成功）")
}
