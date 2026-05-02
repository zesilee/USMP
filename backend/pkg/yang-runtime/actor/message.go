package actor

import (
	"context"
	"time"
)

// MessageType represents the type of message sent to an actor.
type MessageType string

const (
	// MsgTranslate requests translating CR Spec to YANG configuration.
	MsgTranslate MessageType = "translate"
	// MsgValidate requests validation of the configuration.
	MsgValidate MessageType = "validate"
	// MsgPrepare requests a dry run of configuration application (2PC phase 1).
	MsgPrepare MessageType = "prepare"
	// MsgCommit requests committing the prepared configuration (2PC phase 2).
	MsgCommit MessageType = "commit"
	// MsgAbort requests aborting the 2PC transaction (discard candidate config).
	MsgAbort MessageType = "abort"
	// MsgRollback requests rolling back to a previous configuration version.
	MsgRollback MessageType = "rollback"
	// MsgApply requests applying configuration directly (no 2PC).
	MsgApply MessageType = "apply"
	// MsgStatusQuery queries the current actor state.
	MsgStatusQuery MessageType = "status"
)

// Message is the base interface for all actor messages.
type Message interface {
	// Type returns the type of this message.
	Type() MessageType
	// Context returns the context associated with this message.
	Context() context.Context
	// ID returns the unique identifier of this message.
	ID() string
}

// BaseMessage provides common fields for all message types.
type BaseMessage struct {
	msgID   string
	msgType MessageType
	ctx     context.Context
}

// NewBaseMessage creates a new base message with the given ID and type.
func NewBaseMessage(msgID string, msgType MessageType) BaseMessage {
	return BaseMessage{
		msgID:   msgID,
		msgType: msgType,
		ctx:     context.Background(),
	}
}

// NewBaseMessageWithContext creates a new base message with a custom context.
func NewBaseMessageWithContext(msgID string, msgType MessageType, ctx context.Context) BaseMessage {
	return BaseMessage{
		msgID:   msgID,
		msgType: msgType,
		ctx:     ctx,
	}
}

// Type returns the message type.
func (m BaseMessage) Type() MessageType {
	return m.msgType
}

// Context returns the message context.
func (m BaseMessage) Context() context.Context {
	return m.ctx
}

// ID returns the message ID.
func (m BaseMessage) ID() string {
	return m.msgID
}

// TranslateCmd requests translating CR Spec to YANG configuration.
type TranslateCmd struct {
	BaseMessage
	// Path is the YANG schema path for this configuration (optional).
	Path string
	// Payload contains the raw CR Spec data to translate.
	Payload map[string]interface{}
	// Operation specifies how to apply the payload.
	Operation OperationType
}

// OperationType defines how a configuration payload should be applied.
type OperationType string

const (
	// OperationMerge merges the payload with existing configuration.
	OperationMerge OperationType = "merge"
	// OperationReplace replaces existing configuration with the payload.
	OperationReplace OperationType = "replace"
	// OperationDelete deletes the configuration at the specified path.
	OperationDelete OperationType = "delete"
)

// ValidateCmd requests validation of the current desired configuration.
type ValidateCmd struct {
	BaseMessage
}

// PrepareCmd requests a dry run validation of the configuration change (2PC phase 1).
type PrepareCmd struct {
	BaseMessage
	// DryRun if true, only validates without changing any state.
	DryRun bool
}

// CommitCmd requests committing the prepared configuration (2PC phase 2).
type CommitCmd struct {
	BaseMessage
	// ForceCommit forces committing even if some preconditions fail (use with caution).
	ForceCommit bool
}

// RollbackCmd requests rolling back to a previous configuration version.
type RollbackCmd struct {
	BaseMessage
	// TargetVersion is the version number to roll back to (0 = previous version).
	TargetVersion int64
	// TargetChecksum is the SHA256 checksum of the target state (optional, for verification).
	TargetChecksum string
}

// ApplyCmd requests applying configuration directly without 2PC.
type ApplyCmd struct {
	BaseMessage
	// ForceApply bypasses validation checks (use with caution).
	ForceApply bool
}

// StatusQueryCmd requests the actor's current status.
type StatusQueryCmd struct {
	BaseMessage
	// IncludeDetails if true returns additional state details (may be expensive).
	IncludeDetails bool
}

// AbortCmd requests aborting the 2PC transaction and discarding candidate config.
type AbortCmd struct {
	BaseMessage
	// Reason is optional human-readable reason for the abort.
	Reason string
}

// Result represents the outcome of processing a message.
type Result struct {
	// MsgID is the ID of the message this result corresponds to.
	MsgID string
	// Success indicates whether the operation succeeded.
	Success bool
	// Error contains the error if the operation failed.
	Error error
	// Data contains operation-specific result data.
	Data map[string]interface{}
	// Version is the new configuration version after this operation (if applicable).
	Version int64
	// Checksum is the SHA256 checksum of the new state (if applicable).
	Checksum string
}

// ResultPromise is a channel for receiving results asynchronously.
type ResultPromise chan Result

// NewResultPromise creates a new buffered result promise channel.
func NewResultPromise() ResultPromise {
	return make(chan Result, 1)
}

// Success creates a successful result.
func Success(msgID string) Result {
	return Result{
		MsgID:   msgID,
		Success: true,
		Data:    make(map[string]interface{}),
	}
}

// SuccessWithData creates a successful result with additional data.
func SuccessWithData(msgID string, data map[string]interface{}) Result {
	return Result{
		MsgID:   msgID,
		Success: true,
		Data:    data,
	}
}

// Failure creates a failed result with an error.
func Failure(msgID string, err error) Result {
	return Result{
		MsgID:   msgID,
		Success: false,
		Error:   err,
		Data:    make(map[string]interface{}),
	}
}

// ActorStatus represents the current state of an actor.
type ActorStatus string

const (
	// StatusInitializing indicates the actor is initializing.
	StatusInitializing ActorStatus = "initializing"
	// StatusReady indicates the actor is ready to process messages.
	StatusReady ActorStatus = "ready"
	// StatusRunning indicates the actor is processing a message.
	StatusRunning ActorStatus = "running"
	// StatusFailed indicates the actor encountered a non-recoverable error.
	StatusFailed ActorStatus = "failed"
	// StatusStopped indicates the actor has been stopped gracefully.
	StatusStopped ActorStatus = "stopped"
)

// StatusInfo contains comprehensive actor status information.
type StatusInfo struct {
	ActorID      string        `json:"actor_id"`
	Module       string        `json:"module"`
	DeviceID     string        `json:"device_id"`
	Status       ActorStatus   `json:"status"`
	LastError    error         `json:"last_error,omitempty"`
	LastActivity time.Time     `json:"last_activity"`
	MessageCount int64         `json:"message_count"`
	CurrentVersion int64      `json:"current_version"`
	CurrentChecksum string     `json:"current_checksum"`
	Uptime        time.Duration `json:"uptime"`
}
