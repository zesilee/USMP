package intent

import (
	"context"
	"log"
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

// Register wires the business-vlan intent controller into the Stack B manager
// (BIO-01): a CR watch source + periodic resync source (C4) feeding the intent
// Reconciler (C3) with the cross-device TxCoordinator (BIO-03). CRD is
// persistence + watch carrier only — the reconcile architecture is unchanged
// (R01, 禁止复活 Stack A 式 CRD 架构).
//
// Degrades gracefully: without a reachable Kubernetes config it logs, returns
// (nil, nil) and the rest of the process runs unaffected (R08). On success the
// returned cache must be started (crdsource.StartCache 同款).
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

	tx := NewTxCoordinator(mgr.GetClientPool(), mgr.GetDeviceStore(), confirmTimeout)
	rec := NewReconciler(cl).WithPush(tx, tx, mgr.GetConfigStore(), mgr.TriggerReconcile)

	ctrl := controller.ControllerManagedBy("business-vlan-intent").
		WithReconciler(rec).
		WithSource(multiSource{NewSource(c), NewResyncSource(cl, intentResyncInterval)}).
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
