package source

import (
	"context"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/controller"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/predicate"
)

// GNMISubSource is an event source that subscribes to gNMI notifications
// from a device and triggers reconciliation when notifications are received
type GNMISubSource struct {
	BaseSource
	deviceID string
	path     string
	client   client.Client
	done     chan struct{}
	wg       DoneWaitGroup
}

// NewGNMISubSource creates a new gNMI subscription source
func NewGNMISubSource(deviceID, path string, c client.Client) *GNMISubSource {
	return &GNMISubSource{
		deviceID: deviceID,
		path:     path,
		client:   c,
		done:     make(chan struct{}),
	}
}

// Start implements Source interface
func (s *GNMISubSource) Start(ctx context.Context, ctrl controller.Controller) error {
	s.controller = ctrl
	s.done = make(chan struct{})
	s.wg.Add()

	handler := func(notification client.Notification) {
		// When a notification is received, trigger reconciliation
		evt := predicate.Event{
			DeviceID: s.deviceID,
			Path:     notification.Path,
			Type:     predicate.UpdateEvent,
			Metadata: map[string]interface{}{
				"data": notification.Data,
			},
		}
		s.EnqueueEvent(evt)
	}

	go s.run(ctx, handler)
	return nil
}

func (s *GNMISubSource) run(ctx context.Context, handler func(client.Notification)) {
	defer s.wg.Done()

	err := s.client.Subscribe(ctx, s.path, handler)
	if err != nil {
		// Subscription failed, exit
		return
	}

	<-s.done
}

// Stop implements Source interface
func (s *GNMISubSource) Stop() {
	close(s.done)
	s.wg.Wait()
}
