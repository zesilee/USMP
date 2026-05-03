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

// =============================================================================
// Cross-Module 2PC Transaction Coordination
// =============================================================================

// DeviceTransactionState tracks the state of a multi-module 2PC transaction.
type DeviceTransactionState struct {
	TransactionID string
	Status        string // "init", "preparing", "prepared", "committing", "committed", "aborting", "aborted", "failed"
	Modules       []string
	PrepareResults map[string]Result
	CommitResults  map[string]Result
	AbortResults   map[string]Result
	StartedAt     time.Time
	CompletedAt   time.Time
	Error         error
}

// PrepareAll executes Phase 1 (Prepare) of 2PC across all modules.
// If any module fails to prepare, the entire transaction fails and Abort is called.
func (a *DeviceActor) PrepareAll(ctx context.Context, dryRun bool) (*DeviceTransactionState, error) {
	a.mu.Lock()

	// Cannot prepare if device is not running
	if a.status != StatusReady && a.status != StatusRunning {
		a.mu.Unlock()
		return nil, fmt.Errorf("device not ready: %s", a.status)
	}

	// Get all module actors
	moduleNames := make([]string, 0, len(a.modules))
	moduleActors := make(map[string]Actor)
	for name, actor := range a.modules {
		moduleNames = append(moduleNames, name)
		moduleActors[name] = actor
	}
	a.mu.Unlock()

	// Initialize transaction state
	txState := &DeviceTransactionState{
		TransactionID:  fmt.Sprintf("tx-%d", time.Now().UnixNano()),
		Status:         "preparing",
		Modules:        moduleNames,
		PrepareResults: make(map[string]Result),
		CommitResults:  make(map[string]Result),
		AbortResults:   make(map[string]Result),
		StartedAt:      time.Now(),
	}

	// Step 1: Prepare all modules sequentially (parallel could be added later)
	allPrepared := true

	for moduleName, actor := range moduleActors {
		prepareCmd := &PrepareCmd{
			BaseMessage: NewBaseMessageWithContext(
				fmt.Sprintf("prepare-%s-%s", moduleName, txState.TransactionID),
				MsgPrepare,
				ctx,
			),
			DryRun: dryRun,
		}

		promise, err := actor.Send(prepareCmd)
		if err != nil {
			txState.PrepareResults[moduleName] = Result{
				MsgID:   prepareCmd.ID(),
				Success: false,
				Error:   err,
			}
			allPrepared = false
			continue
		}

		result := <-promise
		txState.PrepareResults[moduleName] = result

		if !result.Success {
			allPrepared = false
		}
	}

	// Step 2: Handle prepare outcome
	if !allPrepared {
		txState.Status = "failed"

		// Attempt to abort all modules on prepare failure (best effort)
		for moduleName, actor := range moduleActors {
			result, prepared := txState.PrepareResults[moduleName]
			if prepared && result.Success {
				// Only abort modules that successfully prepared (may have candidate)
				abortCmd := &AbortCmd{
					BaseMessage: NewBaseMessageWithContext(
						fmt.Sprintf("abort-%s-%s", moduleName, txState.TransactionID),
						MsgAbort,
						ctx,
					),
					Reason: "Prepare phase failed for other module",
				}

				promise, _ := actor.Send(abortCmd)
				abortResult := <-promise
				txState.AbortResults[moduleName] = abortResult
			}
		}

		txState.CompletedAt = time.Now()
		txState.Error = fmt.Errorf("prepare phase failed for one or more modules")
		return txState, txState.Error
	}

	txState.Status = "prepared"
	return txState, nil
}

// CommitAll executes Phase 2 (Commit) of 2PC across all modules.
// Must be called after a successful PrepareAll.
func (a *DeviceActor) CommitAll(ctx context.Context, forceCommit bool) (*DeviceTransactionState, error) {
	a.mu.Lock()

	if a.status != StatusReady && a.status != StatusRunning {
		a.mu.Unlock()
		return nil, fmt.Errorf("device not ready: %s", a.status)
	}

	// Get all module actors (consistent set as when prepare was called)
	moduleActors := make(map[string]Actor)
	for name, actor := range a.modules {
		moduleActors[name] = actor
	}
	a.mu.Unlock()

	// Initialize transaction state for commit
	txState := &DeviceTransactionState{
		TransactionID: fmt.Sprintf("tx-commit-%d", time.Now().UnixNano()),
		Status:        "committing",
		Modules:       make([]string, 0, len(moduleActors)),
		CommitResults: make(map[string]Result),
		StartedAt:     time.Now(),
	}

	for name := range moduleActors {
		txState.Modules = append(txState.Modules, name)
	}

	// Commit all modules
	allCommitted := true

	for moduleName, actor := range moduleActors {
		commitCmd := &CommitCmd{
			BaseMessage: NewBaseMessageWithContext(
				fmt.Sprintf("commit-%s-%s", moduleName, txState.TransactionID),
				MsgCommit,
				ctx,
			),
			ForceCommit: forceCommit,
		}

		promise, err := actor.Send(commitCmd)
		if err != nil {
			txState.CommitResults[moduleName] = Result{
				MsgID:   commitCmd.ID(),
				Success: false,
				Error:   err,
			}
			allCommitted = false
			continue
		}

		result := <-promise
		txState.CommitResults[moduleName] = result

		if !result.Success {
			allCommitted = false
		}
	}

	if allCommitted {
		txState.Status = "committed"
	} else {
		txState.Status = "failed"
		txState.Error = fmt.Errorf("commit phase failed for one or more modules")
	}

	txState.CompletedAt = time.Now()

	if !allCommitted {
		return txState, txState.Error
	}

	return txState, nil
}

// AbortAll aborts a multi-module 2PC transaction across all modules.
// This can be called after PrepareAll but before CommitAll.
func (a *DeviceActor) AbortAll(ctx context.Context, reason string) (*DeviceTransactionState, error) {
	a.mu.Lock()

	if a.status != StatusReady && a.status != StatusRunning {
		a.mu.Unlock()
		return nil, fmt.Errorf("device not ready: %s", a.status)
	}

	moduleActors := make(map[string]Actor)
	for name, actor := range a.modules {
		moduleActors[name] = actor
	}
	a.mu.Unlock()

	txState := &DeviceTransactionState{
		TransactionID: fmt.Sprintf("tx-abort-%d", time.Now().UnixNano()),
		Status:        "aborting",
		Modules:       make([]string, 0, len(moduleActors)),
		AbortResults:  make(map[string]Result),
		StartedAt:     time.Now(),
	}

	for name := range moduleActors {
		txState.Modules = append(txState.Modules, name)
	}

	allAborted := true

	for moduleName, actor := range moduleActors {
		abortCmd := &AbortCmd{
			BaseMessage: NewBaseMessageWithContext(
				fmt.Sprintf("abort-%s-%s", moduleName, txState.TransactionID),
				MsgAbort,
				ctx,
			),
			Reason: reason,
		}

		promise, err := actor.Send(abortCmd)
		if err != nil {
			txState.AbortResults[moduleName] = Result{
				MsgID:   abortCmd.ID(),
				Success: false,
				Error:   err,
			}
			allAborted = false
			continue
		}

		result := <-promise
		txState.AbortResults[moduleName] = result

		if !result.Success {
			allAborted = false
		}
	}

	if allAborted {
		txState.Status = "aborted"
	} else {
		txState.Status = "failed"
		txState.Error = fmt.Errorf("abort failed for one or more modules")
	}

	txState.CompletedAt = time.Now()

	if !allAborted {
		return txState, txState.Error
	}

	return txState, nil
}

// PrepareAndCommitAll executes the full 2PC transaction (Prepare -> Commit) atomically.
// This is the primary API for multi-module configuration deployment.
func (a *DeviceActor) PrepareAndCommitAll(ctx context.Context, dryRun bool) (*DeviceTransactionState, error) {
	// Phase 1: Prepare
	txState, err := a.PrepareAll(ctx, dryRun)
	if err != nil {
		return txState, fmt.Errorf("prepare phase failed: %w", err)
	}

	// If dry run, return after prepare without committing
	if dryRun {
		return txState, nil
	}

	// Phase 2: Commit
	txState, err = a.CommitAll(ctx, false)
	if err != nil {
		return txState, fmt.Errorf("commit phase failed: %w", err)
	}

	return txState, nil
}
