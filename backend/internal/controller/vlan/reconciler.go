package vlan

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"

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
	return parseDeviceID(deviceID)
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
			// XML format from NETCONF get-config.
			// ygot 把 vlans/vlan（及 member-port）生成为 Go map 且无 xml tag，encoding/xml
			// 无法解析进 map —— 直接 xml.Unmarshal 会得到空 actual，导致对账永远算出 diff
			// （VLAN「一直漂移」）。改用手写 token 解析器把 <vlan> 填进 ygot map。
			return client.ParseHuaweiVlanVlansXML(data)
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

// parseDeviceID is the legacy fallback that derives connection info from the
// DeviceID string ("ip" | "ip:port" | "user:pass@ip:port"). Kept only for the
// migration window; the DeviceStore is the real source.
func parseDeviceID(deviceID string) client.DeviceConnectionInfo {
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
	return info
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
