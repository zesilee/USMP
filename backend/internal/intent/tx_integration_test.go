package intent

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	vlanctl "github.com/leezesi/usmp/backend/internal/controller/vlan"
	"github.com/leezesi/usmp/backend/internal/generated/business"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
)

// BIO-03 / BVS-03/BVS-04（矩阵 A1/A2/A3/A9/A10①③）—— TxCoordinator 对双模拟网元
// 的跨设备 2PC：全体成功 / prepare 部分失败零残留 / 能力缺失降级 / 并发无竞态。

const (
	devA = "10.0.0.1"
	devB = "10.0.0.2"
)

// startTxSim 起一台模拟网元。addr 用 loopback 别名（127.0.0.1/127.0.0.2）区分
// 两台设备：ClientPool 按 info.IP 键控（生产设备 IP 唯一），同 IP 双 sim 会被
// 池合并成一条连接。
func startTxSim(t *testing.T, addr string, sc *netsim.ScenarioConfig) *netsim.Simulator {
	t.Helper()
	sim := netsim.NewSimulator()
	sim.SetListen(addr, 0)
	if sc != nil {
		sim.SetScenario(sc)
	}
	if err := sim.Start(); err != nil {
		t.Fatalf("start sim on %s: %v", addr, err)
	}
	t.Cleanup(sim.Stop)
	return sim
}

func txStack(t *testing.T, simA, simB *netsim.Simulator) *TxCoordinator {
	t.Helper()
	ds := device.NewStore()
	reg := func(id string, sim *netsim.Simulator) {
		ds.Put(id, client.DeviceConnectionInfo{
			IP: sim.Addr(), Port: sim.Port(),
			Username: sim.Username(), Password: sim.Password(),
			Protocol: client.ProtocolNETCONF, Timeout: 5 * time.Second,
		})
	}
	reg(devA, simA)
	reg(devB, simB)
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	t.Cleanup(func() { _ = pool.CloseAll() })
	return NewTxCoordinator(pool, ds, 2*time.Second)
}

func mustExpand(t *testing.T, vlanID uint16) []Fragment {
	t.Helper()
	id := vlanID
	name := "tx-test"
	spec := &business.UsmpBusinessVlan_BusinessVlanService{
		VlanId: &id,
		Name:   &name,
		Devices: map[string]*business.UsmpBusinessVlan_BusinessVlanService_Devices{
			devA: {Ip: s(devA), AccessPorts: []string{"GE0/0/1"}, TrunkPorts: []string{"GE0/0/2"}},
			devB: {Ip: s(devB), TrunkPorts: []string{"GE0/0/3"}},
		},
	}
	frags, _, err := ExpandBusinessVlan(spec)
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	return frags
}

// 矩阵 A1/A2/A6：全体成功——两台 sim running 逐项断言 vlan 与端口 L2 属性。
func TestTxPushAllSuccess_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	simA, simB := startTxSim(t, "127.0.0.1", nil), startTxSim(t, "127.0.0.2", nil)
	tx := txStack(t, simA, simB)

	results := tx.Push(context.Background(), mustExpand(t, 100))
	for dev, res := range results {
		if res.Err != nil {
			t.Fatalf("device %s failed: %v", dev, res.Err)
		}
		if res.NonTransactional {
			t.Errorf("device %s unexpectedly non-transactional", dev)
		}
	}

	if name, ok := simA.RunningHuaweiVLANs()[100]; !ok || name != "tx-test" {
		t.Fatalf("simA vlan 100 = %q,%v want tx-test", name, ok)
	}
	if _, ok := simB.RunningHuaweiVLANs()[100]; !ok {
		t.Fatal("simB missing vlan 100")
	}
	ifacesA := simA.RunningHuaweiInterfaces()
	acc := ifacesA["GE0/0/1"]
	if acc == nil || acc.L2.LinkType != int(huawei.HuaweiEthernet_LinkType_access) || acc.L2.Pvid != 100 {
		t.Fatalf("simA GE0/0/1 L2 = %+v, want access+pvid100", acc)
	}
	trk := ifacesA["GE0/0/2"]
	if trk == nil || trk.L2.LinkType != int(huawei.HuaweiEthernet_LinkType_trunk) || trk.L2.TrunkVlans != "100" {
		t.Fatalf("simA GE0/0/2 L2 = %+v, want trunk+100", trk)
	}
	if b := simB.RunningHuaweiInterfaces()["GE0/0/3"]; b == nil || b.L2.TrunkVlans != "100" {
		t.Fatalf("simB GE0/0/3 L2 = %+v, want trunk 100", b)
	}
}

// 矩阵 A3：不抹除——设备上既有手工 vlan 300，意图 2PC 后 300 仍在。
func TestTxPushPreservesManualConfig_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	simA, simB := startTxSim(t, "127.0.0.1", nil), startTxSim(t, "127.0.0.2", nil)
	simA.SetRunningConfigXML([]byte(`<vlan xmlns="urn:huawei:params:xml:ns:yang:huawei-vlan"><vlans><vlan><id>300</id><name>manual</name></vlan></vlans></vlan>`))
	tx := txStack(t, simA, simB)

	results := tx.Push(context.Background(), mustExpand(t, 100))
	for dev, res := range results {
		if res.Err != nil {
			t.Fatalf("device %s failed: %v", dev, res.Err)
		}
	}
	vlans := simA.RunningHuaweiVLANs()
	if _, ok := vlans[300]; !ok {
		t.Fatalf("manual vlan 300 clobbered by intent push: %v", vlans)
	}
	if _, ok := vlans[100]; !ok {
		t.Fatalf("intent vlan 100 missing: %v", vlans)
	}
}

// 矩阵 A10①：prepare 部分失败——全体 discard，两台 running 零残留。
func TestTxPushPrepareFailureAborts_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	scB := netsim.NewScenarioConfig()
	scB.ErrorOnRPC["edit-config"] = errInjected
	simA, simB := startTxSim(t, "127.0.0.1", nil), startTxSim(t, "127.0.0.2", scB)
	tx := txStack(t, simA, simB)

	results := tx.Push(context.Background(), mustExpand(t, 100))
	for dev, res := range results {
		if res.Err == nil {
			t.Fatalf("device %s should fail when transaction aborts", dev)
		}
		if !strings.Contains(res.Err.Error(), "prepare failed") {
			t.Errorf("device %s error should carry abort reason, got %v", dev, res.Err)
		}
	}
	if _, ok := simA.RunningHuaweiVLANs()[100]; ok {
		t.Fatal("simA running must stay clean after aborted transaction")
	}
	if _, ok := simB.RunningHuaweiVLANs()[100]; ok {
		t.Fatal("simB running must stay clean after aborted transaction")
	}
}

// 矩阵 A10③：能力缺失降级——simB 无 :confirmed-commit → 普通 commit，标记非事务。
func TestTxPushCapabilityDowngrade_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	scB := netsim.NewScenarioConfig()
	scB.DisableConfirmedCommit = true
	simA, simB := startTxSim(t, "127.0.0.1", nil), startTxSim(t, "127.0.0.2", scB)
	tx := txStack(t, simA, simB)

	results := tx.Push(context.Background(), mustExpand(t, 100))
	if res := results[devA]; res.Err != nil || res.NonTransactional {
		t.Fatalf("devA = %+v, want transactional success", res)
	}
	if res := results[devB]; res.Err != nil || !res.NonTransactional {
		t.Fatalf("devB = %+v, want non-transactional success", res)
	}
	if _, ok := simA.RunningHuaweiVLANs()[100]; !ok {
		t.Fatal("simA missing vlan 100")
	}
	if _, ok := simB.RunningHuaweiVLANs()[100]; !ok {
		t.Fatal("simB missing vlan 100")
	}
}

// 矩阵 A9：并发两意图对同设备——每设备互斥串行化，无竞态无丢更新（-race）。
func TestTxPushConcurrentIntents_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	simA, simB := startTxSim(t, "127.0.0.1", nil), startTxSim(t, "127.0.0.2", nil)
	tx := txStack(t, simA, simB)

	var wg sync.WaitGroup
	for _, id := range []uint16{100, 200} {
		wg.Add(1)
		go func(vlanID uint16) {
			defer wg.Done()
			results := tx.Push(context.Background(), mustExpand(t, vlanID))
			for dev, res := range results {
				if res.Err != nil {
					t.Errorf("vlan %d device %s: %v", vlanID, dev, res.Err)
				}
			}
		}(id)
	}
	wg.Wait()

	vlansA := simA.RunningHuaweiVLANs()
	if _, ok := vlansA[100]; !ok {
		t.Errorf("simA lost vlan 100 under concurrency: %v", vlansA)
	}
	if _, ok := vlansA[200]; !ok {
		t.Errorf("simA lost vlan 200 under concurrency: %v", vlansA)
	}
}

var errInjected = &injectedError{}

type injectedError struct{}

func (*injectedError) Error() string { return "injected edit-config failure" }

// BIO-04（矩阵 A4 变体）：2PC 成功后设备漂移（手工删 vlan）——写入的 desired 驱动
// 原生 vlan 声明式对账单设备修复，不再触发跨设备事务。
func TestDriftRepairSingleDevice_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	simA, simB := startTxSim(t, "127.0.0.1", nil), startTxSim(t, "127.0.0.2", nil)
	ds := device.NewStore()
	reg := func(id string, sim *netsim.Simulator) {
		ds.Put(id, client.DeviceConnectionInfo{
			IP: sim.Addr(), Port: sim.Port(),
			Username: sim.Username(), Password: sim.Password(),
			Protocol: client.ProtocolNETCONF, Timeout: 5 * time.Second,
		})
	}
	reg(devA, simA)
	reg(devB, simB)
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	t.Cleanup(func() { _ = pool.CloseAll() })
	tx := NewTxCoordinator(pool, ds, 2*time.Second)

	frags := mustExpand(t, 100)
	results := tx.Push(context.Background(), frags)
	for dev, res := range results {
		if res.Err != nil {
			t.Fatalf("push %s: %v", dev, res.Err)
		}
	}
	cs := newStore()
	writeDesired(cs, nil, frags)

	// 漂移：设备侧 vlan 100 被手工清掉。
	simA.SetRunningConfigXML([]byte(`<vlan xmlns="urn:huawei:params:xml:ns:yang:huawei-vlan"><vlans><vlan><id>999</id><name>other</name></vlan></vlans></vlan>`))
	if _, ok := simA.RunningHuaweiVLANs()[100]; ok {
		t.Fatal("precondition: drift not injected")
	}

	// 原生 vlan 声明式对账（单设备，非事务）修复。
	rec := vlanctl.New(cs, pool, ds)
	res := rec.Reconcile(context.Background(), reconcile.Request{DeviceID: devA, Path: VlanPath})
	if res.Error != nil {
		t.Fatalf("native reconcile: %v", res.Error)
	}
	if _, ok := simA.RunningHuaweiVLANs()[100]; !ok {
		t.Fatalf("drifted vlan 100 not repaired: %v", simA.RunningHuaweiVLANs())
	}
}
