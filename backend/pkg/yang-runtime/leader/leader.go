// Package leader 提供事件源的 leader election 门控（YR-08，自 intent 面泛化）：
// 多副本部署下仅持有 Lease 的副本启动被包装的事件源（非 leader 不产生
// reconcile 事件），失主即停源。一个 Gate = 一把 Lease = 一个选主循环，可包裹
// 任意多个源（同进程内多控制器共享一次选主，避免多 elector 抢同一把锁）。
package leader

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/controller"
)

// Options 配置一把 Lease 的选主参数。
type Options struct {
	// LeaseName 是 Lease 对象名（如 usmp-native-controllers）。
	LeaseName string
	// Namespace 是 Lease 所在 namespace。
	Namespace string
	// Identity 是本副本标识；空则 hostname（再退 usmp-<pid>）。
	Identity string
	// LeaseDuration/RenewDeadline/RetryPeriod 零值取生产缺省 15s/10s/2s
	// （与 intent 面一致；测试可缩短加速接管）。
	LeaseDuration time.Duration
	RenewDeadline time.Duration
	RetryPeriod   time.Duration
	// LogPrefix 用于日志归属（如 "native"/"intent"）；空则用 LeaseName。
	LogPrefix string
}

// Gate 是一把 Lease 的选主门：Wrap 返回被门控的源。nil Gate 或无集群配置
// （cfg==nil）时 Wrap 透传（现行为零变化，R08 降级）。
type Gate struct {
	cfg  *rest.Config
	opts Options

	mu       sync.Mutex
	sources  []*gatedSource  // 已 Start 注册的源
	leadCtx  context.Context // 非 nil = 正在领导
	electing bool            // 选主循环已启动
}

// NewGate 创建一把 Lease 的选主门。cfg 为 nil 表示无可达集群（Wrap 透传）。
func NewGate(cfg *rest.Config, opts Options) *Gate {
	if opts.Identity == "" {
		if host, _ := os.Hostname(); host != "" {
			opts.Identity = host
		} else {
			opts.Identity = fmt.Sprintf("usmp-%d", os.Getpid())
		}
	}
	if opts.LeaseDuration == 0 {
		opts.LeaseDuration = 15 * time.Second
	}
	if opts.RenewDeadline == 0 {
		opts.RenewDeadline = 10 * time.Second
	}
	if opts.RetryPeriod == 0 {
		opts.RetryPeriod = 2 * time.Second
	}
	if opts.LogPrefix == "" {
		opts.LogPrefix = opts.LeaseName
	}
	return &Gate{cfg: cfg, opts: opts}
}

// Wrap 把 inner 包进本 Gate 的选主门；nil Gate / 无集群配置时透传。
func (g *Gate) Wrap(inner controller.Source) controller.Source {
	if g == nil || g.cfg == nil {
		return inner
	}
	return &gatedSource{gate: g, inner: inner}
}

// gatedSource 把 Start/Stop 委托给共享 Gate：Start 仅登记（并按需启动选主
// 循环），真正的 inner.Start 发生在 OnStartedLeading。
type gatedSource struct {
	gate    *Gate
	inner   controller.Source
	ctrl    controller.Controller
	started bool // inner 当前已启动（由 gate.mu 保护）
}

// Start implements controller.Source.
func (s *gatedSource) Start(ctx context.Context, ctrl controller.Controller) error {
	return s.gate.register(ctx, s, ctrl)
}

// Stop implements controller.Source.
func (s *gatedSource) Stop() error {
	g := s.gate
	g.mu.Lock()
	defer g.mu.Unlock()
	for i, reg := range g.sources {
		if reg == s {
			g.sources = append(g.sources[:i], g.sources[i+1:]...)
			break
		}
	}
	if s.started {
		s.started = false
		return s.inner.Stop()
	}
	return nil
}

// register 登记源；若已在领导则立即启动（controller 注册顺序无关），并保证
// 选主循环恰好启动一次（首个 Start 的 ctx 驱动整个循环生命周期）。
func (g *Gate) register(ctx context.Context, s *gatedSource, ctrl controller.Controller) error {
	g.mu.Lock()
	s.ctrl = ctrl
	g.sources = append(g.sources, s)
	if g.leadCtx != nil {
		g.startLocked(s)
	}
	needElection := !g.electing
	g.electing = true
	g.mu.Unlock()

	if !needElection {
		return nil
	}
	return g.runElection(ctx)
}

// startLocked 启动单个源（调用方持 g.mu）。
func (g *Gate) startLocked(s *gatedSource) {
	if s.started || g.leadCtx == nil {
		return
	}
	if err := s.inner.Start(g.leadCtx, s.ctrl); err != nil {
		log.Printf("%s: start source after leadership: %v", g.opts.LogPrefix, err)
		return
	}
	s.started = true
}

// runElection 起选主循环：赢 → 启动全部已登记源；失主 → 停全部源（队列内
// 已入队事件由 worker 自然排空，reconcile 幂等）。
func (g *Gate) runElection(ctx context.Context) error {
	cs, err := kubernetes.NewForConfig(g.cfg)
	if err != nil {
		return fmt.Errorf("%s leader election: %w", g.opts.LogPrefix, err)
	}
	lock := &resourcelock.LeaseLock{
		LeaseMeta:  metav1.ObjectMeta{Name: g.opts.LeaseName, Namespace: g.opts.Namespace},
		Client:     cs.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{Identity: g.opts.Identity},
	}
	go leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:            lock,
		LeaseDuration:   g.opts.LeaseDuration,
		RenewDeadline:   g.opts.RenewDeadline,
		RetryPeriod:     g.opts.RetryPeriod,
		ReleaseOnCancel: true,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(leadCtx context.Context) {
				log.Printf("%s: leader election won (%s); starting %d gated source(s)",
					g.opts.LogPrefix, g.opts.Identity, g.sourceCount())
				g.mu.Lock()
				g.leadCtx = leadCtx
				for _, s := range g.sources {
					g.startLocked(s)
				}
				g.mu.Unlock()
			},
			OnStoppedLeading: func() {
				// 失主即停源（R09：绝不双 leader 下发）；进程保留只读 API。
				log.Printf("%s: leadership lost (%s); gated sources stopped", g.opts.LogPrefix, g.opts.Identity)
				g.mu.Lock()
				g.leadCtx = nil
				for _, s := range g.sources {
					if s.started {
						s.started = false
						if err := s.inner.Stop(); err != nil {
							log.Printf("%s: stop source after leadership loss: %v", g.opts.LogPrefix, err)
						}
					}
				}
				g.mu.Unlock()
			},
		},
	})
	return nil
}

func (g *Gate) sourceCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.sources)
}
