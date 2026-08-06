package api

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/backend/internal/cache"
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
	// fetchState 状态通道（<get>，include_state=true 的按需读）：绕缓存且不写
	// 缓存。可注入以便无设备测试；默认 fetchStateFromDevice。
	fetchState func(ctx context.Context, ip, path string) (interface{}, error)
	// pushDelete 同步下发单条目删除（BR-09，命令语义）。可注入以便无设备/模拟器
	// 测试；默认 pushDeleteToDevice。
	pushDelete func(ctx context.Context, ip string, target interface{}) error
	// support 返回设备的节点级不支持集视图（BR-12/CN-04），无可用连接返回 nil
	// （行为退化为既有链路）。可注入以便无设备测试；默认经连接池可选接口断言。
	support func(ip string) nodeSupportView
	// stateCache 状态快照缓存（BR-14/CC-07）：include_state 读取的短 TTL 快照，
	// 独立于运行配置缓存——写后失效（InvalidatePrefix）刻意不触及本实例，
	// TTL 经 USMP_STATE_SNAPSHOT_TTL 独立配置（默认 30s）。
	stateCache *cache.TTLLRUCache
}

// nodeSupportView 连接实现的可选能力（与 ServerCapabilities 同款断言模式，
// 不扩 client.Client 接口面）：节点级不支持集的查/标/清。
type nodeSupportView interface {
	IsUnsupportedPath(string) bool
	MarkUnsupportedPath(string)
	ClearUnsupportedPath(string)
	UnsupportedPathsUnder(string) []string
}

// unsupportedTabsFor 该设备在模块根下已学习的不支持子路径，折算为相对模块根
// 的首段局部名（CN-05，前端 Tab 预标记用）。无连接/无学习返回 nil（键省略）。
func unsupportedTabsFor(mgr manager.Manager, deviceID, rootName string) []string {
	view := supportViewFromPool(mgr, deviceID)
	if view == nil {
		return nil
	}
	prefix := rootName + ":" + rootName
	seen := map[string]bool{}
	var out []string
	for _, p := range view.UnsupportedPathsUnder(prefix) {
		rest := strings.TrimPrefix(strings.TrimPrefix(p, prefix), "/")
		if rest == "" {
			continue
		}
		seg := rest
		if i := strings.IndexByte(seg, '/'); i >= 0 {
			seg = seg[:i]
		}
		if i := strings.IndexByte(seg, '['); i >= 0 {
			seg = seg[:i]
		}
		if i := strings.LastIndexByte(seg, ':'); i >= 0 {
			seg = seg[i+1:]
		}
		if seg != "" && !seen[seg] {
			seen[seg] = true
			out = append(out, seg)
		}
	}
	sort.Strings(out)
	return out
}

// reasonNodeUnsupported BR-12 结构化错误标识：设备软件版本无此节点。
const reasonNodeUnsupported = "node-unsupported"

// nodeUnsupportedData 错误响应 data 负载（前端以 reason 判定占位态，FE-24）。
type nodeUnsupportedData struct {
	Reason string `json:"reason"`
}

func rejectNodeUnsupported(c *gin.Context, path string) {
	ErrorWithData(c, 500, "Device does not support this node: "+path,
		nodeUnsupportedData{Reason: reasonNodeUnsupported})
}

// stateSnapshotTTL 解析 USMP_STATE_SNAPSHOT_TTL（CC-07）：Go duration（"10s"）
// 或纯秒数（"10"），非法/缺省回落 30s（R08 配置错误不崩溃）。
func stateSnapshotTTL() time.Duration {
	raw := os.Getenv("USMP_STATE_SNAPSHOT_TTL")
	if raw == "" {
		return 30 * time.Second
	}
	if d, err := time.ParseDuration(raw); err == nil && d > 0 {
		return d
	}
	if sec, err := strconv.Atoi(raw); err == nil && sec > 0 {
		return time.Duration(sec) * time.Second
	}
	return 30 * time.Second
}

// NewConfigHandler creates a new ConfigHandler
func NewConfigHandler(mgr manager.Manager) *ConfigHandler {
	h := &ConfigHandler{manager: mgr}
	h.fetch = h.fetchFromDevice
	h.fetchState = h.fetchStateFromDevice
	h.pushDelete = h.pushDeleteToDevice
	h.support = h.supportFromPool
	// 容量 256：状态快照条目大（万级行整树），上限收紧靠 LRU 兜底淘汰（CC-07）。
	h.stateCache = cache.NewTTLLRUCache(256, stateSnapshotTTL(), time.Minute)
	return h
}

// supportFromPool resolves the device's connection and type-asserts the node
// support capability. 设备未注册/建连失败返回 nil（不因能力查询引入新错误面）。
func (h *ConfigHandler) supportFromPool(ip string) nodeSupportView {
	return supportViewFromPool(h.manager, ip)
}

// supportViewFromPool 包级复用（ConfigHandler/ChangesetHandler 同口径取视图）。
// 只 Peek 不拨号：能力查询绝不能触发建连（未注册/离线设备会白付拨号超时）；
// 无既有连接时返回 nil——学习/清标记的调用点都在设备请求之后，届时连接已存在。
func supportViewFromPool(mgr manager.Manager, ip string) nodeSupportView {
	info, _ := device.ResolveConn(mgr.GetDeviceStore(), ip)
	peeker, ok := mgr.GetClientPool().(interface {
		Peek(string) (client.Client, bool)
	})
	if !ok {
		return nil
	}
	cli, ok := peeker.Peek(info.IP)
	if !ok {
		return nil
	}
	v, ok := cli.(nodeSupportView)
	if !ok {
		return nil
	}
	return v
}

// fetchFromDevice reads running config from the device via the client pool.
// 快通道（真机回归）：<get-config> 只读配置。真机 <get> 在大 list 子树上收集
// 全量硬件状态可 30s+ 不回首字节（wire 抓包实证），列表/基线读一律走此通道；
// 状态经 fetchStateFromDevice 按需单独取（多为单行谓词读）。
func (h *ConfigHandler) fetchFromDevice(ctx context.Context, ip, path string) (interface{}, error) {
	return h.fetchVia(ctx, ip, path, false)
}

// fetchStateFromDevice 状态通道（BR-01/DP-09）：发 <get>，回读同时携带
// config=false 状态子树（接口 dynamic、VLAN status 等），前端只读控件回显。
func (h *ConfigHandler) fetchStateFromDevice(ctx context.Context, ip, path string) (interface{}, error) {
	return h.fetchVia(ctx, ip, path, true)
}

func (h *ConfigHandler) fetchVia(ctx context.Context, ip, path string, withState bool) (interface{}, error) {
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
	opts := []client.GetOption{client.WithDatastore("running")}
	if withState {
		opts = append(opts, client.WithStateData())
	}
	result, err := cli.Get(ctx, path, opts...)
	if err != nil {
		return nil, err
	}
	return decodeRunningConfig(path, result.Data), nil
}

// learnNodeUnsupported 归因学习（CN-04）：设备错误可归因为 unknown-element 时
// 入集并返回 true（当次即转结构化错误，无需第二次请求）。视图在错误发生后
// 再取（Peek）：请求刚打过设备，连接此刻已在池中。不可归因或无视图返回
// false（走既有错误路径）。
func (h *ConfigHandler) learnNodeUnsupported(ip, path string, err error) bool {
	if !client.UnknownElementForPath(path, err) {
		return false
	}
	view := h.support(ip)
	if view == nil {
		return false
	}
	view.MarkUnsupportedPath(path)
	return true
}

// clearUnsupported force 重试成功后的恢复动作（BR-12）。
func (h *ConfigHandler) clearUnsupported(ip, path string) {
	if view := h.support(ip); view != nil {
		view.ClearUnsupportedPath(path)
	}
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
// @Param    ip            path  string true  "设备 IP"
// @Param    path          path  string true  "YANG 路径"
// @Param    force_refresh query bool   false "绕缓存/快照强制回读设备（BR-04/BR-14）"
// @Param    include_state query bool   false "状态通道 <get>：含 config=false 数据，短 TTL 快照缓存（BR-14）"
// @Param    limit         query int    false "分页模式（BR-13，仅 YANG list 路径）：每页行数 1..1000，出现即分页，data 变为 {rows,total,limit,offset}"
// @Param    offset        query int    false "分页起始行（缺省 0）"
// @Param    filter        query []string false "过滤条件，可重复：<leaf>==<值> 等值 / <leaf>~=<值> 包含（AND 组合）" collectionFormat(multi)
// @Param    sort          query string false "排序字段（叶名，支持嵌套路径）"
// @Param    sort_dir      query string false "排序方向 asc|desc（缺省 asc）"
// @Success  200 {object} Response{data=ConfigGetData} "运行配置（无分页参数：data 为子树；分页模式：data 为行切片与总数）"
// @Failure  400 {object} Response "分页参数非法或目标非 list 节点"
// @Failure  500 {object} Response "获取失败"
// @Failure  503 {object} Response "设备未连接"
// @Router   /config/{ip}/{path} [get]
func (h *ConfigHandler) GetConfig(c *gin.Context) {
	ip := c.Param("ip")
	path := c.Param("path") // *path already includes leading slash
	forceRefresh := c.Query("force_refresh") == "true"

	// BR-13：分页参数解析先行——非法参数在触达缓存/设备前 400（不静默忽略）。
	lq, lqErr := parseListQuery(c.Request.URL.Query())
	if lqErr != nil {
		Error(c, 400, "分页参数非法: "+lqErr.Error())
		return
	}

	// 分页模式的谓词路径：设备取数/缓存键截到首个谓词段之前的父容器（整列表
	// 连键取回，快照共享；深路径 filter 选不回祖先 list 键叶，下钻会匹配不到行）。
	// 无参数读取（lq==nil）不受影响——单行谓词读契约原样。
	fetchPath := path
	if lq != nil {
		if fp, ok := predicateFetchPath(path); ok {
			fetchPath = fp
		}
	}

	rc := h.manager.GetRunningCache()
	key := runKey(ip, fetchPath)
	ttlSec := int(rc.TTL().Seconds())

	// 节点级不支持快速失败（BR-12）：已学习路径不再打设备；force_refresh 绕过
	// 重试（成功即清标记——设备升级后的恢复通道）。视图经 Peek 取自既有连接
	// （绝不拨号）；无连接=无学习记忆，检查自然跳过。
	if view := h.support(ip); view != nil && !forceRefresh && view.IsUnsupportedPath(fetchPath) {
		rejectNodeUnsupported(c, fetchPath)
		return
	}

	// include_state=true → 状态通道（<get>，按需）：BR-14 改为短 TTL 快照缓存
	// 优先（万级状态表翻页秒开），force_refresh 绕过直打设备（实时逃生门）。
	if c.Query("include_state") == "true" {
		h.serveStateRead(c, ip, path, fetchPath, key, lq, forceRefresh)
		return
	}

	// Serve fresh cache (§8 TTL 30s) unless a refresh is forced. A hit reflects
	// config freshness only; device liveness is /devices/:ip/status.
	if !forceRefresh {
		if val, age, ok := rc.GetWithAge(key); ok {
			h.respondConfig(c, val, lq, path, true, int(age.Seconds()), ttlSec, "cache", "Configuration retrieved (cached)")
			return
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	data, err := h.fetch(ctx, ip, fetchPath)
	if err != nil {
		if h.learnNodeUnsupported(ip, fetchPath, err) {
			rejectNodeUnsupported(c, fetchPath)
			return
		}
		if errors.Is(err, errDeviceNotConnected) {
			Error(c, 503, "Device is not connected")
			return
		}
		Error(c, 500, "Failed to get configuration: "+err.Error())
		return
	}
	if forceRefresh {
		h.clearUnsupported(ip, fetchPath)
	}

	rc.Set(key, data)
	h.respondConfig(c, data, lq, path, false, 0, ttlSec, "device", "Configuration retrieved")
}

// serveStateRead 状态通道读（BR-14）：快照缓存优先（键 = runKey(ip, fetchPath)，
// 实例独立于运行配置缓存——写后失效不触及），未命中/过期/force_refresh 经
// <get> 全量回读并回填快照。分页参数与状态读可组合：切片作用于快照；
// path（含谓词的原始路径）用于行提取下钻，fetchPath 用于设备取数与学习。
func (h *ConfigHandler) serveStateRead(c *gin.Context, ip, path, fetchPath, key string, lq *ListQueryParams, forceRefresh bool) {
	ttlSec := int(h.stateCache.TTL().Seconds())
	if !forceRefresh {
		if val, age, ok := h.stateCache.GetWithAge(key); ok {
			h.respondConfig(c, val, lq, path, true, int(age.Seconds()), ttlSec, "cache", "Configuration retrieved (cached)")
			return
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	data, err := h.fetchState(ctx, ip, fetchPath)
	if err != nil {
		if h.learnNodeUnsupported(ip, fetchPath, err) {
			rejectNodeUnsupported(c, fetchPath)
			return
		}
		if errors.Is(err, errDeviceNotConnected) {
			Error(c, 503, "Device is not connected")
			return
		}
		Error(c, 500, "Failed to get state: "+err.Error())
		return
	}
	if forceRefresh {
		h.clearUnsupported(ip, fetchPath)
	}
	h.stateCache.Set(key, data)
	h.respondConfig(c, data, lq, path, false, 0, ttlSec, "device", "Configuration retrieved")
}

// respondConfig 组装 GET /config 响应：lq 非 nil 时行提取 + 查询切片（BR-13，
// data 变为 ListPage 形状），目标非 list/无法定位行数组 → 400；lq 为 nil 保持
// 整树形状不变（回读子树剥层契约回归锚点）。
func (h *ConfigHandler) respondConfig(c *gin.Context, data interface{}, lq *ListQueryParams, path string, cached bool, ageSec, ttlSec int, source, msg string) {
	payload := data
	if lq != nil {
		rows, err := extractListRows(h.manager.GetSchema(), path, data)
		if err != nil {
			Error(c, 400, err.Error())
			return
		}
		payload = applyListQuery(rows, *lq)
	}
	Success(c, ConfigGetData{
		Data:            payload,
		Cached:          cached,
		CacheAgeSeconds: ageSec,
		TTLSeconds:      ttlSec,
		Source:          source,
	}, msg)
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

	// 节点不支持写门禁（BR-12）：设备版本没有的节点不接受下发（无 force 逃生——
	// 恢复通道是 GET force_refresh 重试成功清标记后写自然恢复）。
	if view := h.support(ip); view != nil && view.IsUnsupportedPath(path) {
		rejectNodeUnsupported(c, path)
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
