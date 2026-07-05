package status

import (
	"errors"
	"sync"
	"testing"
)

func TestStore_RecordAndGet(t *testing.T) {
	s := NewStore()
	s.Record("10.0.0.1", "/vlans", OutcomeConverged, 0, nil)

	got, ok := s.Get("10.0.0.1", "/vlans")
	if !ok {
		t.Fatalf("expected status found")
	}
	if got.Outcome != OutcomeConverged {
		t.Errorf("outcome = %q, want %q", got.Outcome, OutcomeConverged)
	}
	if got.DeviceID != "10.0.0.1" || got.Path != "/vlans" {
		t.Errorf("device/path = %q/%q, want 10.0.0.1//vlans", got.DeviceID, got.Path)
	}
	if got.LastRun.IsZero() {
		t.Errorf("LastRun should be stamped, got zero")
	}
	if got.LastError != "" {
		t.Errorf("LastError = %q, want empty", got.LastError)
	}
}

func TestStore_GetMissing(t *testing.T) {
	s := NewStore()
	if _, ok := s.Get("10.0.0.9", "/vlans"); ok {
		t.Errorf("expected not found for unrecorded device")
	}
}

func TestStore_Overwrite_LatestWins(t *testing.T) {
	s := NewStore()
	s.Record("10.0.0.1", "/vlans", OutcomeDrifted, 3, nil)
	s.Record("10.0.0.1", "/vlans", OutcomeConverged, 0, nil)

	got, _ := s.Get("10.0.0.1", "/vlans")
	if got.Outcome != OutcomeConverged || got.DiffCount != 0 {
		t.Errorf("got %q/%d, want converged/0", got.Outcome, got.DiffCount)
	}
}

func TestStore_RecordError(t *testing.T) {
	s := NewStore()
	s.Record("10.0.0.1", "/vlans", OutcomeError, 0, errors.New("session timeout"))

	got, _ := s.Get("10.0.0.1", "/vlans")
	if got.Outcome != OutcomeError {
		t.Errorf("outcome = %q, want error", got.Outcome)
	}
	if got.LastError != "session timeout" {
		t.Errorf("LastError = %q, want 'session timeout'", got.LastError)
	}
}

func TestStore_DriftedCarriesDiffCount(t *testing.T) {
	s := NewStore()
	s.Record("10.0.0.1", "/ifm", OutcomeDrifted, 4, nil)
	got, _ := s.Get("10.0.0.1", "/ifm")
	if got.Outcome != OutcomeDrifted || got.DiffCount != 4 {
		t.Errorf("got %q/%d, want drifted/4", got.Outcome, got.DiffCount)
	}
}

func TestStore_ListByDevice(t *testing.T) {
	s := NewStore()
	s.Record("10.0.0.1", "/vlans", OutcomeConverged, 0, nil)
	s.Record("10.0.0.1", "/ifm", OutcomeDrifted, 2, nil)
	s.Record("10.0.0.2", "/vlans", OutcomeError, 0, errors.New("x"))

	list := s.ListByDevice("10.0.0.1")
	if len(list) != 2 {
		t.Fatalf("ListByDevice = %d entries, want 2", len(list))
	}
	for _, st := range list {
		if st.DeviceID != "10.0.0.1" {
			t.Errorf("leaked device %q into ListByDevice(10.0.0.1)", st.DeviceID)
		}
	}
}

func TestStore_Snapshot(t *testing.T) {
	s := NewStore()
	s.Record("10.0.0.1", "/vlans", OutcomeConverged, 0, nil)
	s.Record("10.0.0.2", "/ifm", OutcomeDrifted, 1, nil)
	if got := len(s.Snapshot()); got != 2 {
		t.Errorf("Snapshot = %d, want 2", got)
	}
}

// TestStore_Concurrent asserts the store is race-free under concurrent record/read.
// Run with -race.
func TestStore_Concurrent(t *testing.T) {
	s := NewStore()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			s.Record("10.0.0.1", "/vlans", OutcomeConverged, n, nil)
		}(i)
		go func() {
			defer wg.Done()
			s.Get("10.0.0.1", "/vlans")
			s.ListByDevice("10.0.0.1")
			s.Snapshot()
		}()
	}
	wg.Wait()
}
