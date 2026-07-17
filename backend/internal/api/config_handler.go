package api

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/internal/intent"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/audit"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
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
	// pushDelete 同步下发单条目删除（BR-09，命令语义）。可注入以便无设备/模拟器
	// 测试；默认 pushDeleteToDevice。
	pushDelete func(ctx context.Context, ip string, target interface{}) error
}

// NewConfigHandler creates a new ConfigHandler
func NewConfigHandler(mgr manager.Manager) *ConfigHandler {
	h := &ConfigHandler{manager: mgr}
	h.fetch = h.fetchFromDevice
	h.pushDelete = h.pushDeleteToDevice
	return h
}

// fetchFromDevice reads running config from the device via the client pool.
func (h *ConfigHandler) fetchFromDevice(ctx context.Context, ip, path string) (interface{}, error) {
	// DS-06: resolve full connection info via the shared helper (unregistered
	// devices degrade to AUTO/no-credential).
	info, _ := device.ResolveConn(h.manager.GetDeviceStore(), ip)
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
	// OwnershipWarning 软归属提示（BR-11）：路径被业务意图认领时附带，不拦截。
	OwnershipWarning *OwnershipWarning `json:"ownershipWarning,omitempty"`
}

// OwnershipWarning 标记手改命中业务意图认领路径（BIO-07 软归属：意图收敛会覆盖手改）。
type OwnershipWarning struct {
	// Intents 认领该路径的意图 CR（namespace/name）。
	Intents []string `json:"intents"`
	Message string   `json:"message"`
}

// ownershipWarningFor 查归属索引，未认领返回 nil（响应体零噪音）。
func ownershipWarningFor(device, path string) *OwnershipWarning {
	owners := intent.DefaultOwnership.Owners(device, path)
	if len(owners) == 0 {
		return nil
	}
	return &OwnershipWarning{
		Intents: owners,
		Message: "该路径由业务网络配置管理，意图收敛时会覆盖此手工修改",
	}
}

// OwnershipRejection 是归属硬锁 409 的 data 负载（前端据此渲染阻断确认流）。
type OwnershipRejection struct {
	// Intents 认领该路径的意图 CR（namespace/name）。
	Intents []string `json:"intents"`
}

// rejectOwnedPath 以信封码 409 拒绝命中认领路径的手改（BR-11 二期硬锁）。
func rejectOwnedPath(c *gin.Context, owners []string) {
	ErrorWithData(c, 409,
		"路径由业务意图 "+strings.Join(owners, "、")+" 管理：请先删除/修改对应意图，或携带 force=true 强制下发（意图收敛仍会覆盖）",
		OwnershipRejection{Intents: owners})
}

// forcedOwners 是审计 ForcedOwners 的取值：仅 force 覆盖时留名单，普通下发零噪音。
func forcedOwners(force bool, owners []string) []string {
	if !force || len(owners) == 0 {
		return nil
	}
	return append([]string(nil), owners...)
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
// @Param    force  query bool                   false "覆盖业务意图归属硬锁（force=true，审计留痕）"
// @Success  200 {object} Response{data=ConfigSetData} "已接受，对账进行中"
// @Failure  400 {object} Response "请求或配置解析失败"
// @Failure  409 {object} Response{data=OwnershipRejection} "路径被业务意图认领（无 force 拒绝）"
// @Failure  500 {object} Response "存储失败"
// @Router   /config/{ip}/{path} [post]
func (h *ConfigHandler) SetConfig(c *gin.Context) {
	ip := c.Param("ip")
	path := c.Param("path") // *path already includes leading slash

	// 归属硬锁（BR-11 二期）：认领路径缺省 409 早拒（编解码/建连之前），
	// force=true 放行且后续审计留痕。被拒请求不产生审计记录（OA-01）。
	force := c.Query("force") == "true"
	owners := intent.DefaultOwnership.Owners(ip, path)
	if len(owners) > 0 && !force {
		rejectOwnedPath(c, owners)
		return
	}

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		Error(c, 400, "Invalid request: "+err.Error())
		return
	}

	// Convert the raw data to the appropriate YANG model struct
	// This ensures the ConfigStore stores properly typed data that the
	// Reconciler can work with for diff calculation
	desiredConfig, anchor, err := convertConfigAnchored(path, data)
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
	// desired 以描述符锚点为 key 存储（BR-05）：解码值以锚点为根，子路径下发归一化，
	// 周期对账（模块路径入队）与本次触发读同一 key，不产生分叉副本。
	configStore := h.manager.GetConfigStore()
	if err := storeConfigMerged(configStore, ip, anchor, desiredConfig); err != nil {
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
	controllerFound := h.manager.TriggerReconcile(ip, anchor)

	// 记录操作审计（§8 本地 JSON）。仅在成功接受下发后记录——被拒下发(400)不产生
	// 审计记录。诚实字段：ip/path/提交摘要/是否触发对账/时间(store 自打)/actor(system)；
	// reconcile 结局不在此存，查询 /logs 时按当前态 live-join。
	h.manager.GetAuditStore().Record(audit.Record{
		DeviceIP:  ip,
		Path:      path,
		Summary:   summarizeSubmitted(data),
		Triggered: controllerFound,
		// force 覆盖归属硬锁必须留痕（OA-01 二期）：owners 非空到达此处仅可能是 force。
		Forced:       force && len(owners) > 0,
		ForcedOwners: forcedOwners(force, owners),
	})

	Success(c, ConfigSetData{
		Status: "ACCEPTED",
		Path:   path,
		Reconciliation: ReconcileInfo{
			Triggered: controllerFound,
			Message:   "Configuration stored. Reconciliation will sync device state.",
		},
		OwnershipWarning: ownershipWarningFor(ip, path),
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

// ConfigDeleteData 是 DELETE /config 的 data 负载（命令语义：同步下发成功才返回）。
type ConfigDeleteData struct {
	Status         string        `json:"status"`
	Path           string        `json:"path"`
	Key            string        `json:"key"`
	Reconciliation ReconcileInfo `json:"reconciliation"`
	// OwnershipWarning 软归属提示（BR-11）：条目被业务意图认领时附带，不拦截。
	OwnershipWarning *OwnershipWarning `json:"ownershipWarning,omitempty"`
}

// pushDeleteToDevice 经客户端池同步下发单条目删除（candidate→commit，DP-04/DP-07）。
func (h *ConfigHandler) pushDeleteToDevice(ctx context.Context, ip string, target interface{}) error {
	// DS-06: resolve via the shared helper（未注册降级 AUTO/空凭据）。
	info, _ := device.ResolveConn(h.manager.GetDeviceStore(), ip)
	cli, err := h.manager.GetClientPool().Get(info)
	if err != nil {
		return err
	}
	if !cli.IsConnected() {
		return errDeviceNotConnected
	}
	result, err := cli.Set(ctx, []client.Change{{Type: client.DeleteChange, OldValue: target}}, client.WithCommit(true))
	// per-change 错误优先：聚合错误（"one or more changes failed"）会吞掉设备的
	// data-missing 等细节（§9 诚实透出）。
	if result != nil && !result.Success {
		for _, cr := range result.Changes {
			if cr.Error != nil {
				return cr.Error
			}
		}
	}
	if err != nil {
		return err
	}
	return nil
}

// DeleteConfig 以命令语义删除列表单条目（BR-09/BR-10）
//
// @Summary  删除设备指定 YANG 列表路径下的单条目
// @Tags     config
// @Produce  json
// @Param    ip   path  string true "设备 IP"
// @Param    path path  string true "YANG 列表路径"
// @Param    key  query string true "条目主键（vlan→id，interface→name）"
// @Param    force query bool  false "覆盖业务意图归属硬锁（force=true，审计留痕）"
// @Success  200 {object} Response{data=ConfigDeleteData} "删除成功"
// @Failure  400 {object} Response "非法 key / 未知路径 / 模型门禁拒绝"
// @Failure  409 {object} Response{data=OwnershipRejection} "条目被业务意图认领（无 force 拒绝）"
// @Failure  502 {object} Response "设备删除失败（含 data-missing）"
// @Router   /config/{ip}/{path} [delete]
func (h *ConfigHandler) DeleteConfig(c *gin.Context) {
	ip := c.Param("ip")
	path := c.Param("path")
	key := c.Query("key")

	// 归属硬锁（BR-11 二期）：认领条目缺省 409 早拒，force=true 放行留痕。
	force := c.Query("force") == "true"
	owners := intent.DefaultOwnership.Owners(ip, path)
	if len(owners) > 0 && !force {
		rejectOwnedPath(c, owners)
		return
	}

	// 模型门禁先行（BR-10）：operation-exclude/readonly 拒绝比未知路径更明确。
	if err := deleteGate(h.manager.GetSchema(), path); err != nil {
		Error(c, 400, "删除被模型门禁拒绝: "+err.Error())
		return
	}
	target, err := parseDeleteTarget(path, key)
	if err != nil {
		Error(c, 400, "无法解析删除目标: "+err.Error())
		return
	}

	// 先移 desired 再下发（design D4）：对账不会把刚删的条目加回。
	if err := storeConfigDeleted(h.manager.GetConfigStore(), ip, path, target); err != nil {
		Error(c, 500, "移除期望配置失败: "+err.Error())
		return
	}

	// 同步下发（命令语义）：失败原样透出，不失效缓存、不写审计（§9）。
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	if err := h.pushDelete(ctx, ip, target); err != nil {
		Error(c, 502, "设备删除失败: "+err.Error())
		return
	}

	// 成功：失效运行缓存 + 审计 + 触发对账验证收敛。
	h.manager.GetRunningCache().InvalidatePrefix(ip + "|")
	controllerFound := h.manager.TriggerReconcile(ip, path)
	h.manager.GetAuditStore().Record(audit.Record{
		DeviceIP:  ip,
		Path:      path,
		Summary:   summarizeDeleted(target),
		Triggered: controllerFound,
		Forced:    force && len(owners) > 0,
		// force 覆盖归属硬锁必须留痕（OA-01 二期）。
		ForcedOwners: forcedOwners(force, owners),
	})

	Success(c, ConfigDeleteData{
		Status: "DELETED",
		Path:   path,
		Key:    key,
		Reconciliation: ReconcileInfo{
			Triggered: controllerFound,
			Message:   "Entry deleted on device. Reconciliation will verify convergence.",
		},
		OwnershipWarning: ownershipWarningFor(ip, path),
	}, "Entry deleted")
}
