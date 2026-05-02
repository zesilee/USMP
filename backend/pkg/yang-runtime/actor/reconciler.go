package actor

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
)

// ActorReconciler implements reconcile.Reconciler using the Actor model.
// This bridges the K8s controller-style reconciler interface with the Actor-based configuration management.
type ActorReconciler struct {
	actor       Actor
	configStore reconcile.ConfigStore
}

// NewActorReconciler creates a new ActorReconciler wrapping the given actor.
func NewActorReconciler(actor Actor, configStore reconcile.ConfigStore) *ActorReconciler {
	return &ActorReconciler{
		actor:       actor,
		configStore: configStore,
	}
}

// Reconcile implements reconcile.Reconciler interface.
// It performs a full reconciliation cycle:
// 1. Get desired state from config store
// 2. Translate desired state to YANG struct
// 3. Fetch actual state from device
// 4. Compute diff
// 5. Apply changes to device
func (r *ActorReconciler) Reconcile(ctx context.Context, req reconcile.Request) reconcile.Result {
	// Step 1: Get desired state from config store
	desired, err := r.configStore.Get(req.DeviceID, req.Path)
	if err != nil {
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: 5 * time.Second,
			Error:        err,
		}
	}

	// Step 2: Translate CR Spec to YANG struct
	payload, ok := desired.(map[string]interface{})
	if !ok {
		// Try to convert via JSON serialization
		dataBytes, err := json.Marshal(desired)
		if err != nil {
			return reconcile.Result{
				Requeue:      true,
				RequeueAfter: 5 * time.Second,
				Error:        err,
			}
		}
		if err := json.Unmarshal(dataBytes, &payload); err != nil {
			return reconcile.Result{
				Requeue:      true,
				RequeueAfter: 5 * time.Second,
				Error:        err,
			}
		}
	}

	translateCmd := &TranslateCmd{
		BaseMessage: NewBaseMessageWithContext(req.DeviceID+"-"+req.Path, MsgTranslate, ctx),
		Path:        req.Path,
		Payload:     payload,
		Operation:   OperationMerge,
	}

	promise, err := r.actor.Send(translateCmd)
	if err != nil {
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: 5 * time.Second,
			Error:        err,
		}
	}

	translateResult := <-promise
	if !translateResult.Success {
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: 30 * time.Second,
			Error:        translateResult.Error,
		}
	}

	// Step 3: Apply changes to device
	applyCmd := &ApplyCmd{
		BaseMessage: NewBaseMessageWithContext(req.DeviceID+"-"+req.Path, MsgApply, ctx),
	}

	promise, err = r.actor.Send(applyCmd)
	if err != nil {
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: 5 * time.Second,
			Error:        err,
		}
	}

	applyResult := <-promise
	if !applyResult.Success {
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: 30 * time.Second,
			Error:        applyResult.Error,
		}
	}

	// Success - no requeue needed
	return reconcile.Result{
		Requeue:      false,
		RequeueAfter: 0,
		Error:        nil,
	}
}

// ActorManager manages actors for multiple devices and modules.
// This is designed for integration with K8s controller manager.
type ActorManager struct {
	deviceActors  map[string]*DeviceActor
	clientPool    client.ClientPool
	moduleFactory *ModuleFactory
	mu            sync.RWMutex
	configStore   reconcile.ConfigStore
}

// NewActorManager creates a new ActorManager.
func NewActorManager(clientPool client.ClientPool, configStore reconcile.ConfigStore) *ActorManager {
	return &ActorManager{
		deviceActors:  make(map[string]*DeviceActor),
		clientPool:    clientPool,
		moduleFactory: NewModuleFactory(clientPool),
		configStore:   configStore,
	}
}

// GetDeviceActor returns or creates the DeviceActor for the given device.
// Automatically registers all supported modules for new devices.
func (m *ActorManager) GetDeviceActor(deviceID string) *DeviceActor {
	m.mu.RLock()
	actor, exists := m.deviceActors[deviceID]
	m.mu.RUnlock()

	if exists {
		return actor
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double check after write lock
	if actor, exists := m.deviceActors[deviceID]; exists {
		return actor
	}

	actor = NewDeviceActor(deviceID, m.clientPool)
	// Register all modules for new device
	if err := m.moduleFactory.RegisterAllModules(actor); err != nil {
		// Log error but continue with partially registered actor
	}
	m.deviceActors[deviceID] = actor
	return actor
}

// StartAll starts all managed actors.
func (m *ActorManager) StartAll() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, actor := range m.deviceActors {
		if err := actor.Start(); err != nil {
			return err
		}
	}

	return nil
}

// StopAll stops all managed actors.
func (m *ActorManager) StopAll() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var lastErr error
	for _, actor := range m.deviceActors {
		if err := actor.Stop(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// GetStatus returns aggregated status for all actors.
func (m *ActorManager) GetStatus() map[string]StatusInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statusMap := make(map[string]StatusInfo)
	for deviceID, actor := range m.deviceActors {
		statusMap[deviceID] = actor.Status()
	}

	return statusMap
}

// GetReconcilerForModule returns a reconciler for a specific module on a device.
func (m *ActorManager) GetReconcilerForModule(deviceID, moduleName string) (reconcile.Reconciler, error) {
	deviceActor := m.GetDeviceActor(deviceID)
	moduleActor, exists := deviceActor.GetModuleActor(moduleName)
	if !exists {
		return nil, nil
	}

	return NewActorReconciler(moduleActor, m.configStore), nil
}
