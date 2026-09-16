package manager

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/internal/cache"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	"github.com/stretchr/testify/assert"
)

// 回归用例（T07）：复现「设备慢 → 重试跨过 desired TTL → 读空假收敛」。
// 用真 InMemoryConfigStore + 20ms TTL：首轮下发失败，睡过 TTL 后重试必须仍读到
// desired 并送达；此前的实现在这里读到 nil 并按 Converged 返回。

type flakyDevice struct {
	failSets int32 // 前 N 次 Set 失败
	sets     int32
	actual   interface{}
}

func (d *flakyDevice) Get(context.Context, string) (interface{}, error) { return d.actual, nil }
func (d *flakyDevice) Set(_ context.Context, _ string, _ []reconcile.Change) error {
	n := atomic.AddInt32(&d.sets, 1)
	if n <= atomic.LoadInt32(&d.failSets) {
		return errors.New("device busy")
	}
	d.actual = "desired"
	return nil
}

type strDiff struct{}

func (strDiff) Diff(desired, actual interface{}, path string) ([]reconcile.Change, error) {
	if desired == actual {
		return nil, nil
	}
	return []reconcile.Change{{Path: path, DesiredValue: desired}}, nil
}

func TestRegression_RetryAcrossTTLStillDelivers(t *testing.T) {
	c := cache.NewTTLLRUCache(100, 20*time.Millisecond, 0)
	defer c.Stop()
	cs := NewInMemoryConfigStore(c)
	dev := &flakyDevice{failSets: 1, actual: "stale"}
	r := reconcile.NewGenericReconciler(cs, dev, strDiff{})
	req := reconcile.Request{DeviceID: "10.0.0.1", Path: "/vlan"}

	assert.NoError(t, cs.Set(req.DeviceID, req.Path, "desired"))

	first := r.Reconcile(context.Background(), req)
	assert.True(t, first.Requeue, "首轮设备失败应重投")
	assert.False(t, first.NoDesired)

	time.Sleep(60 * time.Millisecond) // 远超 20ms TTL——旧实现在此丢 desired

	second := r.Reconcile(context.Background(), req)
	assert.False(t, second.NoDesired, "重试跨过 TTL 不得读空（假收敛根因）")
	assert.Nil(t, second.Error)
	assert.Equal(t, 1, second.Changes, "重试应真正下发")
	assert.Equal(t, "desired", dev.actual)

	// YR-05 复验：零变更 → 标记已同步 → 自此按 TTL 释放（口径 A）
	third := r.Reconcile(context.Background(), req)
	assert.Equal(t, 0, third.Changes)
	assert.False(t, third.NoDesired)
	_, _, pending, ok := cs.Track(req.DeviceID, req.Path)
	assert.True(t, ok)
	assert.False(t, pending, "复验收敛后转已同步")

	time.Sleep(60 * time.Millisecond)
	v, _ := cs.Get(req.DeviceID, req.Path)
	assert.Nil(t, v, "已同步后按 TTL 释放")
	fourth := r.Reconcile(context.Background(), req)
	assert.True(t, fourth.NoDesired, "释放后读空以 NoDesired 上报，而非 Converged")
}

func TestRegression_AbandonAfterLimitWithRealStore(t *testing.T) {
	// 200ms：Set → 首轮失败必须落在上限内，给 -race 弱机 ×5 余量
	reconcile.SetAbandonAfter(200 * time.Millisecond)
	defer reconcile.SetAbandonAfter(0)

	c := cache.NewTTLLRUCache(100, time.Second, 0)
	defer c.Stop()
	cs := NewInMemoryConfigStore(c)
	dev := &flakyDevice{failSets: 1 << 30, actual: "stale"} // 永远失败
	r := reconcile.NewGenericReconciler(cs, dev, strDiff{})
	req := reconcile.Request{DeviceID: "10.0.0.1", Path: "/vlan"}
	assert.NoError(t, cs.Set(req.DeviceID, req.Path, "desired"))

	res := r.Reconcile(context.Background(), req)
	assert.True(t, res.Requeue)
	assert.False(t, res.Terminal, "未超限照常重投")

	time.Sleep(250 * time.Millisecond)
	res = r.Reconcile(context.Background(), req)
	assert.True(t, res.Terminal, "超限放弃")
	assert.True(t, errors.Is(res.Error, reconcile.ErrDesiredAbandoned))
	v, _ := cs.Get(req.DeviceID, req.Path)
	assert.Nil(t, v, "放弃后 desired 删除")
	assert.Equal(t, "stale", dev.actual, "放弃不触碰设备")
}
