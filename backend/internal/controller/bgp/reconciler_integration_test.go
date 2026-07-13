package bgp

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

func bgpDesired() *huawei.HuaweiBgp_Bgp {
	return &huawei.HuaweiBgp_Bgp{
		Global: &huawei.HuaweiBgp_Bgp_Global{YangEnable: ygot.Bool(true)},
		BaseProcess: &huawei.HuaweiBgp_Bgp_BaseProcess{
			Enable:       ygot.Bool(true),
			As:           ygot.String("100"),
			CheckFirstAs: ygot.Bool(true),
			AsPathLimit:  ygot.Uint16(50),
			GracefulRestart: &huawei.HuaweiBgp_Bgp_BaseProcess_GracefulRestart{
				Enable:      ygot.Bool(true),
				RestartTime: ygot.Uint16(120),
			},
			Timer: &huawei.HuaweiBgp_Bgp_BaseProcess_Timer{
				HoldTime:      ygot.Uint32(180),
				KeepAliveTime: ygot.Uint32(60),
			},
		},
	}
}

// TestReconciler_Integration_BgpConvergesAndReadable：公网 BGP 意图（标量 + 嵌套
// 容器 graceful-restart/timer）下发→回读→二次对账必须收敛（Changes==0，BGP-04/
// XC-05）。锁死「容器根一直漂移」：若 diff/client/编解码路径有 list 中心假设，第二轮
// 会算出非零 changes（永久漂移）。
func TestReconciler_Integration_BgpConvergesAndReadable(t *testing.T) {
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
	if err := cs.Set(deviceID, BgpPath, bgpDesired()); err != nil {
		t.Fatalf("config store set: %v", err)
	}

	r := New(cs, pool, ds)
	req := reconcile.Request{DeviceID: deviceID, Path: BgpPath}
	ctx := context.Background()

	// 第一轮：设备无 BGP 配置 → 检测漂移并下发
	first := r.Reconcile(ctx, req)
	if first.Error != nil {
		t.Fatalf("first reconcile: %v", first.Error)
	}
	assert.Greater(t, first.Changes, 0, "首轮应检测到漂移并下发 BGP 配置")

	// 回读设备：BGP 配置真正落盘（含嵌套容器字段）
	dc := &deviceClient{clientPool: pool, resolver: ds}
	got, err := dc.Get(ctx, deviceID)
	if err != nil {
		t.Fatalf("readback: %v", err)
	}
	bgp, ok := got.(*huawei.HuaweiBgp_Bgp)
	if !ok {
		t.Fatalf("unexpected readback type %T", got)
	}
	if bgp.BaseProcess == nil || bgp.BaseProcess.As == nil || *bgp.BaseProcess.As != "100" ||
		bgp.BaseProcess.Enable == nil || !*bgp.BaseProcess.Enable {
		t.Fatalf("回读缺基础字段: %#v", bgp.BaseProcess)
	}
	if bgp.BaseProcess.GracefulRestart == nil || bgp.BaseProcess.GracefulRestart.RestartTime == nil ||
		*bgp.BaseProcess.GracefulRestart.RestartTime != 120 {
		t.Fatalf("回读缺嵌套容器 graceful-restart: %#v", bgp.BaseProcess.GracefulRestart)
	}

	// 第二轮：desired==actual → 必须收敛（否则容器根永久漂移）
	second := r.Reconcile(ctx, req)
	if second.Error != nil {
		t.Fatalf("second reconcile: %v", second.Error)
	}
	assert.Equal(t, 0, second.Changes, "二轮必须收敛（Changes==0），否则容器根一直漂移")
}

// TestReconciler_Integration_BgpEditConfigFailure：设备 edit-config 拒绝时对账须
// 明确报错、不 panic（R08/§9：下发失败保留原配置、缓存不更新——此处验对账层诚实
// 透出错误，缓存不更新由 API 层保证）。
func TestReconciler_Integration_BgpEditConfigFailure(t *testing.T) {
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
	if err := cs.Set(deviceID, BgpPath, bgpDesired()); err != nil {
		t.Fatalf("config store set: %v", err)
	}
	r := New(cs, pool, ds)
	result := r.Reconcile(context.Background(), reconcile.Request{DeviceID: deviceID, Path: BgpPath})
	assert.Error(t, result.Error, "edit-config 被拒时对账须报错（诚实透出，非静默成功）")
}
