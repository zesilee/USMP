package actor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/openconfig/ygot/ygot"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/diff"
)

// YANGGoStruct is the interface implemented by all ygot-generated structs.
type YANGGoStruct interface {
	ygot.GoStruct
	Validate(...ygot.ValidationOption) error
}

// Actor defines the core interface for all configuration actors.
type Actor interface {
	// Start initializes and starts the actor message processing loop.
	Start() error
	// Stop gracefully stops the actor.
	Stop() error
	// Send sends a message to the actor. Returns immediately, result is async.
	Send(msg Message) (ResultPromise, error)
	// Status returns the current actor status.
	Status() StatusInfo
}

// DeviceClient provides the interface for interacting with a network device.
type DeviceClient interface {
	// Get retrieves the current configuration from the device.
	Get(ctx context.Context, path string) (YANGGoStruct, error)
	// Set applies the configuration changes to the device.
	Set(ctx context.Context, changes []Change) error
	// Commit commits the candidate configuration to running.
	Commit(ctx context.Context) error
	// Discard discards the candidate configuration.
	Discard(ctx context.Context) error
}

// Change represents a configuration change to apply to the device.
type Change struct {
	Path    string
	Value   interface{}
	Op      ChangeOp
}

// ChangeOp represents the type of configuration change operation.
type ChangeOp string

const (
	ChangeOpMerge  ChangeOp = "merge"
	ChangeOpReplace ChangeOp = "replace"
	ChangeOpDelete  ChangeOp = "delete"
)

// ModelActor is the generic base actor for a specific YANG module type.
// T is the ygot-generated struct type for the module (e.g., *HuaweiVlan_Vlan).
type ModelActor[T YANGGoStruct] struct {
	// Identity
	actorID  string
	deviceID string
	module   string

	// State management
	mu          sync.RWMutex
	desired     T       // Desired configuration from CR Spec
	actual      T       // Actual configuration from device
	state       T       // Working state (merged for reconciliation)
	status      ActorStatus
	lastError   error
	lastActivity time.Time
	messageCount int64
	startTime   time.Time

	// Dependencies
	versionMgr *VersionManager[T]
	translator Translator[T]
	clientPool client.ClientPool

	// Messaging
	msgChan chan msgWithPromise
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
	wg      sync.WaitGroup
}

type msgWithPromise struct {
	msg     Message
	promise ResultPromise
}

// NewModelActor creates a new ModelActor for the given YANG module type.
func NewModelActor[T YANGGoStruct](
	actorID string,
	deviceID string,
	clientPool client.ClientPool,
	translator Translator[T],
) *ModelActor[T] {
	// Create zero values for the YANG structs
	var zero T
	structType := reflect.TypeOf(zero).Elem()

	desired := reflect.New(structType).Interface().(T)
	actual := reflect.New(structType).Interface().(T)
	state := reflect.New(structType).Interface().(T)

	ctx, cancel := context.WithCancel(context.Background())

	return &ModelActor[T]{
		actorID:    actorID,
		deviceID:   deviceID,
		module:     "", // Set by specific actor implementations
		desired:    desired,
		actual:     actual,
		state:      state,
		status:     StatusInitializing,
		versionMgr: NewVersionManager[T](50),
		translator: translator,
		clientPool: clientPool,
		msgChan:    make(chan msgWithPromise, 100),
		ctx:        ctx,
		cancel:     cancel,
		startTime:  time.Now(),
	}
}

// Start implements the Actor interface.
func (a *ModelActor[T]) Start() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return nil
	}

	a.running = true
	a.status = StatusReady
	a.startTime = time.Now()

	a.wg.Add(1)
	go a.runMessageLoop()

	return nil
}

// Stop implements the Actor interface.
func (a *ModelActor[T]) Stop() error {
	a.mu.Lock()
	if !a.running {
		a.mu.Unlock()
		return nil
	}
	a.running = false
	a.cancel()
	a.mu.Unlock()

	a.wg.Wait()

	a.mu.Lock()
	a.status = StatusStopped
	a.mu.Unlock()

	return nil
}

// Send implements the Actor interface.
func (a *ModelActor[T]) Send(msg Message) (ResultPromise, error) {
	a.mu.RLock()
	status := a.status
	a.mu.RUnlock()

	if status == StatusStopped {
		return nil, errors.New("actor is stopped")
	}

	promise := NewResultPromise()
	select {
	case a.msgChan <- msgWithPromise{msg: msg, promise: promise}:
		return promise, nil
	case <-a.ctx.Done():
		return nil, a.ctx.Err()
	case <-time.After(5 * time.Second):
		return nil, errors.New("actor mailbox full")
	}
}

// Status implements the Actor interface.
func (a *ModelActor[T]) Status() StatusInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return StatusInfo{
		ActorID:      a.actorID,
		Module:       a.module,
		DeviceID:     a.deviceID,
		Status:       a.status,
		LastError:    a.lastError,
		LastActivity: a.lastActivity,
		MessageCount: atomic.LoadInt64(&a.messageCount),
		CurrentVersion: a.versionMgr.CurrentVersion(),
		CurrentChecksum: a.versionMgr.CurrentChecksum(),
		Uptime:        time.Since(a.startTime),
	}
}

// runMessageLoop processes incoming messages sequentially.
func (a *ModelActor[T]) runMessageLoop() {
	defer a.wg.Done()

	for {
		select {
		case msgWP := <-a.msgChan:
			result := a.processMessage(msgWP.msg)
			msgWP.promise <- result
			close(msgWP.promise)

		case <-a.ctx.Done():
			return
		}
	}
}

// processMessage routes messages to appropriate handlers.
func (a *ModelActor[T]) processMessage(msg Message) Result {
	a.mu.Lock()
	a.status = StatusRunning
	atomic.AddInt64(&a.messageCount, 1)
	a.lastActivity = time.Now()
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.status = StatusReady
		a.mu.Unlock()
	}()

	var result Result
	switch m := msg.(type) {
	case *TranslateCmd:
		result = a.handleTranslate(m)
	case *ValidateCmd:
		result = a.handleValidate(m)
	case *ApplyCmd:
		result = a.handleApply(m)
	case *PrepareCmd:
		result = a.handlePrepare(m)
	case *CommitCmd:
		result = a.handleCommit(m)
	case *RollbackCmd:
		result = a.handleRollback(m)
	case *StatusQueryCmd:
		result = a.handleStatusQuery(m)
	default:
		result = Failure(msg.ID(), fmt.Errorf("unknown message type: %T", msg))
	}

	if result.Error != nil {
		a.mu.Lock()
		a.lastError = result.Error
		a.mu.Unlock()
	}

	return result
}

// handleTranslate processes a TranslateCmd message.
func (a *ModelActor[T]) handleTranslate(cmd *TranslateCmd) Result {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.translator.Translate(cmd.Payload, a.desired); err != nil {
		return Failure(cmd.ID(), fmt.Errorf("translation failed: %w", err))
	}

	// Validate the translated configuration
	if err := a.desired.Validate(); err != nil {
		return Failure(cmd.ID(), fmt.Errorf("validation failed: %w", err))
	}

	// Create version snapshot
	version, err := a.versionMgr.CreateSnapshot(a.desired, "CR Spec translated", a.actorID)
	if err != nil {
		return Failure(cmd.ID(), fmt.Errorf("failed to create snapshot: %w", err))
	}

	return Result{
		MsgID:    cmd.ID(),
		Success:  true,
		Version:  version.Number,
		Checksum: version.Checksum,
		Data: map[string]interface{}{
			"path":    cmd.Path,
			"op":      cmd.Operation,
			"message": "translation successful",
		},
	}
}

// handleValidate processes a ValidateCmd message.
func (a *ModelActor[T]) handleValidate(cmd *ValidateCmd) Result {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if err := a.desired.Validate(); err != nil {
		return Failure(cmd.ID(), fmt.Errorf("validation failed: %w", err))
	}

	return Success(cmd.ID())
}

// handleApply processes an ApplyCmd message (direct apply without 2PC).
func (a *ModelActor[T]) handleApply(cmd *ApplyCmd) Result {
	a.mu.Lock()
	defer a.mu.Unlock()

	// 1. Fetch actual config from device
	actual, err := a.fetchActualFromDevice(cmd.Context())
	if err != nil {
		return Failure(cmd.ID(), fmt.Errorf("failed to fetch actual config: %w", err))
	}
	a.actual = actual

	// 2. Compute diff between desired and actual
	diffEngine := diff.NewDefaultDiffEngine()
	diffResult, err := diffEngine.Diff(a.desired, a.actual, nil)
	if err != nil {
		return Failure(cmd.ID(), fmt.Errorf("failed to compute diff: %w", err))
	}

	// No changes needed
	if diffResult.Summary.Total == 0 {
		return Success(cmd.ID())
	}

	// 3. Convert diff changes to client changes
	clientChanges := a.convertDiffToClientChanges(diffResult.Changes)

	// 4. Apply changes to device
	if err := a.applyChangesToDevice(cmd.Context(), clientChanges); err != nil {
		return Failure(cmd.ID(), fmt.Errorf("failed to apply changes: %w", err))
	}

	// 5. Verify by refetching actual config from device
	verifiedActual, err := a.fetchActualFromDevice(cmd.Context())
	if err != nil {
		return Failure(cmd.ID(), fmt.Errorf("failed to verify changes: %w", err))
	}
	a.actual = verifiedActual

	// 6. Create version snapshot of the new desired state
	version, err := a.versionMgr.CreateSnapshot(a.desired, "Configuration applied", a.actorID)
	if err != nil {
		return Failure(cmd.ID(), fmt.Errorf("failed to create snapshot: %w", err))
	}

	return Result{
		MsgID:    cmd.ID(),
		Success:  true,
		Version:  version.Number,
		Checksum: version.Checksum,
		Data: map[string]interface{}{
			"changes_applied": diffResult.Summary.Total,
			"adds":            diffResult.Summary.Adds,
			"deletes":         diffResult.Summary.Deletes,
			"modifies":        diffResult.Summary.Modifies,
		},
	}
}

// fetchActualFromDevice retrieves the current configuration from the device.
func (a *ModelActor[T]) fetchActualFromDevice(ctx context.Context) (T, error) {
	var zero T

	// Get device client using device IP as identifier
	// TODO: In real implementation, get connection info from device registry
	info := client.DeviceConnectionInfo{
		IP:       a.deviceID,
		Protocol: client.ProtocolNETCONF,
	}

	deviceClient, err := a.clientPool.Get(info)
	if err != nil {
		return zero, err
	}

	// Fetch config from device
	result, err := deviceClient.Get(ctx, a.module)
	if err != nil {
		return zero, err
	}

	// Convert result to YANG struct type
	// The result.Data should be compatible with T
	converted, ok := result.Data.(T)
	if !ok {
		// Try to deep copy via JSON serialization for type compatibility
		dataBytes, err := json.Marshal(result.Data)
		if err != nil {
			return zero, err
		}

		var target T
		if err := json.Unmarshal(dataBytes, &target); err != nil {
			return zero, err
		}
		return target, nil
	}

	return converted, nil
}

// convertDiffToClientChanges converts diff.Change slice to client.Change slice.
func (a *ModelActor[T]) convertDiffToClientChanges(diffChanges []diff.Change) []client.Change {
	clientChanges := make([]client.Change, len(diffChanges))
	for i, dc := range diffChanges {
		var changeType client.ChangeType
		switch dc.Type {
		case diff.AddChange:
			changeType = client.AddChange
		case diff.DeleteChange:
			changeType = client.DeleteChange
		case diff.ModifyChange:
			changeType = client.ModifyChange
		}

		clientChanges[i] = client.Change{
			Type:       changeType,
			Path:       dc.Path,
			OldValue:   dc.OldValue,
			NewValue:   dc.NewValue,
			SchemaPath: dc.SchemaPath,
		}
	}
	return clientChanges
}

// applyChangesToDevice applies the configuration changes to the device.
func (a *ModelActor[T]) applyChangesToDevice(ctx context.Context, changes []client.Change) error {
	// Get device client
	info := client.DeviceConnectionInfo{
		IP:       a.deviceID,
		Protocol: client.ProtocolNETCONF,
	}

	deviceClient, err := a.clientPool.Get(info)
	if err != nil {
		return err
	}

	// Apply changes with commit
	result, err := deviceClient.Set(ctx, changes, client.WithCommit(true))
	if err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("device returned failure: %s", result.Message)
	}

	return nil
}

// handlePrepare processes a PrepareCmd message (2PC phase 1).
func (a *ModelActor[T]) handlePrepare(cmd *PrepareCmd) Result {
	a.mu.Lock()
	defer a.mu.Unlock()

	// 1. Validate desired configuration
	if err := a.desired.Validate(); err != nil {
		return Failure(cmd.ID(), fmt.Errorf("desired config validation failed: %w", err))
	}

	// 2. Check device connectivity and fetch current config
	actual, err := a.fetchActualFromDevice(cmd.Context())
	if err != nil {
		return Failure(cmd.ID(), fmt.Errorf("device connectivity check failed: %w", err))
	}
	a.actual = actual

	// 3. Compute diff and store in working state
	diffEngine := diff.NewDefaultDiffEngine()
	diffResult, err := diffEngine.Diff(a.desired, a.actual, nil)
	if err != nil {
		return Failure(cmd.ID(), fmt.Errorf("failed to compute diff: %w", err))
	}

	// Store pending changes in working state for commit phase
	// Deep copy desired to state as the candidate
	stateCopy, err := deepCopy(a.desired)
	if err != nil {
		return Failure(cmd.ID(), fmt.Errorf("failed to copy desired state: %w", err))
	}
	a.state = stateCopy

	// 4. In dry run mode, just return changes without applying them
	if cmd.DryRun {
		return Result{
			MsgID:   cmd.ID(),
			Success: true,
			Data: map[string]interface{}{
				"dry_run":    true,
				"can_commit": diffResult.Summary.Total > 0,
				"changes": map[string]interface{}{
					"total":   diffResult.Summary.Total,
					"adds":    diffResult.Summary.Adds,
					"deletes": diffResult.Summary.Deletes,
					"modifies": diffResult.Summary.Modifies,
				},
			},
		}
	}

	// 5. Create candidate config on device (apply to candidate datastore only)
	clientChanges := a.convertDiffToClientChanges(diffResult.Changes)
	if err := a.prepareCandidateOnDevice(cmd.Context(), clientChanges); err != nil {
		return Failure(cmd.ID(), fmt.Errorf("failed to prepare candidate config: %w", err))
	}

	return Result{
		MsgID:   cmd.ID(),
		Success: true,
		Data: map[string]interface{}{
			"dry_run":    false,
			"can_commit": diffResult.Summary.Total > 0,
			"changes": map[string]interface{}{
				"total":   diffResult.Summary.Total,
				"adds":    diffResult.Summary.Adds,
				"deletes": diffResult.Summary.Deletes,
				"modifies": diffResult.Summary.Modifies,
			},
		},
	}
}

// prepareCandidateOnDevice applies changes to candidate datastore without committing.
func (a *ModelActor[T]) prepareCandidateOnDevice(ctx context.Context, changes []client.Change) error {
	info := client.DeviceConnectionInfo{
		IP:       a.deviceID,
		Protocol: client.ProtocolNETCONF,
	}

	deviceClient, err := a.clientPool.Get(info)
	if err != nil {
		return err
	}

	// Apply to candidate datastore only (no commit)
	result, err := deviceClient.Set(ctx, changes, client.WithCommit(false))
	if err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("candidate preparation failed: %s", result.Message)
	}

	return nil
}

// handleCommit processes a CommitCmd message (2PC phase 2).
func (a *ModelActor[T]) handleCommit(cmd *CommitCmd) Result {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Commit the candidate config on device
	if err := a.commitCandidateOnDevice(cmd.Context(), cmd.ForceCommit); err != nil {
		return Failure(cmd.ID(), fmt.Errorf("commit failed: %w", err))
	}

	// Update actual state from device after commit
	actual, err := a.fetchActualFromDevice(cmd.Context())
	if err != nil {
		return Failure(cmd.ID(), fmt.Errorf("failed to verify commit: %w", err))
	}
	a.actual = actual

	// Create version snapshot
	version, err := a.versionMgr.CreateSnapshot(a.desired, "Configuration committed", a.actorID)
	if err != nil {
		return Failure(cmd.ID(), err)
	}

	return Result{
		MsgID:    cmd.ID(),
		Success:  true,
		Version:  version.Number,
		Checksum: version.Checksum,
		Data: map[string]interface{}{
			"force_commit": cmd.ForceCommit,
			"message":      "commit successful",
		},
	}
}

// commitCandidateOnDevice commits the candidate configuration to running.
func (a *ModelActor[T]) commitCandidateOnDevice(ctx context.Context, force bool) error {
	info := client.DeviceConnectionInfo{
		IP:       a.deviceID,
		Protocol: client.ProtocolNETCONF,
	}

	deviceClient, err := a.clientPool.Get(info)
	if err != nil {
		return err
	}

	// Apply empty changes with commit flag to commit the candidate config
	_, err = deviceClient.Set(ctx, []client.Change{}, client.WithCommit(true))
	return err
}

// handleRollback processes a RollbackCmd message.
func (a *ModelActor[T]) handleRollback(cmd *RollbackCmd) Result {
	a.mu.Lock()
	defer a.mu.Unlock()

	var snapshot Snapshot[T]
	var found bool

	if cmd.TargetChecksum != "" {
		snapshot, found = a.versionMgr.GetByChecksum(cmd.TargetChecksum)
	} else if cmd.TargetVersion > 0 {
		snapshot, found = a.versionMgr.GetByNumber(cmd.TargetVersion)
	} else {
		// Rollback to previous version
		history := a.versionMgr.History()
		if len(history) >= 2 {
			snapshot, found = a.versionMgr.GetByNumber(history[len(history)-2].Number)
		} else {
			return Failure(cmd.ID(), errors.New("no previous version to roll back to"))
		}
	}

	if !found {
		return Failure(cmd.ID(), fmt.Errorf("snapshot not found"))
	}

	// Restore desired state from snapshot
	a.desired = snapshot.State

	// Create new version for this rollback
	newVersion, err := a.versionMgr.CreateSnapshot(
		a.desired,
		fmt.Sprintf("rollback to version %d", snapshot.Number),
		a.actorID,
	)
	if err != nil {
		return Failure(cmd.ID(), err)
	}

	return Result{
		MsgID:    cmd.ID(),
		Success:  true,
		Version:  newVersion.Number,
		Checksum: newVersion.Checksum,
		Data: map[string]interface{}{
			"rolled_back_to": snapshot.Number,
		},
	}
}

// handleStatusQuery processes a StatusQueryCmd message.
func (a *ModelActor[T]) handleStatusQuery(cmd *StatusQueryCmd) Result {
	status := a.Status()

	data := map[string]interface{}{
		"actor_id":       status.ActorID,
		"device_id":      status.DeviceID,
		"status":         string(status.Status),
		"message_count":  status.MessageCount,
		"uptime":         status.Uptime.String(),
		"current_version": status.CurrentVersion,
		"checksum":       status.CurrentChecksum,
	}

	if cmd.IncludeDetails {
		data["history_count"] = len(a.versionMgr.History())
		data["last_activity"] = status.LastActivity
		if status.LastError != nil {
			data["last_error"] = status.LastError.Error()
		}
	}

	return Result{
		MsgID:   cmd.ID(),
		Success: true,
		Data:    data,
	}
}

// Desired returns a copy of the current desired configuration.
func (a *ModelActor[T]) Desired() (T, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return deepCopy(a.desired)
}

// Actual returns a copy of the current actual device configuration.
func (a *ModelActor[T]) Actual() (T, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return deepCopy(a.actual)
}
