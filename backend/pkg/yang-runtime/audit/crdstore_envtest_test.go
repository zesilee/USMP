package audit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"k8s.io/client-go/rest"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	usmpv1 "github.com/leezesi/usmp/backend/api/core/v1"
)

// AuditRecord CRD 后端 envtest 矩阵（OA-01/02/03）：真 apiserver 上验证
// 每条一 CR、跨副本可见、重启保留、超限清理幂等、写失败不阻断。

var (
	sharedEnv    *envtest.Environment
	sharedCfg    *rest.Config
	sharedClient ctrlclient.Client
)

func TestMain(m *testing.M) {
	code := m.Run()
	if sharedEnv != nil {
		_ = sharedEnv.Stop()
	}
	os.Exit(code)
}

func envtestConfig(t *testing.T) (*rest.Config, ctrlclient.Client) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping envtest integration in short mode")
	}
	if sharedCfg != nil {
		return sharedCfg, sharedClient
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
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "..", "deploy", "crds")},
	}
	cfg, err := env.Start()
	if err != nil {
		t.Fatalf("start envtest: %v", err)
	}
	scheme, err := CRDStoreScheme()
	if err != nil {
		t.Fatalf("scheme: %v", err)
	}
	cl, err := ctrlclient.New(cfg, ctrlclient.Options{Scheme: scheme})
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	sharedEnv, sharedCfg, sharedClient = env, cfg, cl
	return cfg, cl
}

func newTestStore(t *testing.T, cfg *rest.Config, ns string, max int) Store {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	s, err := NewCRDStore(ctx, cfg, ns, max)
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

func countCRs(t *testing.T, cl ctrlclient.Client, device string) int {
	t.Helper()
	var list usmpv1.AuditRecordList
	if err := cl.List(context.Background(), &list, ctrlclient.InNamespace("default"),
		ctrlclient.MatchingLabels{DeviceIPLabel: device}); err != nil {
		t.Fatalf("list audit CRs: %v", err)
	}
	return len(list.Items)
}

// OA-01/02: Record 落 CR（label 可筛选、spec 完整、无对账结局字段）、镜像即时可读。
func TestCRDAudit_RecordCreatesCR_Integration(t *testing.T) {
	cfg, cl := envtestConfig(t)
	s := newTestStore(t, cfg, "default", 100)

	s.Record(Record{DeviceIP: "10.3.0.1", Path: "/vlan:vlan/vlan:vlans", Summary: "vlan keys [10]", Triggered: true})

	// 写穿：本地 List 立即可见，ID 已分配
	recs := s.List()
	if len(recs) != 1 || recs[0].ID == "" || recs[0].Actor != "system" {
		t.Fatalf("mirror after Record = %+v", recs)
	}
	// CR 最终落库
	waitFor(t, "audit CR created", func() bool { return countCRs(t, cl, "10.3.0.1") == 1 })
	var list usmpv1.AuditRecordList
	_ = cl.List(context.Background(), &list, ctrlclient.InNamespace("default"),
		ctrlclient.MatchingLabels{DeviceIPLabel: "10.3.0.1"})
	spec := list.Items[0].Spec
	if spec.DeviceIP != "10.3.0.1" || spec.Path != "/vlan:vlan/vlan:vlans" || !spec.Triggered || spec.Actor != "system" {
		t.Fatalf("CR spec mismatch: %+v", spec)
	}
}

// OA-02: 跨副本可见（B 副本经 watch 收敛看到 A 副本写入）。
func TestCRDAudit_CrossReplicaVisible_Integration(t *testing.T) {
	cfg, _ := envtestConfig(t)
	sA := newTestStore(t, cfg, "default", 100)
	sB := newTestStore(t, cfg, "default", 100)

	sA.Record(Record{DeviceIP: "10.3.0.2", Summary: "from-A"})
	waitFor(t, "replica B sees record", func() bool {
		for _, r := range sB.ListByDevice("10.3.0.2") {
			if r.Summary == "from-A" {
				return true
			}
		}
		return false
	})
}

// OA-02: 实例重建后审计历史完整保留。
func TestCRDAudit_RestartRetention_Integration(t *testing.T) {
	cfg, _ := envtestConfig(t)
	sA := newTestStore(t, cfg, "default", 100)
	sA.Record(Record{DeviceIP: "10.3.0.3", Summary: "before-restart"})

	waitFor(t, "record persisted", func() bool {
		return countCRs(t, sharedClient, "10.3.0.3") == 1
	})
	sNew := newTestStore(t, cfg, "default", 100) // 模拟重建
	waitFor(t, "history recovered", func() bool {
		recs := sNew.ListByDevice("10.3.0.3")
		return len(recs) == 1 && recs[0].Summary == "before-restart"
	})
}

// OA-03: 超上限删最旧（写入方清理，保最新）。
func TestCRDAudit_OverflowCleanup_Integration(t *testing.T) {
	cfg, cl := envtestConfig(t)
	s := newTestStore(t, cfg, "default", 3)

	for i := 0; i < 5; i++ {
		s.Record(Record{DeviceIP: "10.3.0.4", Summary: fmt.Sprintf("rec-%d", i)})
		time.Sleep(5 * time.Millisecond) // 保证时间序
	}
	waitFor(t, "cleanup to max", func() bool { return countCRs(t, cl, "10.3.0.4") == 3 })
	// 保最新：rec-2..4 在，rec-0/1 没了
	waitFor(t, "newest kept", func() bool {
		recs := s.ListByDevice("10.3.0.4")
		if len(recs) != 3 {
			return false
		}
		return recs[0].Summary == "rec-4" && recs[2].Summary == "rec-2"
	})
}

// OA-03: 并发写入+清理幂等（NotFound 容忍），无竞态无 panic。
func TestCRDAudit_ConcurrentCleanup_Integration(t *testing.T) {
	cfg, cl := envtestConfig(t)
	sA := newTestStore(t, cfg, "default", 2)
	sB := newTestStore(t, cfg, "default", 2)

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			sA.Record(Record{DeviceIP: "10.3.0.5", Summary: fmt.Sprintf("a-%d", n)})
		}(i)
		go func(n int) {
			defer wg.Done()
			sB.Record(Record{DeviceIP: "10.3.0.5", Summary: fmt.Sprintf("b-%d", n)})
		}(i)
	}
	wg.Wait()
	waitFor(t, "converge to max", func() bool { return countCRs(t, cl, "10.3.0.5") <= 2 })
}

// OA-01: 持久化后端不可用（namespace 不存在）时 Record 不阻断、不 panic，
// 镜像仍可读（与旧文件路径「persist failed, keeping in-memory only」一致）。
func TestCRDAudit_WriteFailureNonBlocking_Integration(t *testing.T) {
	cfg, _ := envtestConfig(t)
	s := newTestStore(t, cfg, "no-such-namespace", 10)

	done := make(chan struct{})
	go func() {
		s.Record(Record{DeviceIP: "10.3.0.6", Summary: "doomed"})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("Record must not block on persist failure")
	}
	waitFor(t, "mirror still readable", func() bool {
		return len(s.ListByDevice("10.3.0.6")) == 1
	})
}
