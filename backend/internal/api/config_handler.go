package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/audit"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
)

// errDeviceNotConnected marks a fetch failure due to the device being offline,
// so GetConfig can map it to 503 (vs 500 for other fetch errors).
var errDeviceNotConnected = errors.New("device is not connected")

// ConfigHandler handles configuration API requests
type ConfigHandler struct {
	manager manager.Manager
	// fetch reads a device's running config at a path. Injectable so cache
	// behaviour can be tested without a device/sim; defaults to fetchFromDevice.
	fetch func(ctx context.Context, ip, path string) (interface{}, error)
}

// NewConfigHandler creates a new ConfigHandler
func NewConfigHandler(mgr manager.Manager) *ConfigHandler {
	h := &ConfigHandler{manager: mgr}
	h.fetch = h.fetchFromDevice
	return h
}

// fetchFromDevice reads running config from the device via the client pool.
func (h *ConfigHandler) fetchFromDevice(ctx context.Context, ip, path string) (interface{}, error) {
	// Resolve full connection info (port + credentials + protocol) from the shared
	// DeviceStore. Fall back to an AUTO/no-cred stub for unregistered devices —
	// Protocol must be set or the client factory hits its "unsupported protocol"
	// default branch.
	info := client.DeviceConnectionInfo{IP: ip, Protocol: client.ProtocolAUTO}
	if ds := h.manager.GetDeviceStore(); ds != nil {
		if stored, ok := ds.Get(ip); ok {
			info = stored
		}
	}
	cli, err := h.manager.GetClientPool().Get(info)
	if err != nil {
		return nil, err
	}
	if !cli.IsConnected() {
		return nil, errDeviceNotConnected
	}
	result, err := cli.Get(ctx, path, client.WithDatastore("running"))
	if err != nil {
		return nil, err
	}
	return decodeRunningConfig(path, result.Data), nil
}

// runKey builds the running-cache key "ip|path", normalising a trailing slash
// so "/vlans" and "/vlans/" map to the same entry.
func runKey(ip, path string) string {
	p := strings.TrimRight(path, "/")
	if p == "" {
		p = "/"
	}
	return ip + "|" + p
}

// ConfigGetData 是 GET /config 的 data 负载。Data 为动态 YANG 配置（结构随路径而变）。
// Cached/CacheAgeSeconds/TTLSeconds/Source 描述数据新鲜度（供前端新鲜度环）。
// 注意：命中缓存只表「配置新鲜度」，设备在线/离线判定走 /devices/:ip/status。
type ConfigGetData struct {
	Data            interface{} `json:"data"`
	Cached          bool        `json:"cached"`
	CacheAgeSeconds int         `json:"cache_age_seconds"`
	TTLSeconds      int         `json:"ttl_seconds"`
	Source          string      `json:"source"` // "device" | "cache"
}

// ReconcileInfo 描述下发后的异步对账触发状态。
type ReconcileInfo struct {
	Triggered bool   `json:"triggered"`
	Message   string `json:"message"`
}

// ConfigSetData 是 POST /config 的 data 负载（声明式下发 + 对账）。
type ConfigSetData struct {
	Status         string        `json:"status"`
	Path           string        `json:"path"`
	Reconciliation ReconcileInfo `json:"reconciliation"`
}

// GetConfig gets the configuration for a specific device and YANG path
//
// @Summary  读取设备指定 YANG 路径的运行配置
// @Tags     config
// @Produce  json
// @Param    ip   path string true "设备 IP"
// @Param    path path string true "YANG 路径"
// @Success  200 {object} Response{data=ConfigGetData} "运行配置"
// @Failure  500 {object} Response "获取失败"
// @Failure  503 {object} Response "设备未连接"
// @Router   /config/{ip}/{path} [get]
func (h *ConfigHandler) GetConfig(c *gin.Context) {
	ip := c.Param("ip")
	path := c.Param("path") // *path already includes leading slash
	forceRefresh := c.Query("force_refresh") == "true"

	rc := h.manager.GetRunningCache()
	key := runKey(ip, path)
	ttlSec := int(rc.TTL().Seconds())

	// Serve fresh cache (§8 TTL 30s) unless a refresh is forced. A hit reflects
	// config freshness only; device liveness is /devices/:ip/status.
	if !forceRefresh {
		if val, age, ok := rc.GetWithAge(key); ok {
			Success(c, ConfigGetData{
				Data:            val,
				Cached:          true,
				CacheAgeSeconds: int(age.Seconds()),
				TTLSeconds:      ttlSec,
				Source:          "cache",
			}, "Configuration retrieved (cached)")
			return
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	data, err := h.fetch(ctx, ip, path)
	if err != nil {
		if errors.Is(err, errDeviceNotConnected) {
			Error(c, 503, "Device is not connected")
			return
		}
		Error(c, 500, "Failed to get configuration: "+err.Error())
		return
	}

	rc.Set(key, data)
	Success(c, ConfigGetData{
		Data:            data,
		Cached:          false,
		CacheAgeSeconds: 0,
		TTLSeconds:      ttlSec,
		Source:          "device",
	}, "Configuration retrieved")
}

// SetConfig sets the desired configuration and triggers reconciliation
// This is the DECLARATIVE API: desired state is stored, and the controller
// will asynchronously reconcile the actual device state to match it.
//
// @Summary  声明式下发配置并触发对账
// @Tags     config
// @Accept   json
// @Produce  json
// @Param    ip     path string                 true "设备 IP"
// @Param    path   path string                 true "YANG 路径"
// @Param    config body map[string]interface{} true "期望配置（YANG JSON）"
// @Success  200 {object} Response{data=ConfigSetData} "已接受，对账进行中"
// @Failure  400 {object} Response "请求或配置解析失败"
// @Failure  500 {object} Response "存储失败"
// @Router   /config/{ip}/{path} [post]
func (h *ConfigHandler) SetConfig(c *gin.Context) {
	ip := c.Param("ip")
	path := c.Param("path") // *path already includes leading slash

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		Error(c, 400, "Invalid request: "+err.Error())
		return
	}

	// Convert the raw data to the appropriate YANG model struct
	// This ensures the ConfigStore stores properly typed data that the
	// Reconciler can work with for diff calculation
	desiredConfig, err := convertConfig(path, data)
	if err != nil {
		Error(c, 400, "Failed to parse configuration: "+err.Error())
		return
	}

	// 域约束校验（YANG 模型未编码的业务范围，如 VLAN ID 1-4094）——非法值必须被拒，
	// 不能静默下发到设备（§9 前端表单校验的服务端权威兜底）。
	if verr := validateConfig(desiredConfig); verr != nil {
		Error(c, 400, "配置校验失败: "+verr.Error())
		return
	}

	// Store the desired configuration in ConfigStore.
	//
	// 合并语义（防数据丢失）：UI 每次只提交单个 VLAN/接口，但对账把 desired 当「完整状态」。
	// 若直接覆盖，第二次下发会让对账删除设备上已有但本次未提交的条目。故先并入已存 desired
	// （按 key union），使 desired 累积为完整意图。删除走独立 DELETE 端点，不经此路径。
	configStore := h.manager.GetConfigStore()
	if err := storeConfigMerged(configStore, ip, path, desiredConfig); err != nil {
		Error(c, 500, "Failed to store configuration: "+err.Error())
		return
	}

	// Invalidate this device's cached running config (§8: 下发后主动失效), by
	// prefix so any sub-path reads are cleared too. Only after a successful
	// store — a rejected push must not evict good cache.
	h.manager.GetRunningCache().InvalidatePrefix(ip + "|")

	// Trigger immediate reconciliation
	// The controller will:
	// 1. Get actual config from device
	// 2. Calculate diff between desired and actual
	// 3. Apply changes to device
	// 4. Commit (if supported by protocol)
	controllerFound := h.manager.TriggerReconcile(ip, path)

	// 记录操作审计（§8 本地 JSON）。仅在成功接受下发后记录——被拒下发(400)不产生
	// 审计记录。诚实字段：ip/path/提交摘要/是否触发对账/时间(store 自打)/actor(system)；
	// reconcile 结局不在此存，查询 /logs 时按当前态 live-join。
	h.manager.GetAuditStore().Record(audit.Record{
		DeviceIP:  ip,
		Path:      path,
		Summary:   summarizeSubmitted(data),
		Triggered: controllerFound,
	})

	Success(c, ConfigSetData{
		Status: "ACCEPTED",
		Path:   path,
		Reconciliation: ReconcileInfo{
			Triggered: controllerFound,
			Message:   "Configuration stored. Reconciliation will sync device state.",
		},
	}, "Configuration accepted - reconciliation in progress")
}

// validateConfig 对已转换的配置做 YANG 模型未编码的域约束校验。华为 VLAN 模型未在 schema
// 里编码 VLAN ID 范围，故此处显式校验 1-4094（0/4095+ 为保留/非法，真机会拒绝或误配）。
func validateConfig(cfg interface{}) error {
	if v, ok := cfg.(*huawei.HuaweiVlan_Vlan_Vlans); ok {
		for id := range v.Vlan {
			if id < 1 || id > 4094 {
				return fmt.Errorf("VLAN ID %d 超出有效范围 [1, 4094]", id)
			}
		}
	}
	return nil
}

// configMergeMu 串行化 Get→merge→Set 临界区，避免并发下发时的丢更新与竞态（R09）。
// 配置下发频率低，单锁足够；storeConfigMerged 是唯一并发写入口。
var configMergeMu sync.Mutex

// storeConfigMerged 原子地把 incoming 并入已存 desired 并存回（加锁串行化）。SetConfig
// 与集成测试共用，保证「先读后并再写」不被并发打断。
func storeConfigMerged(cs reconcile.ConfigStore, ip, path string, incoming interface{}) error {
	configMergeMu.Lock()
	defer configMergeMu.Unlock()
	desired := incoming
	if existing, gerr := cs.Get(ip, path); gerr == nil && existing != nil {
		desired = mergeConfig(existing, incoming)
	}
	return cs.Set(ip, path, desired)
}

// mergeConfig 把新提交的配置并入已存 desired（按列表主键 union），使增量 UI 提交不会
// 让声明式对账删除设备上已有条目。同键以新值覆盖（=编辑）。
// 构造全新对象（不原地改共享 stored 对象），使并发读（对账器）不会读到正在变更的 map。
// 非列表类型（如 System 单例）无既有合并语义，直接返回新值。
func mergeConfig(existing, incoming interface{}) interface{} {
	switch inc := incoming.(type) {
	case *huawei.HuaweiVlan_Vlan_Vlans:
		if ex, ok := existing.(*huawei.HuaweiVlan_Vlan_Vlans); ok && ex != nil {
			merged := &huawei.HuaweiVlan_Vlan_Vlans{Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{}}
			for k, v := range ex.Vlan {
				merged.Vlan[k] = v
			}
			for k, v := range inc.Vlan {
				merged.Vlan[k] = v
			}
			return merged
		}
	case *huawei.HuaweiIfm_Ifm_Interfaces:
		if ex, ok := existing.(*huawei.HuaweiIfm_Ifm_Interfaces); ok && ex != nil {
			merged := &huawei.HuaweiIfm_Ifm_Interfaces{Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{}}
			for k, v := range ex.Interface {
				merged.Interface[k] = v
			}
			for k, v := range inc.Interface {
				merged.Interface[k] = v
			}
			return merged
		}
	}
	return incoming
}

// convertToTypedStruct converts raw JSON map to the appropriate strongly-typed
// YANG model struct based on the path. This ensures proper diff calculation.
func convertToTypedStruct(path string, data map[string]interface{}) (interface{}, error) {
	// Huawei System
	if strings.Contains(path, "system:") {
		return convertMapToHuaweiSystem(data)
	}

	// Huawei IFM Interfaces
	if strings.Contains(path, "ifm:ifm") && strings.Contains(path, "interfaces") {
		return convertMapToHuaweiIfm(data)
	}

	// Huawei VLANs
	if strings.Contains(path, "vlan:") && (strings.Contains(path, "vlan") || strings.Contains(path, "vlans")) {
		return convertMapToHuaweiVlan(data)
	}

	// Fallback: return the raw map for unhandled paths. Log a warning so unknown
	// paths are visible rather than silently degraded (R08 — no silent truncation).
	log.Printf("config-api: no typed codec for path %q, storing raw map", path)
	return data, nil
}

// convertMapToHuaweiIfm converts a map to HuaweiIfm_Ifm_Interfaces struct
func convertMapToHuaweiIfm(data map[string]interface{}) (*huawei.HuaweiIfm_Ifm_Interfaces, error) {
	result := &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: make(map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface),
	}

	// Extract interface array/map
	var ifacesData interface{}
	if v, ok := data["interface"]; ok {
		ifacesData = v
	} else if v, ok := data["Interface"]; ok {
		ifacesData = v
	} else if v, ok := data["interfaces"]; ok {
		ifacesData = v
	} else if v, ok := data["Interfaces"]; ok {
		ifacesData = v
	} else {
		// If no interface container, assume the data itself is the interface config
		iface := mapEntryToInterface(data)
		key := "default"
		if iface.Name != nil {
			key = *iface.Name
		}
		result.Interface[key] = iface
		return result, nil
	}

	// Handle both slice and map formats
	// Note: JSON unmarshal always produces []interface{} for arrays
	switch v := ifacesData.(type) {
	case []interface{}:
		for i, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				iface := mapEntryToInterface(m)
				key := fmt.Sprintf("iface-%d", i)
				if iface.Name != nil {
					key = *iface.Name
				}
				result.Interface[key] = iface
			}
		}
	case map[string]interface{}:
		for key, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				iface := mapEntryToInterface(m)
				if iface.Name == nil {
					iface.Name = &key
				}
				result.Interface[key] = iface
			}
		}
	}

	return result, nil
}

// mapEntryToInterface converts a single interface entry map to struct
func mapEntryToInterface(m map[string]interface{}) *huawei.HuaweiIfm_Ifm_Interfaces_Interface {
	result := &huawei.HuaweiIfm_Ifm_Interfaces_Interface{}

	for k, v := range m {
		key := strings.ToLower(strings.ReplaceAll(k, "-", ""))
		switch key {
		// ===== 基础属性 =====
		case "name":
			if s, ok := v.(string); ok {
				result.Name = &s
			}
		case "description":
			if s, ok := v.(string); ok {
				result.Description = &s
			}
		case "index":
			if num, ok := valueToUint(v); ok {
				uint32Val := uint32(num)
				result.Index = &uint32Val
			}
		case "number":
			if s, ok := v.(string); ok {
				result.Number = &s
			}
		case "position":
			if s, ok := v.(string); ok {
				result.Position = &s
			}
		case "parentname":
			if s, ok := v.(string); ok {
				result.ParentName = &s
			}

		// ===== 状态和类型 =====
		case "adminstatus":
			if n, ok := enumInt(v, "E_HuaweiIfm_PortStatus"); ok {
				result.AdminStatus = huawei.E_HuaweiIfm_PortStatus(n)
			}
		case "type":
			if n, ok := enumInt(v, "E_HuaweiIfm_PortType"); ok {
				result.Type = huawei.E_HuaweiIfm_PortType(n)
			}
		case "class":
			if n, ok := enumInt(v, "E_HuaweiIfm_ClassType"); ok {
				result.Class = huawei.E_HuaweiIfm_ClassType(n)
			}
		case "linkprotocol":
			if n, ok := enumInt(v, "E_HuaweiIfm_LinkProtocol"); ok {
				result.LinkProtocol = huawei.E_HuaweiIfm_LinkProtocol(n)
			}
		case "routertype":
			if n, ok := enumInt(v, "E_HuaweiIfm_RouterType"); ok {
				result.RouterType = huawei.E_HuaweiIfm_RouterType(n)
			}
		case "servicetype":
			if n, ok := enumInt(v, "E_HuaweiIfm_ServiceType"); ok {
				result.ServiceType = huawei.E_HuaweiIfm_ServiceType(n)
			}

		// ===== 网络参数 =====
		case "mtu":
			if num, ok := valueToUint(v); ok {
				uint32Val := uint32(num)
				result.Mtu = &uint32Val
			}
		case "macaddress":
			if s, ok := v.(string); ok {
				result.MacAddress = &s
			}
		case "bandwidth":
			if num, ok := valueToUint(v); ok {
				uint32Val := uint32(num)
				result.Bandwidth = &uint32Val
			}
		case "bandwidthkbps":
			if num, ok := valueToUint(v); ok {
				uint32Val := uint32(num)
				result.BandwidthKbps = &uint32Val
			}
		case "vrfname":
			if s, ok := v.(string); ok {
				result.VrfName = &s
			}
		case "vsname":
			if s, ok := v.(string); ok {
				result.VsName = &s
			}

		// ===== 链路聚合 =====
		case "aggregationname":
			if s, ok := v.(string); ok {
				result.AggregationName = &s
			}

		// ===== 定时器和延迟 =====
		case "downdelaytime":
			if num, ok := valueToUint(v); ok {
				uint32Val := uint32(num)
				result.DownDelayTime = &uint32Val
			}
		case "protocolupdelaytime":
			if num, ok := valueToUint(v); ok {
				uint32Val := uint32(num)
				result.ProtocolUpDelayTime = &uint32Val
			}

		// ===== 功能开关 =====
		case "clearipdf":
			if b, ok := v.(bool); ok {
				result.ClearIpDf = &b
			}
		case "isl2switch":
			if b, ok := v.(bool); ok {
				result.IsL2Switch = &b
			}
		case "l2modeenable":
			if b, ok := v.(bool); ok {
				result.L2ModeEnable = &b
			}
		case "linkupdowntrapenable":
			if b, ok := v.(bool); ok {
				result.LinkUpDownTrapEnable = &b
			}
		case "statisticenable":
			if b, ok := v.(bool); ok {
				result.StatisticEnable = &b
			}
		case "spreadmtuflag":
			if b, ok := v.(bool); ok {
				result.SpreadMtuFlag = &b
			}

		// ===== 统计配置 =====
		case "statisticinterval":
			if num, ok := valueToUint(v); ok {
				uint32Val := uint32(num)
				result.StatisticInterval = &uint32Val
			}
		case "statisticmode":
			if num, ok := valueToUint(v); ok {
				result.StatisticMode = huawei.E_HuaweiIfm_StatisticMode(num)
			}

		// ===== 嵌套容器 =====
		case "controlflap":
			if nested, ok := v.(map[string]interface{}); ok {
				result.ControlFlap = mapToInterfaceControlFlap(nested)
			}
		case "damp":
			if nested, ok := v.(map[string]interface{}); ok {
				result.Damp = mapToInterfaceDamp(nested)
			}
		}
	}

	return result
}

// mapToInterfaceControlFlap converts map to ControlFlap struct
func mapToInterfaceControlFlap(m map[string]interface{}) *huawei.HuaweiIfm_Ifm_Interfaces_Interface_ControlFlap {
	result := &huawei.HuaweiIfm_Ifm_Interfaces_Interface_ControlFlap{}

	for k, v := range m {
		key := strings.ToLower(strings.ReplaceAll(k, "-", ""))
		switch key {
		case "ceiling":
			if num, ok := valueToUint(v); ok {
				uint32Val := uint32(num)
				result.Ceiling = &uint32Val
			}
		case "controlflapcount":
			if num, ok := valueToUint(v); ok {
				uint32Val := uint32(num)
				result.ControlFlapCount = &uint32Val
			}
		case "decayng":
			if num, ok := valueToUint(v); ok {
				uint32Val := uint32(num)
				result.DecayNg = &uint32Val
			}
		case "decayok":
			if num, ok := valueToUint(v); ok {
				uint32Val := uint32(num)
				result.DecayOk = &uint32Val
			}
		case "reuse":
			if num, ok := valueToUint(v); ok {
				uint32Val := uint32(num)
				result.Reuse = &uint32Val
			}
		case "suppress":
			if num, ok := valueToUint(v); ok {
				uint32Val := uint32(num)
				result.Suppress = &uint32Val
			}
		}
	}

	return result
}

// mapToInterfaceDamp converts map to Damp struct
func mapToInterfaceDamp(m map[string]interface{}) *huawei.HuaweiIfm_Ifm_Interfaces_Interface_Damp {
	result := &huawei.HuaweiIfm_Ifm_Interfaces_Interface_Damp{}

	for k, v := range m {
		key := strings.ToLower(strings.ReplaceAll(k, "-", ""))
		switch key {
		case "txoff":
			if b, ok := v.(bool); ok {
				result.TxOff = &b
			}
		case "auto":
			if nested, ok := v.(map[string]interface{}); ok {
				result.Auto = mapToInterfaceDampAuto(nested)
			}
		case "manual":
			if nested, ok := v.(map[string]interface{}); ok {
				result.Manual = mapToInterfaceDampManual(nested)
			}
		}
	}

	return result
}

// mapToInterfaceDampAuto converts map to Damp.Auto struct
func mapToInterfaceDampAuto(m map[string]interface{}) *huawei.HuaweiIfm_Ifm_Interfaces_Interface_Damp_Auto {
	result := &huawei.HuaweiIfm_Ifm_Interfaces_Interface_Damp_Auto{}

	for k, v := range m {
		key := strings.ToLower(strings.ReplaceAll(k, "-", ""))
		switch key {
		case "level":
			if num, ok := valueToUint(v); ok {
				result.Level = huawei.E_HuaweiIfm_DampLevelType(num)
			}
		}
	}

	return result
}

// mapToInterfaceDampManual converts map to Damp.Manual struct
func mapToInterfaceDampManual(m map[string]interface{}) *huawei.HuaweiIfm_Ifm_Interfaces_Interface_Damp_Manual {
	result := &huawei.HuaweiIfm_Ifm_Interfaces_Interface_Damp_Manual{}

	for k, v := range m {
		key := strings.ToLower(strings.ReplaceAll(k, "-", ""))
		switch key {
		case "halflifeperiod":
			if num, ok := valueToUint(v); ok {
				uint16Val := uint16(num)
				result.HalfLifePeriod = &uint16Val
			}
		case "maxsuppresstime":
			if num, ok := valueToUint(v); ok {
				uint16Val := uint16(num)
				result.MaxSuppressTime = &uint16Val
			}
		case "reuse":
			if num, ok := valueToUint(v); ok {
				uint32Val := uint32(num)
				result.Reuse = &uint32Val
			}
		case "suppress":
			if num, ok := valueToUint(v); ok {
				uint32Val := uint32(num)
				result.Suppress = &uint32Val
			}
		}
	}

	return result
}

// convertMapToHuaweiVlan converts a map to HuaweiVlan_Vlan_Vlans struct
func convertMapToHuaweiVlan(data map[string]interface{}) (*huawei.HuaweiVlan_Vlan_Vlans, error) {
	result := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: make(map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan),
	}

	// Extract vlans array
	var vlansData interface{}
	if v, ok := data["vlans"]; ok {
		vlansData = v
	} else if v, ok := data["Vlan"]; ok {
		vlansData = v
	} else {
		// If no vlans container, assume the data itself is the vlan list
		vlansData = data
	}

	// Handle both slice and map formats
	switch v := vlansData.(type) {
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				entry := mapEntryToVlan(m)
				if entry.Id != nil {
					result.Vlan[*entry.Id] = entry
				}
			}
		}
	case map[string]interface{}:
		for key, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				entry := mapEntryToVlan(m)
				if entry.Id == nil {
					// Try to parse key as VLAN ID
					if id, err := strconv.ParseUint(key, 10, 16); err == nil {
						id16 := uint16(id)
						entry.Id = &id16
					}
				}
				if entry.Id != nil {
					result.Vlan[*entry.Id] = entry
				}
			}
		}
	}

	return result, nil
}

// enumInt 把前端提交的枚举值转为 ygot 枚举整数值。兼容两种形式：
//
//	数字（旧路径，如 2）→ 直通；字符串枚举名（如 "up"）→ 经生成的 ΛEnum 反查。
//
// 表单动态渲染用字符串名（可读），故字符串路径是主用路径。
func enumInt(v interface{}, enumTypeName string) (int64, bool) {
	if num, ok := valueToUint(v); ok {
		return int64(num), true
	}
	if s, ok := v.(string); ok {
		if m, ok := huawei.ΛEnum[enumTypeName]; ok {
			for val, def := range m {
				if def.Name == s {
					return val, true
				}
			}
		}
	}
	return 0, false
}

// mapToMemberPorts 把端口成员列表转为 MemberPorts 结构。接受 [ {interface-name,
// access-type, tag-mode} ... ]（或以 interface-name 为键的 map），按 interface-name 建键。
func mapToMemberPorts(v interface{}) *huawei.HuaweiVlan_Vlan_Vlans_Vlan_MemberPorts {
	mp := &huawei.HuaweiVlan_Vlan_Vlans_Vlan_MemberPorts{
		MemberPort: map[string]*huawei.HuaweiVlan_Vlan_Vlans_Vlan_MemberPorts_MemberPort{},
	}
	add := func(m map[string]interface{}) {
		port := &huawei.HuaweiVlan_Vlan_Vlans_Vlan_MemberPorts_MemberPort{}
		for k, val := range m {
			switch strings.ToLower(strings.ReplaceAll(k, "-", "")) {
			case "interfacename":
				if s, ok := val.(string); ok {
					port.InterfaceName = &s
				}
			case "accesstype":
				if n, ok := enumInt(val, "E_HuaweiVlan_AccessType"); ok {
					port.AccessType = huawei.E_HuaweiVlan_AccessType(n)
				}
			case "tagmode":
				if n, ok := enumInt(val, "E_HuaweiVlan_TagMode"); ok {
					port.TagMode = huawei.E_HuaweiVlan_TagMode(n)
				}
			}
		}
		if port.InterfaceName != nil {
			mp.MemberPort[*port.InterfaceName] = port
		}
	}
	// member-ports 可能是 { "member-port": [...] } 包一层，先解包
	if outer, ok := v.(map[string]interface{}); ok {
		if inner, ok := outer["member-port"]; ok {
			v = inner
		} else if inner, ok := outer["memberPort"]; ok {
			v = inner
		}
	}
	switch list := v.(type) {
	case []interface{}:
		for _, it := range list {
			if m, ok := it.(map[string]interface{}); ok {
				add(m)
			}
		}
	case map[string]interface{}:
		for _, it := range list {
			if m, ok := it.(map[string]interface{}); ok {
				add(m)
			}
		}
	}
	if len(mp.MemberPort) == 0 {
		return nil
	}
	return mp
}

// mapEntryToVlan converts a single VLAN entry map to struct
func mapEntryToVlan(m map[string]interface{}) *huawei.HuaweiVlan_Vlan_Vlans_Vlan {
	result := &huawei.HuaweiVlan_Vlan_Vlans_Vlan{}

	for k, v := range m {
		key := strings.ToLower(strings.ReplaceAll(k, "-", ""))
		switch key {
		case "id":
			if num, ok := valueToUint(v); ok {
				id := uint16(num)
				result.Id = &id
			}
		case "name":
			if s, ok := v.(string); ok {
				result.Name = &s
			}
		case "description":
			if s, ok := v.(string); ok {
				result.Description = &s
			}
		case "type":
			if n, ok := enumInt(v, "E_HuaweiVlan_VlanType"); ok {
				result.Type = huawei.E_HuaweiVlan_VlanType(n)
			}
		case "adminstatus":
			if n, ok := enumInt(v, "E_HuaweiVlan_AdminStatus"); ok {
				result.AdminStatus = huawei.E_HuaweiVlan_AdminStatus(n)
			}
		case "broadcastdiscard":
			if n, ok := enumInt(v, "E_HuaweiVlan_EnableStatus"); ok {
				result.BroadcastDiscard = huawei.E_HuaweiVlan_EnableStatus(n)
			}
		case "unknownmulticastdiscard":
			if n, ok := enumInt(v, "E_HuaweiVlan_EnableStatus"); ok {
				result.UnknownMulticastDiscard = huawei.E_HuaweiVlan_EnableStatus(n)
			}
		case "memberports":
			if mp := mapToMemberPorts(v); mp != nil {
				result.MemberPorts = mp
			}
		case "maclearning":
			if n, ok := enumInt(v, "E_HuaweiVlan_EnableStatus"); ok {
				result.MacLearning = huawei.E_HuaweiVlan_EnableStatus(n)
			}
		case "macagingtime":
			if num, ok := valueToUint(v); ok {
				val := uint32(num)
				result.MacAgingTime = &val
			}
		case "statisticenable":
			if n, ok := enumInt(v, "E_HuaweiVlan_EnableStatus"); ok {
				result.StatisticEnable = huawei.E_HuaweiVlan_EnableStatus(n)
			}
		case "statisticdiscard":
			if n, ok := enumInt(v, "E_HuaweiVlan_EnableStatus"); ok {
				result.StatisticDiscard = huawei.E_HuaweiVlan_EnableStatus(n)
			}
		case "supervlan":
			if num, ok := valueToUint(v); ok {
				val := uint16(num)
				result.SuperVlan = &val
			}
		case "unkownunicastdiscard", "unknownunicastdiscard": // YANG 模型拼写为 unkown（缺 n），兼容两种
			if nested, ok := v.(map[string]interface{}); ok {
				result.UnkownUnicastDiscard = mapToUnicastDiscard(nested)
			}
		case "suppression":
			if nested, ok := v.(map[string]interface{}); ok {
				result.Suppression = mapToSuppression(nested)
			}
		}
	}

	return result
}

// mapToUnicastDiscard converts map to UnkownUnicastDiscard struct
func mapToUnicastDiscard(m map[string]interface{}) *huawei.HuaweiVlan_Vlan_Vlans_Vlan_UnkownUnicastDiscard {
	result := &huawei.HuaweiVlan_Vlan_Vlans_Vlan_UnkownUnicastDiscard{}

	for k, v := range m {
		key := strings.ToLower(strings.ReplaceAll(k, "-", ""))
		switch key {
		case "discard":
			if n, ok := enumInt(v, "E_HuaweiVlan_EnableStatus"); ok {
				result.Discard = huawei.E_HuaweiVlan_EnableStatus(n)
			}
		case "maclearningenable":
			if n, ok := enumInt(v, "E_HuaweiVlan_EnableStatus"); ok {
				result.MacLearningEnable = huawei.E_HuaweiVlan_EnableStatus(n)
			}
		}
	}

	return result
}

// mapToSuppression converts map to Suppression struct
func mapToSuppression(m map[string]interface{}) *huawei.HuaweiVlan_Vlan_Vlans_Vlan_Suppression {
	result := &huawei.HuaweiVlan_Vlan_Vlans_Vlan_Suppression{}

	for k, v := range m {
		key := strings.ToLower(strings.ReplaceAll(k, "-", ""))
		switch key {
		case "inbound":
			if n, ok := enumInt(v, "E_HuaweiVlan_EnableStatus"); ok {
				result.Inbound = huawei.E_HuaweiVlan_EnableStatus(n)
			}
		case "outbound":
			if n, ok := enumInt(v, "E_HuaweiVlan_EnableStatus"); ok {
				result.Outbound = huawei.E_HuaweiVlan_EnableStatus(n)
			}
		}
	}

	return result
}

// valueToUint converts various numeric types to uint64
func valueToUint(v interface{}) (uint64, bool) {
	switch val := v.(type) {
	case float64:
		return uint64(val), true
	case int:
		return uint64(val), true
	case int64:
		return uint64(val), true
	case uint:
		return uint64(val), true
	case uint32:
		return uint64(val), true
	case uint64:
		return val, true
	case string:
		if num, err := strconv.ParseUint(val, 10, 64); err == nil {
			return num, true
		}
	}
	return 0, false
}

// convertMapToHuaweiSystem converts a map to HuaweiSystem_System struct
func convertMapToHuaweiSystem(data map[string]interface{}) (*huawei.HuaweiSystem_System, error) {
	result := &huawei.HuaweiSystem_System{}

	// Extract system-info container
	var sysInfoData map[string]interface{}
	if v, ok := data["system-info"]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			sysInfoData = m
		}
	} else if v, ok := data["systemInfo"]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			sysInfoData = m
		}
	} else if v, ok := data["SystemInfo"]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			sysInfoData = m
		}
	} else {
		// If no system-info container, assume the data itself is the system-info config
		sysInfoData = data
	}

	if sysInfoData != nil {
		result.SystemInfo = mapEntryToSystemInfo(sysInfoData)
	}

	return result, nil
}

// mapEntryToSystemInfo converts a single system-info entry map to struct
func mapEntryToSystemInfo(m map[string]interface{}) *huawei.HuaweiSystem_System_SystemInfo {
	result := &huawei.HuaweiSystem_System_SystemInfo{}

	for k, v := range m {
		key := strings.ToLower(strings.ReplaceAll(k, "-", ""))
		switch key {
		case "sysname", "name":
			if s, ok := v.(string); ok {
				result.SysName = &s
			}
		case "syscontact", "contact":
			if s, ok := v.(string); ok {
				result.SysContact = &s
			}
		case "syslocation", "location":
			if s, ok := v.(string); ok {
				result.SysLocation = &s
			}
		// Read-only fields - ignore for configuration
		case "sysdesc", "productname", "productversion", "esn", "sysuptime":
			// These are read-only, do not set them
		}
	}

	return result
}

// summarizeSubmitted 生成下发内容的简明审计摘要（诚实、确定性）。
// 对每个顶层键：值为数组 → "键 (N)"（N=提交条目数），否则 "键"。键排序保证稳定。
// 例：{"vlans":[{id:100},{id:200}]} → "vlans (2)"。空提交 → "(空)"。
func summarizeSubmitted(data map[string]interface{}) string {
	if len(data) == 0 {
		return "(空)"
	}
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		if arr, ok := data[k].([]interface{}); ok {
			parts = append(parts, fmt.Sprintf("%s (%d)", k, len(arr)))
		} else {
			parts = append(parts, k)
		}
	}
	return strings.Join(parts, ", ")
}
