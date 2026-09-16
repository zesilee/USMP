package vlan

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/internal/cache"
	"github.com/leezesi/usmp/backend/internal/generated/native/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// B2（YR-09）：设备短暂不可达、重试跨过 desired 存储 TTL 后恢复——意图必须仍在并
// 真正送达模拟网元；复验收敛后转已同步，按 TTL 释放，释放后读空以 NoDesired 上报。
func TestReconciler_Integration_PendingSurvivesDeviceOutage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	sim := netsim.NewSimulator()
	require.NoError(t, sim.Start())
	ds, deviceID := simStore(sim)
	addr, port := sim.Addr(), sim.Port()

	c := cache.NewTTLLRUCache(100, 40*time.Millisecond, 0) // 远短于任何重试间隔
	defer c.Stop()
	cs := manager.NewInMemoryConfigStore(c)
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(2 * time.Second))
	defer pool.CloseAll()

	path := "/vlan:vlan/vlan:vlans"
	desired := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
			410: {Id: uint16Ptr(410), Name: stringPtr("pending-410")},
		},
	}
	require.NoError(t, cs.Set(deviceID, path, desired))

	// 设备「慢/不可达」：写入意图后网元下线
	sim.Stop()

	r := New(cs, pool, ds)
	req := reconcile.Request{DeviceID: deviceID, Path: path}
	ctx := context.Background()

	first := r.Reconcile(ctx, req)
	require.NotNil(t, first.Error, "网元下线首轮应失败")
	assert.True(t, first.Requeue)
	assert.False(t, first.Terminal)

	time.Sleep(120 * time.Millisecond) // 3×TTL：旧实现在此已丢 desired

	// 网元恢复（同地址同端口）
	sim2 := netsim.NewSimulator()
	sim2.SetListen(addr, port)
	require.NoError(t, sim2.Start())
	defer sim2.Stop()

	second := r.Reconcile(ctx, req)
	require.False(t, second.NoDesired, "重试跨过 TTL 不得读空（假收敛根因）")
	require.Nil(t, second.Error, "网元恢复后应下发成功")
	assert.Greater(t, second.Changes, 0, "应真正下发 VLAN 410")

	// 复验收敛 → 已同步
	third := r.Reconcile(ctx, req)
	require.Nil(t, third.Error)
	assert.Equal(t, 0, third.Changes)
	_, _, pending, ok := cs.Track(deviceID, path)
	assert.True(t, ok)
	assert.False(t, pending, "复验收敛后转已同步")

	// 设备侧真的有了
	dc := &deviceClient{clientPool: pool, resolver: ds}
	got, err := dc.Get(ctx, deviceID)
	require.NoError(t, err)
	vlans, ok := got.(*huawei.HuaweiVlan_Vlan_Vlans)
	require.True(t, ok)
	_, present := vlans.Vlan[410]
	assert.True(t, present, "VLAN 410 应已在模拟网元上")

	// 口径 A：已同步后按 TTL 释放，释放后读空不冒充收敛
	time.Sleep(120 * time.Millisecond)
	v, _ := cs.Get(deviceID, path)
	assert.Nil(t, v, "已同步 desired 按 TTL 释放")
	fourth := r.Reconcile(ctx, req)
	assert.True(t, fourth.NoDesired)
	assert.Nil(t, fourth.Error)
}

// B2（YR-09）：网元持续不可达超过放弃上限 → 放弃：desired 删除、终态错误含「已放弃」。
func TestReconciler_Integration_AbandonAfterLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	reconcile.SetAbandonAfter(150 * time.Millisecond)
	defer reconcile.SetAbandonAfter(0)

	sim := netsim.NewSimulator()
	require.NoError(t, sim.Start())
	ds, deviceID := simStore(sim)
	sim.Stop() // 拿到地址后即下线，且不再恢复

	c := cache.NewTTLLRUCache(100, time.Minute, 0)
	defer c.Stop()
	cs := manager.NewInMemoryConfigStore(c)
	pool := client.NewDefaultClientPool(client.DefaultClientFactory(2 * time.Second))
	defer pool.CloseAll()

	path := "/vlan:vlan/vlan:vlans"
	require.NoError(t, cs.Set(deviceID, path, &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
			420: {Id: uint16Ptr(420), Name: stringPtr("abandon-420")},
		},
	}))

	r := New(cs, pool, ds)
	req := reconcile.Request{DeviceID: deviceID, Path: path}
	ctx := context.Background()

	first := r.Reconcile(ctx, req)
	require.NotNil(t, first.Error)
	assert.False(t, first.Terminal, "未超限照常重投")
	v, _ := cs.Get(deviceID, path)
	assert.NotNil(t, v, "未超限 desired 保留")

	time.Sleep(200 * time.Millisecond)

	second := r.Reconcile(ctx, req)
	require.NotNil(t, second.Error)
	assert.True(t, second.Terminal, "超限应放弃为终态")
	assert.False(t, second.Requeue)
	assert.True(t, errors.Is(second.Error, reconcile.ErrDesiredAbandoned))
	assert.Contains(t, second.Error.Error(), "已放弃")
	v, _ = cs.Get(deviceID, path)
	assert.Nil(t, v, "放弃后 desired 删除")
}
