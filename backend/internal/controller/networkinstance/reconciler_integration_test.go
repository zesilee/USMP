package networkinstance

import (
	"context"
	"fmt"
	"sync"
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

func inst(name, desc string) *huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance {
	return &huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance{
		Name: ygot.String(name), Description: ygot.String(desc),
	}
}

// niDesired 构造 global 标量 + 嵌套 instance list（多条目），驱动原生 config-true 面。
func niDesired(instances ...*huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance) *huawei.HuaweiNetworkInstance_NetworkInstance {
	m := map[string]*huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance{}
	for _, in := range instances {
		m[*in.Name] = in
	}
	return &huawei.HuaweiNetworkInstance_NetworkInstance{
		Global: &huawei.HuaweiNetworkInstance_NetworkInstance_Global{
			CfgRouterId:     ygot.String("1.1.1.1"),
			AsNotationPlain: ygot.Bool(true),
		},
		Instances: &huawei.HuaweiNetworkInstance_NetworkInstance_Instances{Instance: m},
	}
}

func newHarness(t *testing.T, sim *netsim.Simulator) (*NetworkInstanceReconciler, reconcile.ConfigStore, device.Store, string, client.ClientPool) {
	t.Helper()
	c := cache.NewTTLLRUCache(100, 30*time.Second, 1*time.Minute)
	cs := manager.NewInMemoryConfigStore(c)
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	ds, deviceID := simStore(sim)
	return New(cs, pool, ds), cs, ds, deviceID, pool
}

// TestReconciler_Integration_NiConvergesAndReadable：network-instance 意图（global
// 标量 + 嵌套 instance list 多条目）下发→回读→二次对账必须收敛（Changes==0，NI-01/
// NI-04/XC-05 嵌套 list）。锁死「容器根+嵌套 list 一直漂移」：若编解码/diff 路径有
// 缺陷，第二轮会算出非零 changes（永久漂移）。
func TestReconciler_Integration_NiConvergesAndReadable(t *testing.T) {
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

	if err := cs.Set(deviceID, NetworkInstancePath, niDesired(inst("vpn-a", "first"), inst("vpn-b", "second"))); err != nil {
		t.Fatalf("config store set: %v", err)
	}

	r2 := r
	req := reconcile.Request{DeviceID: deviceID, Path: NetworkInstancePath}
	ctx := context.Background()

	first := r2.Reconcile(ctx, req)
	if first.Error != nil {
		t.Fatalf("first reconcile: %v", first.Error)
	}
	assert.Greater(t, first.Changes, 0, "首轮应检测到漂移并下发 network-instance 配置")

	// 回读设备：global + 嵌套 list 真正落盘
	dc := &deviceClient{clientPool: pool, resolver: ds}
	got, err := dc.Get(ctx, deviceID)
	if err != nil {
		t.Fatalf("readback: %v", err)
	}
	ni, ok := got.(*huawei.HuaweiNetworkInstance_NetworkInstance)
	if !ok {
		t.Fatalf("unexpected readback type %T", got)
	}
	if ni.Global == nil || ni.Global.CfgRouterId == nil || *ni.Global.CfgRouterId != "1.1.1.1" {
		t.Fatalf("回读缺 global: %#v", ni.Global)
	}
	if ni.Instances == nil || ni.Instances.Instance["vpn-a"] == nil ||
		ni.Instances.Instance["vpn-a"].Description == nil || *ni.Instances.Instance["vpn-a"].Description != "first" {
		t.Fatalf("回读缺嵌套 list 条目 vpn-a: %#v", ni.Instances)
	}
	if ni.Instances.Instance["vpn-b"] == nil {
		t.Fatalf("回读缺嵌套 list 条目 vpn-b: %#v", ni.Instances)
	}

	// 第二轮：desired==actual → 必须收敛（否则容器根+嵌套 list 永久漂移）
	second := r2.Reconcile(ctx, req)
	if second.Error != nil {
		t.Fatalf("second reconcile: %v", second.Error)
	}
	assert.Equal(t, 0, second.Changes, "二轮必须收敛（Changes==0），否则永久漂移")
}

// TestReconciler_Integration_NiAddAndEdit：嵌套 list 增（新增条目）与改（改
// description）在 MVP merge 收敛下均端到端生效并收敛（NI-05 嵌套 list 增/改）。
func TestReconciler_Integration_NiAddAndEdit(t *testing.T) {
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
	dc := &deviceClient{clientPool: pool, resolver: ds}

	// 初始：一条 vpn-a
	_ = cs.Set(deviceID, NetworkInstancePath, niDesired(inst("vpn-a", "v1")))
	if res := r.Reconcile(ctx, req); res.Error != nil {
		t.Fatalf("reconcile#1: %v", res.Error)
	}

	// 增：加 vpn-b；改：vpn-a 描述 v1→v2
	_ = cs.Set(deviceID, NetworkInstancePath, niDesired(inst("vpn-a", "v2"), inst("vpn-b", "b1")))
	res := r.Reconcile(ctx, req)
	if res.Error != nil {
		t.Fatalf("reconcile#2: %v", res.Error)
	}
	assert.Greater(t, res.Changes, 0, "增/改应检测到漂移")

	got, _ := dc.Get(ctx, deviceID)
	ni := got.(*huawei.HuaweiNetworkInstance_NetworkInstance)
	if ni.Instances.Instance["vpn-a"].Description == nil || *ni.Instances.Instance["vpn-a"].Description != "v2" {
		t.Fatalf("改未生效: %#v", ni.Instances.Instance["vpn-a"])
	}
	if ni.Instances.Instance["vpn-b"] == nil {
		t.Fatalf("增未生效: %#v", ni.Instances)
	}

	// 幂等：desired 不变再对账 → 收敛
	second := r.Reconcile(ctx, req)
	assert.Equal(t, 0, second.Changes, "增/改后二轮须幂等收敛")
}

// TestReconciler_Integration_NiDeclarativeRemoveIsSubset：锁死平台契约——声明式对账
// 按 subset 语义**不删除** actual 中多出的 instance（NI-06、config-delete-semantics）。
// 从 desired 移除 vpn-b 后：Changes==0、设备保留 vpn-b、无永久漂移。设备侧删除须走独立
// DELETE 命令通道（本 MVP 推迟债）。`_public_` 由此天然永不被声明式通道删除。
func TestReconciler_Integration_NiDeclarativeRemoveIsSubset(t *testing.T) {
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
	dc := &deviceClient{clientPool: pool, resolver: ds}

	_ = cs.Set(deviceID, NetworkInstancePath, niDesired(inst("vpn-a", "a"), inst("_public_", "")))
	if res := r.Reconcile(ctx, req); res.Error != nil {
		t.Fatalf("seed reconcile: %v", res.Error)
	}

	// desired 移除 vpn-a 与 _public_（只留一条）→ 声明式通道不应删除它们
	_ = cs.Set(deviceID, NetworkInstancePath, niDesired(inst("vpn-c", "c")))
	res := r.Reconcile(ctx, req)
	if res.Error != nil {
		t.Fatalf("remove reconcile: %v", res.Error)
	}

	got, _ := dc.Get(ctx, deviceID)
	ni := got.(*huawei.HuaweiNetworkInstance_NetworkInstance)
	if ni.Instances.Instance["vpn-a"] == nil {
		t.Error("声明式 subset 语义下 vpn-a 不应被删除（删除须走 DELETE 命令通道）")
	}
	if ni.Instances.Instance["_public_"] == nil {
		t.Error("_public_ 不可删——声明式通道下须天然保留")
	}

	// 无永久漂移：再对账收敛
	second := r.Reconcile(ctx, req)
	assert.Equal(t, 0, second.Changes, "subset 语义下无永久漂移")
}

// TestReconciler_Integration_NiConcurrentReconcile：多协程并发对账（go test -race）
// 无数据竞态、无 panic，收敛一致（R09/NI-05）。共享 ConfigStore/cache/clientPool +
// 有状态 sim 的并发安全防线。
func TestReconciler_Integration_NiConcurrentReconcile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim := netsim.NewSimulator()
	if err := sim.Start(); err != nil {
		t.Fatalf("start simulator: %v", err)
	}
	defer sim.Stop()

	r, cs, _, deviceID, pool := newHarness(t, sim)
	defer pool.CloseAll()
	ctx := context.Background()
	req := reconcile.Request{DeviceID: deviceID, Path: NetworkInstancePath}

	_ = cs.Set(deviceID, NetworkInstancePath, niDesired(inst("vpn-a", "a"), inst("vpn-b", "b")))

	const n = 8
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs[idx] = r.Reconcile(ctx, req).Error
		}(i)
	}
	wg.Wait()
	for i, e := range errs {
		if e != nil {
			t.Errorf("并发 reconcile[%d] 报错: %v", i, e)
		}
	}
	// 并发后最终态收敛
	final := r.Reconcile(ctx, req)
	assert.Equal(t, 0, final.Changes, "并发对账后最终须收敛")
}

// TestReconciler_Integration_NiEditConfigFailure：设备 edit-config 拒绝时对账须明确
// 报错、不 panic（R08/§9：下发失败保留原配置、缓存不更新）。
func TestReconciler_Integration_NiEditConfigFailure(t *testing.T) {
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

	r, cs, _, deviceID, pool := newHarness(t, sim)
	defer pool.CloseAll()
	_ = cs.Set(deviceID, NetworkInstancePath, niDesired(inst("vpn-a", "x")))
	result := r.Reconcile(context.Background(), reconcile.Request{DeviceID: deviceID, Path: NetworkInstancePath})
	assert.Error(t, result.Error, "edit-config 被拒时对账须报错（诚实透出，非静默成功）")
}
