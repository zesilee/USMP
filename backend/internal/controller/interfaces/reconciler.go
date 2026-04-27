package interfaces

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net"
	"strconv"

	"github.com/leezesi/usmp/backend/internal/generated/openconfig"
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
	var s schema.Schema = nil // not used for generic reflection-based diff
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

// InterfacesReconciler reconciles the interface configuration between desired state and actual device state
type InterfacesReconciler struct {
	*reconcile.GenericReconciler
}

// New creates a new InterfacesReconciler with the given dependencies
func New(cs reconcile.ConfigStore, clientPool client.ClientPool) *InterfacesReconciler {
	dc := &deviceClient{
		clientPool: clientPool,
	}
	de := &diffEngineAdapter{
		de: diff.NewDefaultDiffEngine(),
	}
	return &InterfacesReconciler{
		GenericReconciler: reconcile.NewGenericReconciler(cs, dc, de),
	}
}

// deviceClient implements reconcile.DeviceClient interface for getting interface configuration from device
type deviceClient struct {
	clientPool client.ClientPool
}

// Get retrieves the actual interface configuration from the device and converts it to the openconfig.Interfaces struct
func (d *deviceClient) Get(ctx context.Context, deviceID string) (interface{}, error) {
	var info client.DeviceConnectionInfo

	// Split credentials if present (@ separates credentials from host:port)
	if atIdx := lastAt(deviceID); atIdx >= 0 {
		creds := deviceID[:atIdx]
		hostPort := deviceID[atIdx+1:]

		// Split credentials into username and password
		if colonIdx := lastColon(creds); colonIdx >= 0 {
			info.Username = creds[:colonIdx]
			info.Password = creds[colonIdx+1:]
		} else {
			info.Username = creds
		}

		// Parse host and port
		if host, portStr, err := splitHostPort(hostPort); err == nil {
			info.IP = host
			if p, err := parseInt(portStr); err == nil {
				info.Port = p
			}
			info.Protocol = client.ProtocolNETCONF
		} else {
			info.IP = hostPort
		}
	} else if host, portStr, err := splitHostPort(deviceID); err == nil {
		// No credentials, just host:port
		info.IP = host
		if p, err := parseInt(portStr); err == nil {
			info.Port = p
		}
		info.Protocol = client.ProtocolNETCONF
	} else {
		// Just IP, use all defaults
		info.IP = deviceID
	}

	c, err := d.clientPool.Get(info)
	if err != nil {
		return nil, err
	}

	result, err := c.Get(ctx, "/interfaces")
	if err != nil {
		return nil, err
	}

	deviceRoot := &openconfig.Device{}

	// Check data type and parse accordingly
	switch data := result.Data.(type) {
	case []byte:
		// Try JSON first (gNMI case), then XML (NETCONF case)
		if len(data) > 0 && data[0] == '<' {
			// XML format from NETCONF
			if err := xml.Unmarshal(data, deviceRoot); err != nil {
				// If direct unmarshal fails, try wrapping the content
				wrapped := []byte(fmt.Sprintf("<data>%s</data>", string(data)))
				if err2 := xml.Unmarshal(wrapped, deviceRoot); err2 != nil {
					return nil, fmt.Errorf("unmarshal wrapped XML failed: %w (original: %w)", err2, err)
				}
			}
			if deviceRoot.Interfaces == nil {
				return &openconfig.OpenconfigInterfaces_Interfaces{
					Interface: map[string]*openconfig.OpenconfigInterfaces_Interfaces_Interface{},
				}, nil
			}
			return deviceRoot.Interfaces, nil
		}
		// JSON format from gNMI
		if err := json.Unmarshal(data, deviceRoot); err != nil {
			return nil, fmt.Errorf("unmarshal JSON failed: %w", err)
		}
		if deviceRoot.Interfaces == nil {
			return &openconfig.OpenconfigInterfaces_Interfaces{
				Interface: map[string]*openconfig.OpenconfigInterfaces_Interfaces_Interface{},
			}, nil
		}
		return deviceRoot.Interfaces, nil
	}

	// If it's already unmarshaled into a struct, check directly
	if interfaces, ok := result.Data.(*openconfig.OpenconfigInterfaces_Interfaces); ok {
		return interfaces, nil
	}
	if deviceRoot, ok := result.Data.(*openconfig.Device); ok && deviceRoot.Interfaces != nil {
		return deviceRoot.Interfaces, nil
	}

	// Unknown data format
	return nil, fmt.Errorf("unknown data format for interfaces config: %T", result.Data)
}

// Set applies the computed changes to the device
func (d *deviceClient) Set(ctx context.Context, deviceID string, changes []reconcile.Change) error {
	var info client.DeviceConnectionInfo

	// Split credentials if present (@ separates credentials from host:port)
	if atIdx := lastAt(deviceID); atIdx >= 0 {
		creds := deviceID[:atIdx]
		hostPort := deviceID[atIdx+1:]

		// Split credentials into username and password
		if colonIdx := lastColon(creds); colonIdx >= 0 {
			info.Username = creds[:colonIdx]
			info.Password = creds[colonIdx+1:]
		} else {
			info.Username = creds
		}

		// Parse host and port
		if host, portStr, err := splitHostPort(hostPort); err == nil {
			info.IP = host
			if p, err := parseInt(portStr); err == nil {
				info.Port = p
			}
			info.Protocol = client.ProtocolNETCONF
		} else {
			info.IP = hostPort
		}
	} else if host, portStr, err := splitHostPort(deviceID); err == nil {
		// No credentials, just host:port
		info.IP = host
		if p, err := parseInt(portStr); err == nil {
			info.Port = p
		}
		info.Protocol = client.ProtocolNETCONF
	} else {
		// Just IP, use all defaults
		info.IP = deviceID
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
			Type:        changeType,
			Path:        rc.Path,
			OldValue:    rc.ActualValue,
			NewValue:    rc.DesiredValue,
			SchemaPath:  rc.Path,
		}
	}

	// Apply changes with commit
	_, err = c.Set(ctx, clientChanges, client.WithCommit(true))
	return err
}

// splitHostPort splits a string into host and port, compatible with net.SplitHostPort
// but handles cases where there's no port
func splitHostPort(deviceID string) (host, port string, err error) {
	// If there's no colon, it's just host
	if i := lastColon(deviceID); i < 0 {
		return "", "", fmt.Errorf("no port in deviceID")
	} else {
		return net.SplitHostPort(deviceID)
	}
}

// lastColon returns the index of the last colon in s
func lastColon(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ':' {
			return i
		}
	}
	return -1
}

// lastAt returns the index of the last @ in s
func lastAt(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '@' {
			return i
		}
	}
	return -1
}

// parseInt parses a string to int
func parseInt(s string) (int, error) {
	p, err := strconv.Atoi(s)
	return p, err
}
