package queue

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewStandardQueue(t *testing.T) {
	q := NewStandardQueue()
	assert.NotNil(t, q)
	assert.Equal(t, 0, q.Len())
	assert.False(t, q.ShuttingDown())
}

func TestStandardQueueAddGet(t *testing.T) {
	q := NewStandardQueue()
	q.Add("test-item")
	assert.Equal(t, 1, q.Len())

	item, shutdown := q.Get()
	assert.False(t, shutdown)
	assert.Equal(t, "test-item", item)
	q.Done(item)
}

func TestStandardQueueShutdown(t *testing.T) {
	q := NewStandardQueue()
	q.Add("test-item")
	q.ShutDown()
	assert.True(t, q.ShuttingDown())

	// First get should return the existing item added before shutdown
	item, shutdown := q.Get()
	assert.False(t, shutdown)
	assert.Equal(t, "test-item", item)

	// Next get after shutdown should return shutdown=true
	item, shutdown = q.Get()
	assert.True(t, shutdown)
	assert.Nil(t, item)
}

func TestDelayingQueueAddAfter(t *testing.T) {
	q := NewDelayingQueue(NewStandardQueue())
	defer q.ShutDown()

	q.AddAfter("fast", 0*time.Millisecond)
	q.AddAfter("slow", 100*time.Millisecond)

	// Fast should be available immediately
	start := time.Now()
	item, shutdown := q.Get()
	assert.False(t, shutdown)
	assert.Equal(t, "fast", item)
	q.Done(item)
	assert.Less(t, time.Since(start), 50*time.Millisecond)

	// Slow should take ~100ms
	start = time.Now()
	item, shutdown = q.Get()
	assert.False(t, shutdown)
	assert.Equal(t, "slow", item)
	q.Done(item)
	duration := time.Since(start)
	assert.GreaterOrEqual(t, duration.Milliseconds(), int64(50))
}

func TestRateLimitingExponential(t *testing.T) {
	rl := NewItemExponentialFailureRateLimiter(1*time.Second, 30*time.Second)

	// First failure: 1s delay (20% jitter allowed)
	delay := rl.When("item")
	assert.InDelta(t, 1000, delay.Milliseconds(), 200)

	// Second failure: 2s (20% jitter allowed)
	delay = rl.When("item")
	assert.InDelta(t, 2000, delay.Milliseconds(), 400)

	// Third failure: 4s (20% jitter allowed)
	delay = rl.When("item")
	assert.InDelta(t, 4000, delay.Milliseconds(), 800)

	// After forget, back to 1s
	rl.Forget("item")
	// Before doing another When, num requeues should be 0 (Forget deleted it)
	assert.Equal(t, 0, rl.NumRequeues("item"))
	delay = rl.When("item")
	assert.InDelta(t, 1000, delay.Milliseconds(), 200)

	// Check num requeues after the additional When - should be 1
	assert.Equal(t, 1, rl.NumRequeues("item"))
}

func TestRateLimitingMaxDelay(t *testing.T) {
	rl := NewItemExponentialFailureRateLimiter(1*time.Second, 10*time.Second)

	// Keep increasing - should cap at 10s (with 20% jitter, max observed can be 12s)
	for i := 0; i < 10; i++ {
		rl.When("item")
	}

	delay := rl.When("item")
	assert.LessOrEqual(t, delay, 12*time.Second)
}

func TestMaxOfRateLimiter(t *testing.T) {
	rl1 := NewItemExponentialFailureRateLimiter(1*time.Second, 10*time.Second)
	rl2 := NewBucketRateLimiter(10, 10)
	combo := NewMaxOfRateLimiter(rl1, rl2)

	// First item should get ~1s from exponential (20% jitter)
	delay := combo.When("item")
	assert.InDelta(t, 1000, delay.Milliseconds(), 200)

	// After forgetting, it should be forgotten in both
	combo.Forget("item")
	assert.Equal(t, 0, combo.NumRequeues("item"))
}

func TestBucketRateLimiter(t *testing.T) {
	// 10 QPS = 100ms per request, capacity 2
	rl := NewBucketRateLimiter(10, 2)

	// First two should have no delay
	assert.Equal(t, time.Duration(0), rl.When("item1"))
	assert.Equal(t, time.Duration(0), rl.When("item2"))

	// Third should require some waiting
	delay := rl.When("item3")
	assert.Greater(t, delay, time.Duration(0))
}

func TestRateLimitingQueue(t *testing.T) {
	rl := NewItemExponentialFailureRateLimiter(10*time.Millisecond, 100*time.Millisecond)
	q := NewRateLimitingQueue(rl)
	defer q.ShutDown()

	q.AddRateLimited("test-item")
	assert.Equal(t, 1, q.Len())

	item, shutdown := q.Get()
	assert.False(t, shutdown)
	assert.Equal(t, "test-item", item)
	assert.Equal(t, 1, q.NumRequeues("test-item"))

	q.Forget("test-item")
	assert.Equal(t, 0, q.NumRequeues("test-item"))
	q.Done(item)
}

func TestQueueLen(t *testing.T) {
	base := NewStandardQueue()
	q := NewDelayingQueue(base)
	defer q.ShutDown()

	q.Add("item1")
	q.AddAfter("item2", 1*time.Second)
	assert.Equal(t, 2, q.Len())
}

func TestConcurrentAccess(t *testing.T) {
	t.Parallel()
	q := NewStandardQueue()
	defer q.ShutDown()

	numWorkers := 5
	numItems := 20
	totalItems := numWorkers * numItems

	// Start workers to add items
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			for j := 0; j < numItems; j++ {
				q.Add(workerID*numItems + j)
			}
		}(i)
	}

	// Drain items in the main goroutine
	seen := make(map[int]bool)
	for len(seen) < totalItems {
		item, shutdown := q.Get()
		if shutdown {
			break
		}
		seen[item.(int)] = true
		q.Done(item)
	}

	assert.Equal(t, totalItems, len(seen))
}
