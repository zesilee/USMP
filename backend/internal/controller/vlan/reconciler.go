package vlan

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	_ "github.com/leezesi/usmp/backend/internal/drivers" // 描述符注册（回读解码经注册表，XC-04）
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/diff"
	yangdriver "github.com/leezesi/usmp/backend/pkg/yang-runtime/driver"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// diffEngineAdapter adapts diff.DiffEngine to reconcile.DiffEngine interface
// because the diff package returns (*diff.DiffResult, error) while reconcile expects
// []reconcile.Change directly.
type diffEngineAdapter struct {
	de *diff.DefaultDiffEngine
}

// Diff implements the reconcile.DiffEngine interface
func (a *diffEngineAdapter) Diff(desired, actual interface{}, path string) ([]reconcile.Change, error) {
	var s schema.Schema = nil // not used since we have it from the manager schema loading
	result, err := a.de.Diff(desired, actual, s)
	if err != nil {
		return nil, err
	}
	// Convert diff.Change to reconcile.Change
	changes := make([]reconcile.Change, len(result.Changes))
	for i, c := range result.Changes {
		changes[i] = reconcile.Change{
			Path:         c.Path,
			Type:         c.Type.String(),
			DesiredValue: c.NewValue,
			ActualValue:  c.OldValue,
		}
	}
	return changes, nil
}

// VlanReconciler reconciles the VLAN configuration between desired state and actual device state.
// It uses the GenericReconciler base implementation that handles the common reconciliation pattern.
type VlanReconciler struct {
	*reconcile.GenericReconciler
}

// New creates a new VlanReconciler with the given dependencies
func New(cs reconcile.ConfigStore, clientPool client.ClientPool, resolver device.Store) *VlanReconciler {
	dc := &deviceClient{
		clientPool: clientPool,
		resolver:   resolver,
	}
	de := &diffEngineAdapter{
		de: diff.NewDefaultDiffEngine(),
	}
	return &VlanReconciler{
		GenericReconciler: reconcile.NewGenericReconciler(cs, dc, de),
	}
}

// deviceClient implements reconcile.DeviceClient interface for getting VLAN configuration from device
type deviceClient struct {
	clientPool client.ClientPool
	resolver   device.Store
}

// resolveConn resolves connection info via the shared DeviceStore (source of
// truth), falling back to parsing the DeviceID string when the device is not
// registered or no store is wired (legacy path, R08 degrade — no crash).
func (d *deviceClient) resolveConn(deviceID string) client.DeviceConnectionInfo {
	if d.resolver != nil {
		if info, ok := d.resolver.Get(deviceID); ok {
			return info
		}
	}
	// Unregistered device (or no store): degrade to an AUTO/no-credential
	// connection; authentication fails cleanly (R08) rather than crash.
	log.Printf("[vlan] device %q not registered in DeviceStore; using AUTO/no-credential connection", deviceID)
	return client.DeviceConnectionInfo{IP: deviceID, Protocol: client.ProtocolAUTO}
}

// Get retrieves the actual VLAN configuration from the device and converts it to the openconfig.VLans struct
func (d *deviceClient) Get(ctx context.Context, deviceID string) (interface{}, error) {
	c, err := d.clientPool.Get(d.resolveConn(deviceID))
	if err != nil {
		return nil, err
	}

	result, err := c.Get(ctx, "/vlan:vlan/vlan:vlans")
	if err != nil {
		return nil, err
	}

	deviceRoot := &huawei.Device{}

	// Check data type and parse accordingly
	switch data := result.Data.(type) {
	case []byte:
		// Try JSON first (gNMI case), then XML (NETCONF case)
		if len(data) > 0 && data[0] == '<' {
			// XML format from NETCONF get-config：经驱动描述符注册表解码
			// （DR-03/XC-02，通用引擎全字段填充；直接 xml.Unmarshal 进 ygot
			// map 会得到空 actual → 对账永远算出 diff、「一直漂移」）。
			d, ok := yangdriver.DecoderFor("/vlan:vlan/vlan:vlans")
			if !ok {
				return nil, fmt.Errorf("no XML decoder registered for vlan readback")
			}
			return d.DecodeXML(data)
		}
		// JSON format from gNMI
		if err := json.Unmarshal(data, deviceRoot); err != nil {
			return nil, fmt.Errorf("unmarshal JSON failed: %w", err)
		}
		if deviceRoot.Vlan == nil || deviceRoot.Vlan.Vlans == nil {
			return &huawei.HuaweiVlan_Vlan_Vlans{}, nil
		}
		return deviceRoot.Vlan.Vlans, nil
	}

	// If it's already unmarshaled into a struct, check directly
	if vlans, ok := result.Data.(*huawei.HuaweiVlan_Vlan_Vlans); ok {
		return vlans, nil
	}
	if deviceRoot, ok := result.Data.(*huawei.Device); ok && deviceRoot.Vlan != nil && deviceRoot.Vlan.Vlans != nil {
		return deviceRoot.Vlan.Vlans, nil
	}

	// Unknown data format
	return nil, fmt.Errorf("unknown data format for vlan config: %T", result.Data)
}

// Set applies the computed changes to the device
func (d *deviceClient) Set(ctx context.Context, deviceID string, changes []reconcile.Change) error {
	c, err := d.clientPool.Get(d.resolveConn(deviceID))
	if err != nil {
		return err
	}

	// Convert reconcile.Change to client.Change
	clientChanges := make([]client.Change, len(changes))
	for i, rc := range changes {
		var changeType client.ChangeType
		switch rc.Type {
		case "ADD":
			changeType = client.AddChange
		case "DELETE":
			changeType = client.DeleteChange
		case "MODIFY":
			changeType = client.ModifyChange
		default:
			changeType = client.ModifyChange
		}
		clientChanges[i] = client.Change{
			Type:       changeType,
			Path:       rc.Path,
			OldValue:   rc.ActualValue,
			NewValue:   rc.DesiredValue,
			SchemaPath: rc.Path,
		}
	}

	// Apply changes with commit
	_, err = c.Set(ctx, clientChanges, client.WithCommit(true))
	return err
}
