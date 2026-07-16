package intent

import (
	"context"
	"fmt"
	"log"
	"time"

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

// crLister lists intent CRs (satisfied by controller-runtime cache/client;
// narrowed for unit tests).
type crLister interface {
	List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error
}

// ResyncSource periodically re-enqueues every intent CR（BIO-04 稳态）：意图
// Reconciler 的幂等短路会重写 desired 并触发原生对账，对冲 desired TTL 过期与
// 漂移（意图管的配置由此获得与原生周期对账等效的稳态保障）。
type ResyncSource struct {
	reader   crLister
	interval time.Duration
	stop     chan struct{}
}

// NewResyncSource builds the periodic intent resync source.
func NewResyncSource(reader crLister, interval time.Duration) *ResyncSource {
	return &ResyncSource{reader: reader, interval: interval, stop: make(chan struct{})}
}

// Start implements controller.Source.
func (s *ResyncSource) Start(ctx context.Context, ctrl controller.Controller) error {
	if s.reader == nil {
		return fmt.Errorf("intent.ResyncSource: no reader configured")
	}
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.stop:
				return
			case <-ticker.C:
				s.enqueueAll(ctx, ctrl)
			}
		}
	}()
	return nil
}

// Stop implements controller.Source.
func (s *ResyncSource) Stop() error {
	close(s.stop)
	return nil
}

// enqueueAll lists all intent CRs and enqueues an update event per CR.
func (s *ResyncSource) enqueueAll(ctx context.Context, ctrl controller.Controller) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(GVK.GroupVersion().WithKind(GVK.Kind + "List"))
	if err := s.reader.List(ctx, list); err != nil {
		log.Printf("intent: resync list: %v", err)
		return
	}
	for i := range list.Items {
		item := &list.Items[i]
		key := item.GetName()
		if ns := item.GetNamespace(); ns != "" {
			key = ns + "/" + item.GetName()
		}
		ctrl.Enqueue(predicate.Event{DeviceID: key, Path: IntentPath, Type: predicate.UpdateEvent})
	}
}

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
