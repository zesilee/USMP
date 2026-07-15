package intent

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	toolscache "k8s.io/client-go/tools/cache"
	crcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/controller"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/predicate"
)

// Source is the C4 event source for intent CRs (BIO-01): it watches
// BusinessVlanService objects and enqueues CR-keyed reconcile events
// (Request.DeviceID = "namespace/name"). Unlike KubernetesCRDSource it does
// NOT project into the ConfigStore — expansion, transaction and desired writes
// are the intent Reconciler's job (validation/2PC/status live there).
type Source struct {
	cache crcache.Cache
	ctrl  controller.Controller
}

// NewSource creates the intent CR watch source over a controller-runtime cache.
func NewSource(c crcache.Cache) *Source {
	return &Source{cache: c}
}

// Start implements controller.Source.
func (s *Source) Start(ctx context.Context, ctrl controller.Controller) error {
	s.ctrl = ctrl
	if s.cache == nil {
		return fmt.Errorf("intent.Source: no cache configured")
	}
	proto := &unstructured.Unstructured{}
	proto.SetGroupVersionKind(GVK)
	inf, err := s.cache.GetInformer(ctx, proto)
	if err != nil {
		return fmt.Errorf("intent.Source: get informer: %w", err)
	}
	_, err = inf.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
		AddFunc:    func(o interface{}) { s.enqueue(o, predicate.UpdateEvent) },
		UpdateFunc: func(_, o interface{}) { s.enqueue(o, predicate.UpdateEvent) },
		// informer Delete 只在 finalizer 摘除后触发；清理已完成，事件仅用于收尾。
		DeleteFunc: func(o interface{}) { s.enqueue(o, predicate.DeleteEvent) },
	})
	return err
}

// Stop implements controller.Source.
func (s *Source) Stop() error { return nil }

func (s *Source) enqueue(o interface{}, t predicate.EventType) {
	obj, ok := o.(client.Object)
	if !ok || s.ctrl == nil {
		return
	}
	key := obj.GetName()
	if ns := obj.GetNamespace(); ns != "" {
		key = ns + "/" + obj.GetName()
	}
	s.ctrl.Enqueue(predicate.Event{DeviceID: key, Path: IntentPath, Type: t})
}
