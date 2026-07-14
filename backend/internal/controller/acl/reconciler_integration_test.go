package acl

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

// aclDesired：groups/group(IPv4, 含 mandatory enum type) + group6s/group6(IPv6)。
// BGP 2b ACL group leafref 引用的目标实例。
func aclDesired() *huawei.HuaweiAcl_Acl {
	a := &huawei.HuaweiAcl_Acl{
		Groups:  &huawei.HuaweiAcl_Acl_Groups{},
		Group6S: &huawei.HuaweiAcl_Acl_Group6S{},
	}
	g, _ := a.Groups.NewGroup("G-A")
	g.Type = huawei.HuaweiAcl_Group4Type_basic
	g.Description = ygot.String("acl group a")
	g6, _ := a.Group6S.NewGroup6("G6-A")
	g6.Type = huawei.HuaweiAcl_Group6Type_basic
	return a
}

// TestReconciler_Integration_AclConvergesAndReadable：acl 意图（嵌套 group list + 枚举
// type）下发→回读→二次对账必须收敛（Changes==0，ACL-04/XC-05）。锁死「容器根一直漂移」
// 与「枚举字段收敛」（枚举值编解码不一致会致永久漂移）。
func TestReconciler_Integration_AclConvergesAndReadable(t *testing.T) {
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
	if err := cs.Set(deviceID, AclPath, aclDesired()); err != nil {
		t.Fatalf("config store set: %v", err)
	}

	r := New(cs, pool, ds)
	req := reconcile.Request{DeviceID: deviceID, Path: AclPath}
	ctx := context.Background()

	// 第一轮：设备无 acl 配置 → 检测漂移并下发
	first := r.Reconcile(ctx, req)
	if first.Error != nil {
		t.Fatalf("first reconcile: %v", first.Error)
	}
	assert.Greater(t, first.Changes, 0, "首轮应检测到漂移并下发 acl 配置")

	// 回读设备：group（含枚举 type）真正落盘
	dc := &deviceClient{clientPool: pool, resolver: ds}
	got, err := dc.Get(ctx, deviceID)
	if err != nil {
		t.Fatalf("readback: %v", err)
	}
	a, ok := got.(*huawei.HuaweiAcl_Acl)
	if !ok {
		t.Fatalf("unexpected readback type %T", got)
	}
	if a.Groups == nil || a.Groups.Group["G-A"] == nil {
		t.Fatalf("回读缺 group 实例: %#v", a.Groups)
	}
	if g := a.Groups.Group["G-A"]; g.Type != huawei.HuaweiAcl_Group4Type_basic {
		t.Fatalf("回读缺 group 枚举 type: %#v", g)
	}
	if a.Group6S == nil || a.Group6S.Group6["G6-A"] == nil {
		t.Fatalf("回读缺 group6 实例: %#v", a.Group6S)
	}

	// 第二轮：desired==actual → 必须收敛（枚举一致则不漂移）
	second := r.Reconcile(ctx, req)
	if second.Error != nil {
		t.Fatalf("second reconcile: %v", second.Error)
	}
	assert.Equal(t, 0, second.Changes, "二轮必须收敛（Changes==0），否则容器根/枚举一直漂移")
}

// TestReconciler_Integration_AclEditConfigFailure：设备 edit-config 拒绝时对账须明确报错、
// 不 panic（R08/§9）。
func TestReconciler_Integration_AclEditConfigFailure(t *testing.T) {
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
	if err := cs.Set(deviceID, AclPath, aclDesired()); err != nil {
		t.Fatalf("config store set: %v", err)
	}
	r := New(cs, pool, ds)
	result := r.Reconcile(context.Background(), reconcile.Request{DeviceID: deviceID, Path: AclPath})
	assert.Error(t, result.Error, "edit-config 被拒时对账须报错（诚实透出，非静默成功）")
}
