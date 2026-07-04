package api

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
)

// DeviceInfo stores device connection information
type DeviceInfo struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// DeviceHandler handles device-related API requests
type DeviceHandler struct {
	manager manager.Manager
	devices map[string]DeviceInfo
	mu      sync.RWMutex
}

// NewDeviceHandler creates a new DeviceHandler
func NewDeviceHandler(manager manager.Manager) *DeviceHandler {
	h := &DeviceHandler{
		manager: manager,
		devices: make(map[string]DeviceInfo),
	}

	// Add default test device for development
	h.devices["192.168.1.1"] = DeviceInfo{
		IP:       "192.168.1.1",
		Port:     830,
		Username: "admin",
		Password: "admin",
	}

	return h
}

// AddDeviceRequest is the request body for adding a device
type AddDeviceRequest struct {
	IP       string `json:"ip" binding:"required"`
	Port     int    `json:"port"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// DeviceStatus is a device plus its live online status (Stack B REST替代
// BusinessSwitch 控制器的 CR-status 探活).
type DeviceStatus struct {
	DeviceInfo
	Online bool `json:"online"`
}

// probeOnline reports whether a device is reachable, via the ClientPool直连
// (Get + IsConnected). A connection error is treated as offline (R08).
func probeOnline(pool client.ClientPool, d DeviceInfo) bool {
	port := d.Port
	if port == 0 {
		port = 830
	}
	c, err := pool.Get(client.DeviceConnectionInfo{
		IP:       d.IP,
		Port:     port,
		Username: d.Username,
		Password: d.Password,
		Timeout:  3 * time.Second,
	})
	return err == nil && c != nil && c.IsConnected()
}

// ListDevices lists all devices with their live online status.
func (h *DeviceHandler) ListDevices(c *gin.Context) {
	h.mu.RLock()
	snapshot := make([]DeviceInfo, 0, len(h.devices))
	for _, d := range h.devices {
		snapshot = append(snapshot, d)
	}
	h.mu.RUnlock()

	pool := h.manager.GetClientPool()
	devices := make([]DeviceStatus, 0, len(snapshot))
	for _, d := range snapshot {
		devices = append(devices, DeviceStatus{DeviceInfo: d, Online: probeOnline(pool, d)})
	}

	stats := pool.Stats()

	Success(c, gin.H{
		"devices": devices,
		"stats": gin.H{
			"active_connections": stats.ActiveConnections,
			"total_connections":  stats.TotalConnections,
			"errors":             stats.Errors,
		},
	}, "Devices retrieved successfully")
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

	h.mu.Lock()
	h.devices[req.IP] = DeviceInfo{
		IP:       req.IP,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
	}
	h.mu.Unlock()

	// Try to create client and connect immediately
	pool := h.manager.GetClientPool()
	_, err := pool.Get(client.DeviceConnectionInfo{
		IP:       req.IP,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
		Protocol: client.ProtocolAUTO,
	})
	if err != nil {
		// We still store the device info but return the error
		Error(c, 500, "Failed to connect to device: "+err.Error())
		return
	}

	Success(c, nil, "Device added successfully")
}

// RemoveDevice removes a device
func (h *DeviceHandler) RemoveDevice(c *gin.Context) {
	ip := c.Param("ip")

	h.mu.Lock()
	delete(h.devices, ip)
	h.mu.Unlock()

	// Close the connection
	pool := h.manager.GetClientPool()
	pool.Release(ip)

	Success(c, nil, "Device removed successfully")
}

// GetStatus gets device status
func (h *DeviceHandler) GetStatus(c *gin.Context) {
	ip := c.Param("ip")

	h.mu.RLock()
	_, exists := h.devices[ip]
	h.mu.RUnlock()

	if !exists {
		Error(c, 404, "Device not found")
		return
	}

	// Get client from pool and check connection status
	pool := h.manager.GetClientPool()
	// We need to get the full info from our devices map
	h.mu.RLock()
	devInfo := h.devices[ip]
	h.mu.RUnlock()

	cli, err := pool.Get(client.DeviceConnectionInfo{
		IP:       devInfo.IP,
		Port:     devInfo.Port,
		Username: devInfo.Username,
		Password: devInfo.Password,
		Protocol: client.ProtocolAUTO,
	})

	connected := false
	if err == nil && cli != nil {
		connected = cli.IsConnected()
	}

	Success(c, gin.H{
		"running":   true, // API server is always running
		"connected": connected,
	}, "Status retrieved")
}
