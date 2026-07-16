package intent

import (
	"log"

	"k8s.io/apimachinery/pkg/runtime"
	crcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/controller"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
)

// Register wires the business-vlan intent controller into the Stack B manager
// (BIO-01): a CR watch source (C4) feeding the intent Reconciler (C3). CRD is
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

	ctrl := controller.ControllerManagedBy("business-vlan-intent").
		WithReconciler(NewReconciler(cl)).
		WithSource(NewSource(c)).
		WithWorkerCount(1).
		Build()
	mgr.AddController(ctrl)

	log.Printf("intent: business-vlan intent controller registered (CRD watch → expand → status)")
	return c, nil
}
