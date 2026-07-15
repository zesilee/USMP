package intent

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/predicate"
)

// BIO-04 —— 周期 resync 源：把每个意图 CR 重新入队（幂等短路重写 desired，对冲
// desired TTL 过期）。

type fakeCtrl struct {
	mu     sync.Mutex
	events []predicate.Event
}

func (f *fakeCtrl) Start(context.Context) error { return nil }
func (f *fakeCtrl) Stop() error                 { return nil }
func (f *fakeCtrl) Name() string                { return "fake" }
func (f *fakeCtrl) Enqueue(evt predicate.Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, evt)
}
func (f *fakeCtrl) snapshot() []predicate.Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]predicate.Event{}, f.events...)
}

func TestResyncSourceEnqueuesAllCRs(t *testing.T) {
	cr2 := newCR(1, validSpec())
	cr2.SetName("biz-200")
	cl := newFakeClient(t, newCR(1, validSpec()), cr2)

	src := NewResyncSource(cl, 20*time.Millisecond)
	ctrl := &fakeCtrl{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := src.Start(ctx, ctrl); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = src.Stop() }()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		evts := ctrl.snapshot()
		seen := map[string]bool{}
		for _, e := range evts {
			if e.Path != IntentPath || e.Type != predicate.UpdateEvent {
				t.Fatalf("unexpected event %+v", e)
			}
			seen[e.DeviceID] = true
		}
		if seen["default/biz-100"] && seen["default/biz-200"] {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("resync did not enqueue both CRs, got %+v", ctrl.snapshot())
}

func TestResyncSourceStopsCleanly(t *testing.T) {
	cl := newFakeClient(t)
	src := NewResyncSource(cl, 10*time.Millisecond)
	ctrl := &fakeCtrl{}
	ctx := context.Background()
	if err := src.Start(ctx, ctrl); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := src.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
	time.Sleep(30 * time.Millisecond) // 停止后不再滴答（无 panic/竞态，-race 兜底）
}
