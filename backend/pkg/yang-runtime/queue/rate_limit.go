package queue

import (
	"math"
	"math/rand"
	"sync"
	"time"
)

// ItemExponentialFailureRateLimiter implements exponential backoff rate limiting
// per failed item
type ItemExponentialFailureRateLimiter struct {
	baseDelay    time.Duration
	maxDelay     time.Duration
	failures     map[interface{}]int
	mu           sync.Mutex
	rng          *rand.Rand
	jitterFactor float64
}

// NewItemExponentialFailureRateLimiter creates a new ItemExponentialFailureRateLimiter
func NewItemExponentialFailureRateLimiter(baseDelay, maxDelay time.Duration) RateLimiter {
	return &ItemExponentialFailureRateLimiter{
		baseDelay:    baseDelay,
		maxDelay:     maxDelay,
		failures:      make(map[interface{}]int),
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
		jitterFactor: 0.2, // 20% jitter
	}
}

// WithJitter sets the jitter factor (0.0 - 1.0)
func (rl *ItemExponentialFailureRateLimiter) WithJitter(factor float64) *ItemExponentialFailureRateLimiter {
	rl.jitterFactor = factor
	return rl
}

// When implements RateLimiter
func (rl *ItemExponentialFailureRateLimiter) When(item interface{}) time.Duration {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	failures := rl.failures[item]
	rl.failures[item] = failures + 1

	// Calculate delay: base * 2^failures clamped at maxDelay
	delay := time.Duration(float64(rl.baseDelay) * math.Pow(2, float64(failures)))
	if delay > rl.maxDelay {
		delay = rl.maxDelay
	}

	// Add jitter
	if rl.jitterFactor > 0 {
		jitter := 1.0 + rl.rng.Float64()*rl.jitterFactor
		delay = time.Duration(float64(delay) * jitter)
	}

	return delay
}

// Forget implements RateLimiter
func (rl *ItemExponentialFailureRateLimiter) Forget(item interface{}) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.failures, item)
}

// NumRequeues implements RateLimiter
func (rl *ItemExponentialFailureRateLimiter) NumRequeues(item interface{}) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.failures[item]
}

// MaxOfRateLimiter combines multiple rate limiters and uses the maximum delay
type MaxOfRateLimiter struct {
	limiters []RateLimiter
}

// NewMaxOfRateLimiter creates a new MaxOfRateLimiter
func NewMaxOfRateLimiter(limiters ...RateLimiter) RateLimiter {
	return &MaxOfRateLimiter{
		limiters: limiters,
	}
}

// When implements RateLimiter
func (rl *MaxOfRateLimiter) When(item interface{}) time.Duration {
	maxDelay := time.Duration(0)
	for _, l := range rl.limiters {
		delay := l.When(item)
		if delay > maxDelay {
			maxDelay = delay
		}
	}
	return maxDelay
}

// Forget implements RateLimiter
func (rl *MaxOfRateLimiter) Forget(item interface{}) {
	for _, l := range rl.limiters {
		l.Forget(item)
	}
}

// NumRequeues implements RateLimiter
func (rl *MaxOfRateLimiter) NumRequeues(item interface{}) int {
	max := 0
	for _, l := range rl.limiters {
		if n := l.NumRequeues(item); n > max {
			max = n
		}
	}
	return max
}

// BucketRateLimiter implements token bucket rate limiting
type BucketRateLimiter struct {
	limit  float64 // tokens per second
	tokens float64
	last   time.Time
	capacity float64
	mu      sync.Mutex
}

// NewBucketRateLimiter creates a new BucketRateLimiter with the given QPS and capacity
func NewBucketRateLimiter(qps float64, capacity int) RateLimiter {
	return &BucketRateLimiter{
		limit:    qps,
		capacity: float64(capacity),
		tokens:   float64(capacity),
		last:     time.Now(),
	}
}

// When implements RateLimiter
func (rl *BucketRateLimiter) When(item interface{}) time.Duration {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.last)
	rl.last = now

	// Add tokens based on elapsed time
	rl.tokens += elapsed.Seconds() * rl.limit
	if rl.tokens > rl.capacity {
		rl.tokens = rl.capacity
	}

	if rl.tokens >= 1.0 {
		rl.tokens--
		return 0
	}

	// Need to wait for a token
	waitSeconds := (1.0 - rl.tokens) / rl.limit
	return time.Duration(waitSeconds * float64(time.Second))
}

// Forget implements RateLimiter
func (rl *BucketRateLimiter) Forget(item interface{}) {
	// No state to forget per item
}

// NumRequeues implements RateLimiter
func (rl *BucketRateLimiter) NumRequeues(item interface{}) int {
	// No per-item tracking
	return 0
}

// DefaultRateLimiter returns the default rate limiter
// - Exponential backoff starting at 1s, max 30s
// - Overall QPS limit of 10 QPS
func DefaultRateLimiter() RateLimiter {
	exp := NewItemExponentialFailureRateLimiter(1*time.Second, 30*time.Second)
	bucket := NewBucketRateLimiter(10, 100)
	return NewMaxOfRateLimiter(exp, bucket)
}
