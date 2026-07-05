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

// rollup returns the worst outcome and most recent run across a set of statuses.
func rollup(list []status.Status) (status.Outcome, time.Time) {
	worst := status.OutcomeUnknown
	var last time.Time
	for _, st := range list {
		if outcomeSeverity[st.Outcome] > outcomeSeverity[worst] {
			worst = st.Outcome
		}
		if st.LastRun.After(last) {
			last = st.LastRun
		}
	}
	return worst, last
}

// GetDeviceReconcile handles GET /devices/:ip/reconcile.
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
