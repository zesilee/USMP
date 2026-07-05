package controller

import (
	"context"
	"errors"
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/queue"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type fakeRecorder struct {
	deviceID string
	path     string
	outcome  status.Outcome
	diff     int
	err      error
	called   bool
}

func (f *fakeRecorder) Record(deviceID, path string, o status.Outcome, diff int, err error) {
	f.deviceID, f.path, f.outcome, f.diff, f.err, f.called = deviceID, path, o, diff, err, true
}

func newCtrlWithRecorder(mr reconcile.Reconciler, rec status.Recorder) *DefaultController {
	q := queue.NewRateLimitingQueue(queue.DefaultRateLimiter())
	c := New("test", nil, mr, q, nil, 1)
	c.SetStatusRecorder(rec)
	return c
}

func TestProcess_RecordsConverged(t *testing.T) {
	req := reconcile.Request{DeviceID: "10.0.0.1", Path: "/vlans"}
	mr := &MockReconciler{}
	mr.On("Reconcile", mock.Anything, req).Return(reconcile.Result{Requeue: false, Changes: 0})
	rec := &fakeRecorder{}
	newCtrlWithRecorder(mr, rec).process(context.Background(), req)

	assert.True(t, rec.called)
	assert.Equal(t, status.OutcomeConverged, rec.outcome)
	assert.Equal(t, 0, rec.diff)
	assert.Equal(t, "10.0.0.1", rec.deviceID)
	assert.Equal(t, "/vlans", rec.path)
}

func TestProcess_RecordsDrifted(t *testing.T) {
	req := reconcile.Request{DeviceID: "10.0.0.1", Path: "/ifm"}
	mr := &MockReconciler{}
	mr.On("Reconcile", mock.Anything, req).Return(reconcile.Result{Requeue: false, Changes: 2})
	rec := &fakeRecorder{}
	newCtrlWithRecorder(mr, rec).process(context.Background(), req)

	assert.Equal(t, status.OutcomeDrifted, rec.outcome)
	assert.Equal(t, 2, rec.diff)
}

func TestProcess_RecordsReconciling(t *testing.T) {
	req := reconcile.Request{DeviceID: "10.0.0.1", Path: "/vlans"}
	mr := &MockReconciler{}
	mr.On("Reconcile", mock.Anything, req).Return(reconcile.Result{Requeue: true}).Once()
	rec := &fakeRecorder{}
	newCtrlWithRecorder(mr, rec).process(context.Background(), req)

	assert.Equal(t, status.OutcomeReconciling, rec.outcome)
}

func TestProcess_RecordsError(t *testing.T) {
	req := reconcile.Request{DeviceID: "10.0.0.1", Path: "/vlans"}
	mr := &MockReconciler{}
	mr.On("Reconcile", mock.Anything, req).Return(reconcile.Result{
		Requeue: true,
		Error:   errors.New("session timeout"),
	}).Once()
	rec := &fakeRecorder{}
	newCtrlWithRecorder(mr, rec).process(context.Background(), req)

	assert.Equal(t, status.OutcomeError, rec.outcome)
	assert.Error(t, rec.err)
}

// Without a recorder set, process must not panic (R08 degradation).
func TestProcess_NilRecorderNoPanic(t *testing.T) {
	req := reconcile.Request{DeviceID: "10.0.0.1", Path: "/vlans"}
	mr := &MockReconciler{}
	mr.On("Reconcile", mock.Anything, req).Return(reconcile.Result{Requeue: false})
	q := queue.NewRateLimitingQueue(queue.DefaultRateLimiter())
	c := New("test", nil, mr, q, nil, 1)

	assert.NotPanics(t, func() { c.process(context.Background(), req) })
}

// DefaultController must satisfy status.RecorderSetter so the Manager can inject.
func TestDefaultController_IsRecorderSetter(t *testing.T) {
	var _ status.RecorderSetter = (*DefaultController)(nil)
}
