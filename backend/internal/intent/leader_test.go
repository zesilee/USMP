package intent

import (
	"context"
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/controller"
)

// BIO-08 接缝：默认关 → 源直通零行为变化；开 → 包一层 Lease 选主门。

type nopSource struct{ started bool }

func (n *nopSource) Start(context.Context, controller.Controller) error {
	n.started = true
	return nil
}
func (n *nopSource) Stop() error { return nil }

func TestGateSourcesDefaultOffPassthrough(t *testing.T) {
	t.Setenv("USMP_INTENT_LEADER_ELECTION", "")
	inner := &nopSource{}
	gated := gateSources(nil, inner)
	if gated != controller.Source(inner) {
		t.Fatal("default-off must pass sources through untouched（单副本零行为变化）")
	}
	if err := gated.Start(context.Background(), nil); err != nil || !inner.started {
		t.Fatalf("passthrough start failed: %v", err)
	}
}

func TestGateSourcesEnabledWraps(t *testing.T) {
	t.Setenv("USMP_INTENT_LEADER_ELECTION", "1")
	inner := &nopSource{}
	gated := gateSources(nil, inner)
	if _, ok := gated.(*leaderGatedSource); !ok {
		t.Fatalf("enabled seam must wrap with leader gate, got %T", gated)
	}
	if inner.started {
		t.Fatal("inner sources must not start before leadership")
	}
}
