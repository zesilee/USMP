package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/backend/internal/uischema"
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

	Success(c, schema, "UI schema retrieved successfully")
}

// ApplyInterfaces applies the interface configuration
func (h *UISchemaHandler) ApplyInterfaces(c *gin.Context) {
	ip := c.Param("ip")

	var req uischema.ApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "Invalid request: "+err.Error())
		return
	}

	// Validate the request
	if err := h.interfaces.ValidateApply(req); err != nil {
		var valErr *uischema.ValidationError
		if errors.As(err, &valErr) {
			// Return validation error details
			c.JSON(http.StatusOK, Response{
				Code:    400,
				Message: valErr.Message,
				Data: gin.H{
					"code":        valErr.Code,
					"fieldErrors": valErr.FieldErrors,
				},
				Success: false,
			})
			return
		}
		Error(c, 500, "Validation failed: "+err.Error())
		return
	}

	// Convert to typed struct
	desiredConfig, err := convertToTypedStruct(uischema.InterfacesTargetPath, map[string]interface{}{
		"interface": req.Values[uischema.InterfacesWidgetID],
	})
	if err != nil {
		Error(c, 400, "Failed to convert configuration: "+err.Error())
		return
	}

	var triggered bool
	if h.manager != nil {
		// Store desired config
		configStore := h.manager.GetConfigStore()
		if err := configStore.Set(ip, uischema.InterfacesTargetPath, desiredConfig); err != nil {
			Error(c, 500, "Failed to store configuration: "+err.Error())
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
