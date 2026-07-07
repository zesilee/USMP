package controller

import (
	"context"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// spyQueue records which requeue path process() takes without real scheduling.
type spyQueue struct {
	rateLimited []interface{}
	afterItems  []interface{}
	forgotten   []interface{}
}

func (q *spyQueue) Add(interface{})                         {}
func (q *spyQueue) Len() int                                { return 0 }
func (q *spyQueue) Get() (interface{}, bool)                { return nil, true }
func (q *spyQueue) Done(interface{})                        {}
func (q *spyQueue) ShutDown()                               {}
func (q *spyQueue) ShutDownWithDrain()                      {}
func (q *spyQueue) ShuttingDown() bool                      { return false }
func (q *spyQueue) AddAfter(i interface{}, _ time.Duration) { q.afterItems = append(q.afterItems, i) }
func (q *spyQueue) AddRateLimited(i interface{})            { q.rateLimited = append(q.rateLimited, i) }
func (q *spyQueue) Forget(i interface{})                    { q.forgotten = append(q.forgotten, i) }
func (q *spyQueue) NumRequeues(interface{}) int             { return 0 }

// TestProcess_DriftRequeuesReverify: after a corrective apply (Changes>0, no
// error), the controller must requeue a re-verify so the recorded outcome can
// settle from "drifted" to "converged". Without this the device stays "drifted"
// forever, because the periodic source is wired with no devices (a no-op). This
// is the root cause of the reported "新增接口后设备一直显示已漂移".
func TestProcess_DriftRequeuesReverify(t *testing.T) {
	req := reconcile.Request{DeviceID: "192.168.1.1", Path: "/ifm:ifm/ifm:interfaces"}
	mr := &MockReconciler{}
	mr.On("Reconcile", mock.Anything, req).Return(reconcile.Result{Requeue: false, Changes: 1})

	q := &spyQueue{}
	c := New("test", nil, mr, q, nil, 1)
	c.process(context.Background(), req)

	assert.Contains(t, q.rateLimited, interface{}(req), "下发有变更后必须 requeue 复验，否则状态永远停在 drifted")
	assert.NotContains(t, q.forgotten, interface{}(req), "有待复验的变更不应被 Forget")
}

// TestProcess_ConvergedForgetsNoRequeue: a converged reconcile (Changes==0) must
// NOT requeue — it forgets the item, so the re-verify chain terminates instead of
// looping forever.
func TestProcess_ConvergedForgetsNoRequeue(t *testing.T) {
	req := reconcile.Request{DeviceID: "192.168.1.1", Path: "/ifm:ifm/ifm:interfaces"}
	mr := &MockReconciler{}
	mr.On("Reconcile", mock.Anything, req).Return(reconcile.Result{Requeue: false, Changes: 0})

	q := &spyQueue{}
	c := New("test", nil, mr, q, nil, 1)
	c.process(context.Background(), req)

	assert.Contains(t, q.forgotten, interface{}(req))
	assert.Empty(t, q.rateLimited, "收敛后不应再入队，避免复验死循环")
}
