package api

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/internal/uischema"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
)

// UISchemaHandler handles UI schema API requests
type UISchemaHandler struct {
	manager    manager.Manager
	interfaces *uischema.InterfacesGenerator
}

// NewUISchemaHandler creates a new UISchemaHandler
func NewUISchemaHandler(m manager.Manager) *UISchemaHandler {
	return &UISchemaHandler{
		manager:    m,
		interfaces: uischema.NewInterfacesGenerator(),
	}
}

// GetInterfaces returns the UI schema for interfaces
func (h *UISchemaHandler) GetInterfaces(c *gin.Context) {
	ip := c.Param("ip")
	schema := h.interfaces.BuildSchema(ip)

	if h.manager != nil {
		rows, err := h.loadInterfaceRows(ip)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"code":    "LOAD_FAILED",
				"message": "Failed to load interface configuration: " + err.Error(),
			})
			return
		}
		schema.Values[uischema.InterfacesWidgetID] = rows
	}

	Success(c, schema, "UI schema retrieved successfully")
}

func (h *UISchemaHandler) loadInterfaceRows(ip string) ([]interface{}, error) {
	cli, err := h.manager.GetClientPool().Get(client.DeviceConnectionInfo{IP: ip})
	if err != nil {
		return nil, err
	}
	if !cli.IsConnected() {
		return nil, errors.New("device is not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := cli.Get(ctx, uischema.InterfacesTargetPath, client.WithDatastore("running"))
	if err != nil {
		return nil, err
	}

	interfaces, ok := result.Data.(*huawei.HuaweiIfm_Ifm_Interfaces)
	if !ok {
		return nil, nil
	}

	names := make([]string, 0, len(interfaces.Interface))
	for name := range interfaces.Interface {
		names = append(names, name)
	}
	sort.Strings(names)

	rows := make([]interface{}, 0, len(names))
	for _, name := range names {
		iface := interfaces.Interface[name]
		row := map[string]interface{}{"name": name}
		if iface.Name != nil {
			row["name"] = *iface.Name
		}
		if iface.Description != nil {
			row["description"] = *iface.Description
		}
		if iface.Mtu != nil {
			row["mtu"] = int(*iface.Mtu)
		}
		if iface.AdminStatus != 0 {
			row["admin-status"] = int(iface.AdminStatus)
		}
		rows = append(rows, row)
	}

	return rows, nil
}

// ApplyInterfaces applies the interface configuration
func (h *UISchemaHandler) ApplyInterfaces(c *gin.Context) {
	ip := c.Param("ip")

	var req uischema.ApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Invalid JSON/request format
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"code":    "INVALID_REQUEST",
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	// Validate the request
	if err := h.interfaces.ValidateApply(req); err != nil {
		var valErr *uischema.ValidationError
		if errors.As(err, &valErr) {
			// Validation error with details
			c.JSON(http.StatusOK, gin.H{
				"success":     false,
				"code":        valErr.Code,
				"message":     valErr.Message,
				"fieldErrors": valErr.FieldErrors,
			})
			return
		}
		// Non-validation error during validation
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"code":    "APPLY_FAILED",
			"message": "Validation failed: " + err.Error(),
		})
		return
	}

	// Convert to typed struct
	desiredConfig, err := convertToTypedStruct(uischema.InterfacesTargetPath, map[string]interface{}{
		"interface": req.Values[uischema.InterfacesWidgetID],
	})
	if err != nil {
		// Conversion failure
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"code":    "CONVERT_FAILED",
			"message": "Failed to convert configuration: " + err.Error(),
		})
		return
	}

	var triggered bool
	if h.manager != nil {
		// Store desired config
		configStore := h.manager.GetConfigStore()
		if err := configStore.Set(ip, uischema.InterfacesTargetPath, desiredConfig); err != nil {
			// Config store failure
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"code":    "STORE_FAILED",
				"message": "Failed to store configuration: " + err.Error(),
			})
			return
		}

		// Trigger reconciliation
		triggered = h.manager.TriggerReconcile(ip, uischema.InterfacesTargetPath)
	}

	Success(c, gin.H{
		"schemaVersion": req.SchemaVersion,
		"values":        req.Values,
		"lastSync":      time.Now().UTC().Format(time.RFC3339),
		"reconciliation": gin.H{
			"triggered": triggered,
		},
	}, "Configuration applied successfully")
}
