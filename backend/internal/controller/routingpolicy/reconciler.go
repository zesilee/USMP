// Package routingpolicy reconciles 华为 routing-policy 配置（/rtp:routing-policy，容器根）
// between desired state and actual device state。结构对齐 xpl/tunnelmgmt/bgp reconciler
// （GenericReconciler + deviceClient + diffEngineAdapter，容器根单条整根 change），差异仅在
// path 与 GoStruct 类型。routing-policy 是 BGP 2b import/export route-policy leafref 的目标
// 模型——越序禁令要求先接成可配模型，policy-definition 实例存在后 BGP 引用才合法。本波次
// 功能面仅 policy-definitions/policy-definition 标量边界。
package routingpolicy

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	_ "github.com/leezesi/usmp/backend/internal/drivers" // 描述符注册（回读解码经注册表，XC-04/05）
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/diff"
	yangdriver "github.com/leezesi/usmp/backend/pkg/yang-runtime/driver"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// RoutingPolicyPath 是 routing-policy 配置根路径（描述符谓词以此锚定，见 internal/drivers）。
const RoutingPolicyPath = "/rtp:routing-policy"

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
	// 容器根收敛为单条整根 change（同 bgp/tunnelmgmt/xpl）：rtp 生成物是标量+子容器、无根
	// list，细粒度 diff 会把每个顶层子容器各自作为 change value 发出——而 client 的
	// XMLEncoderForValue 只登记根类型 *HuaweiRoutingPolicy_RoutingPolicy，匹配不到子容器 →
	// 落 xml.Marshal 兜底发出 Go 类型名（设备树对不上 → 永久漂移）。故有任一漂移即收敛为
	// 「下发整个 desired /rtp:routing-policy」：经描述符 xmlcodec container 模式编码为
	// <routing-policy>…，edit-config merge 语义使其收敛。
	return []reconcile.Change{{
		Type:         "MODIFY",
		Path:         path,
		DesiredValue: desired,
		ActualValue:  actual,
	}}, nil
}

// RoutingPolicyReconciler reconciles routing-policy configuration.
type RoutingPolicyReconciler struct {
	*reconcile.GenericReconciler
}

// New creates a RoutingPolicyReconciler. resolver is the shared DeviceStore for per-device
// connection info; nil / unregistered degrades to AUTO/no-credential (R08).
func New(cs reconcile.ConfigStore, clientPool client.ClientPool, resolver device.Store) *RoutingPolicyReconciler {
	dc := &deviceClient{clientPool: clientPool, resolver: resolver}
	de := &diffEngineAdapter{de: diff.NewDefaultDiffEngine()}
	return &RoutingPolicyReconciler{GenericReconciler: reconcile.NewGenericReconciler(cs, dc, de)}
}

type deviceClient struct {
	clientPool client.ClientPool
	resolver   device.Store
}

func (d *deviceClient) resolveConn(deviceID string) client.DeviceConnectionInfo {
	if d.resolver != nil {
		if info, ok := d.resolver.Get(deviceID); ok {
			return info
		}
	}
	log.Printf("[routingpolicy] device %q not registered in DeviceStore; using AUTO/no-credential connection", deviceID)
	return client.DeviceConnectionInfo{IP: deviceID, Protocol: client.ProtocolAUTO}
}

// Get retrieves the actual rtp config as *HuaweiRoutingPolicy_RoutingPolicy.
func (d *deviceClient) Get(ctx context.Context, deviceID string) (interface{}, error) {
	c, err := d.clientPool.Get(d.resolveConn(deviceID))
	if err != nil {
		return nil, err
	}
	result, err := c.Get(ctx, RoutingPolicyPath)
	if err != nil {
		return nil, err
	}

	switch data := result.Data.(type) {
	case []byte:
		if len(data) > 0 && data[0] == '<' {
			// NETCONF get-config XML：经驱动描述符注册表 container 模式解码（DR-03/XC-05，
			// 全字段填充；直接 xml.Unmarshal 会得空 actual → 永远漂移）。
			dec, ok := yangdriver.DecoderFor(RoutingPolicyPath)
			if !ok {
				return nil, fmt.Errorf("no XML decoder registered for routing-policy readback")
			}
			return dec.DecodeXML(data)
		}
		// gNMI JSON
		deviceRoot := &huawei.Device{}
		if err := json.Unmarshal(data, deviceRoot); err != nil {
			return nil, fmt.Errorf("unmarshal JSON failed: %w", err)
		}
		if deviceRoot.RoutingPolicy == nil {
			return &huawei.HuaweiRoutingPolicy_RoutingPolicy{}, nil
		}
		return deviceRoot.RoutingPolicy, nil
	}

	if rp, ok := result.Data.(*huawei.HuaweiRoutingPolicy_RoutingPolicy); ok {
		return rp, nil
	}
	if deviceRoot, ok := result.Data.(*huawei.Device); ok && deviceRoot.RoutingPolicy != nil {
		return deviceRoot.RoutingPolicy, nil
	}
	return nil, fmt.Errorf("unknown data format for routing-policy config: %T", result.Data)
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
