package api

import (
	"context"
	"sync"
	"testing"

	vlanctl "github.com/leezesi/usmp/backend/internal/controller/vlan"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	testsupport "github.com/leezesi/usmp/backend/simulator/netconfsim/testsupport"
)

// P3：全属性矩阵——一个 VLAN 配齐所有可配置属性（枚举用字符串名），逐项断言到设备。
func TestVlanConfig_Integration_AllAttributes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim, cs, pool, ds, deviceID := newVlanSimStack(t)
	defer sim.Stop()
	defer pool.CloseAll()

	applyVlan(t, cs, pool, ds, deviceID, []interface{}{
		map[string]interface{}{
			"id":                        float64(100),
			"name":                      "full",
			"description":               "all-attrs",
			"type":                      "common",
			"admin-status":              "up",
			"mac-learning":              "enable",
			"mac-aging-time":            float64(300),
			"broadcast-discard":         "enable",
			"statistic-enable":          "enable",
			"statistic-discard":         "disable",
			"unknown-multicast-discard": "enable",
			"suppression":               map[string]interface{}{"inbound": "enable", "outbound": "disable"},
			"unkown-unicast-discard":    map[string]interface{}{"discard": "enable", "mac-learning-enable": "disable"},
		},
	})

	en := int(huawei.HuaweiVlan_EnableStatus_enable)
	dis := int(huawei.HuaweiVlan_EnableStatus_disable)
	testsupport.AssertHuaweiVlanName(t, sim, 100, "full")
	testsupport.AssertHuaweiVlanDescription(t, sim, 100, "all-attrs")
	testsupport.AssertHuaweiVlanType(t, sim, 100, int(huawei.HuaweiVlan_VlanType_common))
	testsupport.AssertHuaweiVlanAdminStatus(t, sim, 100, int(huawei.HuaweiVlan_AdminStatus_up))
	testsupport.AssertHuaweiVlanMacLearning(t, sim, 100, en)
	testsupport.AssertHuaweiVlanMacAgingTime(t, sim, 100, 300)
	testsupport.AssertHuaweiVlanBroadcastDiscard(t, sim, 100, en)
	testsupport.AssertHuaweiVlanStatisticEnable(t, sim, 100, en)
	testsupport.AssertHuaweiVlanStatisticDiscard(t, sim, 100, dis)
	testsupport.AssertHuaweiVlanUnknownMulticastDiscard(t, sim, 100, en)
	testsupport.AssertHuaweiVlanSuppression(t, sim, 100, en, dis)
	testsupport.AssertHuaweiVlanUnkownUnicastDiscard(t, sim, 100, en, dis)
}

// P2：幂等——同一 VLAN 二次对账应无变化、不报错。
func TestVlanConfig_Integration_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim, cs, pool, ds, deviceID := newVlanSimStack(t)
	defer sim.Stop()
	defer pool.CloseAll()

	applyVlan(t, cs, pool, ds, deviceID, []interface{}{
		map[string]interface{}{"id": float64(50), "name": "idem"},
	})
	// 第二次对账（desired 未变）——应无错、无 requeue
	res := reconcileVlan(cs, pool, ds, deviceID)
	if res.Error != nil {
		t.Fatalf("second reconcile errored: %v", res.Error)
	}
	if res.Requeue {
		t.Errorf("idempotent reconcile should not requeue")
	}
	testsupport.AssertHuaweiVlanExists(t, sim, 50)
	testsupport.AssertHuaweiVlanCount(t, sim, 1)
}

// P2：编辑（发完整条目）应保留未改动的属性。
func TestVlanConfig_Integration_EditPreservesAttributes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim, cs, pool, ds, deviceID := newVlanSimStack(t)
	defer sim.Stop()
	defer pool.CloseAll()

	applyVlan(t, cs, pool, ds, deviceID, []interface{}{
		map[string]interface{}{"id": float64(60), "name": "orig", "description": "keep-me", "admin-status": "up"},
	})
	// 编辑：改 name，其余按 UI 行为回填完整条目
	applyVlan(t, cs, pool, ds, deviceID, []interface{}{
		map[string]interface{}{"id": float64(60), "name": "edited", "description": "keep-me", "admin-status": "up"},
	})
	testsupport.AssertHuaweiVlanName(t, sim, 60, "edited")
	testsupport.AssertHuaweiVlanDescription(t, sim, 60, "keep-me")
	testsupport.AssertHuaweiVlanAdminStatus(t, sim, 60, int(huawei.HuaweiVlan_AdminStatus_up))
}

// P2：member-ports 多端口 + 改为单端口（整条 VLAN 替换语义）。
func TestVlanConfig_Integration_MemberPortsReplace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim, cs, pool, ds, deviceID := newVlanSimStack(t)
	defer sim.Stop()
	defer pool.CloseAll()

	twoPorts := map[string]interface{}{
		"id": float64(70), "name": "ports",
		"member-ports": map[string]interface{}{"member-port": []interface{}{
			map[string]interface{}{"interface-name": "GE0/0/1", "access-type": "access"},
			map[string]interface{}{"interface-name": "GE0/0/2", "access-type": "trunk"},
		}},
	}
	applyVlan(t, cs, pool, ds, deviceID, []interface{}{twoPorts})
	testsupport.AssertHuaweiVlanMemberPort(t, sim, 70, "GE0/0/1", int(huawei.HuaweiVlan_AccessType_access))
	testsupport.AssertHuaweiVlanMemberPort(t, sim, 70, "GE0/0/2", int(huawei.HuaweiVlan_AccessType_trunk))

	// 改为单端口（回填完整条目，仅保留 GE0/0/1）
	applyVlan(t, cs, pool, ds, deviceID, []interface{}{
		map[string]interface{}{"id": float64(70), "name": "ports",
			"member-ports": map[string]interface{}{"member-port": []interface{}{
				map[string]interface{}{"interface-name": "GE0/0/1", "access-type": "access"},
			}}},
	})
	testsupport.AssertHuaweiVlanMemberPort(t, sim, 70, "GE0/0/1", int(huawei.HuaweiVlan_AccessType_access))
}

// P2：读回——配置后设备运行配置可读回且一致（GET 路径）。
func TestVlanConfig_Integration_ReadBack(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim, cs, pool, ds, deviceID := newVlanSimStack(t)
	defer sim.Stop()
	defer pool.CloseAll()

	applyVlan(t, cs, pool, ds, deviceID, []interface{}{
		map[string]interface{}{"id": float64(80), "name": "readback", "description": "rb"},
	})
	// 通过设备运行配置读回（asserts 走 sim running config，即真实 edit-config 结果）
	vlans := sim.RunningHuaweiVLANsFull()
	v, ok := vlans[80]
	if !ok {
		t.Fatal("VLAN 80 读回失败")
	}
	if v.Name != "readback" || v.Description != "rb" {
		t.Errorf("读回属性不一致: name=%q desc=%q", v.Name, v.Description)
	}
}

// P2/R09：并发存储不同 VLAN 应无数据竞态、无丢更新（go test -race）。
//
// 只并发「配置存储」(storeConfigMerged，生产中并发的 SetConfig 处理器)，对账在最后单次执行
// ——对应生产中对账由控制器事件队列串行化，不会直接并发用同一 NETCONF 连接。此测试专测
// 合并存储在并发下的原子性(锁 + 非原地合并)：无丢更新、无竞态。
func TestVlanConfig_Integration_ConcurrentNoRace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim, cs, pool, ds, deviceID := newVlanSimStack(t)
	defer sim.Stop()
	defer pool.CloseAll()

	const n = 16
	var wg sync.WaitGroup
	for i := 1; i <= n; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			typed, err := convertConfig(vlanPath, map[string]interface{}{
				"vlan": []interface{}{map[string]interface{}{"id": float64(id), "name": "c"}},
			})
			if err != nil {
				return
			}
			_ = storeConfigMerged(cs, deviceID, vlanPath, typed)
		}(i)
	}
	wg.Wait()

	// 无丢更新：并发合并后 desired 应含全部 n 个 VLAN
	desired, _ := cs.Get(deviceID, vlanPath)
	vlans, ok := desired.(*huawei.HuaweiVlan_Vlan_Vlans)
	if !ok || vlans == nil {
		t.Fatal("desired 类型错误")
	}
	if len(vlans.Vlan) != n {
		t.Fatalf("并发合并丢更新：期望 %d 个 VLAN，实得 %d", n, len(vlans.Vlan))
	}

	// 单次对账后设备应有全部 VLAN
	reconcileVlan(cs, pool, ds, deviceID)
	for i := 1; i <= n; i++ {
		testsupport.AssertHuaweiVlanExists(t, sim, uint16(i))
	}
}

// 畸形输入：id 为非数字字符串——RFC7951 严格解码显式拒绝（BR-06，不静默跳过）。
func TestConvertVlan_MalformedRejected(t *testing.T) {
	_, err := convertConfig("/vlan:vlan/vlan:vlans", map[string]interface{}{
		"vlan": []interface{}{
			map[string]interface{}{"id": "not-a-number", "name": "bad"},
			map[string]interface{}{"id": float64(200), "name": "good"},
		},
	})
	if err == nil {
		t.Fatal("畸形 id 应被显式拒绝（RFC7951 严格解码），不得静默跳过")
	}
}

// reconcileVlan 直接触发一次 VLAN 对账（幂等测试用，desired 不变）。
func reconcileVlan(cs *manager.InMemoryConfigStore, pool client.ClientPool, ds device.Store, deviceID string) reconcile.Result {
	return vlanctl.New(cs, pool, ds).Reconcile(context.Background(), reconcile.Request{DeviceID: deviceID, Path: vlanPath})
}
