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

	// Add default test device for development. Write it to the shared DeviceStore
	// (single source of truth) so reconcile/config/periodic can resolve its
	// credentials; the local map is kept in parallel during migration.
	seed := DeviceInfo{
		IP:       "192.168.1.1",
		Port:     830,
		Username: "admin",
		Password: "admin",
	}
	h.devices[seed.IP] = seed
	h.putStore(seed)

	return h
}

// toConnInfo maps a DeviceInfo to the shared store's connection-info type,
// defaulting Protocol to AUTO (port-based NETCONF/gNMI selection).
func toConnInfo(d DeviceInfo) client.DeviceConnectionInfo {
	return client.DeviceConnectionInfo{
		IP:       d.IP,
		Port:     d.Port,
		Username: d.Username,
		Password: d.Password,
		Protocol: client.ProtocolAUTO,
	}
}

// deviceInfoFromConn is the inverse of toConnInfo for API responses.
func deviceInfoFromConn(info client.DeviceConnectionInfo) DeviceInfo {
	return DeviceInfo{IP: info.IP, Port: info.Port, Username: info.Username, Password: info.Password}
}

// putStore registers a device in the shared DeviceStore (no-op if the manager
// exposes none, keeping the handler usable in minimal test setups).
func (h *DeviceHandler) putStore(d DeviceInfo) {
	if ds := h.manager.GetDeviceStore(); ds != nil {
		ds.Put(d.IP, toConnInfo(d))
	}
}

// snapshotDevices returns all registered devices, reading from the shared
// DeviceStore (single source of truth) and falling back to the local map when no
// store is available (minimal test setups).
func (h *DeviceHandler) snapshotDevices() []DeviceInfo {
	if ds := h.manager.GetDeviceStore(); ds != nil {
		ids := ds.List()
		out := make([]DeviceInfo, 0, len(ids))
		for _, id := range ids {
			if info, ok := ds.Get(id); ok {
				out = append(out, deviceInfoFromConn(info))
			}
		}
		return out
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]DeviceInfo, 0, len(h.devices))
	for _, d := range h.devices {
		out = append(out, d)
	}
	return out
}

// lookupDevice resolves a single device by IP from the shared DeviceStore, with
// the same local-map fallback as snapshotDevices.
func (h *DeviceHandler) lookupDevice(ip string) (DeviceInfo, bool) {
	if ds := h.manager.GetDeviceStore(); ds != nil {
		if info, ok := ds.Get(ip); ok {
			return deviceInfoFromConn(info), true
		}
		return DeviceInfo{}, false
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	d, ok := h.devices[ip]
	return d, ok
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
		// Protocol must be set or the factory hits its default ("unsupported
		// protocol") branch and the probe would falsely report the device offline.
		Protocol: client.ProtocolAUTO,
		Timeout:  3 * time.Second,
	})
	return err == nil && c != nil && c.IsConnected()
}

// PoolStatsDTO 是连接池统计（对齐 client.PoolStats，仅暴露 JSON 契约）。
type PoolStatsDTO struct {
	ActiveConnections int `json:"active_connections"`
	TotalConnections  int `json:"total_connections"`
	Errors            int `json:"errors"`
}

// DeviceListData 是 GET /devices 的 data 字段负载（前端 res.data.data）。
type DeviceListData struct {
	Devices []DeviceStatus `json:"devices"`
	Stats   PoolStatsDTO   `json:"stats"`
}

// DeviceConnStatus 是 GET /devices/:ip/status 的 data 字段负载。
type DeviceConnStatus struct {
	Running   bool `json:"running"`
	Connected bool `json:"connected"`
}

// ListDevices lists all devices with their live online status.
//
// @Summary  列出所有设备及在线状态
// @Tags     devices
// @Produce  json
// @Success  200 {object} Response{data=DeviceListData} "设备列表 + 连接池统计"
// @Router   /devices [get]
func (h *DeviceHandler) ListDevices(c *gin.Context) {
	snapshot := h.snapshotDevices()

	pool := h.manager.GetClientPool()
	devices := make([]DeviceStatus, 0, len(snapshot))
	for _, d := range snapshot {
		devices = append(devices, DeviceStatus{DeviceInfo: d, Online: probeOnline(pool, d)})
	}

	stats := pool.Stats()

	Success(c, DeviceListData{
		Devices: devices,
		Stats: PoolStatsDTO{
			ActiveConnections: stats.ActiveConnections,
			TotalConnections:  stats.TotalConnections,
			Errors:            stats.Errors,
		},
	}, "Devices retrieved successfully")
}

// AddDevice adds a new device
//
// @Summary  添加设备并即时建连
// @Tags     devices
// @Accept   json
// @Produce  json
// @Param    device body AddDeviceRequest true "设备连接信息"
// @Success  200 {object} Response "添加成功"
// @Failure  400 {object} Response "请求非法"
// @Failure  500 {object} Response "连接设备失败"
// @Router   /devices [post]
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

	dev := DeviceInfo{
		IP:       req.IP,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
	}
	h.mu.Lock()
	h.devices[req.IP] = dev
	h.mu.Unlock()
	h.putStore(dev)

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
//
// @Summary  移除设备并释放连接
// @Tags     devices
// @Produce  json
// @Param    ip path string true "设备 IP"
// @Success  200 {object} Response "移除成功"
// @Router   /devices/{ip} [delete]
func (h *DeviceHandler) RemoveDevice(c *gin.Context) {
	ip := c.Param("ip")

	h.mu.Lock()
	delete(h.devices, ip)
	h.mu.Unlock()
	if ds := h.manager.GetDeviceStore(); ds != nil {
		ds.Delete(ip)
	}

	// Close the connection
	pool := h.manager.GetClientPool()
	pool.Release(ip)

	Success(c, nil, "Device removed successfully")
}

// GetStatus gets device status
//
// @Summary  查询设备运行/连接状态
// @Tags     devices
// @Produce  json
// @Param    ip path string true "设备 IP"
// @Success  200 {object} Response{data=DeviceConnStatus} "运行与连接状态"
// @Failure  404 {object} Response "设备不存在"
// @Router   /devices/{ip}/status [get]
func (h *DeviceHandler) GetStatus(c *gin.Context) {
	ip := c.Param("ip")

	devInfo, exists := h.lookupDevice(ip)
	if !exists {
		Error(c, 404, "Device not found")
		return
	}

	// Get client from pool and check connection status
	pool := h.manager.GetClientPool()

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

	Success(c, DeviceConnStatus{
		Running:   true, // API server is always running
		Connected: connected,
	}, "Status retrieved")
}
