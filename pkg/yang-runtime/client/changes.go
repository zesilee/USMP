package client

import "time"

// ChangeType represents the type of configuration change
type ChangeType int

const (
	// AddChange represents adding a new configuration node
	AddChange ChangeType = iota
	// DeleteChange represents deleting an existing configuration node
	DeleteChange
	// ModifyChange represents modifying an existing configuration node
	ModifyChange
)

// String returns the string representation of ChangeType
func (t ChangeType) String() string {
	switch t {
	case AddChange:
		return "ADD"
	case DeleteChange:
		return "DELETE"
	case ModifyChange:
		return "MODIFY"
	default:
		return "UNKNOWN"
	}
}

// Change represents a single configuration change between desired and actual
type Change struct {
	// Type is the change type
	Type ChangeType
	// Path is the YANG path to the changed node
	Path string
	// OldValue is the actual value before the change
	OldValue interface{}
	// NewValue is the desired value after the change
	NewValue interface{}
	// SchemaPath is the path in the schema
	SchemaPath string
}

// GetResult contains the result of a get operation
type GetResult struct {
	// Path is the YANG path that was retrieved
	Path string
	// Data is the retrieved configuration data
	Data interface{}
	// Timestamp is when the data was retrieved
	Timestamp time.Time
	// Error is any error that occurred
	Error error
}

// SetResult contains the result of a set operation
type SetResult struct {
	// Success indicates whether all changes were applied successfully
	Success bool
	// Message provides additional information about the result
	Message string
	// Changes contains the result of each individual change
	Changes []ChangeResult
	// Timestamp is when the operation completed
	Timestamp time.Time
}

// ChangeResult contains the result of a single change
type ChangeResult struct {
	// Change that was attempted
	Change Change
	// Success indicates whether the change was applied successfully
	Success bool
	// Error is any error that occurred
	Error error
}

// Notification represents a notification from the device
type Notification struct {
	// Path is the YANG path the notification relates to
	Path string
	// Data is the notification data
	Data interface{}
	// Timestamp is when the notification was received
	Timestamp time.Time
}
