package ifm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/diff"
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

// IfmReconciler reconciles the Interface configuration between desired state and actual device state.
// It uses the GenericReconciler base implementation that handles the common reconciliation pattern.
type IfmReconciler struct {
	*reconcile.GenericReconciler
}

// New creates a new IfmReconciler with the given dependencies. resolver is the
// shared DeviceStore used to look up per-device connection info (credentials,
// port, protocol) by DeviceID; when nil or a device is unregistered the client
// degrades to an AUTO/no-credential connection.
func New(cs reconcile.ConfigStore, clientPool client.ClientPool, resolver device.Store) *IfmReconciler {
	dc := &deviceClient{
		clientPool: clientPool,
		resolver:   resolver,
	}
	de := &diffEngineAdapter{
		de: diff.NewDefaultDiffEngine(),
	}
	return &IfmReconciler{
		GenericReconciler: reconcile.NewGenericReconciler(cs, dc, de),
	}
}

// deviceClient implements reconcile.DeviceClient interface for getting Interface configuration from device
type deviceClient struct {
	clientPool client.ClientPool
	resolver   device.Store
}

// resolveConn resolves connection info for a device from the shared DeviceStore
// (source of truth). An unregistered device (or no store) degrades to an AUTO/
// no-credential connection — authentication fails cleanly rather than crashing
// (R08).
func (d *deviceClient) resolveConn(deviceID string) client.DeviceConnectionInfo {
	if d.resolver != nil {
		if info, ok := d.resolver.Get(deviceID); ok {
			return info
		}
	}
	// Unregistered device (or no store): degrade to an AUTO/no-credential
	// connection. Authentication will fail cleanly (R08) rather than crash.
	log.Printf("[ifm] device %q not registered in DeviceStore; using AUTO/no-credential connection", deviceID)
	return client.DeviceConnectionInfo{IP: deviceID, Protocol: client.ProtocolAUTO}
}

// Get retrieves the actual Interface configuration from the device and converts it to the huawei.HuaweiIfm_Ifm_Interfaces struct
func (d *deviceClient) Get(ctx context.Context, deviceID string) (interface{}, error) {
	c, err := d.clientPool.Get(d.resolveConn(deviceID))
	if err != nil {
		return nil, err
	}

	result, err := c.Get(ctx, "/ifm:ifm/ifm:interfaces")
	if err != nil {
		return nil, err
	}

	deviceRoot := &huawei.Device{}

	// Check data type and parse accordingly
	switch data := result.Data.(type) {
	case []byte:
		// Try JSON first (gNMI case), then XML (NETCONF case)
		if len(data) > 0 && data[0] == '<' {
			// XML format from NETCONF get-config.
			// ygot 结构体把 interfaces/interface 生成为 Go map 且无 xml tag，encoding/xml
			// 无法解析进 map —— 直接 xml.Unmarshal 会得到空 actual，导致对账永远算出 diff
			// （前端「一直漂移」）。改用手写 token 解析器把 <interface> 填进 ygot map。
			return client.ParseHuaweiIfmInterfacesXML(data)
		}
		// JSON format from gNMI
		if err := json.Unmarshal(data, deviceRoot); err != nil {
			return nil, fmt.Errorf("unmarshal JSON failed: %w", err)
		}
		if deviceRoot.Ifm == nil || deviceRoot.Ifm.Interfaces == nil {
			return &huawei.HuaweiIfm_Ifm_Interfaces{}, nil
		}
		return deviceRoot.Ifm.Interfaces, nil
	}

	// If it's already unmarshaled into a struct, check directly
	if ifm, ok := result.Data.(*huawei.HuaweiIfm_Ifm_Interfaces); ok {
		return ifm, nil
	}
	if deviceRoot, ok := result.Data.(*huawei.Device); ok && deviceRoot.Ifm != nil && deviceRoot.Ifm.Interfaces != nil {
		return deviceRoot.Ifm.Interfaces, nil
	}

	// Unknown data format
	return nil, fmt.Errorf("unknown data format for ifm config: %T", result.Data)
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
