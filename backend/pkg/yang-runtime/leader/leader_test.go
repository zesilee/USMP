package leader

import (
	"context"
	"sync"
	"testing"

	"k8s.io/client-go/rest"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/controller"
)

// YR-08 单元面：无集群/未启用时透传（现行为零变化），启用时包裹且
// 未获 Lease 前绝不启动内部源。

type fakeSource struct {
	mu      sync.Mutex
	started bool
	stops   int
}

func (f *fakeSource) Start(context.Context, controller.Controller) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = true
	return nil
}

func (f *fakeSource) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = false
	f.stops++
	return nil
}

func (f *fakeSource) isStarted() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.started
}

func TestWrapNilGatePassthrough(t *testing.T) {
	var g *Gate
	inner := &fakeSource{}
	if g.Wrap(inner) != controller.Source(inner) {
		t.Fatal("nil gate must pass sources through untouched")
	}
}

func TestWrapNoClusterPassthrough(t *testing.T) {
	g := NewGate(nil, Options{LeaseName: "x", Namespace: "default"})
	inner := &fakeSource{}
	if g.Wrap(inner) != controller.Source(inner) {
		t.Fatal("nil rest.Config must pass sources through untouched (YR-08 无集群透传)")
	}
}

func TestWrapGatesBeforeLeadership(t *testing.T) {
	g := NewGate(&rest.Config{Host: "https://127.0.0.1:1"}, Options{LeaseName: "x", Namespace: "default"})
	inner := &fakeSource{}
	gated := g.Wrap(inner)
	if gated == controller.Source(inner) {
		t.Fatal("with cluster config the source must be wrapped")
	}
	if inner.isStarted() {
		t.Fatal("inner source must not start before leadership")
	}
}
