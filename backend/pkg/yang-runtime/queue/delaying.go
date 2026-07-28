package queue

import (
	"container/heap"
	"sync"
	"sync/atomic"
	"time"

	"k8s.io/utils/clock"
)

// delayingQueue implements a queue that supports delayed item processing
type delayingQueue struct {
	base         Interface
	delayedItems delayedHeap
	stop         chan struct{}
	// newItem wakes process() whenever AddAfter enqueues an item, so a shorter
	// delay added while the loop is parked on a longer wait is honored promptly
	// (fixes the race/latency where a fresh 0-delay item waited on a stale timer).
	newItem chan struct{}
	// clock is injectable so tests can drive time deterministically (RealClock
	// in production). Set once at construction, never mutated → lock-free reads.
	clock    clock.Clock
	wg       sync.WaitGroup
	mu       sync.Mutex
	shutdown bool
}

// NewDelayingQueue creates a new delaying queue backed by the real clock.
func NewDelayingQueue(base Interface) RateLimitingInterface {
	return newDelayingQueueWithClock(base, clock.RealClock{})
}

// newDelayingQueueWithClock creates a delaying queue with an injectable clock.
// Unexported: only in-package tests need a fake clock; production always uses
// NewDelayingQueue (real clock).
func newDelayingQueueWithClock(base Interface, clk clock.Clock) RateLimitingInterface {
	dq := &delayingQueue{
		base:         base,
		delayedItems: make(delayedHeap, 0),
		stop:         make(chan struct{}),
		newItem:      make(chan struct{}, 1),
		clock:        clk,
	}
	heap.Init(&dq.delayedItems)
	dq.wg.Add(1)
	go dq.process()
	return dq
}

// Add implements Interface
func (dq *delayingQueue) Add(item interface{}) {
	dq.base.Add(item)
}

// Len implements Interface
func (dq *delayingQueue) Len() int {
	// delayedItems is mutated by process() under dq.mu; read it under the same
	// lock to stay race-free. base has its own synchronization.
	dq.mu.Lock()
	delayed := dq.delayedItems.Len()
	dq.mu.Unlock()
	return dq.base.Len() + delayed
}

// Get implements Interface
func (dq *delayingQueue) Get() (item interface{}, shutdown bool) {
	return dq.base.Get()
}

// Done implements Interface
func (dq *delayingQueue) Done(item interface{}) {
	dq.base.Done(item)
}

// ShutDown implements Interface
func (dq *delayingQueue) ShutDown() {
	dq.mu.Lock()
	dq.shutdown = true
	dq.mu.Unlock()
	close(dq.stop)
	dq.base.ShutDown()
	dq.wg.Wait()
}

// ShutDownWithDrain implements Interface
func (dq *delayingQueue) ShutDownWithDrain() {
	dq.mu.Lock()
	dq.shutdown = true
	dq.mu.Unlock()
	close(dq.stop)
	// Wait for all delayed items to be processed and drained
	dq.base.ShutDownWithDrain()
	dq.wg.Wait()
}

// ShuttingDown implements Interface
func (dq *delayingQueue) ShuttingDown() bool {
	dq.mu.Lock()
	defer dq.mu.Unlock()
	return dq.shutdown || dq.base.ShuttingDown()
}

// AddAfter implements RateLimitingInterface
func (dq *delayingQueue) AddAfter(item interface{}, delay time.Duration) {
	when := dq.clock.Now().Add(delay)
	dq.mu.Lock()
	heap.Push(&dq.delayedItems, &delayedEntry{
		item:  item,
		ready: when,
	})
	dq.mu.Unlock()

	// Wake process() to re-evaluate: the new item may be ready now or sooner
	// than whatever wait it is currently parked on. Non-blocking — one pending
	// signal is enough to force a full re-evaluation of the heap.
	select {
	case dq.newItem <- struct{}{}:
	default:
	}
}

// AddRateLimited implements RateLimitingInterface
func (dq *delayingQueue) AddRateLimited(item interface{}) {
	// This is handled by the rate limiting queue wrapper
	dq.Add(item)
}

// Forget implements RateLimitingInterface
func (dq *delayingQueue) Forget(item interface{}) {
	// No-op on delaying queue, handled by rate limiter
}

// NumRequeues implements RateLimitingInterface
func (dq *delayingQueue) NumRequeues(item interface{}) int {
	// Handled by rate limiter
	return 0
}

func (dq *delayingQueue) process() {
	defer dq.wg.Done()

	// Fallback wait when the heap is empty. The newItem signal wakes the loop the
	// instant anything is added, so this only bounds idle time if a signal were
	// ever missed — kept long (not a busy poll) since correctness rides on newItem.
	const idleWait = time.Hour

	for {
		select {
		case <-dq.stop:
			return
		default:
		}

		now := dq.clock.Now()

		// Deliver every item whose ready time has arrived (drains all currently
		// ready items, not just one per tick).
		dq.mu.Lock()
		for dq.delayedItems.Len() > 0 && !dq.delayedItems[0].ready.After(now) {
			entry := heap.Pop(&dq.delayedItems).(*delayedEntry)
			dq.mu.Unlock()
			dq.base.Add(entry.item) // may block (unbuffered base) — back-pressure, by design
			dq.mu.Lock()
		}
		wait := idleWait
		if dq.delayedItems.Len() > 0 {
			wait = dq.delayedItems[0].ready.Sub(now)
		}
		dq.mu.Unlock()

		if wait <= 0 {
			continue // an item became ready while delivering; re-evaluate now
		}

		timer := dq.clock.NewTimer(wait)
		select {
		case <-dq.stop:
			timer.Stop()
			return
		case <-dq.newItem:
			timer.Stop() // a new (possibly sooner) item arrived — re-evaluate
		case <-timer.C():
			// next item's ready time reached
		}
	}
}

// delayedEntry represents an entry in the delayed heap
type delayedEntry struct {
	item  interface{}
	ready time.Time
	index int
}

// delayedHeap is a min-heap based on ready time
type delayedHeap []*delayedEntry

func (h delayedHeap) Len() int { return len(h) }

func (h delayedHeap) Less(i, j int) bool {
	return h[i].ready.Before(h[j].ready)
}

func (h delayedHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *delayedHeap) Push(x interface{}) {
	entry := x.(*delayedEntry)
	entry.index = len(*h)
	*h = append(*h, entry)
}

func (h *delayedHeap) Pop() interface{} {
	old := *h
	n := len(old)
	entry := old[n-1]
	old[n-1] = nil
	entry.index = -1
	*h = old[0 : n-1]
	return entry
}

// rateLimitingQueue wraps a delaying queue with rate limiting
type rateLimitingQueue struct {
	RateLimitingInterface
	rateLimiter RateLimiter
}

// NewRateLimitingQueue creates a new rate limiting queue
func NewRateLimitingQueue(rateLimiter RateLimiter) RateLimitingInterface {
	base := NewDelayingQueue(&standardQueue{
		queue: make(chan interface{}),
	})
	return &rateLimitingQueue{
		RateLimitingInterface: base.(RateLimitingInterface),
		rateLimiter:           rateLimiter,
	}
}

// AddRateLimited implements RateLimitingInterface
func (rlq *rateLimitingQueue) AddRateLimited(item interface{}) {
	delay := rlq.rateLimiter.When(item)
	rlq.AddAfter(item, delay)
}

// Forget implements RateLimitingInterface
func (rlq *rateLimitingQueue) Forget(item interface{}) {
	rlq.rateLimiter.Forget(item)
}

// NumRequeues implements RateLimitingInterface
func (rlq *rateLimitingQueue) NumRequeues(item interface{}) int {
	return rlq.rateLimiter.NumRequeues(item)
}

// standardQueue is the base blocking queue implementation
type standardQueue struct {
	queue    chan interface{}
	shutdown atomic.Bool
}

// Add implements Interface
func (q *standardQueue) Add(item interface{}) {
	if !q.shutdown.Load() {
		q.queue <- item
	}
}

// Len implements Interface
func (q *standardQueue) Len() int {
	return len(q.queue)
}

// Get implements Interface
func (q *standardQueue) Get() (item interface{}, shutdown bool) {
	item, ok := <-q.queue
	return item, q.shutdown.Load() && ok == false
}

// Done implements Interface
func (q *standardQueue) Done(item interface{}) {
	// No-op for standard queue, all items are processed when popped
}

// ShutDown implements Interface
func (q *standardQueue) ShutDown() {
	q.shutdown.Store(true)
	close(q.queue)
}

// ShutDownWithDrain implements Interface
func (q *standardQueue) ShutDownWithDrain() {
	q.ShutDown()
}

// ShuttingDown implements Interface
func (q *standardQueue) ShuttingDown() bool {
	return q.shutdown.Load()
}

// NewStandardQueue creates a new standard queue
func NewStandardQueue() Interface {
	return &standardQueue{
		queue: make(chan interface{}, 10000),
	}
}
