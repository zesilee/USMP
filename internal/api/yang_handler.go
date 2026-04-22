package api

import (
	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/pkg/yang-runtime/manager"
)

// YangModuleInfo represents information about a YANG module
type YangModuleInfo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

// YangHandler handles YANG model API requests
type YangHandler struct {
	manager manager.Manager
}

// NewYangHandler creates a new YangHandler
func NewYangHandler(manager manager.Manager) *YangHandler {
	return &YangHandler{
		manager: manager,
	}
}

// ListModules lists all supported YANG modules
func (h *YangHandler) ListModules(c *gin.Context) {
	s := h.manager.GetSchema()
	modules := make([]YangModuleInfo, 0)

	for _, mod := range s.Modules() {
		// Get the root node
		root := mod.Root()
		if root == nil {
			continue
		}

		info := YangModuleInfo{
			Name:        mod.Name(),
			Path:        "/" + root.Name(),
			Description: root.Description(),
			Type:        string(root.Type()),
		}
		modules = append(modules, info)
	}

	// If no modules are loaded, return some example for now
	if len(modules) == 0 {
		modules = []YangModuleInfo{
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
	}

	Success(c, modules, "YANG modules retrieved successfully")
}