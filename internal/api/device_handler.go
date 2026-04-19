package api

import (
	"time"

	protoactor "github.com/asynkron/protoactor-go/actor"
	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/internal/actor"
)

// DeviceHandler handles device-related API requests
type DeviceHandler struct {
	root        *protoactor.RootContext
	managerPID  *protoactor.PID
}

// NewDeviceHandler creates a new DeviceHandler
func NewDeviceHandler(root *protoactor.RootContext, managerPID *protoactor.PID) *DeviceHandler {
	return &DeviceHandler{
		root:        root,
	managerPID:  managerPID,
}
}

// AddDeviceRequest is the request body for adding a device
type AddDeviceRequest struct {
	IP       string `json:"ip" binding:"required"`
	Port     int    `json:"port"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// ListDevices lists all devices
func (h *DeviceHandler) ListDevices(c *gin.Context) {
	future := h.root.RequestFuture(h.managerPID, &actor.ListDevicesRequest{}, 5*time.Second)
	res, err := future.Result()
	if err != nil {
		Error(c, 500, "Failed to get devices: "+err.Error())
		return
	}

	listRes, ok := res.(*actor.ListDevicesResponse)
	if !ok {
		Error(c, 500, "Invalid response from actor")
		return
	}

	Success(c, listRes.Devices, "Devices retrieved successfully")
}

// AddDevice adds a new device
func (h *DeviceHandler) AddDevice(c *gin.Context) {
	var req AddDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "Invalid request: "+err.Error())
		return
	}

	// Default port 830 for NETCONF
	if req.Port == 0 {
		req.Port = 830
	}

	future := h.root.RequestFuture(h.managerPID, &actor.AddDeviceRequest{
		Device: actor.DeviceInfo{
			IP:       req.IP,
			Port:     req.Port,
			Username: req.Username,
			Password: req.Password,
		},
	}, 10*time.Second)

	res, err := future.Result()
	if err != nil {
		Error(c, 500, "Failed to add device: "+err.Error())
		return
	}

	addRes, ok := res.(*actor.AddDeviceResponse)
	if !ok {
		Error(c, 500, "Invalid response from actor")
		return
	}

	if !addRes.Success {
		Error(c, 400, addRes.Message)
		return
	}

	Success(c, nil, addRes.Message)
}

// RemoveDevice removes a device
func (h *DeviceHandler) RemoveDevice(c *gin.Context) {
	ip := c.Param("ip")

	future := h.root.RequestFuture(h.managerPID, &actor.RemoveDeviceRequest{IP: ip}, 5*time.Second)
	res, err := future.Result()
	if err != nil {
		Error(c, 500, "Failed to remove device: "+err.Error())
		return
	}

	removeRes, ok := res.(*actor.RemoveDeviceResponse)
	if !ok {
		Error(c, 500, "Invalid response from actor")
		return
	}

	if !removeRes.Success {
		Error(c, 400, removeRes.Message)
		return
	}

	Success(c, nil, removeRes.Message)
}

// GetStatus gets device status
func (h *DeviceHandler) GetStatus(c *gin.Context) {
	ip := c.Param("ip")

	// First get device PID from manager
	future := h.root.RequestFuture(h.managerPID, &actor.GetDeviceRequest{IP: ip}, 5*time.Second)
	res, err := future.Result()
	if err != nil {
		Error(c, 500, "Failed to get device: "+err.Error())
		return
	}

	getRes, ok := res.(*actor.GetDeviceResponse)
	if !ok {
		Error(c, 500, "Invalid response from actor")
		return
	}

	if !getRes.Exists {
		Error(c, 404, "Device not found")
		return
	}

	// Get status from device
	statusFuture := h.root.RequestFuture(getRes.PID, &actor.GetDeviceStatusRequest{}, 5*time.Second)
	statusRes, err := statusFuture.Result()
	if err != nil {
		Error(c, 500, "Failed to get device status: "+err.Error())
		return
	}

	status, ok := statusRes.(*actor.GetDeviceStatusResponse)
	if !ok {
		Error(c, 500, "Invalid response from device actor")
		return
	}

	Success(c, gin.H{
		"running":   status.Running,
		"connected": status.Connected,
	}, "Status retrieved")
}
