package source

import (
	"context"
	"time"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/controller"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/predicate"
)

// DeviceLister yields the current set of device IDs to reconcile. A DeviceStore
// satisfies it, so the periodic source can poll whatever devices are registered
// at each tick (new devices need no restart). Defined here to avoid a source→
// device package dependency.
type DeviceLister interface {
	List() []string
}

// PeriodicSource is an event source that triggers reconciliation at fixed intervals
// It can be used for periodic polling of device configuration to detect drift
type PeriodicSource struct {
	BaseSource
	interval  time.Duration
	deviceIDs []string
	lister    DeviceLister
	path      string
	ticker    *time.Ticker
	done      chan struct{}
	wg        DoneWaitGroup
}

// DoneWaitGroup is a wait group for done channels
type DoneWaitGroup struct {
	done chan struct{}
	wg   int
}

// Add increments the wait group counter
func (wg *DoneWaitGroup) Add() {
	wg.done = make(chan struct{})
	wg.wg++
}

// Done decrements the wait group counter
func (wg *DoneWaitGroup) Done() {
	wg.wg--
	if wg.wg <= 0 {
		close(wg.done)
	}
}

// Wait waits for all goroutines to finish
func (wg *DoneWaitGroup) Wait() {
	if wg.done != nil {
		<-wg.done
	}
}

// NewPeriodicSource creates a periodic source over a static device list.
func NewPeriodicSource(interval time.Duration, deviceIDs []string, path string) *PeriodicSource {
	return &PeriodicSource{
		interval:  interval,
		deviceIDs: deviceIDs,
		path:      path,
		done:      make(chan struct{}),
	}
}

// NewPeriodicSourceWithLister creates a periodic source that polls whatever
// devices the lister reports at each tick (dynamic; the shared DeviceStore is
// the lister in production). Replaces the previous nil-device-list wiring that
// made the periodic source a no-op, so out-of-band drift is actually detected.
func NewPeriodicSourceWithLister(interval time.Duration, lister DeviceLister, path string) *PeriodicSource {
	return &PeriodicSource{
		interval: interval,
		lister:   lister,
		path:     path,
		done:     make(chan struct{}),
	}
}

// deviceList returns the devices to reconcile this tick: the dynamic lister when
// set, otherwise the static list.
func (s *PeriodicSource) deviceList() []string {
	if s.lister != nil {
		return s.lister.List()
	}
	return s.deviceIDs
}

// Start implements Source interface
func (s *PeriodicSource) Start(ctx context.Context, ctrl controller.Controller) error {
	s.controller = ctrl
	s.ticker = time.NewTicker(s.interval)
	s.done = make(chan struct{})
	s.wg.Add()

	go s.run(ctx)
	return nil
}

func (s *PeriodicSource) run(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			s.ticker.Stop()
			return
		case <-s.done:
			s.ticker.Stop()
			return
		case <-s.ticker.C:
			// Enqueue all currently-registered devices for reconciliation
			for _, deviceID := range s.deviceList() {
				evt := predicate.Event{
					DeviceID: deviceID,
					Path:     s.path,
					Type:     predicate.GenericEvent,
				}
				s.EnqueueEvent(evt)
			}
		}
	}
}

// Stop implements Source interface
func (s *PeriodicSource) Stop() error {
	close(s.done)
	s.ticker.Stop()
	s.wg.Wait()
	return nil
}
