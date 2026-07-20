// Package plainmodule 提供「单容器根、走通用 XML 引擎」模块的泛型 Reconciler
// （full-yang-onboarding design D4）：xpl/tunnelmgmt/routingpolicy/acl reconciler
// 完全同构（仅锚点与 GoStruct 类型异），收敛为按驱动描述符参数化的单实现——
// 新模块零 reconciler 代码，main.go 按注册表循环装配控制器。
//
// 本包 1:N 服务 drivers/huawei_modules.go plainModules 表内全部模块（57 个）；
// 仅 system/vlan/ifm/bgp/network-instance 五个形态特殊模块保留专属包。
// 完整映射关系与「加模块/拆专属包」操作守则见 ../README.md。
package plainmodule

import (
	"context"
	"fmt"
	"reflect"

	_ "github.com/leezesi/usmp/backend/internal/drivers" // 描述符注册（回读解码经注册表）
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/diff"
	yangdriver "github.com/leezesi/usmp/backend/pkg/yang-runtime/driver"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// diffEngineAdapter adapts diff.DiffEngine to reconcile.DiffEngine。
// 容器根收敛为单条整根 change（同 bgp/xpl 先例）：细粒度 diff 会把顶层子容器
// 各自作为 change value 发出，而 client 的 XMLEncoderForValue 只登记根类型，
// 匹配不到子容器 → 落 xml.Marshal 兜底发出 Go 类型名（设备树对不上 → 永久漂移）。
type diffEngineAdapter struct {
	de *diff.DefaultDiffEngine
}

func (a *diffEngineAdapter) Diff(desired, actual interface{}, path string) ([]reconcile.Change, error) {
	var s schema.Schema
	result, err := a.de.Diff(desired, actual, s)
	if err != nil {
		return nil, err
	}
	if len(result.Changes) == 0 {
		return nil, nil
	}
	return []reconcile.Change{{
		Type:         "MODIFY",
		Path:         path,
		DesiredValue: desired,
		ActualValue:  actual,
	}}, nil
}

// Reconciler reconciles one plain-container module keyed by its anchor path.
type Reconciler struct {
	*reconcile.GenericReconciler
	dc *deviceClient
	de *diffEngineAdapter
}

// New 构造泛型 reconciler：anchor 为模块规范根路径（如 "/ntp:ntp"），解码与
// 构造子经驱动描述符注册表按 anchor 查得。resolver 为共享 device store；
// nil/未注册降级 AUTO/无凭据（R08）。
func New(cs reconcile.ConfigStore, clientPool client.ClientPool, resolver device.Store, anchor string) *Reconciler {
	dc := &deviceClient{clientPool: clientPool, resolver: resolver, anchor: anchor}
	de := &diffEngineAdapter{de: diff.NewDefaultDiffEngine()}
	return &Reconciler{
		GenericReconciler: reconcile.NewGenericReconciler(cs, dc, de),
		dc:                dc,
		de:                de,
	}
}

// client/diff 暴露内部件给单测（不进公共 API 面）。
func (r *Reconciler) client() *deviceClient    { return r.dc }
func (r *Reconciler) diff() *diffEngineAdapter { return r.de }

type deviceClient struct {
	clientPool client.ClientPool
	resolver   device.Store
	anchor     string
}

func (d *deviceClient) resolveConn(deviceID string) client.DeviceConnectionInfo {
	info, _ := device.ResolveConn(d.resolver, deviceID)
	return info
}

// Get 以锚点取实际配置并经描述符注册表解码为模块 GoStruct。
func (d *deviceClient) Get(ctx context.Context, deviceID string) (interface{}, error) {
	c, err := d.clientPool.Get(d.resolveConn(deviceID))
	if err != nil {
		return nil, err
	}
	result, err := c.Get(ctx, d.anchor)
	if err != nil {
		return nil, err
	}

	dec, ok := yangdriver.DecoderFor(d.anchor)
	if !ok {
		return nil, fmt.Errorf("no XML decoder registered for %s", d.anchor)
	}
	if raw, isRaw := result.Data.([]byte); isRaw {
		if len(raw) == 0 {
			// 空回读 = 设备该模块无配置：空容器为合法初态
			return dec.NewStruct(), nil
		}
		if raw[0] == '<' {
			return dec.DecodeXML(raw)
		}
		// 非 XML 字节（如 gNMI JSON，规划能力）：显式报错而非静默空容器——
		// 静默空会让 diff 永远全量漂移、每周期整树重发（R08 要求诚实透出）。
		return nil, fmt.Errorf("unexpected non-XML readback for %s (len=%d)", d.anchor, len(raw))
	}
	if result.Data == nil {
		return dec.NewStruct(), nil
	}
	// 已解码（如 sim/测试注入）：仅接受本模块容器类型，异型显式报错
	if reflect.TypeOf(result.Data) == reflect.TypeOf(dec.NewStruct()) {
		return result.Data, nil
	}
	return nil, fmt.Errorf("unknown readback data format for %s: %T", d.anchor, result.Data)
}

// Set applies the computed changes to the device (candidate→commit).
func (d *deviceClient) Set(ctx context.Context, deviceID string, changes []reconcile.Change) error {
	c, err := d.clientPool.Get(d.resolveConn(deviceID))
	if err != nil {
		return err
	}
	clientChanges := make([]client.Change, len(changes))
	for i, rc := range changes {
		var ct client.ChangeType
		switch rc.Type {
		case "ADD":
			ct = client.AddChange
		case "DELETE":
			ct = client.DeleteChange
		default:
			ct = client.ModifyChange
		}
		clientChanges[i] = client.Change{
			Type:       ct,
			Path:       rc.Path,
			OldValue:   rc.ActualValue,
			NewValue:   rc.DesiredValue,
			SchemaPath: rc.Path,
		}
	}
	_, err = c.Set(ctx, clientChanges, client.WithCommit(true))
	return err
}
