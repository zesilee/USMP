package source

import (
	"context"
	"fmt"
	"log"

	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/controller"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/predicate"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
)

// ProjectFunc maps a watched K8s CRD object to a desired device configuration:
// the target deviceID, the YANG path, and the desired ygot struct. It is supplied
// by the application (which knows the concrete CRD type and translator), keeping
// this framework source generic — it does not import any concrete CRD/translator.
type ProjectFunc func(obj client.Object) (deviceID, path string, desired interface{}, err error)

// KubernetesCRDSource is a C4 event source that projects K8s CRD intent into the
// ConfigStore and enqueues reconciliation: on CRD add/update it translates the CR
// (via ProjectFunc) into a desired ygot config, writes it to the in-memory
// ConfigStore, and enqueues an Update event; on delete it clears the store entry
// and enqueues a Delete event. This lets Stack B consume CRD intent through the
// same ConfigStore → GenericReconciler → NETCONF core (no Actor/2PC).
type KubernetesCRDSource struct {
	store   reconcile.ConfigStore
	cache   cache.Cache
	object  client.Object
	project ProjectFunc
	ctrl    controller.Controller
}

// NewKubernetesCRDSource creates a CRD source. cache is a controller-runtime cache
// (nil is allowed for unit-testing the projection core without a cluster); object
// is the CRD prototype to watch; project maps a CR to (deviceID, path, desired).
func NewKubernetesCRDSource(store reconcile.ConfigStore, c cache.Cache, object client.Object, project ProjectFunc) *KubernetesCRDSource {
	return &KubernetesCRDSource{store: store, cache: c, object: object, project: project}
}

// Start implements controller.Source: it registers informer handlers on the cache
// for the CRD type that project/clear into the ConfigStore and enqueue reconcile.
func (s *KubernetesCRDSource) Start(ctx context.Context, ctrl controller.Controller) error {
	s.ctrl = ctrl
	if s.cache == nil {
		return fmt.Errorf("KubernetesCRDSource: no cache configured")
	}
	inf, err := s.cache.GetInformer(ctx, s.object)
	if err != nil {
		return fmt.Errorf("KubernetesCRDSource: get informer: %w", err)
	}
	_, err = inf.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
		AddFunc:    func(o interface{}) { s.onUpsert(o) },
		UpdateFunc: func(_, o interface{}) { s.onUpsert(o) },
		DeleteFunc: func(o interface{}) { s.onDelete(o) },
	})
	return err
}

// Stop implements controller.Source.
func (s *KubernetesCRDSource) Stop() error { return nil }

func (s *KubernetesCRDSource) onUpsert(o interface{}) {
	obj, ok := o.(client.Object)
	if !ok {
		return
	}
	if evt, ok := s.handleUpsert(obj); ok && s.ctrl != nil {
		s.ctrl.Enqueue(evt)
	}
}

func (s *KubernetesCRDSource) onDelete(o interface{}) {
	obj, ok := o.(client.Object)
	if !ok {
		return
	}
	if evt, ok := s.handleDelete(obj); ok && s.ctrl != nil {
		s.ctrl.Enqueue(evt)
	}
}

// handleUpsert translates the CR, projects the desired config into the store, and
// returns the reconcile event to enqueue. Errors are logged (not fatal, R08).
func (s *KubernetesCRDSource) handleUpsert(obj client.Object) (predicate.Event, bool) {
	deviceID, path, desired, err := s.project(obj)
	if err != nil {
		log.Printf("crd-source: project %T: %v", obj, err)
		return predicate.Event{}, false
	}
	if deviceID == "" {
		log.Printf("crd-source: %T has empty deviceID, skipping", obj)
		return predicate.Event{}, false
	}
	if err := s.store.Set(deviceID, path, desired); err != nil {
		log.Printf("crd-source: ConfigStore.Set(%s,%s): %v", deviceID, path, err)
		return predicate.Event{}, false
	}
	return predicate.Event{DeviceID: deviceID, Path: path, Type: predicate.UpdateEvent}, true
}

// handleDelete clears the store entry for the CR and returns the delete event.
func (s *KubernetesCRDSource) handleDelete(obj client.Object) (predicate.Event, bool) {
	deviceID, path, _, err := s.project(obj)
	if err != nil || deviceID == "" {
		return predicate.Event{}, false
	}
	_ = s.store.Delete(deviceID, path)
	return predicate.Event{DeviceID: deviceID, Path: path, Type: predicate.DeleteEvent}, true
}
