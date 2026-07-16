package crdsource

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
)

// 回归（kind 部署实测崩溃，T07）：集群可达但旧桥接 CRD（biz.usmp.io/v1
// BusinessVlan/BusinessInterface，manifest 从未进 deploy/crds）未安装时，
// RegisterIntentSources 必须跳过旧源而非让 mgr.Start Fatalf（R08 降级）。

func envtestConfig(t *testing.T) *rest.Config {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping envtest integration in short mode")
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
	env := &envtest.Environment{
		BinaryAssetsDirectory: assets,
		// 装 deploy/crds 全部 CRD——与真实集群一致：旧桥接 CRD 不在其中
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "deploy", "crds")},
	}
	cfg, err := env.Start()
	if err != nil {
		t.Fatalf("start envtest: %v", err)
	}
	t.Cleanup(func() { _ = env.Stop() })
	return cfg
}

func TestRegisterSkipsMissingLegacyCRDs_Integration(t *testing.T) {
	cfg := envtestConfig(t)
	mgr := manager.New()

	ca, err := registerIntentSourcesWithConfig(mgr, cfg)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if ca != nil {
		go StartCache(ctx, ca)
	}
	// 崩溃形态：Start 返回 `no matches for kind "BusinessVlan"` → main Fatalf。
	if err := mgr.Start(ctx); err != nil {
		t.Fatalf("mgr.Start must succeed when legacy CRDs are absent (R08 degrade), got: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Stop() })

	// 旧桥接控制器不应注册（无匹配 controller 可触发）
	if mgr.TriggerReconcile("10.0.0.1", "/vlan:vlan/vlan:vlans") {
		t.Fatal("legacy bridge controller must be skipped when its CRD is absent")
	}
}
