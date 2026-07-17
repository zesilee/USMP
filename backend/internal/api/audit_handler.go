package api

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/status"
)

// AuditHandler serves the operation-audit log (config-delivery records).
type AuditHandler struct {
	manager manager.Manager
}

// NewAuditHandler creates a new AuditHandler.
func NewAuditHandler(mgr manager.Manager) *AuditHandler {
	return &AuditHandler{manager: mgr}
}

// LogEntry 是 GET /logs 的单条记录：审计事实 + 当前对账结局（live-join）。
// outcome/diff_count 是**查询时**的对账态（异步变化），非下发瞬间快照——诚实标注。
type LogEntry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	DeviceIP  string    `json:"device_ip"`
	Path      string    `json:"path"`
	Summary   string    `json:"summary"`
	Triggered bool      `json:"triggered"`
	Actor     string    `json:"actor"`
	Outcome   string    `json:"outcome"`    // 当前对账结局（live-join；无记录为 unknown）
	DiffCount int       `json:"diff_count"` // 当前差异数（live-join）
	// Forced/ForcedOwners：force 覆盖归属硬锁留痕（OA-01 二期，零值省略）。
	Forced       bool     `json:"forced,omitempty"`
	ForcedOwners []string `json:"forcedOwners,omitempty"`
}

// AuditListData 是 GET /logs 的 data 负载。Total 为筛选后总数（分页前）。
type AuditListData struct {
	Logs  []LogEntry `json:"logs"`
	Total int        `json:"total"`
}

const (
	defaultLogLimit = 50
	maxLogLimit     = 500
)

// ListLogs lists config-delivery audit records, joined live with the current
// reconcile outcome, with optional device/status filters and limit/offset paging.
//
// @Summary  操作日志（配置下发审计 + 当前对账结局）
// @Tags     logs
// @Produce  json
// @Param    device query string false "按设备 IP 筛选"
// @Param    status query string false "按当前对账结局筛选(converged/drifted/error/reconciling/unknown)"
// @Param    limit  query int    false "每页条数(默认 50，上限 500)"
// @Param    offset query int    false "偏移(默认 0)"
// @Success  200 {object} Response{data=AuditListData} "操作日志"
// @Router   /logs [get]
func (h *AuditHandler) ListLogs(c *gin.Context) {
	device := c.Query("device")
	statusFilter := c.Query("status")
	limit := parsePositive(c.Query("limit"), defaultLogLimit, maxLogLimit)
	offset := parsePositive(c.Query("offset"), 0, 1<<31-1)

	reader := h.manager.GetReconcileStatus()
	records := h.manager.GetAuditStore().List() // newest-first

	// 事实记录 → 富化当前对账态 → 按设备/结局筛选。
	entries := make([]LogEntry, 0, len(records))
	for _, r := range records {
		if device != "" && r.DeviceIP != device {
			continue
		}
		outcome := string(status.OutcomeUnknown)
		diffCount := 0
		if st, ok := reader.Get(r.DeviceIP, r.Path); ok {
			outcome = string(st.Outcome)
			diffCount = st.DiffCount
		}
		if statusFilter != "" && outcome != statusFilter {
			continue
		}
		entries = append(entries, LogEntry{
			ID:           r.ID,
			Timestamp:    r.Timestamp,
			DeviceIP:     r.DeviceIP,
			Path:         r.Path,
			Summary:      r.Summary,
			Triggered:    r.Triggered,
			Actor:        r.Actor,
			Outcome:      outcome,
			DiffCount:    diffCount,
			Forced:       r.Forced,
			ForcedOwners: r.ForcedOwners,
		})
	}

	total := len(entries)
	// 分页（越界安全裁剪，R08）
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	page := entries[start:end]

	Success(c, AuditListData{Logs: page, Total: total}, "操作日志")
}

// parsePositive 解析非负整数 query，非法/为空回退 def，超过 max 截到 max。
func parsePositive(raw string, def, max int) int {
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}
