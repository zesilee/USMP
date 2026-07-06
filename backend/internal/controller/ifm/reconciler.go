package ifm

import (
	"context"
	"encoding/json"
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

// New creates a new IfmReconciler with the given dependencies
func New(cs reconcile.ConfigStore, clientPool client.ClientPool) *IfmReconciler {
	dc := &deviceClient{
		clientPool: clientPool,
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
}

// Get retrieves the actual Interface configuration from the device and converts it to the huawei.HuaweiIfm_Ifm_Interfaces struct
func (d *deviceClient) Get(ctx context.Context, deviceID string) (interface{}, error) {
	// deviceID format supports:
	// - "ip" - just IP, use default port (830) and default credentials
	// - "ip:port" - custom port, use default credentials
	// - "user:pass@ip:port" - custom port and credentials (for integration testing)
	var info client.DeviceConnectionInfo

	// Split credentials if present (@ separates credentials from host:port)
	if atIdx := lastAt(deviceID); atIdx >= 0 {
		// credentials part is everything before @
		creds := deviceID[:atIdx]
		// host:port part is everything after @
		hostPort := deviceID[atIdx+1:]

		// Split credentials into username and password
		if colonIdx := lastColon(creds); colonIdx >= 0 {
			info.Username = creds[:colonIdx]
			info.Password = creds[colonIdx+1:]
		} else {
			info.Username = creds
			// no password provided
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
			info.Protocol = client.ProtocolAUTO
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
		info.Protocol = client.ProtocolAUTO
	}
	c, err := d.clientPool.Get(info)
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
	// deviceID format supports:
	// - "ip" - just IP, use default port (830) and default credentials
	// - "ip:port" - custom port, use default credentials
	// - "user:pass@ip:port" - custom port and credentials (for integration testing)
	var info client.DeviceConnectionInfo

	// Split credentials if present (@ separates credentials from host:port)
	if atIdx := lastAt(deviceID); atIdx >= 0 {
		// credentials part is everything before @
		creds := deviceID[:atIdx]
		// host:port part is everything after @
		hostPort := deviceID[atIdx+1:]

		// Split credentials into username and password
		if colonIdx := lastColon(creds); colonIdx >= 0 {
			info.Username = creds[:colonIdx]
			info.Password = creds[colonIdx+1:]
		} else {
			info.Username = creds
			// no password provided
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
			info.Protocol = client.ProtocolAUTO
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
		info.Protocol = client.ProtocolAUTO
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

// splitHostPort splits a string into host and port, compatible with net.SplitHostPort
// but handles cases where there's no port.
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
