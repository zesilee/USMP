package manager

import (
	"time"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/queue"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
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
	// Schema is a pre-built YANG schema to use (takes precedence over SchemeDir).
	// Typically built from generated ygot models via schema.AddYgotSchema.
	Schema schema.Schema
	// EnableDebug enables debug logging
	EnableDebug bool
	// AuditFile is the local JSON file the operation-audit log persists to
	// (§8). Empty ("") keeps the audit log in memory only.
	AuditFile string
	// DeviceStore overrides the shared device registry backend（集群模式注入
	// CRD store，DS-01）。nil 时使用缺省内存实现。
	DeviceStore device.Store
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

// WithAuditFile sets the local JSON file the operation-audit log persists to
// (§8). Empty keeps it in memory only.
func WithAuditFile(path string) Option {
	return func(o *Options) {
		o.AuditFile = path
	}
}

// WithSchema injects a pre-built YANG schema (e.g. built from generated ygot
// models). Takes precedence over SchemeDir. Fixes the empty-schema-tree gap.
func WithSchema(s schema.Schema) Option {
	return func(o *Options) {
		o.Schema = s
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

// WithDeviceStore overrides the shared device registry backend（集群模式注入
// CRD store，DS-01；缺省为进程内存实现）。
func WithDeviceStore(s device.Store) Option {
	return func(o *Options) {
		o.DeviceStore = s
	}
}
