package api

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
)

// ConfigHandler handles configuration API requests
type ConfigHandler struct {
	manager manager.Manager
}

// NewConfigHandler creates a new ConfigHandler
func NewConfigHandler(manager manager.Manager) *ConfigHandler {
	return &ConfigHandler{
		manager: manager,
	}
}

// GetConfig gets the configuration for a specific device and YANG path
func (h *ConfigHandler) GetConfig(c *gin.Context) {
	ip := c.Param("ip")
	path := c.Param("path") // *path already includes leading slash
	_ = c.Query("force_refresh") == "true" // TODO: Implement cache invalidation when we have caching

	// Get the device info from device handler
	// We need to get it from the device registry
	// For now, we just get the client from pool
	pool := h.manager.GetClientPool()

	cli, err := pool.Get(client.DeviceConnectionInfo{
		IP: ip,
	})
	if err != nil {
		Error(c, 500, "Failed to get device client: "+err.Error())
		return
	}

	if !cli.IsConnected() {
		Error(c, 503, "Device is not connected")
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get configuration from device
	result, err := cli.Get(ctx, path, client.WithDatastore("running"))
	if err != nil {
		Error(c, 500, "Failed to get configuration: "+err.Error())
		return
	}

	Success(c, gin.H{
		"data": result.Data,
	}, "Configuration retrieved")
}

// SetConfig sets the desired configuration and triggers reconciliation
// This is the DECLARATIVE API: desired state is stored, and the controller
// will asynchronously reconcile the actual device state to match it.
func (h *ConfigHandler) SetConfig(c *gin.Context) {
	ip := c.Param("ip")
	path := c.Param("path") // *path already includes leading slash

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		Error(c, 400, "Invalid request: "+err.Error())
		return
	}

	// Convert the raw data to the appropriate YANG model struct
	// This ensures the ConfigStore stores properly typed data that the
	// Reconciler can work with for diff calculation
	desiredConfig, err := convertToTypedStruct(path, data)
	if err != nil {
		Error(c, 400, "Failed to parse configuration: "+err.Error())
		return
	}

	// Store the desired configuration in ConfigStore
	// This is the source of truth for what configuration SHOULD be
	configStore := h.manager.GetConfigStore()
	if err := configStore.Set(ip, path, desiredConfig); err != nil {
		Error(c, 500, "Failed to store configuration: "+err.Error())
		return
	}

	// Trigger immediate reconciliation
	// The controller will:
	// 1. Get actual config from device
	// 2. Calculate diff between desired and actual
	// 3. Apply changes to device
	// 4. Commit (if supported by protocol)
	controllerFound := h.manager.TriggerReconcile(ip, path)

	Success(c, gin.H{
		"status":        "ACCEPTED",
		"path":          path,
		"reconciliation": map[string]interface{}{
			"triggered": controllerFound,
			"message":   "Configuration stored. Reconciliation will sync device state.",
		},
	}, "Configuration accepted - reconciliation in progress")
}

// convertToTypedStruct converts raw JSON map to the appropriate strongly-typed
// YANG model struct based on the path. This ensures proper diff calculation.
func convertToTypedStruct(path string, data map[string]interface{}) (interface{}, error) {
	// Huawei IFM Interfaces
	if strings.Contains(path, "ifm:ifm") && strings.Contains(path, "interfaces") {
		return convertMapToHuaweiIfm(data)
	}

	// Huawei VLANs
	if strings.Contains(path, "vlan:") && (strings.Contains(path, "vlan") || strings.Contains(path, "vlans")) {
		return convertMapToHuaweiVlan(data)
	}

	// Fallback: return the raw map for unhandled paths
	// Reconciler will handle it or report error appropriately
	return data, nil
}

// convertMapToHuaweiIfm converts a map to HuaweiIfm_Ifm_Interfaces struct
func convertMapToHuaweiIfm(data map[string]interface{}) (*huawei.HuaweiIfm_Ifm_Interfaces, error) {
	result := &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: make(map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface),
	}

	// Extract interface array/map
	var ifacesData interface{}
	if v, ok := data["interface"]; ok {
		ifacesData = v
	} else if v, ok := data["Interface"]; ok {
		ifacesData = v
	} else if v, ok := data["interfaces"]; ok {
		ifacesData = v
	} else if v, ok := data["Interfaces"]; ok {
		ifacesData = v
	} else {
		// If no interface container, assume the data itself is the interface config
		iface := mapEntryToInterface(data)
		key := "default"
		if iface.Name != nil {
			key = *iface.Name
		}
		result.Interface[key] = iface
		return result, nil
	}

	// Handle both slice and map formats
	// Note: JSON unmarshal always produces []interface{} for arrays
	switch v := ifacesData.(type) {
	case []interface{}:
		for i, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				iface := mapEntryToInterface(m)
				key := fmt.Sprintf("iface-%d", i)
				if iface.Name != nil {
					key = *iface.Name
				}
				result.Interface[key] = iface
			}
		}
	case map[string]interface{}:
		for key, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				iface := mapEntryToInterface(m)
				if iface.Name == nil {
					iface.Name = &key
				}
				result.Interface[key] = iface
			}
		}
	}

	return result, nil
}

// mapEntryToInterface converts a single interface entry map to struct
func mapEntryToInterface(m map[string]interface{}) *huawei.HuaweiIfm_Ifm_Interfaces_Interface {
	result := &huawei.HuaweiIfm_Ifm_Interfaces_Interface{}

	for k, v := range m {
		key := strings.ToLower(strings.ReplaceAll(k, "-", ""))
		switch key {
		case "name":
			if s, ok := v.(string); ok {
				result.Name = &s
			}
		case "description":
			if s, ok := v.(string); ok {
				result.Description = &s
			}
		case "adminstatus":
			if num, ok := valueToUint(v); ok {
				result.AdminStatus = huawei.E_HuaweiIfm_PortStatus(num)
			}
		case "mtu":
			if num, ok := valueToUint(v); ok {
				uint32Val := uint32(num)
				result.Mtu = &uint32Val
			}
		case "type":
			if num, ok := valueToUint(v); ok {
				result.Type = huawei.E_HuaweiIfm_PortType(num)
			}
		}
	}

	return result
}

// convertMapToHuaweiVlan - placeholder for VLAN support (future)
func convertMapToHuaweiVlan(data map[string]interface{}) (interface{}, error) {
	// Return raw map for now - VLAN reconciler will handle
	return data, nil
}

// valueToUint converts various numeric types to uint64
func valueToUint(v interface{}) (uint64, bool) {
	switch val := v.(type) {
	case float64:
		return uint64(val), true
	case int:
		return uint64(val), true
	case int64:
		return uint64(val), true
	case uint:
		return uint64(val), true
	case uint32:
		return uint64(val), true
	case uint64:
		return val, true
	case string:
		if num, err := strconv.ParseUint(val, 10, 64); err == nil {
			return num, true
		}
	}
	return 0, false
}