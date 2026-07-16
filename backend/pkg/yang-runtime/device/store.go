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
	// Put registers/updates the connection info for id. 持久化后端（CRD）写
	// 失败时返回错误（DS-04 写失败可见）；内存实现恒返回 nil。
	Put(id string, info client.DeviceConnectionInfo) error
	// Delete removes id from the registry (no-op if absent)。失败语义同 Put。
	Delete(id string) error
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

func (s *memStore) Put(id string, info client.DeviceConnectionInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.devices[id] = info
	return nil
}

func (s *memStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.devices, id)
	return nil
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
