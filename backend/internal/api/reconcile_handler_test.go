package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/status"
	"github.com/stretchr/testify/assert"
)

func init() { gin.SetMode(gin.TestMode) }

// fakeManager is a Manager test double that only serves reconcile status; the
// read-only Reader interface (by design) blocks seeding through a real manager.
type fakeManager struct {
	manager.Manager
	store status.Reader
}

func (f fakeManager) GetReconcileStatus() status.Reader { return f.store }

// seedManager returns a manager whose reconcile store is pre-populated.
func seedManager() manager.Manager {
	st := status.NewStore()
	// device .1: one converged, one drifted -> device rollup should be drifted
	st.Record("10.0.0.1", "/vlans", status.OutcomeConverged, 0, nil)
	st.Record("10.0.0.1", "/ifm", status.OutcomeDrifted, 2, nil)
	// device .2: error -> rollup error
	st.Record("10.0.0.2", "/vlans", status.OutcomeError, 0, errors.New("session timeout"))
	// device .3: converged only
	st.Record("10.0.0.3", "/vlans", status.OutcomeConverged, 0, nil)
	return fakeManager{store: st}
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

func TestRollup_SeverityCombinations(t *testing.T) {
	base := time.Unix(1000, 0)
	mk := func(o status.Outcome) status.Status { return status.Status{Outcome: o, LastRun: base} }
	cases := []struct {
		name string
		in   []status.Status
		want status.Outcome
	}{
		{"empty->unknown", nil, status.OutcomeUnknown},
		{"single converged", []status.Status{mk(status.OutcomeConverged)}, status.OutcomeConverged},
		{"single reconciling", []status.Status{mk(status.OutcomeReconciling)}, status.OutcomeReconciling},
		{"reconciling>converged", []status.Status{mk(status.OutcomeConverged), mk(status.OutcomeReconciling)}, status.OutcomeReconciling},
		{"drifted>reconciling", []status.Status{mk(status.OutcomeReconciling), mk(status.OutcomeDrifted)}, status.OutcomeDrifted},
		{"error>drifted", []status.Status{mk(status.OutcomeDrifted), mk(status.OutcomeError)}, status.OutcomeError},
		{"error>all", []status.Status{mk(status.OutcomeConverged), mk(status.OutcomeReconciling), mk(status.OutcomeDrifted), mk(status.OutcomeError)}, status.OutcomeError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := rollup(tc.in)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRollup_LastRunFollowsWorst(t *testing.T) {
	old := time.Unix(1000, 0)
	recent := time.Unix(2000, 0)
	// worst is error(old); a more-recent converged(recent) must NOT set LastRun.
	list := []status.Status{
		{Outcome: status.OutcomeError, LastRun: old},
		{Outcome: status.OutcomeConverged, LastRun: recent},
	}
	outcome, last := rollup(list)
	assert.Equal(t, status.OutcomeError, outcome)
	assert.Equal(t, old, last, "LastRun must track the worst outcome, not the newest entry")
}

func TestReconcile_Fleet_EmptyStore(t *testing.T) {
	h := NewReconcileHandler(fakeManager{store: status.NewStore()})
	w := doGet(h.GetFleetReconcile, nil)

	var data FleetReconcileData
	recDecode(t, w, &data)
	assert.Empty(t, data.Devices)
	assert.Empty(t, data.Summary)
}
