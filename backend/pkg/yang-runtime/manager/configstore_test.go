package manager

import (
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/internal/cache"
)

func newTestStore() *InMemoryConfigStore {
	return NewInMemoryConfigStore(cache.NewTTLLRUCache(1000, 30*time.Second, 0))
}

// TestConfigStoreListPaths verifies List returns the paths stored for a device,
// excluding other devices' paths (D7).
func TestConfigStoreListPaths(t *testing.T) {
	s := newTestStore()
	_ = s.Set("10.0.0.1", "huawei-vlan:vlan/vlans", "v1")
	_ = s.Set("10.0.0.1", "huawei-ifm:ifm/interfaces", "v2")
	_ = s.Set("10.0.0.2", "huawei-system:system", "v3")

	got, err := s.List("10.0.0.1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	sort.Strings(got)
	want := []string{"huawei-ifm:ifm/interfaces", "huawei-vlan:vlan/vlans"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("List(10.0.0.1) = %v, want %v", got, want)
	}
}

// TestConfigStoreListEmpty verifies List/ListDevices on an empty store return
// empty (not nil error).
func TestConfigStoreListEmpty(t *testing.T) {
	s := newTestStore()
	if paths, err := s.List("nope"); err != nil || len(paths) != 0 {
		t.Fatalf("List empty = %v, %v; want empty,nil", paths, err)
	}
	if devs, err := s.ListDevices(); err != nil || len(devs) != 0 {
		t.Fatalf("ListDevices empty = %v, %v; want empty,nil", devs, err)
	}
}

// TestConfigStoreListDevices verifies ListDevices returns the distinct device set.
func TestConfigStoreListDevices(t *testing.T) {
	s := newTestStore()
	_ = s.Set("10.0.0.1", "huawei-vlan:vlan/vlans", "v1")
	_ = s.Set("10.0.0.1", "huawei-ifm:ifm/interfaces", "v2") // same device, 2nd path
	_ = s.Set("10.0.0.2", "huawei-system:system", "v3")

	got, err := s.ListDevices()
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	sort.Strings(got)
	want := []string{"10.0.0.1", "10.0.0.2"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("ListDevices = %v, want %v (distinct)", got, want)
	}
}

// TestConfigStoreConcurrent exercises List/ListDevices/Set under -race.
func TestConfigStoreConcurrent(t *testing.T) {
	s := newTestStore()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(3)
		go func() { defer wg.Done(); _ = s.Set("10.0.0.1", "p", "v") }()
		go func() { defer wg.Done(); _, _ = s.List("10.0.0.1") }()
		go func() { defer wg.Done(); _, _ = s.ListDevices() }()
	}
	wg.Wait()
}
