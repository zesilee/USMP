// Package networkinstance reconciles 华为 network-instance L3VPN 实例配置
// （/ni:network-instance，容器根 + 嵌套 list）between desired state and actual
// device state。结构对齐 bgp reconciler（容器根收敛模式）——差异仅在 path 与
// GoStruct 类型。network-instance 是 BGP 二期 peering 的唯一硬前置（peers/afs/
// peer-groups 均 augment 于此根下），本期只驱动原生 config-true 字段（global +
// instance name/description），augment 子树保持 nil、不下发（design D1/D2/D3）。
package networkinstance

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

// NetworkInstancePath 是 network-instance 配置根路径（描述符谓词以此锚定）。
const NetworkInstancePath = "/ni:network-instance"

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
	// 容器根收敛为单条整根 change（同 BGP，区别于 VLAN/IFM 的 list 根）：
	// network-instance 生成物是子容器（global）+ 嵌套 list（instances/instance）、
	// 无根 list，细粒度 diff 会把每个顶层子容器各自作为 change value 发出——而
	// client 的 XMLEncoderForValue 只登记根类型 *HuaweiNetworkInstance_NetworkInstance，
	// 匹配不到子容器 → 落 xml.Marshal 兜底发 Go 类型名（设备树对不上、回读解不出 →
	// 永久漂移）。故有任一漂移即收敛为「下发整个 desired /ni:network-instance」：经
	// 描述符 xmlcodec container 模式编码为 <network-instance>…，edit-config merge 收敛。
	return []reconcile.Change{{
		Type:         "MODIFY", // 对齐 deviceClient.Set 的类型 switch → edit-config merge
		Path:         path,
		DesiredValue: desired,
		ActualValue:  actual,
	}}, nil
}

// NetworkInstanceReconciler reconciles network-instance configuration.
type NetworkInstanceReconciler struct {
	*reconcile.GenericReconciler
}

// New creates a NetworkInstanceReconciler. resolver is the shared DeviceStore for
// per-device connection info; nil / unregistered degrades to AUTO/no-credential (R08).
func New(cs reconcile.ConfigStore, clientPool client.ClientPool, resolver device.Store) *NetworkInstanceReconciler {
	dc := &deviceClient{clientPool: clientPool, resolver: resolver}
	de := &diffEngineAdapter{de: diff.NewDefaultDiffEngine()}
	return &NetworkInstanceReconciler{GenericReconciler: reconcile.NewGenericReconciler(cs, dc, de)}
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

// Get retrieves the actual network-instance config as *HuaweiNetworkInstance_NetworkInstance.
func (d *deviceClient) Get(ctx context.Context, deviceID string) (interface{}, error) {
	c, err := d.clientPool.Get(d.resolveConn(deviceID))
	if err != nil {
		return nil, err
	}
	result, err := c.Get(ctx, NetworkInstancePath)
	if err != nil {
		return nil, err
	}

	switch data := result.Data.(type) {
	case []byte:
		if len(data) > 0 && data[0] == '<' {
			// NETCONF get-config XML：经驱动描述符注册表 container 模式解码
			// （DR-03/XC-05，全字段填充；直接 xml.Unmarshal 会得空 actual → 永远漂移）。
			dec, ok := yangdriver.DecoderFor(NetworkInstancePath)
			if !ok {
				return nil, fmt.Errorf("no XML decoder registered for network-instance readback")
			}
			return dec.DecodeXML(data)
		}
		// gNMI JSON
		deviceRoot := &huawei.Device{}
		if err := json.Unmarshal(data, deviceRoot); err != nil {
			return nil, fmt.Errorf("unmarshal JSON failed: %w", err)
		}
		if deviceRoot.NetworkInstance == nil {
			return &huawei.HuaweiNetworkInstance_NetworkInstance{}, nil
		}
		return deviceRoot.NetworkInstance, nil
	}

	if ni, ok := result.Data.(*huawei.HuaweiNetworkInstance_NetworkInstance); ok {
		return ni, nil
	}
	if deviceRoot, ok := result.Data.(*huawei.Device); ok && deviceRoot.NetworkInstance != nil {
		return deviceRoot.NetworkInstance, nil
	}
	return nil, fmt.Errorf("unknown data format for network-instance config: %T", result.Data)
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
