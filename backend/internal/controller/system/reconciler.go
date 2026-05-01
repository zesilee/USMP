package system

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net"
	"strconv"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
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
func New(cs reconcile.ConfigStore, clientPool client.ClientPool) *SystemReconciler {
	dc := &deviceClient{
		clientPool: clientPool,
	}
	de := &diffEngineAdapter{
		de: diff.NewDefaultDiffEngine(),
	}
	return &SystemReconciler{
		GenericReconciler: reconcile.NewGenericReconciler(cs, dc, de),
	}
}

// deviceClient implements reconcile.DeviceClient interface for getting system configuration from device
type deviceClient struct {
	clientPool client.ClientPool
}

// Get retrieves the actual system configuration from the device
func (d *deviceClient) Get(ctx context.Context, deviceID string) (interface{}, error) {
	var info client.DeviceConnectionInfo

	// Parse device ID using the same logic as VLAN
	if atIdx := lastAt(deviceID); atIdx >= 0 {
		creds := deviceID[:atIdx]
		hostPort := deviceID[atIdx+1:]

		if colonIdx := lastColon(creds); colonIdx >= 0 {
			info.Username = creds[:colonIdx]
			info.Password = creds[colonIdx+1:]
		} else {
			info.Username = creds
		}

		if host, portStr, err := splitHostPort(hostPort); err == nil {
			info.IP = host
			if p, err := parseInt(portStr); err == nil {
				info.Port = p
			}
			info.Protocol = client.ProtocolNETCONF
		} else {
			info.IP = hostPort
			info.Protocol = client.ProtocolAUTO
		}
	} else if host, portStr, err := splitHostPort(deviceID); err == nil {
		info.IP = host
		if p, err := parseInt(portStr); err == nil {
			info.Port = p
		}
		info.Protocol = client.ProtocolNETCONF
	} else {
		info.IP = deviceID
		info.Protocol = client.ProtocolAUTO
	}

	c, err := d.clientPool.Get(info)
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
	var info client.DeviceConnectionInfo

	if atIdx := lastAt(deviceID); atIdx >= 0 {
		creds := deviceID[:atIdx]
		hostPort := deviceID[atIdx+1:]

		if colonIdx := lastColon(creds); colonIdx >= 0 {
			info.Username = creds[:colonIdx]
			info.Password = creds[colonIdx+1:]
		} else {
			info.Username = creds
		}

		if host, portStr, err := splitHostPort(hostPort); err == nil {
			info.IP = host
			if p, err := parseInt(portStr); err == nil {
				info.Port = p
			}
			info.Protocol = client.ProtocolNETCONF
		} else {
			info.IP = hostPort
			info.Protocol = client.ProtocolAUTO
		}
	} else if host, portStr, err := splitHostPort(deviceID); err == nil {
		info.IP = host
		if p, err := parseInt(portStr); err == nil {
			info.Port = p
		}
		info.Protocol = client.ProtocolNETCONF
	} else {
		info.IP = deviceID
		info.Protocol = client.ProtocolAUTO
	}
	c, err := d.clientPool.Get(info)
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

// Helper functions from VLAN reconciler
func lastAt(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '@' {
			return i
		}
	}
	return -1
}

func lastColon(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ':' {
			return i
		}
	}
	return -1
}

func splitHostPort(s string) (host string, port string, err error) {
	if lastColon(s) < 0 {
		return "", "", fmt.Errorf("no port in deviceID")
	}
	return net.SplitHostPort(s)
}

func parseInt(s string) (int, error) {
	p, err := strconv.Atoi(s)
	return p, err
}
