package device

import (
	"fmt"
	"sync"
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
)

func TestStore_PutGet_FullInfo(t *testing.T) {
	s := NewStore()
	info := client.DeviceConnectionInfo{
		IP: "192.168.1.1", Port: 830, Username: "admin", Password: "admin", Protocol: client.ProtocolAUTO,
	}
	s.Put("192.168.1.1", info)

	got, ok := s.Get("192.168.1.1")
	if !ok {
		t.Fatal("registered device must be present")
	}
	if got.Port != 830 || got.Username != "admin" || got.Password != "admin" || got.Protocol != client.ProtocolAUTO {
		t.Fatalf("connection info must round-trip complete, got %+v", got)
	}
}

func TestStore_Get_Miss(t *testing.T) {
	s := NewStore()
	if _, ok := s.Get("10.0.0.9"); ok {
		t.Fatal("unregistered device must return ok=false")
	}
}

func TestStore_Delete(t *testing.T) {
	s := NewStore()
	s.Put("a", client.DeviceConnectionInfo{IP: "a"})
	s.Delete("a")
	if _, ok := s.Get("a"); ok {
		t.Fatal("deleted device must be gone")
	}
}

func TestStore_List(t *testing.T) {
	s := NewStore()
	s.Put("a", client.DeviceConnectionInfo{IP: "a"})
	s.Put("b", client.DeviceConnectionInfo{IP: "b"})
	s.Put("a", client.DeviceConnectionInfo{IP: "a"}) // overwrite, still one key
	if got := s.List(); len(got) != 2 {
		t.Fatalf("want 2 unique devices, got %d (%v)", len(got), got)
	}
}

// TestStore_ConcurrentSafe exercises R09: concurrent Put/Get/Delete/List must
// not race (run with -race).
func TestStore_ConcurrentSafe(t *testing.T) {
	s := NewStore()
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("d%d", i%8)
			s.Put(id, client.DeviceConnectionInfo{IP: id, Port: 830})
			_, _ = s.Get(id)
			_ = s.List()
			if i%3 == 0 {
				s.Delete(id)
			}
		}(i)
	}
	wg.Wait()
}
