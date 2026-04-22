package source

import (
	"context"
	"time"

	"github.com/leezesi/usmp/pkg/yang-runtime/controller"
	"github.com/leezesi/usmp/pkg/yang-runtime/predicate"
)

// PeriodicSource is an event source that triggers reconciliation at fixed intervals
// It can be used for periodic polling of device configuration to detect drift
type PeriodicSource struct {
	BaseSource
	interval  time.Duration
	deviceIDs []string
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

// NewPeriodicSource creates a new periodic source that triggers reconciliation
// at the specified interval for all devices/paths
func NewPeriodicSource(interval time.Duration, deviceIDs []string, path string) *PeriodicSource {
	return &PeriodicSource{
		interval:  interval,
		deviceIDs: deviceIDs,
		path:      path,
		done:      make(chan struct{}),
	}
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
			// Enqueue all devices for reconciliation
			for _, deviceID := range s.deviceIDs {
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
