package api

import (
	"context"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/internal/cache"
	systemctl "github.com/leezesi/usmp/backend/internal/controller/system"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
	testsupport "github.com/leezesi/usmp/backend/simulator/netconfsim/testsupport"
)

// B2（BR-05/BR-06/DR-05）：system form-tab 形状端到端——子路径扁平载荷（模块控制台
// 「基本属性」tab 的真实形状）经锚点包裹解码 → 锚点 key 入库 → 对账 → NETCONF
// edit-config → 模拟网元运行配置可见。这是新契约下唯一依赖包裹的真实生产流。
func TestSystemFormTab_Integration_SubpathFlatToDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	sim := netsim.NewSimulator()
	if err := sim.Start(); err != nil {
		t.Fatalf("start sim: %v", err)
	}
	defer sim.Stop()

	c := cache.NewTTLLRUCache(100, 30*time.Second, 1*time.Minute)
	cs := manager.NewInMemoryConfigStore(c)
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()

	// form-tab 真实形状：POST /config/:ip/system:system/system:system-info + 扁平叶。
	typed, anchor, err := convertConfigAnchored("/system:system/system:system-info",
		map[string]interface{}{"sys-name": "formtab-sw", "sys-contact": "ops@usmp", "sys-location": "Lab-3F"})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if anchor != "/system:system" {
		t.Fatalf("anchor = %q, want /system:system", anchor)
	}

	deviceID := "sim-system"
	ds := device.NewStore()
	_ = ds.Put(deviceID, client.DeviceConnectionInfo{
		IP: sim.Addr(), Port: sim.Port(), Username: sim.Username(), Password: sim.Password(), Protocol: client.ProtocolNETCONF,
	})
	// 与 SetConfig 同语义：锚点 key 合并入库。
	if err := storeConfigMerged(cs, deviceID, anchor, typed); err != nil {
		t.Fatalf("store: %v", err)
	}

	// 周期对账形状：模块路径入队（锚点归一化后必然命中同一 key）。
	res := systemctl.New(cs, pool, ds).Reconcile(context.Background(), reconcile.Request{DeviceID: deviceID, Path: anchor})
	if res.Error != nil {
		t.Fatalf("reconcile: %v", res.Error)
	}

	testsupport.AssertHuaweiSystem(t, sim, "formtab-sw", "ops@usmp", "Lab-3F")
}
