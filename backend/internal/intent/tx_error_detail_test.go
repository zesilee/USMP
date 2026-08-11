package intent

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/internal/generated/native/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/object"
)

// rejectingTxClient 模拟真机拒绝 edit-config：Set 返回 per-change rpc-error 细节
// + 聚合错误（与 NETCONFClient.Set 的真实返回形态一致）。
type rejectingTxClient struct {
	recordingTxClient
	detail error
}

func (r *rejectingTxClient) Set(ctx context.Context, changes []client.Change, opts ...client.SetOption) (*client.SetResult, error) {
	res := &client.SetResult{
		Success: false,
		Changes: []client.ChangeResult{{Success: false, Error: r.detail}},
	}
	return res, errors.New("one or more changes failed to apply")
}

// 真机回归（T07，§9 诚实透出）：设备 rpc-error 细节（如「Unexpected element:
// statistic-mode」）必须出现在 prepare 的错误里——聚合错误「one or more changes
// failed to apply」会把根因吞掉，用户在界面上只能看到无信息量的失败。
// pushDeleteToDevice 已修过同款（per-change 错误优先），此测试把 2PC prepare
// 对齐到同一口径。
func TestPrepareSurfacesDeviceErrorDetail(t *testing.T) {
	detail := errors.New("netconfcore: rpc-error [unknown-element/error]: Unexpected element: statistic-mode.（共 1 条）")
	rej := &rejectingTxClient{detail: detail}
	tc := NewTxCoordinator(nil, nil, time.Second)

	cfg := &huawei.HuaweiIfm_Ifm_Interfaces{}
	err := tc.prepare(context.Background(), rej, []Fragment{
		{Device: "d", Module: "ifm", Path: IfmPath, Config: cfg},
	})
	if err == nil {
		t.Fatal("prepare 必须失败")
	}
	if !strings.Contains(err.Error(), "Unexpected element: statistic-mode") {
		t.Errorf("prepare 错误吞掉了设备 rpc-error 细节：%v", err)
	}
	if !errors.Is(err, detail) {
		t.Errorf("per-change 错误须以 %%w 包装保留错误链，got: %v", err)
	}
}

// 边界：Set 无 per-change 细节（result=nil + 传输层错误）→ 原错误原样透出。
func TestPrepareTransportErrorPassthrough(t *testing.T) {
	transport := &transportErrTxClient{err: errors.New("connection reset by peer")}
	tc := NewTxCoordinator(nil, nil, time.Second)
	err := tc.prepare(context.Background(), transport, []Fragment{
		{Device: "d", Module: "vlan", Path: VlanPath, Config: &huawei.HuaweiVlan_Vlan_Vlans{
			Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{10: {Id: object.Uint16(10)}},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "connection reset by peer") {
		t.Errorf("传输层错误须原样透出，got: %v", err)
	}
}

type transportErrTxClient struct {
	recordingTxClient
	err error
}

func (r *transportErrTxClient) Set(ctx context.Context, changes []client.Change, opts ...client.SetOption) (*client.SetResult, error) {
	return nil, r.err
}
