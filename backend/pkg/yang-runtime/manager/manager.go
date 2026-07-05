package manager

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/leezesi/usmp/backend/internal/cache"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/controller"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/plugin"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/predicate"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/status"
)

// InMemoryConfigStore is an in-memory ConfigStore implementation backed by TTL+LRU cache
type InMemoryConfigStore struct {
	cache *cache.TTLLRUCache
}

// NewInMemoryConfigStore creates a new in-memory config store
func NewInMemoryConfigStore(c *cache.TTLLRUCache) *InMemoryConfigStore {
	return &InMemoryConfigStore{
		cache: c,
	}
}

// Get retrieves the desired configuration at the given path for a device
func (s *InMemoryConfigStore) Get(deviceID, path string) (interface{}, error) {
	key := fmt.Sprintf("%s:%s", deviceID, path)
	val, ok := s.cache.Get(key)
	if !ok {
		return nil, nil // nil means no config exists
	}
	return val, nil
}

// Set stores the desired configuration at the given path for a device
func (s *InMemoryConfigStore) Set(deviceID, path string, value interface{}) error {
	key := fmt.Sprintf("%s:%s", deviceID, path)
	s.cache.Set(key, value)
	return nil
}

// Delete removes the desired configuration at the given path for a device
func (s *InMemoryConfigStore) Delete(deviceID, path string) error {
	key := fmt.Sprintf("%s:%s", deviceID, path)
	s.cache.Delete(key)
	return nil
}

// List lists all paths that have desired configuration for a device. Keys are
// stored as "deviceID:path" (see Set); the deviceID is assumed colon-free (IPv4),
// so the first colon separates deviceID from path.
func (s *InMemoryConfigStore) List(deviceID string) ([]string, error) {
	prefix := deviceID + ":"
	paths := make([]string, 0)
	for _, k := range s.cache.Keys() {
		if strings.HasPrefix(k, prefix) {
			paths = append(paths, strings.TrimPrefix(k, prefix))
		}
	}
	return paths, nil
}

// ListDevices lists the distinct devices that have desired configuration.
func (s *InMemoryConfigStore) ListDevices() ([]string, error) {
	seen := make(map[string]struct{})
	devices := make([]string, 0)
	for _, k := range s.cache.Keys() {
		i := strings.Index(k, ":")
		if i < 0 {
			continue
		}
		dev := k[:i]
		if _, ok := seen[dev]; !ok {
			seen[dev] = struct{}{}
			devices = append(devices, dev)
		}
	}
	return devices, nil
}

// Manager is the main entry point for the yang-controller-runtime framework
// It manages:
// - Global lifecycle (start/stop)
// - Schema loading and caching
// - Client connection pool
// - Controller registration and starting
// - Plugin management
type Manager interface {
	// Start starts the manager and all registered controllers
	Start(ctx context.Context) error
	// Stop stops the manager and all registered controllers
	Stop() error
	// AddController adds a controller to the manager
	AddController(ctrl controller.Controller)
	// GetSchema returns the loaded schema
	GetSchema() schema.Schema
	// GetClientPool returns the client connection pool
	GetClientPool() client.ClientPool
	// GetConfigStore returns the desired configuration store
	GetConfigStore() reconcile.ConfigStore
	// GetReconcileStatus returns the store of most-recent reconcile outcomes
	GetReconcileStatus() *status.Store
	// GetPluginManager returns the plugin manager
	GetPluginManager() *plugin.Manager
	// AddPlugin adds a plugin
	AddPlugin(p plugin.Plugin)
	// TriggerReconcile triggers immediate reconciliation for a device and path
	// Returns true if a matching controller was found and triggered
	TriggerReconcile(deviceID, path string) bool
}

// DefaultManager is the default implementation of Manager
type DefaultManager struct {
	options         Options
	schema          schema.Schema
	clientPool      client.ClientPool
	configStore     reconcile.ConfigStore
	reconcileStatus *status.Store
	controllers     []controller.Controller
	pluginManager   *plugin.Manager
	started         bool
	ctx             context.Context
	cancel          context.CancelFunc
}

// New creates a new DefaultManager with the given options
func New(opts ...Option) *DefaultManager {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	var s schema.Schema
	if options.Schema != nil {
		// Pre-built schema (e.g. from generated ygot models) takes precedence.
		s = options.Schema
	} else {
		// Empty; may be loaded from SchemeDir in Start.
		s = schema.NewSchema()
	}

	// Use the existing TTL+LRU cache as backing store for config store
	// max entries 1000, cleanup every minute, entry TTL 5 minutes
	cache := cache.NewTTLLRUCache(1000, 1*time.Minute, 5*time.Minute)
	cs := NewInMemoryConfigStore(cache)

	m := &DefaultManager{
		options:         options,
		schema:          s,
		clientPool:      client.NewDefaultClientPool(options.ClientFactory),
		configStore:     cs,
		reconcileStatus: status.NewStore(),
		controllers:     make([]controller.Controller, 0),
		pluginManager:   plugin.NewManager(),
	}

	return m
}

// Start implements Manager interface
func (m *DefaultManager) Start(ctx context.Context) error {
	if m.started {
		return nil
	}

	m.ctx, m.cancel = context.WithCancel(ctx)

	// Load schemas from directory if provided and no pre-built schema was injected.
	if m.options.Schema == nil && m.options.SchemeDir != "" {
		loader := schema.NewLoader(m.options.SchemeDir)
		loadedSchema, err := loader.Load()
		if err != nil {
			return err
		}
		m.schema = loadedSchema
	}

	// Start all controllers
	for _, ctrl := range m.controllers {
		if err := ctrl.Start(m.ctx); err != nil {
			return err
		}
	}

	m.started = true
	return nil
}

// Stop implements Manager interface
func (m *DefaultManager) Stop() error {
	if !m.started {
		return nil
	}

	// Stop all controllers
	for _, ctrl := range m.controllers {
		ctrl.Stop()
	}

	// Close all client connections
	if err := m.clientPool.CloseAll(); err != nil {
		// Log but continue shutdown
	}

	m.cancel()
	m.started = false
	return nil
}

// AddController implements Manager interface.
// If the controller can accept a status recorder, it is wired to the manager's
// shared reconcile-status store so its outcomes become queryable. Controllers
// that do not implement status.RecorderSetter simply do not record (R08).
func (m *DefaultManager) AddController(ctrl controller.Controller) {
	if setter, ok := ctrl.(status.RecorderSetter); ok {
		setter.SetStatusRecorder(m.reconcileStatus)
	}
	m.controllers = append(m.controllers, ctrl)
}

// GetReconcileStatus implements Manager interface
func (m *DefaultManager) GetReconcileStatus() *status.Store {
	return m.reconcileStatus
}

// GetSchema implements Manager interface
func (m *DefaultManager) GetSchema() schema.Schema {
	return m.schema
}

// GetClientPool implements Manager interface
func (m *DefaultManager) GetClientPool() client.ClientPool {
	return m.clientPool
}

// GetConfigStore implements Manager interface
func (m *DefaultManager) GetConfigStore() reconcile.ConfigStore {
	return m.configStore
}

// GetPluginManager implements Manager interface
func (m *DefaultManager) GetPluginManager() *plugin.Manager {
	return m.pluginManager
}

// AddPlugin implements Manager interface
func (m *DefaultManager) AddPlugin(p plugin.Plugin) {
	switch pl := p.(type) {
	case plugin.ValidationPlugin:
		m.pluginManager.AddValidationPlugin(pl)
	case plugin.MutationPlugin:
		m.pluginManager.AddMutationPlugin(pl)
	case plugin.NotificationPlugin:
		m.pluginManager.AddNotificationPlugin(pl)
	case plugin.ReconciliationHookPlugin:
		m.pluginManager.AddReconciliationHook(pl)
	}
}

// Controllers returns the list of registered controllers
func (m *DefaultManager) Controllers() []controller.Controller {
	return m.controllers
}

// TriggerReconcile triggers immediate reconciliation for a device and path
// It finds the controller that handles the given path prefix and enqueues an event
func (m *DefaultManager) TriggerReconcile(deviceID, path string) bool {
	for _, ctrl := range m.controllers {
		// Check if this controller handles the path using its predicates
		// We use a simple prefix match heuristic since we can't inspect predicates directly
		// The controller name typically contains the module identifier
		ctrlName := ctrl.Name()

		// Match based on path prefixes - this aligns with our controller registration
		if strings.Contains(path, "system:") && strings.Contains(ctrlName, "system") {
			ctrl.Enqueue(predicate.Event{
				Type:     predicate.UpdateEvent,
				DeviceID: deviceID,
				Path:     path,
			})
			return true
		}
		if (strings.Contains(path, "vlan:") || strings.Contains(path, "vlans")) &&
			strings.Contains(ctrlName, "vlan") {
			ctrl.Enqueue(predicate.Event{
				Type:     predicate.UpdateEvent,
				DeviceID: deviceID,
				Path:     path,
			})
			return true
		}
		if (strings.Contains(path, "ifm:") || strings.Contains(path, "interfaces")) &&
			strings.Contains(ctrlName, "ifm") {
			ctrl.Enqueue(predicate.Event{
				Type:     predicate.UpdateEvent,
				DeviceID: deviceID,
				Path:     path,
			})
			return true
		}
	}
	return false
}
