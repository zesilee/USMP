package manager

import (
	"time"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/audit"
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
	// AuditFile 已退役（OA-05/SC-06）：非空仅产生弃用警告（不再写文件）。
	AuditFile string
	// AuditStore overrides the operation-audit backend（集群模式注入 CRD
	// 实现，OA-02）。nil 时按 AuditFile 缺省为内存实现。
	AuditStore audit.Store
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

// WithAuditFile 已退役（OA-05）：仅保留兼容，非空路径产生弃用警告。
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

// WithAuditStore overrides the operation-audit backend（集群模式注入
// AuditRecord CRD 实现，OA-02；缺省为进程内存实现）。
func WithAuditStore(s audit.Store) Option {
	return func(o *Options) {
		o.AuditStore = s
	}
}
