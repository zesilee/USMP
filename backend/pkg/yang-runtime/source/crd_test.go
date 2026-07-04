package source

import (
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/predicate"
)

// fakeStore is a minimal in-memory ConfigStore for testing the CRD source's
// projection without importing the manager package.
type fakeStore struct{ m map[string]interface{} }

func newFakeStore() *fakeStore { return &fakeStore{m: map[string]interface{}{}} }
func key(d, p string) string   { return d + "|" + p }

func (s *fakeStore) Get(d, p string) (interface{}, error) { return s.m[key(d, p)], nil }
func (s *fakeStore) Set(d, p string, v interface{}) error { s.m[key(d, p)] = v; return nil }
func (s *fakeStore) Delete(d, p string) error             { delete(s.m, key(d, p)); return nil }
func (s *fakeStore) List(string) ([]string, error)        { return nil, nil }
func (s *fakeStore) ListDevices() ([]string, error)       { return nil, nil }

const testPath = "/vlan:vlan/vlan:vlans"

func projectOK(_ client.Object) (string, string, interface{}, error) {
	return "10.0.0.1:830", testPath, "DESIRED", nil
}

// TestHandleUpsert: a CR projects its desired config into the store and yields an
// Update reconcile event.
func TestHandleUpsert(t *testing.T) {
	store := newFakeStore()
	s := NewKubernetesCRDSource(store, nil, &corev1.ConfigMap{}, projectOK)

	evt, ok := s.handleUpsert(&corev1.ConfigMap{})
	if !ok {
		t.Fatal("handleUpsert should succeed")
	}
	if evt.DeviceID != "10.0.0.1:830" || evt.Path != testPath || evt.Type != predicate.UpdateEvent {
		t.Fatalf("event = %+v, want update for 10.0.0.1:830", evt)
	}
	if v, _ := store.Get("10.0.0.1:830", testPath); v != "DESIRED" {
		t.Fatalf("store not projected: %v", v)
	}
}

// TestHandleUpsertTranslateError: a projection error is logged and yields no event
// or store change (R08 graceful).
func TestHandleUpsertTranslateError(t *testing.T) {
	store := newFakeStore()
	proj := func(client.Object) (string, string, interface{}, error) {
		return "", "", nil, errors.New("boom")
	}
	s := NewKubernetesCRDSource(store, nil, &corev1.ConfigMap{}, proj)
	if _, ok := s.handleUpsert(&corev1.ConfigMap{}); ok {
		t.Fatal("translate error must not produce an event")
	}
	if len(store.m) != 0 {
		t.Fatal("store must be unchanged on translate error")
	}
}

// TestHandleUpsertEmptyDeviceID: empty deviceID is skipped.
func TestHandleUpsertEmptyDeviceID(t *testing.T) {
	store := newFakeStore()
	proj := func(client.Object) (string, string, interface{}, error) {
		return "", testPath, "X", nil
	}
	s := NewKubernetesCRDSource(store, nil, &corev1.ConfigMap{}, proj)
	if _, ok := s.handleUpsert(&corev1.ConfigMap{}); ok {
		t.Fatal("empty deviceID must be skipped")
	}
	if len(store.m) != 0 {
		t.Fatal("store must be unchanged")
	}
}

// TestHandleDelete: deleting a CR clears the store entry and yields a Delete event.
func TestHandleDelete(t *testing.T) {
	store := newFakeStore()
	_ = store.Set("10.0.0.1:830", testPath, "DESIRED")
	s := NewKubernetesCRDSource(store, nil, &corev1.ConfigMap{}, projectOK)

	evt, ok := s.handleDelete(&corev1.ConfigMap{})
	if !ok {
		t.Fatal("handleDelete should succeed")
	}
	if evt.Type != predicate.DeleteEvent {
		t.Fatalf("event type = %v, want delete", evt.Type)
	}
	if v, _ := store.Get("10.0.0.1:830", testPath); v != nil {
		t.Fatalf("store entry not cleared: %v", v)
	}
}

// TestStartRequiresCache: Start without a cache errors (real runs need a cache).
func TestStartRequiresCache(t *testing.T) {
	s := NewKubernetesCRDSource(newFakeStore(), nil, &corev1.ConfigMap{}, projectOK)
	if err := s.Start(nil, nil); err == nil {
		t.Fatal("Start without cache should error")
	}
}
