package manager

import (
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
)

// TestManager_GetDeviceStore: the Manager must expose a usable shared device
// store so reconcilers / config reads / the periodic source have a single source
// of truth for device connection info.
func TestManager_GetDeviceStore(t *testing.T) {
	m := New()
	ds := m.GetDeviceStore()
	if ds == nil {
		t.Fatal("GetDeviceStore must not be nil")
	}
	ds.Put("192.168.1.1", client.DeviceConnectionInfo{IP: "192.168.1.1", Port: 830, Username: "admin", Password: "admin", Protocol: client.ProtocolAUTO})
	got, ok := ds.Get("192.168.1.1")
	if !ok || got.Username != "admin" || got.Port != 830 {
		t.Fatalf("device store must round-trip via Manager, got %+v ok=%v", got, ok)
	}
	// same instance across calls (not a fresh store each time)
	if _, ok := m.GetDeviceStore().Get("192.168.1.1"); !ok {
		t.Fatal("GetDeviceStore must return the same shared instance")
	}
}
