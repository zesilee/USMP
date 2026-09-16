package controller

import (
	"context"
	"errors"
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newSpyCtrl(mr reconcile.Reconciler, rec status.Recorder) (*DefaultController, *spyQueue) {
	q := &spyQueue{}
	c := New("test", nil, mr, q, nil, 1)
	c.SetStatusRecorder(rec)
	return c, q
}

// YR-02：desired 读空 → 不记录任何结局（不冒充收敛、不覆盖上次真实结局）、Forget、不重投。
func TestProcess_NoDesired_NoRecordForgets(t *testing.T) {
	req := reconcile.Request{DeviceID: "10.0.0.1", Path: "/vlan"}
	mr := &MockReconciler{}
	mr.On("Reconcile", mock.Anything, req).Return(reconcile.Result{NoDesired: true})
	rec := &fakeRecorder{}
	c, q := newSpyCtrl(mr, rec)

	c.process(context.Background(), req)

	assert.False(t, rec.called, "读空不得记录结局")
	assert.Contains(t, q.forgotten, interface{}(req))
	assert.Empty(t, q.rateLimited)
	assert.Empty(t, q.afterItems)
}

// YR-04：Terminal error → 记 error、Forget、不重投。
func TestProcess_TerminalError_RecordsErrorForgets(t *testing.T) {
	req := reconcile.Request{DeviceID: "10.0.0.1", Path: "/vlan"}
	terr := &reconcile.ReconcileError{DeviceID: req.DeviceID, Path: req.Path, Err: errors.New("desired 已放弃")}
	mr := &MockReconciler{}
	mr.On("Reconcile", mock.Anything, req).Return(reconcile.Result{Terminal: true, Error: terr})
	rec := &fakeRecorder{}
	c, q := newSpyCtrl(mr, rec)

	c.process(context.Background(), req)

	assert.True(t, rec.called)
	assert.Equal(t, status.OutcomeError, rec.outcome)
	assert.Contains(t, rec.err.Error(), "已放弃")
	assert.Contains(t, q.forgotten, interface{}(req))
	assert.Empty(t, q.rateLimited, "终态错误不得重投")
	assert.Empty(t, q.afterItems)
}

// 防回归：非 Terminal 的 error 仍按 YR-04 退避重投。
func TestProcess_NonTerminalError_StillRequeues(t *testing.T) {
	req := reconcile.Request{DeviceID: "10.0.0.1", Path: "/vlan"}
	mr := &MockReconciler{}
	mr.On("Reconcile", mock.Anything, req).Return(reconcile.Result{Requeue: true, Error: errors.New("slow")})
	rec := &fakeRecorder{}
	c, q := newSpyCtrl(mr, rec)

	c.process(context.Background(), req)

	assert.Equal(t, status.OutcomeError, rec.outcome)
	assert.Contains(t, q.rateLimited, interface{}(req))
	assert.Empty(t, q.forgotten)
}
