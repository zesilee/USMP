// Package device provides the shared, in-memory registry of device connection
// information (the single source of truth used by reconcilers, the config-read
// path and the periodic source to build device connections). No database (R03);
// concurrency-safe (R09).
package device

import (
	"sync"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
)

// Store is the single source of truth for device connection info, keyed by
// device ID (the bare IP, matching the ConfigStore desired key).
type Store interface {
	// Get returns the connection info for id, or ok=false if not registered.
	Get(id string) (client.DeviceConnectionInfo, bool)
	// Put registers/updates the connection info for id.
	Put(id string, info client.DeviceConnectionInfo)
	// Delete removes id from the registry (no-op if absent).
	Delete(id string)
	// List returns all registered device IDs.
	List() []string
}

// memStore is the default in-memory, concurrency-safe Store.
type memStore struct {
	mu      sync.RWMutex
	devices map[string]client.DeviceConnectionInfo
}

// NewStore creates an empty in-memory device store.
func NewStore() Store {
	return &memStore{devices: make(map[string]client.DeviceConnectionInfo)}
}

func (s *memStore) Get(id string) (client.DeviceConnectionInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	info, ok := s.devices[id]
	return info, ok
}

func (s *memStore) Put(id string, info client.DeviceConnectionInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.devices[id] = info
}

func (s *memStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.devices, id)
}

func (s *memStore) List() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.devices))
	for id := range s.devices {
		ids = append(ids, id)
	}
	return ids
}
