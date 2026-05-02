package actor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Version represents a specific configuration version with metadata.
type Version struct {
	Number    int64     `json:"number"`     // Monotonically increasing version number
	Checksum  string    `json:"checksum"`   // SHA256 checksum of the serialized state
	CreatedAt time.Time `json:"created_at"` // When this version was created
	CreatedBy string    `json:"created_by"` // Actor/module that created this version
	Message   string    `json:"message"`    // Optional commit message
}

// Versioned represents a state with version information.
type Versioned struct {
	CurrentVersion Version `json:"current_version"`
}

// Snapshot represents a complete state snapshot for rollback.
type Snapshot[T YANGGoStruct] struct {
	Version
	State T `json:"state"`
}

// VersionManager manages version history and rollbacks for YANG state.
type VersionManager[T YANGGoStruct] struct {
	mu          sync.RWMutex
	history     []Snapshot[T]
	maxHistory  int
	nextVersion int64
}

// NewVersionManager creates a new VersionManager with the specified history limit.
func NewVersionManager[T YANGGoStruct](maxHistory int) *VersionManager[T] {
	if maxHistory <= 0 {
		maxHistory = 50 // Default history size
	}
	return &VersionManager[T]{
		history:     make([]Snapshot[T], 0, maxHistory),
		maxHistory:  maxHistory,
		nextVersion: 1,
	}
}

// CreateSnapshot creates a new version snapshot of the given state.
func (vm *VersionManager[T]) CreateSnapshot(state T, message string, creator string) (Version, error) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	// Compute checksum of the state
	checksum, err := computeChecksum(state)
	if err != nil {
		return Version{}, fmt.Errorf("failed to compute state checksum: %w", err)
	}

	// Create deep copy of the state for the snapshot
	stateCopy, err := deepCopy(state)
	if err != nil {
		return Version{}, fmt.Errorf("failed to copy state: %w", err)
	}

	version := Version{
		Number:    vm.nextVersion,
		Checksum:  checksum,
		CreatedAt: time.Now(),
		CreatedBy: creator,
		Message:   message,
	}

	snapshot := Snapshot[T]{
		Version: version,
		State:   stateCopy,
	}

	// Add to history, trimming if needed
	vm.history = append(vm.history, snapshot)
	if len(vm.history) > vm.maxHistory {
		vm.history = vm.history[1:]
	}

	vm.nextVersion++
	return version, nil
}

// GetLatest returns the most recent snapshot.
func (vm *VersionManager[T]) GetLatest() (Snapshot[T], bool) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	if len(vm.history) == 0 {
		return Snapshot[T]{}, false
	}
	return vm.history[len(vm.history)-1], true
}

// GetByNumber returns the snapshot with the given version number.
func (vm *VersionManager[T]) GetByNumber(versionNum int64) (Snapshot[T], bool) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	for _, snap := range vm.history {
		if snap.Number == versionNum {
			return snap, true
		}
	}
	return Snapshot[T]{}, false
}

// GetByChecksum returns the snapshot with the given checksum.
func (vm *VersionManager[T]) GetByChecksum(checksum string) (Snapshot[T], bool) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	for _, snap := range vm.history {
		if snap.Checksum == checksum {
			return snap, true
		}
	}
	return Snapshot[T]{}, false
}

// RollbackToVersion rolls back to the specified version number.
// Returns the state snapshot for that version.
func (vm *VersionManager[T]) RollbackToVersion(versionNum int64) (Snapshot[T], error) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	for i, snap := range vm.history {
		if snap.Number == versionNum {
			// Trim history after this version
			vm.history = vm.history[:i+1]
			return snap, nil
		}
	}
	return Snapshot[T]{}, fmt.Errorf("version %d not found in history", versionNum)
}

// History returns all version history entries.
func (vm *VersionManager[T]) History() []Version {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	result := make([]Version, len(vm.history))
	for i, snap := range vm.history {
		result[i] = snap.Version
	}
	return result
}

// CurrentVersion returns the latest version number.
func (vm *VersionManager[T]) CurrentVersion() int64 {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	if len(vm.history) == 0 {
		return 0
	}
	return vm.history[len(vm.history)-1].Number
}

// CurrentChecksum returns the latest state checksum.
func (vm *VersionManager[T]) CurrentChecksum() string {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	if len(vm.history) == 0 {
		return ""
	}
	return vm.history[len(vm.history)-1].Checksum
}

// ClearHistory removes all snapshots except the most recent one.
func (vm *VersionManager[T]) ClearHistory() {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if len(vm.history) > 1 {
		vm.history = vm.history[len(vm.history)-1:]
	}
}

// computeChecksum computes SHA256 checksum of the JSON-serialized state.
func computeChecksum[T YANGGoStruct](state T) (string, error) {
	data, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// deepCopy creates a deep copy of the state using JSON serialization.
func deepCopy[T YANGGoStruct](state T) (T, error) {
	data, err := json.Marshal(state)
	if err != nil {
		var zero T
		return zero, err
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		var zero T
		return zero, err
	}
	return result, nil
}

// ValidateChecksum verifies that the state matches the expected checksum.
func ValidateChecksum[T YANGGoStruct](state T, expectedChecksum string) (bool, error) {
	actual, err := computeChecksum(state)
	if err != nil {
		return false, err
	}
	return actual == expectedChecksum, nil
}
