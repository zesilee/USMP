package controllers

import (
	"context"
	"errors"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	bizv1 "github.com/leezesi/usmp/backend/api/v1"
	netconfclient "github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
)

// fakeClient / fakePool exercise probeDevice without a real device or Actor.
type fakeClient struct{ connected bool }

func (f *fakeClient) Get(context.Context, string, ...netconfclient.GetOption) (*netconfclient.GetResult, error) {
	return nil, nil
}
func (f *fakeClient) Set(context.Context, []netconfclient.Change, ...netconfclient.SetOption) (*netconfclient.SetResult, error) {
	return nil, nil
}
func (f *fakeClient) Subscribe(context.Context, string, func(netconfclient.Notification)) error {
	return nil
}
func (f *fakeClient) Close() error                           { return nil }
func (f *fakeClient) IsConnected() bool                      { return f.connected }
func (f *fakeClient) DiscardCandidate(context.Context) error { return nil }

type fakePool struct {
	client netconfclient.Client
	err    error
}

func (p *fakePool) Get(netconfclient.DeviceConnectionInfo) (netconfclient.Client, error) {
	return p.client, p.err
}
func (p *fakePool) Release(string)                 {}
func (p *fakePool) CloseAll() error                { return nil }
func (p *fakePool) Stats() netconfclient.PoolStats { return netconfclient.PoolStats{} }

func switchCR() *bizv1.BusinessSwitch {
	return &bizv1.BusinessSwitch{
		ObjectMeta: metav1.ObjectMeta{Name: "sw-1"},
		Spec: bizv1.BusinessSwitchSpec{
			DeviceIP:    "10.0.0.1",
			Port:        830,
			Credentials: bizv1.Credentials{Username: "u", Password: "p"},
		},
	}
}

// TestProbeDeviceOnline: a connected client → online (no error, no Actor).
func TestProbeDeviceOnline(t *testing.T) {
	r := &BusinessSwitchReconciler{ClientPool: &fakePool{client: &fakeClient{connected: true}}}
	if err := r.probeDevice(context.Background(), switchCR()); err != nil {
		t.Fatalf("online device should probe ok: %v", err)
	}
}

// TestProbeDeviceConnectError: ClientPool.Get error → offline (R08 graceful).
func TestProbeDeviceConnectError(t *testing.T) {
	r := &BusinessSwitchReconciler{ClientPool: &fakePool{err: errors.New("connection refused")}}
	if err := r.probeDevice(context.Background(), switchCR()); err == nil {
		t.Fatal("connect error should return offline error")
	}
}

// TestProbeDeviceNotConnected: a disconnected client → offline.
func TestProbeDeviceNotConnected(t *testing.T) {
	r := &BusinessSwitchReconciler{ClientPool: &fakePool{client: &fakeClient{connected: false}}}
	if err := r.probeDevice(context.Background(), switchCR()); err == nil {
		t.Fatal("disconnected client should return error")
	}
}

// TestProbeDeviceDefaultPort: Port=0 defaults to 830 (no panic).
func TestProbeDeviceDefaultPort(t *testing.T) {
	cr := switchCR()
	cr.Spec.Port = 0
	r := &BusinessSwitchReconciler{ClientPool: &fakePool{client: &fakeClient{connected: true}}}
	if err := r.probeDevice(context.Background(), cr); err != nil {
		t.Fatalf("default port should probe ok: %v", err)
	}
}
