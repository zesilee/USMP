package device

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	usmpv1 "github.com/leezesi/usmp/backend/api/core/v1"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
)

// crdStore envtest 矩阵（DS-01/04/05）：真 apiserver 上验证双资源写穿、凭据
// Secret 引用、watch 跨副本可见、重启恢复、失败可见与并发安全。

func envtestAssets(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("KUBEBUILDER_ASSETS"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	matches, _ := filepath.Glob(filepath.Join(home, ".local/share/kubebuilder-envtest/k8s/*"))
	if len(matches) > 0 {
		return matches[0]
	}
	t.Skip("envtest binaries not installed (run: go run sigs.k8s.io/controller-runtime/tools/setup-envtest@release-0.19 use 1.31.0)")
	return ""
}

// 共享单次 envtest apiserver（矩阵 8 场景各自起一次会超出 pre-commit 30s
// 门禁；用例间以不同设备 IP 隔离）。TestMain 负责停靠。
var (
	sharedEnv    *envtest.Environment
	sharedCfg    *rest.Config
	sharedClient ctrlclient.Client
	sharedErr    error
	sharedOnce   sync.Once
)

func TestMain(m *testing.M) {
	code := m.Run()
	if sharedEnv != nil {
		_ = sharedEnv.Stop()
	}
	os.Exit(code)
}

// startEnvtest 起（或复用）真 apiserver（装 deploy/crds 全部 CRD），返回 rest
// 配置与直连 client（测试断言用，不经被测 store）。
func startEnvtest(t *testing.T) (*rest.Config, ctrlclient.Client) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping envtest integration in short mode")
	}
	assets := envtestAssets(t)
	sharedOnce.Do(func() {
		env := &envtest.Environment{
			BinaryAssetsDirectory: assets,
			CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "..", "deploy", "crds")},
		}
		cfg, err := env.Start()
		if err != nil {
			sharedErr = fmt.Errorf("start envtest: %w", err)
			return
		}
		scheme, err := CRDStoreScheme()
		if err != nil {
			sharedErr = fmt.Errorf("scheme: %w", err)
			return
		}
		cl, err := ctrlclient.New(cfg, ctrlclient.Options{Scheme: scheme})
		if err != nil {
			sharedErr = fmt.Errorf("client: %w", err)
			return
		}
		sharedEnv, sharedCfg, sharedClient = env, cfg, cl
	})
	if sharedErr != nil {
		t.Fatal(sharedErr)
	}
	return sharedCfg, sharedClient
}

func newTestStore(t *testing.T, cfg *rest.Config, ns string) Store {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	s, err := NewCRDStore(ctx, cfg, ns)
	if err != nil {
		t.Fatalf("NewCRDStore: %v", err)
	}
	return s
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", what)
}

func fullInfo(ip string) client.DeviceConnectionInfo {
	return client.DeviceConnectionInfo{
		IP: ip, Port: 830, Username: "op", Password: "s3cret",
		Protocol: client.ProtocolNETCONF, Timeout: 5 * time.Second, Vendor: "huawei", Role: "DCGW",
	}
}

// DS-01/DS-04: Put 双资源落库（CR 无明文凭据 + Secret 承载凭据），写穿后本地
// 即时可读，Get 还原完整信息。
func TestCRDStore_PutGetRoundTrip_Integration(t *testing.T) {
	cfg, cl := startEnvtest(t)
	s := newTestStore(t, cfg, "default")

	if err := s.Put("10.0.0.1", fullInfo("10.0.0.1")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// 写穿后立即可读（不等 watch 回流）
	got, ok := s.Get("10.0.0.1")
	if !ok {
		t.Fatal("Get right after Put: not found (write-through miss)")
	}
	if got != fullInfo("10.0.0.1") {
		t.Fatalf("Get = %+v, want %+v", got, fullInfo("10.0.0.1"))
	}

	// CR 存在且不含明文凭据；Secret 承载凭据
	ctx := context.Background()
	var dev usmpv1.Device
	if err := cl.Get(ctx, types.NamespacedName{Namespace: "default", Name: DeviceCRName("10.0.0.1")}, &dev); err != nil {
		t.Fatalf("Device CR: %v", err)
	}
	if dev.Spec.ManagementIP != "10.0.0.1" || dev.Spec.Protocol != "netconf" ||
		dev.Spec.Port != 830 || dev.Spec.TimeoutSeconds != 5 || dev.Spec.Vendor != "huawei" || dev.Spec.Role != "DCGW" {
		t.Fatalf("CR spec mismatch: %+v", dev.Spec)
	}
	if dev.Spec.CredentialsSecretRef == nil {
		t.Fatal("CR must reference a credentials Secret")
	}
	var sec corev1.Secret
	if err := cl.Get(ctx, types.NamespacedName{Namespace: "default", Name: dev.Spec.CredentialsSecretRef.Name}, &sec); err != nil {
		t.Fatalf("Secret: %v", err)
	}
	if string(sec.Data["username"]) != "op" || string(sec.Data["password"]) != "s3cret" {
		t.Fatalf("Secret data mismatch: %v", sec.Data)
	}

	// 未注册设备
	if _, ok := s.Get("10.9.9.9"); ok {
		t.Fatal("unregistered device must return ok=false")
	}

	// BR-05: 重复 IP 覆盖（upsert Update 分支）——凭据与 spec 均更新
	updated := fullInfo("10.0.0.1")
	updated.Password, updated.Port = "rotated", 831
	if err := s.Put("10.0.0.1", updated); err != nil {
		t.Fatalf("re-Put: %v", err)
	}
	got, _ = s.Get("10.0.0.1")
	if got != updated {
		t.Fatalf("after re-Put Get = %+v, want %+v", got, updated)
	}
	if err := cl.Get(ctx, types.NamespacedName{Namespace: "default", Name: DeviceCRName("10.0.0.1")}, &dev); err != nil {
		t.Fatalf("Device CR after re-Put: %v", err)
	}
	if dev.Spec.Port != 831 {
		t.Fatalf("CR spec not overwritten: %+v", dev.Spec)
	}
	if err := cl.Get(ctx, types.NamespacedName{Namespace: "default", Name: dev.Spec.CredentialsSecretRef.Name}, &sec); err != nil {
		t.Fatalf("Secret after re-Put: %v", err)
	}
	if string(sec.Data["password"]) != "rotated" {
		t.Fatalf("Secret not overwritten: %v", sec.Data)
	}
}

// DS-04: Secret 被外部删除 → Get 降级空凭据（ok=true），不 panic。
func TestCRDStore_SecretMissingDegradesEmptyCreds_Integration(t *testing.T) {
	cfg, cl := startEnvtest(t)
	s := newTestStore(t, cfg, "default")

	if err := s.Put("10.0.0.2", fullInfo("10.0.0.2")); err != nil {
		t.Fatalf("Put: %v", err)
	}
	ctx := context.Background()
	var dev usmpv1.Device
	if err := cl.Get(ctx, types.NamespacedName{Namespace: "default", Name: DeviceCRName("10.0.0.2")}, &dev); err != nil {
		t.Fatalf("Device CR: %v", err)
	}
	sec := &corev1.Secret{}
	sec.Namespace, sec.Name = "default", dev.Spec.CredentialsSecretRef.Name
	if err := cl.Delete(ctx, sec); err != nil {
		t.Fatalf("delete secret: %v", err)
	}
	// 触发 watch 重建镜像：touch CR（外部改动路径）
	dev.Labels = map[string]string{"touch": "1"}
	if err := cl.Update(ctx, &dev); err != nil {
		t.Fatalf("update CR: %v", err)
	}
	waitFor(t, "empty-credential degrade", func() bool {
		info, ok := s.Get("10.0.0.2")
		return ok && info.Username == "" && info.Password == "" && info.IP == "10.0.0.2"
	})
}

// DS-04: Delete 反向清理两资源，镜像即时移除。
func TestCRDStore_DeleteRemovesBoth_Integration(t *testing.T) {
	cfg, cl := startEnvtest(t)
	s := newTestStore(t, cfg, "default")

	if err := s.Put("10.0.0.3", fullInfo("10.0.0.3")); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := s.Delete("10.0.0.3"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok := s.Get("10.0.0.3"); ok {
		t.Fatal("Get after Delete must be ok=false")
	}
	ctx := context.Background()
	var dev usmpv1.Device
	err := cl.Get(ctx, types.NamespacedName{Namespace: "default", Name: DeviceCRName("10.0.0.3")}, &dev)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("Device CR should be gone, err=%v", err)
	}
	var secs corev1.SecretList
	if err := cl.List(ctx, &secs, ctrlclient.InNamespace("default")); err != nil {
		t.Fatalf("list secrets: %v", err)
	}
	for _, sc := range secs.Items {
		if sc.Labels[DeviceIPLabel] == "10.0.0.3" {
			t.Fatalf("credentials Secret should be gone: %s", sc.Name)
		}
	}
}

// DS-05: 副本间经 watch 可见（含凭据还原）；删除同样收敛。
func TestCRDStore_CrossReplicaVisibility_Integration(t *testing.T) {
	cfg, _ := startEnvtest(t)
	sA := newTestStore(t, cfg, "default")
	sB := newTestStore(t, cfg, "default")

	if err := sA.Put("10.0.0.4", fullInfo("10.0.0.4")); err != nil {
		t.Fatalf("Put on A: %v", err)
	}
	waitFor(t, "replica B sees device", func() bool {
		info, ok := sB.Get("10.0.0.4")
		return ok && info == fullInfo("10.0.0.4")
	})
	if err := sB.Delete("10.0.0.4"); err != nil {
		t.Fatalf("Delete on B: %v", err)
	}
	waitFor(t, "replica A sees deletion", func() bool {
		_, ok := sA.Get("10.0.0.4")
		return !ok
	})
}

// DS-05: 实例重建后从 CR 完整恢复（含经 Secret 还原的凭据）。
func TestCRDStore_RestartRecovery_Integration(t *testing.T) {
	cfg, _ := startEnvtest(t)
	sA := newTestStore(t, cfg, "default")
	if err := sA.Put("10.0.0.5", fullInfo("10.0.0.5")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	sNew := newTestStore(t, cfg, "default") // 模拟重建实例
	waitFor(t, "recovered device with credentials", func() bool {
		info, ok := sNew.Get("10.0.0.5")
		return ok && info == fullInfo("10.0.0.5")
	})
	waitFor(t, "recovered list", func() bool {
		for _, id := range sNew.List() {
			if id == "10.0.0.5" {
				return true
			}
		}
		return false
	})
}

// DS-04: 目标 namespace 不存在（apiserver 拒绝写）→ Put 返回错误、镜像不变更。
func TestCRDStore_WriteFailureVisible_Integration(t *testing.T) {
	cfg, _ := startEnvtest(t)
	s := newTestStore(t, cfg, "no-such-namespace")

	if err := s.Put("10.0.0.6", fullInfo("10.0.0.6")); err == nil {
		t.Fatal("Put into missing namespace must return error")
	}
	if _, ok := s.Get("10.0.0.6"); ok {
		t.Fatal("mirror must not be updated on failed Put")
	}
}

// R09: 并发 Put/Get/Delete/List 无数据竞态（-race）。
func TestCRDStore_Concurrent_Integration(t *testing.T) {
	cfg, _ := startEnvtest(t)
	s := newTestStore(t, cfg, "default")

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			ip := fmt.Sprintf("10.2.0.%d", n)
			for j := 0; j < 5; j++ {
				_ = s.Put(ip, fullInfo(ip))
				_ = s.Delete(ip)
			}
		}(i)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				s.Get(fmt.Sprintf("10.2.0.%d", n))
				s.List()
			}
		}(i)
	}
	wg.Wait()
}
