package intent

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/controller"
)

// leaderElectionEnabled reads the BIO-08 seam switch（默认关：单副本部署零行为
// 变化；多副本上生产前置 1 开启，仅 leader 产生意图事件 → 展开/2PC/清理单点执行）。
func leaderElectionEnabled() bool {
	return os.Getenv("USMP_INTENT_LEADER_ELECTION") == "1"
}

// gateSources wraps the intent event sources behind leader election when the
// seam is enabled; disabled (default) passes through untouched.
func gateSources(cfg *rest.Config, inner controller.Source) controller.Source {
	if !leaderElectionEnabled() {
		return inner
	}
	return &leaderGatedSource{cfg: cfg, inner: inner}
}

// leaderGatedSource starts the inner sources only after acquiring the Lease;
// losing leadership stops them (队列内已入队事件由 worker 自然排空——展开/推送
// 幂等，且新 leader 的 resync 会重放全量意图)。
type leaderGatedSource struct {
	cfg   *rest.Config
	inner controller.Source
}

// Start implements controller.Source.
func (s *leaderGatedSource) Start(ctx context.Context, ctrl controller.Controller) error {
	cs, err := kubernetes.NewForConfig(s.cfg)
	if err != nil {
		return fmt.Errorf("intent leader election: %w", err)
	}
	id, _ := os.Hostname()
	if id == "" {
		id = fmt.Sprintf("usmp-%d", os.Getpid())
	}
	lock := &resourcelock.LeaseLock{
		LeaseMeta:  metav1.ObjectMeta{Name: "usmp-business-intent", Namespace: Namespace()},
		Client:     cs.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{Identity: id},
	}
	go leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:            lock,
		LeaseDuration:   15 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     2 * time.Second,
		ReleaseOnCancel: true,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(leadCtx context.Context) {
				log.Printf("intent: leader election won (%s); starting intent sources", id)
				if err := s.inner.Start(leadCtx, ctrl); err != nil {
					log.Printf("intent: start sources after leadership: %v", err)
				}
			},
			OnStoppedLeading: func() {
				// 失主即停源（R09：绝不双 leader 下发）；进程保留服务只读 API。
				log.Printf("intent: leadership lost (%s); intent sources stopped", id)
				_ = s.inner.Stop()
			},
		},
	})
	return nil
}

// Stop implements controller.Source.
func (s *leaderGatedSource) Stop() error { return s.inner.Stop() }
