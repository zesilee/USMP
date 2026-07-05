package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/status"
	"github.com/stretchr/testify/assert"
)

func init() { gin.SetMode(gin.TestMode) }

// seedManager returns a manager whose reconcile store is pre-populated.
func seedManager() manager.Manager {
	m := manager.New()
	st := m.GetReconcileStatus()
	// device .1: one converged, one drifted -> device rollup should be drifted
	st.Record("10.0.0.1", "/vlans", status.OutcomeConverged, 0, nil)
	st.Record("10.0.0.1", "/ifm", status.OutcomeDrifted, 2, nil)
	// device .2: error -> rollup error
	st.Record("10.0.0.2", "/vlans", status.OutcomeError, 0, errors.New("session timeout"))
	// device .3: converged only
	st.Record("10.0.0.3", "/vlans", status.OutcomeConverged, 0, nil)
	return m
}

func doGet(h gin.HandlerFunc, params gin.Params) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = params
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	h(c)
	return w
}

func recDecode(t *testing.T, w *httptest.ResponseRecorder, into interface{}) {
	t.Helper()
	assert.Equal(t, http.StatusOK, w.Code)
	var env struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	assert.True(t, env.Success)
	if err := json.Unmarshal(env.Data, into); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
}

func TestReconcile_Device_RollupWorstOutcome(t *testing.T) {
	h := NewReconcileHandler(seedManager())
	w := doGet(h.GetDeviceReconcile, gin.Params{{Key: "ip", Value: "10.0.0.1"}})

	var data DeviceReconcileData
	recDecode(t, w, &data)
	assert.Equal(t, "10.0.0.1", data.DeviceID)
	assert.Equal(t, string(status.OutcomeDrifted), data.Outcome, "converged+drifted rolls up to drifted")
	assert.Len(t, data.Statuses, 2)
}

func TestReconcile_Device_Unknown(t *testing.T) {
	h := NewReconcileHandler(seedManager())
	w := doGet(h.GetDeviceReconcile, gin.Params{{Key: "ip", Value: "10.0.0.99"}})

	var data DeviceReconcileData
	recDecode(t, w, &data)
	assert.Equal(t, string(status.OutcomeUnknown), data.Outcome, "never-reconciled device is unknown, not an error")
	assert.Empty(t, data.Statuses)
}

func TestReconcile_Fleet_Summary(t *testing.T) {
	h := NewReconcileHandler(seedManager())
	w := doGet(h.GetFleetReconcile, nil)

	var data FleetReconcileData
	recDecode(t, w, &data)
	// 3 devices: .1 drifted, .2 error, .3 converged
	assert.Equal(t, 1, data.Summary[string(status.OutcomeDrifted)])
	assert.Equal(t, 1, data.Summary[string(status.OutcomeError)])
	assert.Equal(t, 1, data.Summary[string(status.OutcomeConverged)])
	assert.Len(t, data.Devices, 3)
}
