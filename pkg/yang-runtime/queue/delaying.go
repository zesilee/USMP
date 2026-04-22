package queue

import (
	"container/heap"
	"sync"
	"sync/atomic"
	"time"
)

// delayingQueue implements a queue that supports delayed item processing
type delayingQueue struct {
	base         Interface
	delayedItems delayedHeap
	next         chan interface{}
	stop         chan struct{}
	wg           sync.WaitGroup
	mu           sync.Mutex
	shutdown     bool
}

// NewDelayingQueue creates a new delaying queue
func NewDelayingQueue(base Interface) RateLimitingInterface {
	dq := &delayingQueue{
		base:     base,
		delayedItems: make(delayedHeap, 0),
		next:     make(chan interface{}),
		stop:     make(chan struct{}),
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
	return dq.base.Len() + dq.delayedItems.Len()
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
	when := time.Now().Add(delay)
	dq.mu.Lock()
	defer dq.mu.Unlock()
	heap.Push(&dq.delayedItems, &delayedEntry{
		item:  item,
		ready: when,
	})
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

	for {
		select {
		case <-dq.stop:
			return
		default:
		}

		dq.mu.Lock()
		var nextItem *delayedEntry
		if dq.delayedItems.Len() > 0 {
			nextItem = dq.delayedItems[0]
			if time.Now().After(nextItem.ready) {
				heap.Pop(&dq.delayedItems)
			} else {
				nextItem = nil
			}
		}
		dq.mu.Unlock()

		if nextItem != nil {
			dq.base.Add(nextItem.item)
		}

		// Wait until next item is ready or we're stopped
		dq.mu.Lock()
		var waitTime time.Duration
		if dq.delayedItems.Len() > 0 {
			nextReady := dq.delayedItems[0].ready
			waitTime = nextReady.Sub(time.Now())
			if waitTime < 0 {
				waitTime = 0
			}
		} else {
			waitTime = 100 * time.Millisecond
		}
		dq.mu.Unlock()

		select {
		case <-dq.stop:
			return
		case <-time.After(waitTime):
			// Continue
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
	queue     chan interface{}
	shutdown  atomic.Bool
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
