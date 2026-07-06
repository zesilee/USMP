package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/status"
)

// ReconcileHandler exposes the most-recent reconcile outcomes (desired↔actual
// convergence) tracked in memory by the Manager. Read-only; degrades to
// "unknown" for devices that have never been reconciled (R08).
type ReconcileHandler struct {
	manager manager.Manager
}

// NewReconcileHandler creates a new ReconcileHandler.
func NewReconcileHandler(m manager.Manager) *ReconcileHandler {
	return &ReconcileHandler{manager: m}
}

// DeviceReconcileData is the per-device reconcile rollup plus per-path detail.
type DeviceReconcileData struct {
	DeviceID string          `json:"device_id"`
	Outcome  string          `json:"outcome"`
	Statuses []status.Status `json:"statuses"`
}

// DeviceRollup is a device's single worst-case outcome across all its paths.
type DeviceRollup struct {
	DeviceID string    `json:"device_id"`
	Outcome  string    `json:"outcome"`
	LastRun  time.Time `json:"last_run"`
}

// FleetReconcileData aggregates reconcile outcomes across all devices that have
// been reconciled. Summary maps outcome -> device count; the "unknown" devices
// (never reconciled) are not represented here — callers combine with the device
// list to derive them.
type FleetReconcileData struct {
	Summary map[string]int `json:"summary"`
	Devices []DeviceRollup `json:"devices"`
}

// outcomeSeverity ranks outcomes so a device rolls up to its worst path.
var outcomeSeverity = map[status.Outcome]int{
	status.OutcomeUnknown:     0,
	status.OutcomeConverged:   1,
	status.OutcomeReconciling: 2,
	status.OutcomeDrifted:     3,
	status.OutcomeError:       4,
}

// rollup returns the worst outcome across a set of statuses and the most recent
// run time among entries at that worst outcome (so Outcome and LastRun are
// consistent — LastRun tells you when the reported worst state was observed).
func rollup(list []status.Status) (status.Outcome, time.Time) {
	worst := status.OutcomeUnknown
	for _, st := range list {
		if outcomeSeverity[st.Outcome] > outcomeSeverity[worst] {
			worst = st.Outcome
		}
	}
	var last time.Time
	for _, st := range list {
		if st.Outcome == worst && st.LastRun.After(last) {
			last = st.LastRun
		}
	}
	return worst, last
}

// GetDeviceReconcile handles GET /devices/:ip/reconcile.
//
// @Summary  读取单设备对账结局（desired↔actual，含各 YANG 路径明细）
// @Tags     reconcile
// @Produce  json
// @Param    ip path string true "设备 IP"
// @Success  200 {object} Response{data=DeviceReconcileData} "设备对账结局；从未对账返回 outcome=unknown"
// @Router   /devices/{ip}/reconcile [get]
func (h *ReconcileHandler) GetDeviceReconcile(c *gin.Context) {
	ip := c.Param("ip")
	list := h.manager.GetReconcileStatus().ListByDevice(ip)
	outcome, _ := rollup(list)
	Success(c, DeviceReconcileData{
		DeviceID: ip,
		Outcome:  string(outcome),
		Statuses: list,
	}, "")
}

// GetFleetReconcile handles GET /reconcile/status.
//
// @Summary  车队级对账结局聚合（收敛台账 / 概览大盘用）
// @Tags     reconcile
// @Produce  json
// @Success  200 {object} Response{data=FleetReconcileData} "车队对账聚合；summary 按结局计数，仅含已对账设备（unknown 需与设备列表相减派生）"
// @Router   /reconcile/status [get]
func (h *ReconcileHandler) GetFleetReconcile(c *gin.Context) {
	all := h.manager.GetReconcileStatus().Snapshot()

	byDevice := make(map[string][]status.Status)
	for _, st := range all {
		byDevice[st.DeviceID] = append(byDevice[st.DeviceID], st)
	}

	summary := make(map[string]int)
	devices := make([]DeviceRollup, 0, len(byDevice))
	for id, list := range byDevice {
		outcome, last := rollup(list)
		summary[string(outcome)]++
		devices = append(devices, DeviceRollup{
			DeviceID: id,
			Outcome:  string(outcome),
			LastRun:  last,
		})
	}

	Success(c, FleetReconcileData{
		Summary: summary,
		Devices: devices,
	}, "")
}
