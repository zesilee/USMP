package cache

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// CC-08 / CC-02 例外：待同步条目不计 TTL，MarkExpiring 后自当下起算。

func TestPending_NeverExpiresUntilMarked(t *testing.T) {
	c := NewTTLLRUCache(100, 20*time.Millisecond, 0)
	defer c.Stop()

	c.SetPending("k", "v")
	time.Sleep(60 * time.Millisecond) // 远超 TTL

	val, ok := c.Get("k")
	assert.True(t, ok, "待同步条目不得因 TTL 过期")
	assert.Equal(t, "v", val)

	_, age, ok := c.GetWithAge("k")
	assert.True(t, ok)
	assert.GreaterOrEqual(t, age, 60*time.Millisecond)

	c.ClearExpired()
	_, ok = c.Get("k")
	assert.True(t, ok, "ClearExpired 不得清除待同步条目")
	assert.Contains(t, c.Keys(), "k", "Keys 不得剔除待同步条目")
}

func TestPending_TrackReportsPendingAndGen(t *testing.T) {
	c := NewTTLLRUCache(100, time.Second, 0)
	defer c.Stop()

	_, _, _, ok := c.Track("missing")
	assert.False(t, ok)

	before := time.Now()
	c.SetPending("k", "v1")
	gen, since, pending, ok := c.Track("k")
	assert.True(t, ok)
	assert.True(t, pending)
	assert.Equal(t, uint64(1), gen)
	assert.False(t, since.Before(before))

	c.SetPending("k", "v2")
	gen2, _, pending2, _ := c.Track("k")
	assert.Equal(t, uint64(2), gen2, "再次写入 gen 递增")
	assert.True(t, pending2)
}

func TestPending_MarkExpiringStartsTTLFromNow(t *testing.T) {
	// TTL 取 200ms：标记 → 「TTL 内命中」这段要给 -race 弱机 ×5 的调度余量
	c := NewTTLLRUCache(100, 200*time.Millisecond, 0)
	defer c.Stop()

	c.SetPending("k", "v")
	time.Sleep(400 * time.Millisecond) // 2×TTL，仍应命中
	gen, _, _, _ := c.Track("k")
	assert.True(t, c.MarkExpiring("k", gen))

	_, _, pending, ok := c.Track("k")
	assert.True(t, ok)
	assert.False(t, pending)

	_, ok = c.Get("k")
	assert.True(t, ok, "标记后 TTL 内应命中")
	time.Sleep(300 * time.Millisecond)
	_, ok = c.Get("k")
	assert.False(t, ok, "标记后超过 TTL 应未命中")
}

func TestPending_MarkExpiringGenGuard(t *testing.T) {
	c := NewTTLLRUCache(100, 20*time.Millisecond, 0)
	defer c.Stop()

	c.SetPending("k", "v1")
	oldGen, _, _, _ := c.Track("k")
	c.SetPending("k", "v2") // gen 变

	assert.False(t, c.MarkExpiring("k", oldGen), "旧 gen 不得标记")
	assert.False(t, c.MarkExpiring("missing", 1))

	time.Sleep(50 * time.Millisecond)
	val, ok := c.Get("k")
	assert.True(t, ok, "标记失败的条目仍待同步、不过期")
	assert.Equal(t, "v2", val)
	_, _, pending, _ := c.Track("k")
	assert.True(t, pending)
}

func TestPending_SetPendingAfterMarkRevertsToPending(t *testing.T) {
	c := NewTTLLRUCache(100, 20*time.Millisecond, 0)
	defer c.Stop()

	c.SetPending("k", "v1")
	gen, _, _, _ := c.Track("k")
	assert.True(t, c.MarkExpiring("k", gen))

	t0 := time.Now()
	c.SetPending("k", "v2")
	gen2, since, pending, _ := c.Track("k")
	assert.True(t, pending)
	assert.Equal(t, gen+1, gen2)
	assert.False(t, since.Before(t0), "pendingSince 为本次写入时刻")

	time.Sleep(50 * time.Millisecond)
	_, ok := c.Get("k")
	assert.True(t, ok, "退回待同步后不再过期")
}

func TestPending_PlainSetUnchanged(t *testing.T) {
	c := NewTTLLRUCache(100, 20*time.Millisecond, 0)
	defer c.Stop()

	c.Set("k", "v")
	gen, _, pending, ok := c.Track("k")
	assert.True(t, ok)
	assert.False(t, pending, "既有 Set 写入即计 TTL")
	assert.Equal(t, uint64(1), gen)

	time.Sleep(50 * time.Millisecond)
	_, ok = c.Get("k")
	assert.False(t, ok, "Set 语义不变：过期未命中")

	// Set 覆盖待同步条目也退出待同步（Set = 写入即计时）
	c.SetPending("p", "v")
	c.Set("p", "v2")
	_, _, pending, _ = c.Track("p")
	assert.False(t, pending)
}

func TestPending_DeleteIfGen(t *testing.T) {
	c := NewTTLLRUCache(100, time.Second, 0)
	defer c.Stop()

	c.SetPending("k", "v1")
	old, _, _, _ := c.Track("k")
	c.SetPending("k", "v2")
	assert.False(t, c.DeleteIfGen("k", old), "旧 gen 不得删新值")
	assert.False(t, c.DeleteIfGen("missing", 1))
	cur, _, _, _ := c.Track("k")
	assert.True(t, c.DeleteIfGen("k", cur))
	_, ok := c.Get("k")
	assert.False(t, ok)
}

func TestPending_Concurrent(t *testing.T) {
	c := NewTTLLRUCache(1000, 5*time.Millisecond, time.Millisecond)
	defer c.Stop()

	var wg sync.WaitGroup
	keys := []string{"a", "b", "c", "d"}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for n := 0; n < 200; n++ {
				k := keys[(i+n)%len(keys)]
				c.SetPending(k, n)
				if gen, _, _, ok := c.Track(k); ok {
					c.MarkExpiring(k, gen)
				}
				c.Get(k)
				c.GetWithAge(k)
				c.Keys()
				c.ClearExpired()
			}
		}(i)
	}
	wg.Wait()
}
