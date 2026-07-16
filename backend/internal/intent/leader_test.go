package intent

import (
	"context"
	"testing"

	"k8s.io/client-go/rest"

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
	gated := gateSources(&rest.Config{Host: "https://127.0.0.1:1"}, inner)
	if gated == controller.Source(inner) {
		t.Fatal("enabled seam must wrap with leader gate")
	}
	if inner.started {
		t.Fatal("inner sources must not start before leadership")
	}
}

// YR-08 无集群降级：开关开但无集群配置 → 透传（不崩溃，R08）。
func TestGateSourcesEnabledNoClusterPassthrough(t *testing.T) {
	t.Setenv("USMP_INTENT_LEADER_ELECTION", "1")
	inner := &nopSource{}
	if gateSources(nil, inner) != controller.Source(inner) {
		t.Fatal("enabled without cluster config must pass through (YR-08 degrade)")
	}
}
