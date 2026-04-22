package vlan

import (
	"context"
	"encoding/json"

	"github.com/leezesi/usmp/internal/generated/openconfig"
	"github.com/leezesi/usmp/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/pkg/yang-runtime/diff"
	"github.com/leezesi/usmp/pkg/yang-runtime/reconcile"
	"github.com/leezesi/usmp/pkg/yang-runtime/schema"
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
func New(cs reconcile.ConfigStore, clientPool client.ClientPool) *VlanReconciler {
	dc := &deviceClient{
		clientPool: clientPool,
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
}

// Get retrieves the actual VLAN configuration from the device and converts it to the openconfig.VLans struct
func (d *deviceClient) Get(ctx context.Context, deviceID string) (interface{}, error) {
	// deviceID is the IP address of the device
	// Create DeviceConnectionInfo with deviceID as IP
	info := client.DeviceConnectionInfo{
		IP: deviceID,
	}
	c, err := d.clientPool.Get(info)
	if err != nil {
		return nil, err
	}

	result, err := c.Get(ctx, "/vlans")
	if err != nil {
		return nil, err
	}

	deviceRoot := &openconfig.Device{}
	// Result.Data is JSON bytes
	jsonBytes, ok := result.Data.([]byte)
	if ok {
		if err := json.Unmarshal(jsonBytes, deviceRoot); err != nil {
			return nil, err
		}
		return deviceRoot.Vlans, nil
	}

	// If it's already unmarshaled into a struct, check directly
	if vlans, ok := result.Data.(*openconfig.OpenconfigVlan_Vlans); ok {
		return vlans, nil
	}
	if deviceRoot, ok := result.Data.(*openconfig.Device); ok && deviceRoot.Vlans != nil {
		return deviceRoot.Vlans, nil
	}

	// Unknown data format
	return nil, nil
}

// Set applies the computed changes to the device
func (d *deviceClient) Set(ctx context.Context, deviceID string, changes []reconcile.Change) error {
	// deviceID is the IP address of the device
	info := client.DeviceConnectionInfo{
		IP: deviceID,
	}
	c, err := d.clientPool.Get(info)
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
			Type:      changeType,
			Path:      rc.Path,
			OldValue:  rc.ActualValue,
			NewValue:  rc.DesiredValue,
			SchemaPath: rc.Path,
		}
	}

	// Apply changes with commit
	_, err = c.Set(ctx, clientChanges, client.WithCommit(true))
	return err
}

