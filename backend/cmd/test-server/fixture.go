package main

import (
	"sort"
	"sync"
)

// VLAN is an in-memory VLAN record served by the E2E REST fixture.
type VLAN struct {
	ID            int
	Name          string
	AdminState    string
	TaggedPorts   []string
	UntaggedPorts []string
}

// vlanStore is a concurrency-safe in-memory VLAN store backing the E2E REST
// fixture. The test server never spoke NETCONF — it only needs somewhere to keep
// VLAN state for the frontend Playwright tests — so this honestly-named store
// replaces the former netsim fake "simulator". The mutex guards concurrent Gin
// request goroutines (R09).
type vlanStore struct {
	mu    sync.RWMutex
	vlans map[int]*VLAN
}

// newVLANStore returns a store seeded with the VLANs the E2E suite expects
// (identical to the seed the previous netsim fixture provided).
func newVLANStore() *vlanStore {
	return &vlanStore{vlans: map[int]*VLAN{
		1:  {ID: 1, Name: "default", AdminState: "UP", UntaggedPorts: []string{"GE0/1", "GE0/2"}},
		10: {ID: 10, Name: "Management", AdminState: "UP", TaggedPorts: []string{"GE0/3"}},
		20: {ID: 20, Name: "User_Network", AdminState: "UP", TaggedPorts: []string{"GE0/4", "GE0/5"}},
		30: {ID: 30, Name: "Guest", AdminState: "DOWN"},
	}}
}

// all returns every VLAN ordered by ID for deterministic responses.
func (s *vlanStore) all() []*VLAN {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]int, 0, len(s.vlans))
	for id := range s.vlans {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	out := make([]*VLAN, 0, len(ids))
	for _, id := range ids {
		out = append(out, s.vlans[id])
	}
	return out
}

// get returns a copy of the VLAN with the given ID, or nil.
func (s *vlanStore) get(id int) *VLAN {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.vlans[id]
	if !ok {
		return nil
	}
	cp := *v
	return &cp
}

// put inserts or replaces a VLAN.
func (s *vlanStore) put(v *VLAN) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vlans[v.ID] = v
}

// remove deletes the VLAN with the given ID.
func (s *vlanStore) remove(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.vlans, id)
}
