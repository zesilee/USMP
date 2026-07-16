package leader

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// YR-08 envtest 面：真 apiserver Lease——双副本仅 leader 启源、失主停源、
// 另一副本接管；不同 Lease 互不干扰。

var (
	sharedEnv *envtest.Environment
	sharedCfg *rest.Config
)

func TestMain(m *testing.M) {
	code := m.Run()
	if sharedEnv != nil {
		_ = sharedEnv.Stop()
	}
	os.Exit(code)
}

func envtestConfig(t *testing.T) *rest.Config {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping envtest integration in short mode")
	}
	if sharedCfg != nil {
		return sharedCfg
	}
	assets := os.Getenv("KUBEBUILDER_ASSETS")
	if assets == "" {
		home, _ := os.UserHomeDir()
		matches, _ := filepath.Glob(filepath.Join(home, ".local/share/kubebuilder-envtest/k8s/*"))
		if len(matches) == 0 {
			t.Skip("envtest binaries not installed")
		}
		assets = matches[0]
	}
	env := &envtest.Environment{BinaryAssetsDirectory: assets}
	cfg, err := env.Start()
	if err != nil {
		t.Fatalf("start envtest: %v", err)
	}
	sharedEnv, sharedCfg = env, cfg
	return cfg
}

// fastOpts 用短 Lease 周期让选主/接管在秒级完成（pre-commit 30s 门禁内）。
func fastOpts(lease, id string) Options {
	return Options{
		LeaseName: lease, Namespace: "default", Identity: id,
		LeaseDuration: 1 * time.Second, RenewDeadline: 800 * time.Millisecond, RetryPeriod: 200 * time.Millisecond,
	}
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", what)
}

// 双副本同一 Lease：仅 leader 副本的全部源启动；杀 leader 后另一副本接管。
func TestSingleLeaderAndFailover_Integration(t *testing.T) {
	cfg := envtestConfig(t)

	gateA := NewGate(cfg, fastOpts("gate-failover", "replica-a"))
	gateB := NewGate(cfg, fastOpts("gate-failover", "replica-b"))
	a1, a2 := &fakeSource{}, &fakeSource{}
	b1, b2 := &fakeSource{}, &fakeSource{}

	ctxA, cancelA := context.WithCancel(context.Background())
	defer cancelA()
	ctxB, cancelB := context.WithCancel(context.Background())
	defer cancelB()

	for _, s := range []struct {
		gate *Gate
		ctx  context.Context
		srcs []*fakeSource
	}{{gateA, ctxA, []*fakeSource{a1, a2}}, {gateB, ctxB, []*fakeSource{b1, b2}}} {
		for _, inner := range s.srcs {
			if err := s.gate.Wrap(inner).Start(s.ctx, nil); err != nil {
				t.Fatalf("start gated source: %v", err)
			}
		}
	}

	// 恰好一个副本的全部源启动（多源共享一次选主）
	waitFor(t, "exactly one leader", func() bool {
		a, b := a1.isStarted() && a2.isStarted(), b1.isStarted() && b2.isStarted()
		return (a || b) && !(a && b)
	})

	// 杀 leader → 另一副本接管，原 leader 源已停
	var loser1, loser2 *fakeSource
	if a1.isStarted() {
		cancelA()
		loser1, loser2 = b1, b2
	} else {
		cancelB()
		loser1, loser2 = a1, a2
	}
	waitFor(t, "failover to the other replica", func() bool {
		return loser1.isStarted() && loser2.isStarted()
	})
}

// 不同 Lease 的两个 gate 互不干扰：各自都能成为自己 Lease 的 leader。
func TestIndependentLeases_Integration(t *testing.T) {
	cfg := envtestConfig(t)

	gateX := NewGate(cfg, fastOpts("gate-x", "only-x"))
	gateY := NewGate(cfg, fastOpts("gate-y", "only-y"))
	x, y := &fakeSource{}, &fakeSource{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := gateX.Wrap(x).Start(ctx, nil); err != nil {
		t.Fatalf("start x: %v", err)
	}
	if err := gateY.Wrap(y).Start(ctx, nil); err != nil {
		t.Fatalf("start y: %v", err)
	}
	waitFor(t, "both independent leaders", func() bool {
		return x.isStarted() && y.isStarted()
	})
}

// 晚注册的源在已持有 Lease 时立即启动（controller 注册顺序无关性）。
func TestLateWrapStartsImmediatelyWhenLeading_Integration(t *testing.T) {
	cfg := envtestConfig(t)

	gate := NewGate(cfg, fastOpts("gate-late", "solo"))
	first := &fakeSource{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := gate.Wrap(first).Start(ctx, nil); err != nil {
		t.Fatalf("start first: %v", err)
	}
	waitFor(t, "first source started", first.isStarted)

	late := &fakeSource{}
	if err := gate.Wrap(late).Start(ctx, nil); err != nil {
		t.Fatalf("start late: %v", err)
	}
	waitFor(t, "late source started under existing leadership", late.isStarted)
}
