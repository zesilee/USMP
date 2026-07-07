package api

import (
	"context"
	"errors"
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
)

// fakePoolManager is a Manager test double that serves a custom ClientPool and
// DeviceStore so tests can inspect the DeviceConnectionInfo passed when reading
// device config.
type fakePoolManager struct {
	manager.Manager
	pool  client.ClientPool
	store device.Store
}

func (m fakePoolManager) GetClientPool() client.ClientPool { return m.pool }
func (m fakePoolManager) GetDeviceStore() device.Store     { return m.store }

// TestFetchFromDevice_ResolvesFromDeviceStore: for a registered device the read
// path must build the connection from the shared DeviceStore (port + credentials
// + protocol), not just an AUTO/no-cred stub — otherwise force_refresh readback
// can only auth by luck (pool cache hit).
func TestFetchFromDevice_ResolvesFromDeviceStore(t *testing.T) {
	ds := device.NewStore()
	ds.Put("192.168.1.1", client.DeviceConnectionInfo{
		IP: "192.168.1.1", Port: 830, Username: "admin", Password: "admin", Protocol: client.ProtocolNETCONF,
	})
	p := &fakePool{err: errors.New("stop after capture")}
	h := NewConfigHandler(fakePoolManager{pool: p, store: ds})

	_, _ = h.fetchFromDevice(context.Background(), "192.168.1.1", "/ifm:ifm/ifm:interfaces")

	if p.lastInfo.Username != "admin" || p.lastInfo.Password != "admin" || p.lastInfo.Port != 830 {
		t.Fatalf("fetchFromDevice must resolve creds/port from DeviceStore, got %+v", p.lastInfo)
	}
}

// TestFetchFromDevice_PassesProtocolAuto: the running-config read path must set
// Protocol on the connection info. An empty Protocol hits the factory default
// branch and fails with "unsupported protocol:" (regression: force_refresh 500).
func TestFetchFromDevice_PassesProtocolAuto(t *testing.T) {
	// Return an error so the read short-circuits right after Get captures the info,
	// avoiding a real device round-trip.
	p := &fakePool{err: errors.New("stop after capture")}
	h := NewConfigHandler(fakePoolManager{pool: p})

	_, _ = h.fetchFromDevice(context.Background(), "192.168.1.1", "/ifm:ifm/ifm:interfaces")

	if p.lastInfo.Protocol != client.ProtocolAUTO {
		t.Fatalf("fetchFromDevice must pass Protocol=AUTO, got %q", p.lastInfo.Protocol)
	}
}
