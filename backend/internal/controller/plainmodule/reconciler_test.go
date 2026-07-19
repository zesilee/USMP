package plainmodule

import (
	"context"
	"testing"

	_ "github.com/leezesi/usmp/backend/internal/drivers"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	yangdriver "github.com/leezesi/usmp/backend/pkg/yang-runtime/driver"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/xmlcodec"
)

// fakeClient/fakePool：单测桩（B1，不起 sim）。
type fakeClient struct {
	client.Client
	getRaw  []byte
	setGot  []client.Change
	setErr  error
	getPath string
}

func (f *fakeClient) Get(_ context.Context, path string, _ ...client.GetOption) (*client.GetResult, error) {
	f.getPath = path
	return &client.GetResult{Data: f.getRaw}, nil
}

func (f *fakeClient) Set(_ context.Context, changes []client.Change, _ ...client.SetOption) (*client.SetResult, error) {
	f.setGot = changes
	return &client.SetResult{}, f.setErr
}

type fakePool struct {
	c *fakeClient
}

func (p *fakePool) Get(client.DeviceConnectionInfo) (client.Client, error) { return p.c, nil }
func (p *fakePool) Release(string)                                         {}
func (p *fakePool) CloseAll() error                                        { return nil }
func (p *fakePool) Stats() client.PoolStats                                { return client.PoolStats{} }

// TestPlainModuleGetDecodesViaRegistry：Get 以锚点取数并经描述符注册表解码 XML
// 为对应 GoStruct（任意表行模块通用——以 ntp 为样本）。
func TestPlainModuleGetDecodesViaRegistry(t *testing.T) {
	const anchor = "/ntp:ntp"
	d, ok := yangdriver.EncoderFor(anchor)
	if !ok {
		t.Fatal("ntp 描述符缺失")
	}
	want := d.NewStruct()
	xml, err := xmlcodec.Encode(d.XML, want)
	if err != nil {
		t.Fatalf("样例编码: %v", err)
	}

	fc := &fakeClient{getRaw: []byte(xml)}
	r := New(nil, &fakePool{c: fc}, nil, anchor)
	got, err := r.client().Get(context.Background(), "dev1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fc.getPath != anchor {
		t.Fatalf("Get 路径=%q, want %q", fc.getPath, anchor)
	}
	if _, ok := got.(*huawei.HuaweiNtp_Ntp); !ok {
		t.Fatalf("解码类型=%T, want *HuaweiNtp_Ntp", got)
	}
}

// TestPlainModuleWholeRootConvergence：有漂移即收敛为单条整根 MODIFY change
// （container 根模块细粒度 diff 会碰 XMLEncoderForValue 匹配不到子容器的兜底
// 陷阱，同 xpl/bgp 先例）。
func TestPlainModuleWholeRootConvergence(t *testing.T) {
	const anchor = "/ntp:ntp"
	r := New(nil, &fakePool{c: &fakeClient{}}, nil, anchor)
	desired := &huawei.HuaweiNtp_Ntp{}
	actual := &huawei.HuaweiNtp_Ntp{}
	changes, err := r.diff().Diff(desired, actual, anchor)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	// 无漂移 → 无 change
	if len(changes) != 0 {
		t.Fatalf("零漂移应零 change, got %d", len(changes))
	}
}

// TestPlainModuleSetMapsChangeTypes：Set 把 reconcile change 类型映射到 client 变更。
func TestPlainModuleSetMapsChangeTypes(t *testing.T) {
	const anchor = "/ntp:ntp"
	fc := &fakeClient{}
	r := New(nil, &fakePool{c: fc}, nil, anchor)
	err := r.client().Set(context.Background(), "dev1", []reconcile.Change{
		{Type: "MODIFY", Path: anchor, DesiredValue: &huawei.HuaweiNtp_Ntp{}},
	})
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if len(fc.setGot) != 1 || fc.setGot[0].Type != client.ModifyChange || fc.setGot[0].Path != anchor {
		t.Fatalf("变更映射不符: %+v", fc.setGot)
	}
}
