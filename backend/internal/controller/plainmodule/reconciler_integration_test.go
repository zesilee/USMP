package plainmodule

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/openconfig/ygot/ygot"
	"github.com/stretchr/testify/assert"

	"github.com/leezesi/usmp/backend/internal/cache"
	"github.com/leezesi/usmp/backend/internal/testutil/yangsample"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
	yangdriver "github.com/leezesi/usmp/backend/pkg/yang-runtime/driver"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
)

// 代表模块（每任务域≥1，B2 参数化矩阵）：全量 57 模块的编解码面已由
// drivers 往返矩阵覆盖，sim 端到端抽样覆盖「下发→回读→收敛」链路——
// 含 2b 迁表四模块（tunnel-management/xpl/routing-policy/acl，承接其旧
// per-module 集成测试职责）与各域新模块。
var integrationAnchors = []string{
	"/tunnel-management:tunnel-management", // 隧道（2b 迁表，根名口径断链回归）
	"/xpl:xpl",                             // 路由策略-xpl（2b 迁表）
	"/routing-policy:routing-policy",       // 路由策略（2b 迁表，根名口径断链回归）
	"/acl:acl",                             // 安全-ACL（2b 迁表）
	"/ntp:ntp",                             // 系统管理-时间
	"/syslog:syslog",                       // 系统管理-日志
	"/mstp:mstp",                           // 以太交换-生成树
	"/vrrp:vrrp",                           // 可靠性
	"/sflow:sflow",                         // 网络管理与监控
	"/ospfv2:ospfv2",                       // IP 路由
	"/arp:arp",                             // IP 业务
	"/evpn:evpn",                           // overlay
	"/hwtacacs:hwtacacs",                   // 安全-AAA
	"/qos:qos",                             // QoS（deviation 豁免模块代表）
	"/lldp:lldp",                           // 网络发现（deviation leafref 豁免代表）
	"/bfd:bfd",                             // 可靠性-检测（存量漏网叶代表）
}

func simStore(sim *netsim.Simulator) (device.Store, string) {
	ds := device.NewStore()
	id := "sim"
	ds.Put(id, client.DeviceConnectionInfo{
		IP: sim.Addr(), Port: sim.Port(), Username: sim.Username(), Password: sim.Password(), Protocol: client.ProtocolNETCONF,
	})
	return ds, id
}

// TestPlainModule_Integration_ConvergeAndReadback：对每个代表模块，
// schema 采样 desired → 首轮下发（Changes>0）→ 回读与 desired 无差 →
// 二轮收敛（Changes==0，锁「容器根永久漂移」，同 tunnelmgmt 先例 TNLM-04）。
func TestPlainModule_Integration_ConvergeAndReadback(t *testing.T) {
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
	ctx := context.Background()

	for _, anchor := range integrationAnchors {
		anchor := anchor
		t.Run(anchor, func(t *testing.T) {
			d, ok := yangdriver.EncoderFor(anchor)
			if !ok {
				t.Fatalf("描述符缺失: %s", anchor)
			}
			desired := d.NewStruct()
			if !yangsample.Populate(desired, d.XML.Schema()) {
				t.Skipf("无可采样标量（enum/union 键模块），编解码面由往返矩阵覆盖")
			}
			if err := cs.Set(deviceID, anchor, desired); err != nil {
				t.Fatalf("config store set: %v", err)
			}

			r := New(cs, pool, ds, anchor)
			req := reconcile.Request{DeviceID: deviceID, Path: anchor}

			first := r.Reconcile(ctx, req)
			if first.Error != nil {
				t.Fatalf("first reconcile: %v", first.Error)
			}
			assert.Greater(t, first.Changes, 0, "首轮应检测漂移并下发")

			got, err := r.client().Get(ctx, deviceID)
			if err != nil {
				t.Fatalf("readback: %v", err)
			}
			n, err := ygot.Diff(got.(ygot.GoStruct), desired.(ygot.GoStruct))
			if err != nil {
				t.Fatalf("diff: %v", err)
			}
			assert.Empty(t, n.GetUpdate(), "回读应与 desired 无差")

			second := r.Reconcile(ctx, req)
			if second.Error != nil {
				t.Fatalf("second reconcile: %v", second.Error)
			}
			assert.Equal(t, 0, second.Changes, "二轮必须收敛（容器根永久漂移防线）")
		})
	}
}

// TestPlainModule_Integration_EditConfigFailure：设备拒绝 edit-config 时对账
// 须明确报错不 panic（R08/§9，负路径，以 ntp 为样本）。
func TestPlainModule_Integration_EditConfigFailure(t *testing.T) {
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

	const anchor = "/mstp:mstp"
	d, _ := yangdriver.EncoderFor(anchor)
	desired := d.NewStruct()
	if !yangsample.Populate(desired, d.XML.Schema()) {
		t.Fatal("样本模块必须可采样（否则 desired 为空、无下发即无负路径）")
	}
	if err := cs.Set(deviceID, anchor, desired); err != nil {
		t.Fatalf("config store set: %v", err)
	}
	r := New(cs, pool, ds, anchor)
	result := r.Reconcile(context.Background(), reconcile.Request{DeviceID: deviceID, Path: anchor})
	assert.Error(t, result.Error, "edit-config 被拒时须诚实报错")
}
