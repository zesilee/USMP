package crdsource

import (
	"context"
	"log"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	crcache "sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"

	apiv1 "github.com/leezesi/usmp/backend/api/v1"
	"github.com/leezesi/usmp/backend/internal/controller/vlan"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/controller"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/source"
)

// RegisterVlanIntentSource wires the BusinessVlan CRD intent source into the
// Stack B manager: on CRD change it projects the translated desired config into
// the ConfigStore and enqueues reconciliation through the existing Huawei VLAN
// reconciler. This runs in parallel with the legacy Actor path (§5.3) until the
// Actor path is retired.
//
// It degrades gracefully: if no Kubernetes config is reachable (e.g. local dev or
// CI without a cluster), CRD intent sources are disabled and (nil, nil) is
// returned — the rest of the manager (device-native reconcilers, REST API) runs
// unaffected. On success it returns the controller-runtime cache to Start.
func RegisterVlanIntentSource(mgr manager.Manager) (crcache.Cache, error) {
	cfg, err := ctrlcfg.GetConfig()
	if err != nil {
		log.Printf("crdsource: no Kubernetes config (%v); CRD intent sources disabled", err)
		return nil, nil
	}

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := apiv1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	c, err := crcache.New(cfg, crcache.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	src := source.NewKubernetesCRDSource(mgr.GetConfigStore(), c, VlanObject(), VlanProjectFunc)
	ctrl := controller.ControllerManagedBy("huawei-vlan-crd").
		WithReconciler(vlan.New(mgr.GetConfigStore(), mgr.GetClientPool())).
		WithSource(src).
		Build()
	mgr.AddController(ctrl)

	log.Printf("crdsource: BusinessVlan intent source registered (parallel to Actor path)")
	return c, nil
}

// StartCache starts a controller-runtime cache (blocking) if non-nil. Intended to
// be run in a goroutine alongside the manager.
func StartCache(ctx context.Context, c crcache.Cache) {
	if c == nil {
		return
	}
	if err := c.Start(ctx); err != nil {
		log.Printf("crdsource: cache stopped: %v", err)
	}
}
