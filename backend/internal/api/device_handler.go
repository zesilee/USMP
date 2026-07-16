package api

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/backend/pkg/translator"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
)

// defaultVendor is the vendor assumed when registration omits one — keeps the
// pre-registry behavior intact (all existing devices are Huawei switches).
const defaultVendor = "huawei"

// DeviceInfo stores device connection information
type DeviceInfo struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	// Vendor 厂商标识（SND 驱动选择，缺省 huawei）
	Vendor string `json:"vendor,omitempty"`
}

// DeviceHandler handles device-related API requests. Device connection info
// lives solely in the shared DeviceStore (single source of truth); the handler
// holds no private device map.
type DeviceHandler struct {
	manager manager.Manager
}

// NewDeviceHandler creates a new DeviceHandler
func NewDeviceHandler(manager manager.Manager) *DeviceHandler {
	h := &DeviceHandler{manager: manager}

	// DS-03: 种子设备不再硬编码，仅经 USMP_SEED_DEVICE 注入（供本地开发/E2E；
	// 集群模式下设备集合来自 Device CR，该变量被忽略——W2 装配时生效）。
	if seed, ok := parseSeedDevice(os.Getenv("USMP_SEED_DEVICE")); ok {
		h.putStore(seed)
	}

	return h
}

// parseSeedDevice parses USMP_SEED_DEVICE: "ip[:port],username,password[,vendor]"
// (port 缺省 830，vendor 缺省 huawei)。格式错误仅告警不入库（R08）。
func parseSeedDevice(raw string) (DeviceInfo, bool) {
	if raw == "" {
		return DeviceInfo{}, false
	}
	parts := strings.Split(raw, ",")
	if len(parts) < 3 || len(parts) > 4 {
		log.Printf("device: invalid USMP_SEED_DEVICE %q (want ip[:port],user,pass[,vendor]); no seed", raw)
		return DeviceInfo{}, false
	}
	host, user, pass := strings.TrimSpace(parts[0]), parts[1], parts[2]
	port := 830
	if ip, portStr, found := strings.Cut(host, ":"); found {
		p, err := strconv.Atoi(portStr)
		if err != nil || p <= 0 || p > 65535 {
			log.Printf("device: invalid port in USMP_SEED_DEVICE %q; no seed", raw)
			return DeviceInfo{}, false
		}
		host, port = ip, p
	}
	if host == "" || user == "" || pass == "" {
		log.Printf("device: empty field in USMP_SEED_DEVICE %q; no seed", raw)
		return DeviceInfo{}, false
	}
	vendor := defaultVendor
	if len(parts) == 4 && strings.TrimSpace(parts[3]) != "" {
		vendor = strings.TrimSpace(parts[3])
	}
	return DeviceInfo{IP: host, Port: port, Username: user, Password: pass, Vendor: vendor}, true
}

// toConnInfo maps a DeviceInfo to the shared store's connection-info type,
// defaulting Protocol to AUTO (port-based NETCONF/gNMI selection) and Vendor
// to huawei (DS-01 零值缺省语义).
func toConnInfo(d DeviceInfo) client.DeviceConnectionInfo {
	vendor := d.Vendor
	if vendor == "" {
		vendor = defaultVendor
	}
	return client.DeviceConnectionInfo{
		IP:       d.IP,
		Port:     d.Port,
		Username: d.Username,
		Password: d.Password,
		Protocol: client.ProtocolAUTO,
		Vendor:   vendor,
	}
}

// deviceInfoFromConn is the inverse of toConnInfo for API responses.
func deviceInfoFromConn(info client.DeviceConnectionInfo) DeviceInfo {
	return DeviceInfo{IP: info.IP, Port: info.Port, Username: info.Username, Password: info.Password, Vendor: info.Vendor}
}

// putStore registers a device in the shared DeviceStore (no-op if the manager
// exposes none, keeping the handler usable in minimal test setups).
func (h *DeviceHandler) putStore(d DeviceInfo) {
	if ds := h.manager.GetDeviceStore(); ds != nil {
		ds.Put(d.IP, toConnInfo(d))
	}
}

// snapshotDevices returns all registered devices from the shared DeviceStore.
func (h *DeviceHandler) snapshotDevices() []DeviceInfo {
	ds := h.manager.GetDeviceStore()
	if ds == nil {
		return nil
	}
	ids := ds.List()
	out := make([]DeviceInfo, 0, len(ids))
	for _, id := range ids {
		if info, ok := ds.Get(id); ok {
			out = append(out, deviceInfoFromConn(info))
		}
	}
	return out
}

// lookupDevice resolves a single device by IP from the shared DeviceStore.
func (h *DeviceHandler) lookupDevice(ip string) (DeviceInfo, bool) {
	ds := h.manager.GetDeviceStore()
	if ds == nil {
		return DeviceInfo{}, false
	}
	if info, ok := ds.Get(ip); ok {
		return deviceInfoFromConn(info), true
	}
	return DeviceInfo{}, false
}

// AddDeviceRequest is the request body for adding a device
type AddDeviceRequest struct {
	IP       string `json:"ip" binding:"required"`
	Port     int    `json:"port"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	// Vendor 可选厂商标识：缺省 huawei；无已注册驱动的厂商将被拒绝（400）
	Vendor string `json:"vendor,omitempty"`
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
	info := toConnInfo(d)
	if info.Port == 0 {
		info.Port = 830
	}
	// Short probe timeout — an unreachable device must not hang the API.
	info.Timeout = 3 * time.Second
	c, err := pool.Get(info)
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

	// Vendor 门禁（BR-04）：缺省 huawei；无已注册驱动的厂商早失败（优于下发时报错）。
	// 标签大小写无关（"huawei" → 枚举 "Huawei"），存储侧保留小写标签。
	if req.Vendor == "" {
		req.Vendor = defaultVendor
	}
	vt, known := translator.VendorFromString(req.Vendor)
	if !known || !translator.IsVendorSupported(vt) {
		Error(c, 400, "Invalid request: unsupported vendor '"+req.Vendor+"'")
		return
	}

	added := DeviceInfo{
		IP:       req.IP,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
		Vendor:   req.Vendor,
	}
	h.putStore(added)

	// Try to create client and connect immediately
	pool := h.manager.GetClientPool()
	_, err := pool.Get(toConnInfo(added))
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
