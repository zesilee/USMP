package plugin

import (
	"context"
	"sync"

	"github.com/leezesi/usmp/pkg/yang-runtime/reconcile"
)

// Manager manages all plugins and dispatches events to them
type Manager struct {
	validationPlugins  []ValidationPlugin
	mutationPlugins    []MutationPlugin
	notificationPlugins []NotificationPlugin
	reconciliationHooks []ReconciliationHookPlugin
	mu                  sync.RWMutex
}

// NewManager creates a new plugin manager
func NewManager() *Manager {
	return &Manager{
		validationPlugins:  make([]ValidationPlugin, 0),
		mutationPlugins:    make([]MutationPlugin, 0),
		notificationPlugins: make([]NotificationPlugin, 0),
		reconciliationHooks: make([]ReconciliationHookPlugin, 0),
	}
}

// AddValidationPlugin adds a validation plugin
func (m *Manager) AddValidationPlugin(p ValidationPlugin) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.validationPlugins = append(m.validationPlugins, p)
}

// AddMutationPlugin adds a mutation plugin
func (m *Manager) AddMutationPlugin(p MutationPlugin) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mutationPlugins = append(m.mutationPlugins, p)
}

// AddNotificationPlugin adds a notification plugin
func (m *Manager) AddNotificationPlugin(p NotificationPlugin) {
	m.mu.Lock()
	defer m.mu.Unlock();
	m.notificationPlugins = append(m.notificationPlugins, p)
}

// AddReconciliationHook adds a reconciliation hook plugin
func (m *Manager) AddReconciliationHook(p ReconciliationHookPlugin) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reconciliationHooks = append(m.reconciliationHooks, p)
}

// Validate runs all validation plugins on the change
// Returns the first error encountered, or nil if all validations pass
func (m *Manager) Validate(ctx context.Context, req reconcile.Request, change *Change) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.validationPlugins {
		if err := p.Validate(ctx, req, change); err != nil {
			return err
		}
	}
	return nil
}

// Mutate runs all mutation plugins on the change
// Returns the mutated change, or the original if no mutation plugins changed it
func (m *Manager) Mutate(ctx context.Context, req reconcile.Request, change *Change) (*Change, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	current := change
	for _, p := range m.mutationPlugins {
		mutated, err := p.Mutate(ctx, req, current)
		if err != nil {
			return nil, err
		}
		current = mutated
	}
	return current, nil
}

// OnSuccess notifies all notification plugins of a successful change
func (m *Manager) OnSuccess(ctx context.Context, req reconcile.Request, change *Change) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.notificationPlugins {
		p.OnSuccess(ctx, req, change)
	}
}

// OnFailure notifies all notification plugins of a failed change
func (m *Manager) OnFailure(ctx context.Context, req reconcile.Request, change *Change, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.notificationPlugins {
		p.OnFailure(ctx, req, change, err)
	}
}

// PreReconcile runs all pre-reconciliation hooks
func (m *Manager) PreReconcile(ctx context.Context, req reconcile.Request) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.reconciliationHooks {
		if err := p.PreReconcile(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

// PostReconcile runs all post-reconciliation hooks
func (m *Manager) PostReconcile(ctx context.Context, req reconcile.Request, result reconcile.Result) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.reconciliationHooks {
		if err := p.PostReconcile(ctx, req, result); err != nil {
			return err
		}
	}
	return nil
}

// ValidationPlugins returns a copy of the validation plugins list
func (m *Manager) ValidationPlugins() []ValidationPlugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cpy := make([]ValidationPlugin, len(m.validationPlugins))
	copy(cpy, m.validationPlugins)
	return cpy
}

// MutationPlugins returns a copy of the mutation plugins list
func (m *Manager) MutationPlugins() []MutationPlugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cpy := make([]MutationPlugin, len(m.mutationPlugins))
	copy(cpy, m.mutationPlugins)
	return cpy
}
