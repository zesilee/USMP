// Package xpl reconciles 华为 xpl 配置（/xpl:xpl，容器根）between desired state and
// actual device state。结构对齐 tunnelmgmt/bgp reconciler（GenericReconciler +
// deviceClient + diffEngineAdapter，容器根单条整根 change），差异仅在 path 与 GoStruct
// 类型。xpl 是 BGP 2b route-filter leafref 的目标模型——越序禁令要求先接成可配模型，
// route-filter 实例存在后 BGP 引用才合法。本波次功能面仅 route-filters/route-filter。
package xpl

import (
	"context"
	"encoding/json"
	"fmt"

	_ "github.com/leezesi/usmp/backend/internal/drivers" // 描述符注册（回读解码经注册表，XC-04/05）
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/diff"
	yangdriver "github.com/leezesi/usmp/backend/pkg/yang-runtime/driver"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// XplPath 是 xpl 配置根路径（描述符谓词以此锚定，见 internal/drivers）。
const XplPath = "/xpl:xpl"

// diffEngineAdapter adapts diff.DiffEngine to reconcile.DiffEngine.
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
	// 容器根收敛为单条整根 change（同 bgp/tunnelmgmt）：xpl 生成物是标量+子容器、无根 list，
	// 细粒度 diff 会把每个顶层子容器各自作为 change value 发出——而 client 的
	// XMLEncoderForValue 只登记根类型 *HuaweiXpl_Xpl，匹配不到子容器 → 落 xml.Marshal 兜底
	// 发出 Go 类型名（设备树对不上 → 永久漂移）。故有任一漂移即收敛为「下发整个 desired
	// /xpl:xpl」：经描述符 xmlcodec container 模式编码为 <xpl>…，edit-config merge 收敛。
	return []reconcile.Change{{
		Type:         "MODIFY",
		Path:         path,
		DesiredValue: desired,
		ActualValue:  actual,
	}}, nil
}

// XplReconciler reconciles xpl configuration.
type XplReconciler struct {
	*reconcile.GenericReconciler
}

// New creates an XplReconciler. resolver is the shared DeviceStore for per-device
// connection info; nil / unregistered degrades to AUTO/no-credential (R08).
func New(cs reconcile.ConfigStore, clientPool client.ClientPool, resolver device.Store) *XplReconciler {
	dc := &deviceClient{clientPool: clientPool, resolver: resolver}
	de := &diffEngineAdapter{de: diff.NewDefaultDiffEngine()}
	return &XplReconciler{GenericReconciler: reconcile.NewGenericReconciler(cs, dc, de)}
}

type deviceClient struct {
	clientPool client.ClientPool
	resolver   device.Store
}

// resolveConn delegates to the shared device.ResolveConn helper (DS-06):
// registered devices use stored info, unregistered degrade to
// AUTO/no-credential (R08).
func (d *deviceClient) resolveConn(deviceID string) client.DeviceConnectionInfo {
	info, _ := device.ResolveConn(d.resolver, deviceID)
	return info
}

// Get retrieves the actual xpl config and returns it as *HuaweiXpl_Xpl.
func (d *deviceClient) Get(ctx context.Context, deviceID string) (interface{}, error) {
	c, err := d.clientPool.Get(d.resolveConn(deviceID))
	if err != nil {
		return nil, err
	}
	result, err := c.Get(ctx, XplPath)
	if err != nil {
		return nil, err
	}

	switch data := result.Data.(type) {
	case []byte:
		if len(data) > 0 && data[0] == '<' {
			// NETCONF get-config XML：经驱动描述符注册表 container 模式解码（DR-03/XC-05，
			// 全字段填充；直接 xml.Unmarshal 会得空 actual → 永远漂移）。
			dec, ok := yangdriver.DecoderFor(XplPath)
			if !ok {
				return nil, fmt.Errorf("no XML decoder registered for xpl readback")
			}
			return dec.DecodeXML(data)
		}
		// gNMI JSON
		deviceRoot := &huawei.Device{}
		if err := json.Unmarshal(data, deviceRoot); err != nil {
			return nil, fmt.Errorf("unmarshal JSON failed: %w", err)
		}
		if deviceRoot.Xpl == nil {
			return &huawei.HuaweiXpl_Xpl{}, nil
		}
		return deviceRoot.Xpl, nil
	}

	if xp, ok := result.Data.(*huawei.HuaweiXpl_Xpl); ok {
		return xp, nil
	}
	if deviceRoot, ok := result.Data.(*huawei.Device); ok && deviceRoot.Xpl != nil {
		return deviceRoot.Xpl, nil
	}
	return nil, fmt.Errorf("unknown data format for xpl config: %T", result.Data)
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
		case "MODIFY":
			ct = client.ModifyChange
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
