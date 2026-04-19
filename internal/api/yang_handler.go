package api

import (
	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/internal/actor"
)

// YangHandler handles YANG model API requests
type YangHandler struct {
}

// NewYangHandler creates a new YangHandler
func NewYangHandler() *YangHandler {
	return &YangHandler{}
}

// ListModules lists all supported YANG modules
func (h *YangHandler) ListModules(c *gin.Context) {
	// The list of supported YANG modules is hardcoded for now
	// When we have dynamic YANG loading, this will be generated from the model
	modules := []actor.YangModuleInfo{
		{
			Name:        "Interfaces",
			Path:        "/interfaces",
			Description: "Network interfaces configuration",
			Type:        "container",
		},
		{
			Name:        "VLANs",
			Path:        "/vlans",
			Description: "VLAN configuration",
			Type:        "container",
		},
		{
			Name:        "System",
			Path:        "/system",
			Description: "System information and configuration",
			Type:        "container",
		},
	}

	Success(c, modules, "YANG modules retrieved successfully")
}
