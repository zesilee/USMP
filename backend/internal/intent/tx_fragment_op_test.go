package intent

import (
	"context"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/internal/generated/native/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/object"
)

// recordingTxClient 记录 prepare 发出的 Set 调用（CS-04 Fragment Op 映射防线）。
type recordingTxClient struct {
	calls []recordedSet
}

type recordedSet struct {
	changes []client.Change
	opts    client.SetOptions
}

func (r *recordingTxClient) Set(ctx context.Context, changes []client.Change, opts ...client.SetOption) (*client.SetResult, error) {
	var so client.SetOptions
	for _, o := range opts {
		o.Apply(&so)
	}
	r.calls = append(r.calls, recordedSet{changes: changes, opts: so})
	return &client.SetResult{Success: true}, nil
}

func (r *recordingTxClient) Get(ctx context.Context, path string, opts ...client.GetOption) (*client.GetResult, error) {
	return &client.GetResult{}, nil
}
func (r *recordingTxClient) ExecuteRPC(ctx context.Context, ns, name string, in []client.RPCInput) (*client.RPCResult, error) {
	return nil, nil
}
func (r *recordingTxClient) Subscribe(ctx context.Context, path string, h func(client.Notification)) error {
	return nil
}
func (r *recordingTxClient) Close() error                               { return nil }
func (r *recordingTxClient) IsConnected() bool                          { return true }
func (r *recordingTxClient) DiscardCandidate(ctx context.Context) error { return nil }
func (r *recordingTxClient) ConfirmCommit(ctx context.Context) error    { return nil }
func (r *recordingTxClient) CommitConfirmed(ctx context.Context, d time.Duration) error {
	return nil
}

// TestPrepareFragmentOpMapping：Fragment 三形态映射（CS-04）——
// 缺省(merge)→AddChange、Op=delete→DeleteChange(OldValue)、RawXML→字符串
// 透传 AddChange；全部 WithCommit(false)（candidate 两阶段前置）。
func TestPrepareFragmentOpMapping(t *testing.T) {
	mergeCfg := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{10: {Id: object.Uint16(10)}},
	}
	delCfg := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{20: {Id: object.Uint16(20)}},
	}
	rec := &recordingTxClient{}
	tc := NewTxCoordinator(nil, nil, time.Second)

	frags := []Fragment{
		{Device: "d", Module: "vlan", Path: VlanPath, Config: mergeCfg},
		{Device: "d", Module: "vlan", Path: VlanPath, Config: delCfg, Op: FragmentOpDelete},
		{Device: "d", Module: "vlan", Path: VlanPath, RawXML: "<pre-encoded/>"},
	}
	if err := tc.prepare(context.Background(), rec, frags); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if len(rec.calls) != 3 {
		t.Fatalf("want 3 Set calls, got %d", len(rec.calls))
	}
	for i, call := range rec.calls {
		if call.opts.Commit {
			t.Errorf("call %d must be WithCommit(false)", i)
		}
		if len(call.changes) != 1 {
			t.Fatalf("call %d: want 1 change, got %d", i, len(call.changes))
		}
	}
	// 缺省 merge → AddChange，NewValue=Config（既有语义零变化，BIO-03 回归锚点）
	c0 := rec.calls[0].changes[0]
	if c0.Type != client.AddChange || c0.NewValue == nil || c0.OldValue != nil {
		t.Errorf("merge fragment mapped wrong: %+v", c0)
	}
	// Op=delete → DeleteChange，目标在 OldValue（与 marshalDeleteChange 契约一致）
	c1 := rec.calls[1].changes[0]
	if c1.Type != client.DeleteChange {
		t.Errorf("delete fragment must map to DeleteChange, got %v", c1.Type)
	}
	if c1.OldValue != delCfg {
		t.Errorf("delete fragment OldValue must carry the keyed target, got %+v", c1.OldValue)
	}
	// RawXML → 字符串透传（EncodeChangeXML passthrough 通道，叶级删除预编码用）
	c2 := rec.calls[2].changes[0]
	if c2.Type != client.AddChange {
		t.Errorf("raw fragment must map to AddChange, got %v", c2.Type)
	}
	if s, ok := c2.NewValue.(string); !ok || s != "<pre-encoded/>" {
		t.Errorf("raw fragment must pass string through, got %T %v", c2.NewValue, c2.NewValue)
	}
}

// TestPrepareFragmentOpUnknown：未知 Op 明确报错，不猜语义（R08）。
func TestPrepareFragmentOpUnknown(t *testing.T) {
	rec := &recordingTxClient{}
	tc := NewTxCoordinator(nil, nil, time.Second)
	err := tc.prepare(context.Background(), rec, []Fragment{
		{Device: "d", Path: VlanPath, Config: &huawei.HuaweiVlan_Vlan_Vlans{}, Op: "replace"},
	})
	if err == nil {
		t.Fatal("want error for unknown fragment op")
	}
	if len(rec.calls) != 0 {
		t.Errorf("unknown op must not reach the device, got %d Set calls", len(rec.calls))
	}
}

// TestPrepareFragmentDeleteNilConfig：delete 片段缺目标必须报错，绝不发送
// 无目标删除（R08，与 marshalDeleteChange 同口径）。
func TestPrepareFragmentDeleteNilConfig(t *testing.T) {
	rec := &recordingTxClient{}
	tc := NewTxCoordinator(nil, nil, time.Second)
	err := tc.prepare(context.Background(), rec, []Fragment{
		{Device: "d", Path: VlanPath, Op: FragmentOpDelete},
	})
	if err == nil {
		t.Fatal("want error for delete fragment without target")
	}
	if len(rec.calls) != 0 {
		t.Errorf("nil-target delete must not reach the device, got %d Set calls", len(rec.calls))
	}
}
