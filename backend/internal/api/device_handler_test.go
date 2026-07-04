package api

import (
	"context"
	"errors"
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
)

// fakeClient / fakePool exercise probeOnline without a real device.
type fakeClient struct{ connected bool }

func (f *fakeClient) Get(context.Context, string, ...client.GetOption) (*client.GetResult, error) {
	return nil, nil
}
func (f *fakeClient) Set(context.Context, []client.Change, ...client.SetOption) (*client.SetResult, error) {
	return nil, nil
}
func (f *fakeClient) Subscribe(context.Context, string, func(client.Notification)) error { return nil }
func (f *fakeClient) Close() error                                                       { return nil }
func (f *fakeClient) IsConnected() bool                                                  { return f.connected }
func (f *fakeClient) DiscardCandidate(context.Context) error                             { return nil }

type fakePool struct {
	client client.Client
	err    error
}

func (p *fakePool) Get(client.DeviceConnectionInfo) (client.Client, error) { return p.client, p.err }
func (p *fakePool) Release(string)                                         {}
func (p *fakePool) CloseAll() error                                        { return nil }
func (p *fakePool) Stats() client.PoolStats                                { return client.PoolStats{} }

// TestProbeOnline: a connected client → online.
func TestProbeOnline(t *testing.T) {
	if !probeOnline(&fakePool{client: &fakeClient{connected: true}}, DeviceInfo{IP: "1.1.1.1", Port: 830}) {
		t.Fatal("reachable device should be online")
	}
}

// TestProbeOfflineOnError: ClientPool.Get error → offline (R08 graceful).
func TestProbeOfflineOnError(t *testing.T) {
	if probeOnline(&fakePool{err: errors.New("connection refused")}, DeviceInfo{IP: "1.1.1.1", Port: 830}) {
		t.Fatal("connect error should be offline")
	}
}

// TestProbeOfflineNotConnected: a disconnected client → offline.
func TestProbeOfflineNotConnected(t *testing.T) {
	if probeOnline(&fakePool{client: &fakeClient{connected: false}}, DeviceInfo{IP: "1.1.1.1", Port: 830}) {
		t.Fatal("disconnected client should be offline")
	}
}

// TestProbeDefaultPort: Port=0 defaults to 830 (no panic, online).
func TestProbeDefaultPort(t *testing.T) {
	if !probeOnline(&fakePool{client: &fakeClient{connected: true}}, DeviceInfo{IP: "1.1.1.1"}) {
		t.Fatal("default-port device should be online")
	}
}
