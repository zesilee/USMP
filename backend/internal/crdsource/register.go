package crdsource

import (
	"context"
	"log"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	crcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"

	apiv1 "github.com/leezesi/usmp/backend/api/biz/v1"
	"github.com/leezesi/usmp/backend/internal/controller/ifm"
	"github.com/leezesi/usmp/backend/internal/controller/vlan"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/controller"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/source"
)

// RegisterIntentSources wires the business CRD intent sources into the Stack B
// manager: on CRD change each source projects the translated desired config into
// the ConfigStore and enqueues reconciliation through the matching device-native
// reconciler. These run in parallel with the legacy Actor path (§5.3) until it is
// retired. Sources with only stub translations (Route/System — D5/D8) are not
// registered yet: projecting a non-ygot desired would break diff (R04).
//
// It degrades gracefully: if no Kubernetes config is reachable (e.g. local dev or
// CI without a cluster), CRD intent sources are disabled and (nil, nil) is
// returned — the rest of the manager (device-native reconcilers, REST API) runs
// unaffected. On success it returns the controller-runtime cache to Start.
func RegisterIntentSources(mgr manager.Manager) (crcache.Cache, error) {
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

	cs, pool := mgr.GetConfigStore(), mgr.GetClientPool()
	addIntentController(mgr, c, "huawei-vlan-crd", VlanObject(), VlanProjectFunc, vlan.New(cs, pool))
	addIntentController(mgr, c, "huawei-ifm-crd", InterfaceObject(), InterfaceProjectFunc, ifm.New(cs, pool))

	log.Printf("crdsource: CRD intent sources registered (BusinessVlan, BusinessInterface; parallel to Actor path)")
	return c, nil
}

// addIntentController builds a controller whose CRD source projects intent into the
// ConfigStore and whose reconciler aligns the device, then registers it.
func addIntentController(mgr manager.Manager, c crcache.Cache, name string, obj client.Object, project source.ProjectFunc, reconciler reconcile.Reconciler) {
	src := source.NewKubernetesCRDSource(mgr.GetConfigStore(), c, obj, project)
	ctrl := controller.ControllerManagedBy(name).
		WithReconciler(reconciler).
		WithSource(src).
		Build()
	mgr.AddController(ctrl)
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
