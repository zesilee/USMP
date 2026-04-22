package queue

import "time"

// Interface is the core work queue interface
type Interface interface {
	// Add adds an item to the queue
	Add(item interface{})
	// Len returns the current queue length
	Len() int
	// Get blocks until an item is available
	Get() (item interface{}, shutdown bool)
	// Done marks the item as processed
	Done(item interface{})
	// ShutDown stops the queue
	ShutDown()
	// ShutDownWithDrain stops the queue after processing all items
	ShutDownWithDrain()
	// ShuttingDown returns true if the queue is shutting down
	ShuttingDown() bool
}

// RateLimitingInterface adds rate limiting capabilities to the queue
type RateLimitingInterface interface {
	Interface
	// AddAfter adds an item after the specified delay
	AddAfter(item interface{}, delay time.Duration)
	// AddRateLimited adds an item with rate limiting
	AddRateLimited(item interface{})
	// Forget removes the item from rate limiter tracking
	Forget(item interface{})
	// NumRequeues returns how many times the item was requeued
	NumRequeues(item interface{}) int
}

// RateLimiter defines the interface for rate limiting strategies
type RateLimiter interface {
	// When returns the delay before the next retry
	When(item interface{}) time.Duration
	// Forget clears rate limit state for an item
	Forget(item interface{})
	// NumRequeues returns the number of times item was retried
	NumRequeues(item interface{}) int
}
