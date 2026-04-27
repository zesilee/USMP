package api

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
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
	path := "/" + c.Param("path")
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

// SetConfig sets the configuration for a specific device and YANG path
func (h *ConfigHandler) SetConfig(c *gin.Context) {
	ip := c.Param("ip")
	path := "/" + c.Param("path")

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		Error(c, 400, "Invalid request: "+err.Error())
		return
	}

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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// For simplicity in the new API, we create a single change that covers the path
	// The actual YANG parsing and diff should be handled by the controller reconciliation
	changes := []client.Change{{
		Type:     client.ModifyChange,
		Path:     path,
		NewValue: data,
	}}

	result, err := cli.Set(ctx, changes, client.WithCommit(true))
	if err != nil {
		Error(c, 500, "Failed to apply configuration: "+err.Error())
		return
	}

	if !result.Success {
		Error(c, 500, result.Message)
		return
	}

	Success(c, nil, "Configuration applied successfully")
}