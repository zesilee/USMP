package reconcile

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// 锁定：desired 缺失时对账是空操作，**不得**推导出删除。
//
// 这不是 bug，是契约（code-todo-backlog B3 已定性）。
// spec business-intent-orchestration BIO-05 写死：删除走 DELETE 命令通道，
// 「声明式通道不承载删除」。
//
// 之所以值得专门加锁定用例：diff 引擎里确实存在 desiredNil && !actualNil →
// DeleteChange 的分支（diff.go），单看那段很容易得出「对账应该能删」的结论，
// 进而把这里的提前返回当成逻辑缺口"修"掉。真那么改，任何一次 desired 过期
// 或读取落空都会被翻译成删设备配置——是删真机配置级别的事故。
//
// 用例同时锁死两件事：不碰设备（不发起回读）、不进 diff。

func TestReconcileDesiredAbsentIsNoOp(t *testing.T) {
	cs := new(MockConfigStore)
	dc := new(MockDeviceClient)
	de := new(MockDiffEngine)

	req := Request{DeviceID: "10.0.0.1", Path: "/vlan:vlan/vlan:vlans"}
	cs.On("Get", req.DeviceID, req.Path).Return(nil, nil)

	res := NewGenericReconciler(cs, dc, de).Reconcile(context.Background(), req)

	assert.False(t, res.Requeue, "desired 缺失属稳态，不应重排队")
	assert.Nil(t, res.Error, "desired 缺失不是错误")

	// 关键断言：一步都不能往下走。
	dc.AssertNotCalled(t, "Get", mock.Anything, mock.Anything)
	de.AssertNotCalled(t, "Diff", mock.Anything, mock.Anything, mock.Anything)
	cs.AssertExpectations(t)
}

// 对照组：desired 存在时才进入「回读→diff」正常链路，
// 确保上面的空操作是 desired 缺失专属，而不是整个对账都被短路了。
func TestReconcileDesiredPresentProceedsToDiff(t *testing.T) {
	cs := new(MockConfigStore)
	dc := new(MockDeviceClient)
	de := new(MockDiffEngine)

	req := Request{DeviceID: "10.0.0.1", Path: "/vlan:vlan/vlan:vlans"}
	desired := map[string]interface{}{"id": 100}
	actual := map[string]interface{}{"id": 100}

	cs.On("Get", req.DeviceID, req.Path).Return(desired, nil)
	dc.On("Get", mock.Anything, req.DeviceID).Return(actual, nil)
	de.On("Diff", desired, actual, req.Path).Return([]Change{}, nil)

	NewGenericReconciler(cs, dc, de).Reconcile(context.Background(), req)

	dc.AssertExpectations(t)
	de.AssertExpectations(t)
}
