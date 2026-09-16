package manager

import (
	"sync"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/internal/cache"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	"github.com/stretchr/testify/assert"
)

// D2：InMemoryConfigStore 经可选接口 SyncTracker 暴露两态生命周期（YR-09）。
var _ reconcile.SyncTracker = (*InMemoryConfigStore)(nil)

func newSyncStore(ttl time.Duration) (*InMemoryConfigStore, *cache.TTLLRUCache) {
	c := cache.NewTTLLRUCache(100, ttl, 0)
	return NewInMemoryConfigStore(c), c
}

func TestConfigStore_SetIsPendingAndOutlivesTTL(t *testing.T) {
	cs, c := newSyncStore(20 * time.Millisecond)
	defer c.Stop()

	assert.NoError(t, cs.Set("d", "/p", "v"))
	gen, since, pending, ok := cs.Track("d", "/p")
	assert.True(t, ok)
	assert.True(t, pending, "Set 写入即待同步")
	assert.Equal(t, uint64(1), gen)
	assert.False(t, since.IsZero())

	time.Sleep(60 * time.Millisecond)
	v, err := cs.Get("d", "/p")
	assert.NoError(t, err)
	assert.Equal(t, "v", v, "待同步 desired 不因 TTL 丢失")

	paths, _ := cs.List("d")
	assert.Equal(t, []string{"/p"}, paths)
	devs, _ := cs.ListDevices()
	assert.Equal(t, []string{"d"}, devs)
}

func TestConfigStore_MarkSyncedStartsTTL(t *testing.T) {
	cs, c := newSyncStore(30 * time.Millisecond)
	defer c.Stop()

	_ = cs.Set("d", "/p", "v")
	gen, _, _, _ := cs.Track("d", "/p")
	assert.True(t, cs.MarkSynced("d", "/p", gen))
	_, _, pending, _ := cs.Track("d", "/p")
	assert.False(t, pending)

	time.Sleep(70 * time.Millisecond)
	v, _ := cs.Get("d", "/p")
	assert.Nil(t, v, "已同步后按 TTL 释放（口径 A）")
	_, _, _, ok := cs.Track("d", "/p")
	assert.False(t, ok)
}

func TestConfigStore_MarkSyncedGenMismatch(t *testing.T) {
	cs, c := newSyncStore(20 * time.Millisecond)
	defer c.Stop()

	_ = cs.Set("d", "/p", "v1")
	old, _, _, _ := cs.Track("d", "/p")
	_ = cs.Set("d", "/p", "v2")
	assert.False(t, cs.MarkSynced("d", "/p", old))
	assert.False(t, cs.MarkSynced("d", "/missing", 1))

	time.Sleep(50 * time.Millisecond)
	v, _ := cs.Get("d", "/p")
	assert.Equal(t, "v2", v, "新值仍待同步")
}

func TestConfigStore_AbandonGenGuard(t *testing.T) {
	cs, c := newSyncStore(time.Second)
	defer c.Stop()

	_ = cs.Set("d", "/p", "v1")
	old, _, _, _ := cs.Track("d", "/p")
	_ = cs.Set("d", "/p", "v2")
	assert.False(t, cs.Abandon("d", "/p", old), "写代不匹配不得放弃新值")
	v, _ := cs.Get("d", "/p")
	assert.Equal(t, "v2", v)

	cur, _, _, _ := cs.Track("d", "/p")
	assert.True(t, cs.Abandon("d", "/p", cur))
	v, _ = cs.Get("d", "/p")
	assert.Nil(t, v)
	assert.False(t, cs.Abandon("d", "/p", cur), "已删幂等返回 false")
}

func TestConfigStore_SyncConcurrent(t *testing.T) {
	cs, c := newSyncStore(5 * time.Millisecond)
	defer c.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for n := 0; n < 200; n++ {
				_ = cs.Set("d", "/p", n)
				if gen, _, pending, ok := cs.Track("d", "/p"); ok && pending {
					if i%2 == 0 {
						cs.MarkSynced("d", "/p", gen)
					} else {
						cs.Abandon("d", "/p", gen)
					}
				}
				_, _ = cs.Get("d", "/p")
				_, _ = cs.List("d")
			}
		}(i)
	}
	wg.Wait()
}
