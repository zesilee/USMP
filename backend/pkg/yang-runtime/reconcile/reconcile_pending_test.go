package reconcile

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// trackingStore: 实现 ConfigStore + SyncTracker 的可控替身（YR-09 逻辑用例）。
// 真实过期行为由 manager 包用真 InMemoryConfigStore 的回归用例覆盖。
type trackingStore struct {
	val          interface{}
	gen          uint64
	pendingSince time.Time
	pending      bool
	markedGen    []uint64
	abandonedGen []uint64
	// mutateOnTrack 模拟「对账读入后用户又写了一版」：Track 之后写代跳变。
	mutateOnTrack bool
}

func (s *trackingStore) Get(string, string) (interface{}, error) { return s.val, nil }
func (s *trackingStore) Set(_, _ string, v interface{}) error {
	s.val, s.gen, s.pending, s.pendingSince = v, s.gen+1, true, time.Now()
	return nil
}
func (s *trackingStore) Delete(string, string) error    { s.val = nil; return nil }
func (s *trackingStore) List(string) ([]string, error)  { return nil, nil }
func (s *trackingStore) ListDevices() ([]string, error) { return nil, nil }
func (s *trackingStore) Track(string, string) (uint64, time.Time, bool, bool) {
	if s.val == nil {
		return 0, time.Time{}, false, false
	}
	g := s.gen
	if s.mutateOnTrack {
		s.gen++
	}
	return g, s.pendingSince, s.pending, true
}
func (s *trackingStore) MarkSynced(_, _ string, gen uint64) bool {
	s.markedGen = append(s.markedGen, gen)
	if gen != s.gen {
		return false
	}
	s.pending = false
	return true
}
func (s *trackingStore) Abandon(_, _ string, gen uint64) bool {
	s.abandonedGen = append(s.abandonedGen, gen)
	if gen != s.gen {
		return false
	}
	s.val = nil
	return true
}

func pendingStore(v interface{}, since time.Time) *trackingStore {
	return &trackingStore{val: v, gen: 1, pending: true, pendingSince: since}
}

var req = Request{DeviceID: "10.0.0.1", Path: "/vlan"}

func TestPending_ConvergedMarksSyncedWithGen(t *testing.T) {
	cs := pendingStore("desired", time.Now())
	dc := &MockDeviceClient{}
	de := &MockDiffEngine{}
	dc.On("Get", mock.Anything, req.DeviceID).Return("actual", nil)
	de.On("Diff", "desired", "actual", req.Path).Return([]Change{}, nil)

	res := NewGenericReconciler(cs, dc, de).Reconcile(context.Background(), req)

	assert.False(t, res.Requeue)
	assert.Nil(t, res.Error)
	assert.Equal(t, []uint64{1}, cs.markedGen, "零变更须以读入时的 gen 标记已同步")
	assert.False(t, cs.pending)
}

func TestPending_ChangesAppliedDoesNotMarkYet(t *testing.T) {
	cs := pendingStore("desired", time.Now())
	dc := &MockDeviceClient{}
	de := &MockDiffEngine{}
	dc.On("Get", mock.Anything, req.DeviceID).Return("actual", nil)
	de.On("Diff", "desired", "actual", req.Path).Return([]Change{{Path: "/x"}}, nil)
	dc.On("Set", mock.Anything, req.DeviceID, mock.Anything).Return(nil)

	res := NewGenericReconciler(cs, dc, de).Reconcile(context.Background(), req)

	assert.Equal(t, 1, res.Changes)
	assert.Empty(t, cs.markedGen, "首轮下发不标记，由 YR-05 复验确认")
	assert.True(t, cs.pending)
}

func TestPending_RewrittenDuringReconcileNotMarked(t *testing.T) {
	cs := pendingStore("desired", time.Now())
	cs.mutateOnTrack = true // 读入 gen=1 后用户改成 gen=2
	dc := &MockDeviceClient{}
	de := &MockDiffEngine{}
	dc.On("Get", mock.Anything, req.DeviceID).Return("actual", nil)
	de.On("Diff", "desired", "actual", req.Path).Return([]Change{}, nil)

	NewGenericReconciler(cs, dc, de).Reconcile(context.Background(), req)

	assert.Equal(t, []uint64{1}, cs.markedGen)
	assert.True(t, cs.pending, "写代不匹配：新值仍待同步")
}

func TestPending_FailureWithinLimitRequeuesKeepsDesired(t *testing.T) {
	SetAbandonAfter(time.Hour)
	defer SetAbandonAfter(0)
	cs := pendingStore("desired", time.Now())
	dc := &MockDeviceClient{}
	dc.On("Get", mock.Anything, req.DeviceID).Return(nil, errors.New("device slow"))

	res := NewGenericReconciler(cs, dc, &MockDiffEngine{}).Reconcile(context.Background(), req)

	assert.True(t, res.Requeue)
	assert.False(t, res.Terminal)
	assert.NotNil(t, res.Error)
	assert.Empty(t, cs.abandonedGen)
	assert.Equal(t, "desired", cs.val)
}

func TestPending_FailurePastLimitAbandons(t *testing.T) {
	SetAbandonAfter(50 * time.Millisecond)
	defer SetAbandonAfter(0)
	cs := pendingStore("desired", time.Now().Add(-time.Second))

	for name, setup := range map[string]func(dc *MockDeviceClient, de *MockDiffEngine){
		"回读失败": func(dc *MockDeviceClient, de *MockDiffEngine) {
			dc.On("Get", mock.Anything, req.DeviceID).Return(nil, errors.New("unreachable"))
		},
		"diff失败": func(dc *MockDeviceClient, de *MockDiffEngine) {
			dc.On("Get", mock.Anything, req.DeviceID).Return("actual", nil)
			de.On("Diff", "desired", "actual", req.Path).Return([]Change{}, errors.New("diff"))
		},
		"下发失败": func(dc *MockDeviceClient, de *MockDiffEngine) {
			dc.On("Get", mock.Anything, req.DeviceID).Return("actual", nil)
			de.On("Diff", "desired", "actual", req.Path).Return([]Change{{Path: "/x"}}, nil)
			dc.On("Set", mock.Anything, req.DeviceID, mock.Anything).Return(errors.New("rejected"))
		},
	} {
		t.Run(name, func(t *testing.T) {
			store := *cs
			store.abandonedGen = nil
			dc, de := &MockDeviceClient{}, &MockDiffEngine{}
			setup(dc, de)

			res := NewGenericReconciler(&store, dc, de).Reconcile(context.Background(), req)

			assert.True(t, res.Terminal, "超限放弃须为终态")
			assert.False(t, res.Requeue)
			if assert.NotNil(t, res.Error) {
				assert.True(t, errors.Is(res.Error, ErrDesiredAbandoned))
				assert.Contains(t, res.Error.Error(), "已放弃")
			}
			assert.Equal(t, []uint64{1}, store.abandonedGen)
			assert.Nil(t, store.val, "放弃后 desired 删除")
		})
	}
}

func TestPending_AbandonGenMismatchFallsBackToRequeue(t *testing.T) {
	SetAbandonAfter(time.Millisecond)
	defer SetAbandonAfter(0)
	cs := pendingStore("desired", time.Now().Add(-time.Second))
	cs.mutateOnTrack = true
	dc := &MockDeviceClient{}
	dc.On("Get", mock.Anything, req.DeviceID).Return(nil, errors.New("unreachable"))

	res := NewGenericReconciler(cs, dc, &MockDiffEngine{}).Reconcile(context.Background(), req)

	assert.False(t, res.Terminal, "新值不属于本轮，不得终态")
	assert.True(t, res.Requeue)
	assert.Equal(t, "desired", cs.val)
}

func TestPending_NilDesiredReportsNoDesired(t *testing.T) {
	cs := &trackingStore{}
	res := NewGenericReconciler(cs, &MockDeviceClient{}, &MockDiffEngine{}).Reconcile(context.Background(), req)
	assert.True(t, res.NoDesired)
	assert.False(t, res.Requeue)
	assert.Nil(t, res.Error)
}

// 不实现 SyncTracker 的 ConfigStore（既有 MockConfigStore）：行为与改前一致。
func TestPending_PlainStoreUnaffected(t *testing.T) {
	SetAbandonAfter(time.Millisecond)
	defer SetAbandonAfter(0)
	cs := &MockConfigStore{}
	cs.On("Get", req.DeviceID, req.Path).Return("desired", nil)
	dc := &MockDeviceClient{}
	dc.On("Get", mock.Anything, req.DeviceID).Return(nil, errors.New("unreachable"))

	res := NewGenericReconciler(cs, dc, &MockDiffEngine{}).Reconcile(context.Background(), req)

	assert.True(t, res.Requeue)
	assert.False(t, res.Terminal)
	cs.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestAbandonAfter_EnvParsing(t *testing.T) {
	defer SetAbandonAfter(0)

	assert.Equal(t, DefaultAbandonAfter, parseAbandonAfter(""))
	assert.Equal(t, 200*time.Millisecond, parseAbandonAfter("200ms"))
	assert.Equal(t, DefaultAbandonAfter, parseAbandonAfter("banana"), "非法回退默认")
	assert.Equal(t, DefaultAbandonAfter, parseAbandonAfter("-5s"), "非正数回退默认")

	os.Setenv("USMP_DESIRED_ABANDON_AFTER", "200ms")
	defer os.Unsetenv("USMP_DESIRED_ABANDON_AFTER")
	SetAbandonAfter(0) // 0 = 重新按环境变量/默认解析
	assert.Equal(t, 200*time.Millisecond, AbandonAfter())
}
