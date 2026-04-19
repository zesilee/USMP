package api

import (
	"time"

	protoactor "github.com/asynkron/protoactor-go/actor"
	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/internal/actor"
)

// ConfigHandler handles configuration API requests
type ConfigHandler struct {
	root       *protoactor.RootContext
	managerPID *protoactor.PID
}

// NewConfigHandler creates a new ConfigHandler
func NewConfigHandler(root *protoactor.RootContext, managerPID *protoactor.PID) *ConfigHandler {
	return &ConfigHandler{
		root:       root,
		managerPID: managerPID,
	}
}

// GetConfig gets the configuration for a specific device and YANG path
func (h *ConfigHandler) GetConfig(c *gin.Context) {
	ip := c.Param("ip")
	path := "/" + c.Param("path")
	forceRefresh := c.Query("force_refresh") == "true"

	// First get device PID from manager
	future := h.root.RequestFuture(h.managerPID, &actor.GetDeviceRequest{IP: ip}, 5*time.Second)
	res, err := future.Result()
	if err != nil {
		Error(c, 500, "Failed to get device: "+err.Error())
		return
	}

	deviceRes, ok := res.(*actor.GetDeviceResponse)
	if !ok {
		Error(c, 500, "Invalid response from manager")
		return
	}

	if !deviceRes.Exists {
		Error(c, 404, "Device not found")
		return
	}

	// Get YANG object actor from device
	yangFuture := h.root.RequestFuture(deviceRes.PID, &actor.GetYANGObjectActorRequest{Path: path}, 5*time.Second)
	yangRes, err := yangFuture.Result()
	if err != nil {
		Error(c, 500, "Failed to get YANG actor: "+err.Error())
		return
	}

	yangResp, ok := yangRes.(*actor.GetYANGObjectActorResponse)
	if !ok {
		Error(c, 500, "Invalid response from device")
		return
	}

	if !yangResp.Exists {
		Error(c, 404, "YANG path not found: "+path)
		return
	}

	// Request configuration from YANG actor
	configFuture := h.root.RequestFuture(yangResp.PID, &actor.GetConfigRequest{ForceRefresh: forceRefresh}, 10*time.Second)
	configRes, err := configFuture.Result()
	if err != nil {
		Error(c, 500, "Failed to get config: "+err.Error())
		return
	}

	configResp, ok := configRes.(*actor.GetConfigResponse)
	if !ok {
		Error(c, 500, "Invalid response from YANG actor")
		return
	}

	if !configResp.Success {
		Error(c, 500, configResp.Message)
		return
	}

	Success(c, gin.H{
		"data":        configResp.Data,
		"from_cache":  configResp.FromCache,
	}, "Configuration retrieved")
}

// SetConfig sets the configuration for a specific device and YANG path
func (h *ConfigHandler) SetConfig(c *gin.Context) {
	ip := c.Param("ip")
	path := "/" + c.Param("path")

	var data interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		Error(c, 400, "Invalid request: "+err.Error())
		return
	}

	// First get device PID from manager
	future := h.root.RequestFuture(h.managerPID, &actor.GetDeviceRequest{IP: ip}, 5*time.Second)
	res, err := future.Result()
	if err != nil {
		Error(c, 500, "Failed to get device: "+err.Error())
		return
	}

	deviceRes, ok := res.(*actor.GetDeviceResponse)
	if !ok {
		Error(c, 500, "Invalid response from manager")
		return
	}

	if !deviceRes.Exists {
		Error(c, 404, "Device not found")
		return
	}

	// Get YANG object actor from device
	yangFuture := h.root.RequestFuture(deviceRes.PID, &actor.GetYANGObjectActorRequest{Path: path}, 5*time.Second)
	yangRes, err := yangFuture.Result()
	if err != nil {
		Error(c, 500, "Failed to get YANG actor: "+err.Error())
		return
	}

	yangResp, ok := yangRes.(*actor.GetYANGObjectActorResponse)
	if !ok {
		Error(c, 500, "Invalid response from device")
		return
	}

	if !yangResp.Exists {
		Error(c, 404, "YANG path not found: "+path)
		return
	}

	// Send configuration to YANG actor
	setFuture := h.root.RequestFuture(yangResp.PID, &actor.SetConfigRequest{Data: data}, 15*time.Second)
	setRes, err := setFuture.Result()
	if err != nil {
		Error(c, 500, "Failed to set config: "+err.Error())
		return
	}

	setResp, ok := setRes.(*actor.SetConfigResponse)
	if !ok {
		Error(c, 500, "Invalid response from YANG actor")
		return
	}

	if !setResp.Success {
		Error(c, 500, setResp.Message)
		return
	}

	Success(c, nil, setResp.Message)
}
