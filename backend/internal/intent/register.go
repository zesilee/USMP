package intent

import (
	"context"
	"log"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	crcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/controller"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
)

// intentResyncInterval paces the periodic re-reconcile of every intent CR
// (BIO-04 稳态：幂等短路重写 desired，对冲 desired TTL 过期与设备漂移).
const intentResyncInterval = 5 * time.Minute

// confirmTimeout is the confirmed-commit window for the cross-device 2PC
// (design open-question 初值 60s).
const confirmTimeout = 60 * time.Second

// apiClient is the controller-runtime client wired at Register, consumed by
// the config-api business proxy (nil without a cluster → API 降级 503).
var apiClient client.Client

// APIClient returns the shared CR client (nil when no cluster is reachable).
func APIClient() client.Client { return apiClient }

// Namespace returns the namespace intent CRs live in (USMP_INTENT_NAMESPACE,
// default "default").
func Namespace() string {
	if ns := os.Getenv("USMP_INTENT_NAMESPACE"); ns != "" {
		return ns
	}
	return "default"
}

// Register wires the business-vlan intent controller into the Stack B manager
// (BIO-01): a CR watch source + periodic resync source (C4) feeding the intent
// Reconciler (C3) with the cross-device TxCoordinator (BIO-03). CRD is
// persistence + watch carrier only — the reconcile architecture is unchanged
// (R01, 禁止复活 Stack A 式 CRD 架构).
//
// Degrades gracefully: without a reachable Kubernetes config it logs, returns
// (nil, nil) and the rest of the process runs unaffected (R08). On success the
// returned cache must be started (StartCache).
func Register(mgr manager.Manager) (crcache.Cache, error) {
	cfg, err := ctrlcfg.GetConfig()
	if err != nil {
		log.Printf("intent: no Kubernetes config (%v); business intent controller disabled", err)
		return nil, nil
	}

	scheme := runtime.NewScheme()
	c, err := crcache.New(cfg, crcache.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}
	cl, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}
	apiClient = cl

	tx := NewTxCoordinator(mgr.GetClientPool(), mgr.GetDeviceStore(), confirmTimeout)
	rec := NewReconciler(cl).WithPush(tx, tx, mgr.GetConfigStore(), mgr.TriggerReconcile)

	ctrl := controller.ControllerManagedBy("business-vlan-intent").
		WithReconciler(rec).
		WithSource(gateSources(cfg, multiSource{NewSource(c), NewResyncSource(cl, intentResyncInterval)})).
		WithWorkerCount(1).
		Build()
	mgr.AddController(ctrl)

	log.Printf("intent: business-vlan intent controller registered (CRD watch + resync → expand → 2PC → status)")
	return c, nil
}

// multiSource fans a controller into several event sources (builder 单源限制
// 的最小组合器).
type multiSource []controller.Source

// Start implements controller.Source.
func (m multiSource) Start(ctx context.Context, ctrl controller.Controller) error {
	for _, s := range m {
		if err := s.Start(ctx, ctrl); err != nil {
			return err
		}
	}
	return nil
}

// Stop implements controller.Source.
func (m multiSource) Stop() error {
	var last error
	for _, s := range m {
		if err := s.Stop(); err != nil {
			last = err
		}
	}
	return last
}

// cacheStarter is the minimal surface StartCache needs (crcache.Cache 满足)，
// narrow 接口便于单测注入。
type cacheStarter interface {
	Start(ctx context.Context) error
}

// StartCache starts a controller-runtime cache (blocking) if non-nil. Intended
// to be run in a goroutine alongside the manager（自 crdsource 平移，旧桥接退役后
// 本包是唯一消费方）。nil cache 是无集群降级路径，直接返回（R08）。
func StartCache(ctx context.Context, c cacheStarter) {
	if c == nil {
		return
	}
	if err := c.Start(ctx); err != nil {
		log.Printf("intent: cache stopped: %v", err)
	}
}
