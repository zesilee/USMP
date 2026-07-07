package system

import (
	"context"
	"encoding/json"
	"encoding/xml"
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
type diffEngineAdapter struct {
	de *diff.DefaultDiffEngine
}

// Diff implements the reconcile.DiffEngine interface
func (a *diffEngineAdapter) Diff(desired, actual interface{}, path string) ([]reconcile.Change, error) {
	var s schema.Schema = nil
	result, err := a.de.Diff(desired, actual, s)
	if err != nil {
		return nil, err
	}
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

// SystemReconciler reconciles the system configuration between desired state and actual device state.
type SystemReconciler struct {
	*reconcile.GenericReconciler
}

// New creates a new SystemReconciler with the given dependencies
func New(cs reconcile.ConfigStore, clientPool client.ClientPool, resolver device.Store) *SystemReconciler {
	dc := &deviceClient{
		clientPool: clientPool,
		resolver:   resolver,
	}
	de := &diffEngineAdapter{
		de: diff.NewDefaultDiffEngine(),
	}
	return &SystemReconciler{
		GenericReconciler: reconcile.NewGenericReconciler(cs, dc, de),
	}
}

// resolveConn resolves connection info from the shared DeviceStore (source of
// truth). An unregistered device (or no store) degrades to an AUTO/no-credential
// connection — authentication fails cleanly rather than crashing (R08).
func (d *deviceClient) resolveConn(deviceID string) client.DeviceConnectionInfo {
	if d.resolver != nil {
		if info, ok := d.resolver.Get(deviceID); ok {
			return info
		}
	}
	log.Printf("[system] device %q not registered in DeviceStore; using AUTO/no-credential connection", deviceID)
	return client.DeviceConnectionInfo{IP: deviceID, Protocol: client.ProtocolAUTO}
}

// deviceClient implements reconcile.DeviceClient interface for getting system configuration from device
type deviceClient struct {
	clientPool client.ClientPool
	resolver   device.Store
}

// Get retrieves the actual system configuration from the device
func (d *deviceClient) Get(ctx context.Context, deviceID string) (interface{}, error) {
	c, err := d.clientPool.Get(d.resolveConn(deviceID))
	if err != nil {
		return nil, err
	}

	result, err := c.Get(ctx, "/system:system")
	if err != nil {
		return nil, err
	}

	deviceRoot := &huawei.Device{}

	switch data := result.Data.(type) {
	case []byte:
		if len(data) > 0 && data[0] == '<' {
			// XML format from NETCONF
			if err := xml.Unmarshal(data, deviceRoot); err != nil {
				wrapped := []byte(fmt.Sprintf("<data>%s</data>", string(data)))
				if err2 := xml.Unmarshal(wrapped, deviceRoot); err2 != nil {
					return nil, fmt.Errorf("unmarshal XML failed: %w (original: %w)", err2, err)
				}
			}
			if deviceRoot.System == nil || deviceRoot.System.SystemInfo == nil {
				return &huawei.HuaweiSystem_System{}, nil
			}
			return deviceRoot.System, nil
		}
		// JSON format from gNMI
		if err := json.Unmarshal(data, deviceRoot); err != nil {
			return nil, fmt.Errorf("unmarshal JSON failed: %w", err)
		}
		if deviceRoot.System == nil || deviceRoot.System.SystemInfo == nil {
			return &huawei.HuaweiSystem_System{}, nil
		}
		return deviceRoot.System, nil
	default:
		return &huawei.HuaweiSystem_System{}, nil
	}
}

// Set applies the configuration changes to the device
func (d *deviceClient) Set(ctx context.Context, deviceID string, changes []reconcile.Change) error {
	c, err := d.clientPool.Get(d.resolveConn(deviceID))
	if err != nil {
		return err
	}

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
