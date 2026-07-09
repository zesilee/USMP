package api

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	"github.com/leezesi/usmp/backend/simulator/netconfsim/testsupport"
	"github.com/stretchr/testify/assert"

	ifmctl "github.com/leezesi/usmp/backend/internal/controller/ifm"
	vlanctl "github.com/leezesi/usmp/backend/internal/controller/vlan"
)

// pushDeleteViaPool 走真实客户端（candidate→commit）下发删除——与
// ConfigHandler.pushDeleteToDevice 同链路（B3 已测 seam，此处测导线）。
func pushDeleteViaPool(t *testing.T, pool client.ClientPool, ds device.Store, deviceID string, target interface{}) error {
	t.Helper()
	info, ok := ds.Get(deviceID)
	assert.True(t, ok)
	cli, err := pool.Get(info)
	assert.NoError(t, err)
	result, err := cli.Set(context.Background(), []client.Change{{Type: client.DeleteChange, OldValue: target}}, client.WithCommit(true))
	// per-change 错误优先于聚合错误（与 pushDeleteToDevice 同规，保留 data-missing 细节）。
	if result != nil && !result.Success {
		for _, cr := range result.Changes {
			if cr.Error != nil {
				return cr.Error
			}
		}
	}
	return err
}

// BR-09 端到端（vlan）：建→删→回读消失→二轮对账 0 change（不复活、不误删同表其它键）。
func TestDeleteConfig_Integration_VlanEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim, cs, pool, ds, deviceID := newVlanSimStack(t)
	defer sim.Stop()
	defer pool.CloseAll()

	applyVlan(t, cs, pool, ds, deviceID, []interface{}{
		map[string]interface{}{"id": float64(30), "name": "to-delete"},
		map[string]interface{}{"id": float64(40), "name": "to-keep"},
	})
	testsupport.AssertHuaweiVlanExists(t, sim, 30)
	testsupport.AssertHuaweiVlanExists(t, sim, 40)

	// 与 DeleteConfig handler 同序：先移 desired，再下发删除。
	target, err := parseDeleteTarget(vlanPath, "30")
	assert.NoError(t, err)
	assert.NoError(t, storeConfigDeleted(cs, deviceID, vlanPath, target))
	assert.NoError(t, pushDeleteViaPool(t, pool, ds, deviceID, target))

	// 设备侧：30 消失、40 存活
	vlans := sim.RunningHuaweiVLANs()
	assert.NotContains(t, vlans, uint16(30))
	assert.Contains(t, vlans, uint16(40))

	// 二轮对账：0 change（desired 与 actual 均无 30 → 不复活、不漂移）
	res := vlanctl.New(cs, pool, ds).Reconcile(context.Background(), reconcile.Request{DeviceID: deviceID, Path: vlanPath})
	assert.Nil(t, res.Error)
	assert.Equal(t, 0, res.Changes, "post-delete reconcile must be converged")
	assert.NotContains(t, sim.RunningHuaweiVLANs(), uint16(30), "deleted vlan must not resurrect")
}

// BR-09 负路径：删设备上不存在的条目 → data-missing 如实透出（RFC 6241 §7.2）。
func TestDeleteConfig_Integration_DataMissing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim, cs, pool, ds, deviceID := newVlanSimStack(t)
	defer sim.Stop()
	defer pool.CloseAll()

	applyVlan(t, cs, pool, ds, deviceID, []interface{}{
		map[string]interface{}{"id": float64(50), "name": "only-one"},
	})

	target, err := parseDeleteTarget(vlanPath, "999")
	assert.NoError(t, err)
	err = pushDeleteViaPool(t, pool, ds, deviceID, target)
	if assert.Error(t, err, "deleting a nonexistent entry must surface a device error") {
		assert.True(t, strings.Contains(err.Error(), "data-missing") || strings.Contains(err.Error(), "not found"),
			"err = %v, want data-missing", err)
	}
	// 既有条目不受影响
	testsupport.AssertHuaweiVlanExists(t, sim, 50)
}

// BR-09 端到端（ifm）：接口建→删→回读消失。
func TestDeleteConfig_Integration_IfmEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim, cs, pool, ds, deviceID := newVlanSimStack(t)
	defer sim.Stop()
	defer pool.CloseAll()

	raw := map[string]interface{}{
		"interface": []interface{}{
			map[string]interface{}{"name": "GigabitEthernet0/0/9", "description": "temp"},
		},
	}
	typed, err := convertMapToHuaweiIfm(raw)
	assert.NoError(t, err)
	ifmPath := "/ifm:ifm/ifm:interfaces"
	assert.NoError(t, storeConfigMerged(cs, deviceID, ifmPath, typed))
	res := ifmctl.New(cs, pool, ds).Reconcile(context.Background(), reconcile.Request{DeviceID: deviceID, Path: ifmPath})
	assert.Nil(t, res.Error)
	testsupport.AssertHuaweiInterfaceExists(t, sim, "GigabitEthernet0/0/9")

	target, err := parseDeleteTarget(ifmPath, "GigabitEthernet0/0/9")
	assert.NoError(t, err)
	assert.NoError(t, storeConfigDeleted(cs, deviceID, ifmPath, target))
	assert.NoError(t, pushDeleteViaPool(t, pool, ds, deviceID, target))

	assert.NotContains(t, sim.RunningHuaweiInterfaces(), "GigabitEthernet0/0/9")

	// 二轮对账收敛
	res = ifmctl.New(cs, pool, ds).Reconcile(context.Background(), reconcile.Request{DeviceID: deviceID, Path: ifmPath})
	assert.Nil(t, res.Error)
	assert.Equal(t, 0, res.Changes)
}

// R09：并发删除不同键与合并下发交错——desired 串行化、设备终态正确（-race）。
func TestDeleteConfig_Integration_ConcurrentDeleteAndMerge(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim, cs, pool, ds, deviceID := newVlanSimStack(t)
	defer sim.Stop()
	defer pool.CloseAll()

	// 预置 4 条待删 + 保留位
	applyVlan(t, cs, pool, ds, deviceID, []interface{}{
		map[string]interface{}{"id": float64(101), "name": "d1"},
		map[string]interface{}{"id": float64(102), "name": "d2"},
		map[string]interface{}{"id": float64(103), "name": "d3"},
		map[string]interface{}{"id": float64(104), "name": "d4"},
	})

	var wg sync.WaitGroup
	for _, id := range []string{"101", "102", "103", "104"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			target, err := parseDeleteTarget(vlanPath, key)
			if err != nil {
				t.Error(err)
				return
			}
			if err := storeConfigDeleted(cs, deviceID, vlanPath, target); err != nil {
				t.Error(err)
				return
			}
			if err := pushDeleteViaPool(t, pool, ds, deviceID, target); err != nil {
				t.Error(err)
			}
		}(id)
	}
	// 同时并发合并一条新配置
	wg.Add(1)
	go func() {
		defer wg.Done()
		typed, err := convertMapToHuaweiVlan(map[string]interface{}{"vlans": []interface{}{
			map[string]interface{}{"id": float64(200), "name": "merged"},
		}})
		if err != nil {
			t.Error(err)
			return
		}
		_ = storeConfigMerged(cs, deviceID, vlanPath, typed)
	}()
	wg.Wait()

	// 终态：4 条删除键都不在设备上
	vlans := sim.RunningHuaweiVLANs()
	for _, id := range []uint16{101, 102, 103, 104} {
		assert.NotContains(t, vlans, id, "vlan %d must be deleted", id)
	}
	// 合并的 200 经一次对账落到设备且不复活已删条目
	res := vlanctl.New(cs, pool, ds).Reconcile(context.Background(), reconcile.Request{DeviceID: deviceID, Path: vlanPath})
	assert.Nil(t, res.Error)
	testsupport.AssertHuaweiVlanExists(t, sim, 200)
	for _, id := range []uint16{101, 102, 103, 104} {
		assert.NotContains(t, sim.RunningHuaweiVLANs(), id)
	}
}
