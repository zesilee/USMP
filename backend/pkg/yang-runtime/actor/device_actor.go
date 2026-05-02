package actor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
)

// DeviceActor manages configuration for an entire network device.
// It coordinates multiple ModelActors for different YANG modules.
type DeviceActor struct {
	mu          sync.RWMutex
	deviceID    string
	clientPool  client.ClientPool
	modules     map[string]Actor // module name -> actor
	status      ActorStatus
	lastError   error
	lastActivity time.Time
	startTime   time.Time
	running     bool
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewDeviceActor creates a new DeviceActor for the specified device.
func NewDeviceActor(deviceID string, clientPool client.ClientPool) *DeviceActor {
	ctx, cancel := context.WithCancel(context.Background())
	return &DeviceActor{
		deviceID:   deviceID,
		clientPool: clientPool,
		modules:    make(map[string]Actor),
		status:     StatusInitializing,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// RegisterModuleActor registers a ModelActor for a specific YANG module.
func (a *DeviceActor) RegisterModuleActor(moduleName string, actor Actor) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.modules[moduleName]; exists {
		return fmt.Errorf("module %s already registered", moduleName)
	}

	a.modules[moduleName] = actor
	return nil
}

// GetModuleActor returns the actor for the specified module.
func (a *DeviceActor) GetModuleActor(moduleName string) (Actor, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	actor, exists := a.modules[moduleName]
	return actor, exists
}

// Start implements the Actor interface.
func (a *DeviceActor) Start() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return nil
	}

	// Start all module actors
	for module, actor := range a.modules {
		if err := actor.Start(); err != nil {
			return fmt.Errorf("failed to start module %s: %w", module, err)
		}
	}

	a.running = true
	a.status = StatusReady
	a.startTime = time.Now()

	return nil
}

// Stop implements the Actor interface.
func (a *DeviceActor) Stop() error {
	a.mu.Lock()
	if !a.running {
		a.mu.Unlock()
		return nil
	}
	a.running = false
	a.cancel()
	a.mu.Unlock()

	a.wg.Wait()

	// Stop all module actors
	a.mu.Lock()
	defer a.mu.Unlock()

	var lastErr error
	for module, actor := range a.modules {
		if err := actor.Stop(); err != nil {
			lastErr = fmt.Errorf("module %s stop error: %w", module, err)
		}
	}

	a.status = StatusStopped
	return lastErr
}

// Send implements the Actor interface.
// The message path determines which module actor to route to.
func (a *DeviceActor) Send(msg Message) (ResultPromise, error) {
	a.mu.RLock()
	status := a.status
	a.mu.RUnlock()

	if status == StatusStopped {
		return nil, fmt.Errorf("device actor is stopped")
	}

	// Determine target module from message path
	// For TranslateCmd, the path prefix indicates the module
	var moduleName string
	switch m := msg.(type) {
	case *TranslateCmd:
		moduleName = extractModuleFromPath(m.Path)
	case *ApplyCmd:
		moduleName = extractModuleFromPath("") // Default module for ApplyCmd
	case *PrepareCmd:
		moduleName = extractModuleFromPath("")
	case *CommitCmd:
		moduleName = extractModuleFromPath("")
	case *RollbackCmd:
		moduleName = extractModuleFromPath("")
	case *ValidateCmd:
		moduleName = extractModuleFromPath("")
	case *StatusQueryCmd:
		return a.handleDeviceStatusQuery(m), nil
	default:
		return nil, fmt.Errorf("unsupported message type: %T", msg)
	}

	// Route to appropriate module actor
	a.mu.RLock()
	moduleActor, exists := a.modules[moduleName]
	a.mu.RUnlock()

	if !exists {
		// Fallback to first module if specific module not found
		a.mu.RLock()
		if len(a.modules) == 0 {
			a.mu.RUnlock()
			return nil, fmt.Errorf("no modules registered")
		}
		for _, moduleActor = range a.modules {
			break // Take first module
		}
		a.mu.RUnlock()
	}

	promise, err := moduleActor.Send(msg)
	if err != nil {
		return nil, err
	}

	// Update device last activity when result is received
	devicePromise := make(chan Result, 1)
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		result := <-promise
		a.mu.Lock()
		a.lastActivity = time.Now()
		if result.Error != nil {
			a.lastError = result.Error
		}
		a.mu.Unlock()
		devicePromise <- result
		close(devicePromise)
	}()

	return devicePromise, nil
}

// Status implements the Actor interface.
func (a *DeviceActor) Status() StatusInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	aggregatedMessageCount := int64(0)
	for _, actor := range a.modules {
		moduleStatus := actor.Status()
		aggregatedMessageCount += moduleStatus.MessageCount
	}

	return StatusInfo{
		ActorID:      a.deviceID,
		Module:       "device",
		DeviceID:     a.deviceID,
		Status:       a.status,
		LastError:    a.lastError,
		LastActivity: a.lastActivity,
		MessageCount: aggregatedMessageCount,
		Uptime:       time.Since(a.startTime),
	}
}

// handleDeviceStatusQuery handles status queries for the device-level actor.
func (a *DeviceActor) handleDeviceStatusQuery(cmd *StatusQueryCmd) ResultPromise {
	promise := make(chan Result, 1)

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()

		status := a.Status()

		data := map[string]interface{}{
			"device_id":      status.DeviceID,
			"status":         string(status.Status),
			"message_count":  status.MessageCount,
			"uptime":         status.Uptime.String(),
			"modules_count":  len(a.modules),
		}

		if cmd.IncludeDetails {
			data["last_activity"] = status.LastActivity
			if status.LastError != nil {
				data["last_error"] = status.LastError.Error()
			}

			// Get status from all module actors
			moduleStatuses := make(map[string]interface{})
			a.mu.RLock()
			for moduleName, actor := range a.modules {
				moduleStatus := actor.Status()
				moduleStatuses[moduleName] = map[string]interface{}{
					"status":         string(moduleStatus.Status),
					"message_count":  moduleStatus.MessageCount,
					"current_version": moduleStatus.CurrentVersion,
				}
			}
			a.mu.RUnlock()
			data["modules"] = moduleStatuses
		}

		promise <- Result{
			MsgID:   cmd.ID(),
			Success: true,
			Data:    data,
		}
		close(promise)
	}()

	return promise
}

// extractModuleFromPath extracts the module name from a YANG path.
// This is a simplified implementation - can be enhanced based on actual path structure.
func extractModuleFromPath(path string) string {
	// For now, return a default module name
	// In a real implementation, this would parse the path and extract the module
	return "default"
}

// ApplyAll applies configuration changes to all modules.
// This is useful for bulk configuration operations.
func (a *DeviceActor) ApplyAll(ctx context.Context) ([]Result, error) {
	a.mu.RLock()
	modules := make([]Actor, 0, len(a.modules))
	for _, actor := range a.modules {
		modules = append(modules, actor)
	}
	a.mu.RUnlock()

	results := make([]Result, 0, len(modules))

	for _, actor := range modules {
		cmd := &ApplyCmd{
			BaseMessage: NewBaseMessageWithContext("apply-all", MsgApply, ctx),
		}

		promise, err := actor.Send(cmd)
		if err != nil {
			return results, err
		}

		result := <-promise
		results = append(results, result)
	}

	return results, nil
}

// RollbackAll rolls back all modules to the specified version.
func (a *DeviceActor) RollbackAll(ctx context.Context, targetVersion int64) ([]Result, error) {
	a.mu.RLock()
	modules := make([]Actor, 0, len(a.modules))
	for _, actor := range a.modules {
		modules = append(modules, actor)
	}
	a.mu.RUnlock()

	results := make([]Result, 0, len(modules))

	for _, actor := range modules {
		cmd := &RollbackCmd{
			BaseMessage:   NewBaseMessageWithContext("rollback-all", MsgRollback, ctx),
			TargetVersion: targetVersion,
		}

		promise, err := actor.Send(cmd)
		if err != nil {
			return results, err
		}

		result := <-promise
		results = append(results, result)
	}

	return results, nil
}
