package manager

import (
	"time"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/queue"
)

// Options contains options for creating a new Manager
type Options struct {
	// ClientFactory is the factory for creating device clients
	ClientFactory client.ClientFactory
	// DefaultTimeout is the default connection timeout
	DefaultTimeout time.Duration
	// RateLimiter is the default rate limiter for controller queues
	RateLimiter queue.RateLimiter
	// SchemeDir is the directory containing YANG schema files
	SchemeDir string
	// EnableDebug enables debug logging
	EnableDebug bool
}

// Option is a function that modifies Options
type Option func(*Options)

// WithClientFactory sets the client factory
func WithClientFactory(f client.ClientFactory) Option {
	return func(o *Options) {
		o.ClientFactory = f
	}
}

// WithDefaultTimeout sets the default connection timeout
func WithDefaultTimeout(t time.Duration) Option {
	return func(o *Options) {
		o.DefaultTimeout = t
	}
}

// WithRateLimiter sets the default rate limiter
func WithRateLimiter(r queue.RateLimiter) Option {
	return func(o *Options) {
		o.RateLimiter = r
	}
}

// WithSchemeDir sets the directory containing YANG schema files
func WithSchemeDir(dir string) Option {
	return func(o *Options) {
		o.SchemeDir = dir
	}
}

// WithDebug enables debug logging
func WithDebug() Option {
	return func(o *Options) {
		o.EnableDebug = true
	}
}

// DefaultOptions returns the default options
func DefaultOptions() Options {
	return Options{
		ClientFactory:  client.DefaultClientFactory(10 * time.Second),
		DefaultTimeout: 10 * time.Second,
		RateLimiter:    queue.DefaultRateLimiter(),
		EnableDebug:    false,
	}
}
