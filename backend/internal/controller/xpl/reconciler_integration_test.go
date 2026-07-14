package xpl

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

// xplDesired：route-filters/route-filter(name+content) 嵌套 list。BGP 2b route-filter
// leafref 引用的目标实例。
func xplDesired() *huawei.HuaweiXpl_Xpl {
	xp := &huawei.HuaweiXpl_Xpl{
		RouteFilters: &huawei.HuaweiXpl_Xpl_RouteFilters{},
	}
	rf, _ := xp.RouteFilters.NewRouteFilter("RF-A")
	rf.Content = ygot.String("if-match protocol bgp\napply cost 100")
	return xp
}

// TestReconciler_Integration_XplConvergesAndReadable：xpl 意图（嵌套 route-filter list）
// 下发→回读→二次对账必须收敛（Changes==0，XPL-04/XC-05）。锁死「容器根一直漂移」。
func TestReconciler_Integration_XplConvergesAndReadable(t *testing.T) {
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
	if err := cs.Set(deviceID, XplPath, xplDesired()); err != nil {
		t.Fatalf("config store set: %v", err)
	}

	r := New(cs, pool, ds)
	req := reconcile.Request{DeviceID: deviceID, Path: XplPath}
	ctx := context.Background()

	// 第一轮：设备无 xpl 配置 → 检测漂移并下发
	first := r.Reconcile(ctx, req)
	if first.Error != nil {
		t.Fatalf("first reconcile: %v", first.Error)
	}
	assert.Greater(t, first.Changes, 0, "首轮应检测到漂移并下发 xpl 配置")

	// 回读设备：route-filter 真正落盘（含嵌套 list + content 正文）
	dc := &deviceClient{clientPool: pool, resolver: ds}
	got, err := dc.Get(ctx, deviceID)
	if err != nil {
		t.Fatalf("readback: %v", err)
	}
	xp, ok := got.(*huawei.HuaweiXpl_Xpl)
	if !ok {
		t.Fatalf("unexpected readback type %T", got)
	}
	if xp.RouteFilters == nil || xp.RouteFilters.RouteFilter["RF-A"] == nil {
		t.Fatalf("回读缺 route-filter 实例: %#v", xp.RouteFilters)
	}
	if rf := xp.RouteFilters.RouteFilter["RF-A"]; rf.Content == nil || *rf.Content != "if-match protocol bgp\napply cost 100" {
		t.Fatalf("回读缺 route-filter content: %#v", rf)
	}

	// 第二轮：desired==actual → 必须收敛
	second := r.Reconcile(ctx, req)
	if second.Error != nil {
		t.Fatalf("second reconcile: %v", second.Error)
	}
	assert.Equal(t, 0, second.Changes, "二轮必须收敛（Changes==0），否则容器根一直漂移")
}

// TestReconciler_Integration_XplEditConfigFailure：设备 edit-config 拒绝时对账须明确报错、
// 不 panic（R08/§9）。
func TestReconciler_Integration_XplEditConfigFailure(t *testing.T) {
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
	if err := cs.Set(deviceID, XplPath, xplDesired()); err != nil {
		t.Fatalf("config store set: %v", err)
	}
	r := New(cs, pool, ds)
	result := r.Reconcile(context.Background(), reconcile.Request{DeviceID: deviceID, Path: XplPath})
	assert.Error(t, result.Error, "edit-config 被拒时对账须报错（诚实透出，非静默成功）")
}
