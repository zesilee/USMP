package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/audit"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/status"
	"github.com/stretchr/testify/assert"
)

// fakeAuditMgr 只覆写 ListLogs 用到的两个 getter，其余方法零值（不会被调用）。
type fakeAuditMgr struct {
	manager.Manager
	audit  audit.Store
	status *status.Store
}

func (f fakeAuditMgr) GetAuditStore() audit.Store        { return f.audit }
func (f fakeAuditMgr) GetReconcileStatus() status.Reader { return f.status }

func listLogsReq(h *AuditHandler, query string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/logs"+query, nil)
	h.ListLogs(c)
	return w
}

func decodeAudit(t *testing.T, w *httptest.ResponseRecorder) AuditListData {
	t.Helper()
	assert.Equal(t, http.StatusOK, w.Code)
	var env struct {
		Data AuditListData `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return env.Data
}

func seedHandler() *AuditHandler {
	as := audit.NewStore("", 100)
	as.Record(audit.Record{DeviceIP: "10.0.0.1", Path: "/vlan", Summary: "vlans (1)"})
	as.Record(audit.Record{DeviceIP: "10.0.0.2", Path: "/ifm", Summary: "interface (1)"})
	as.Record(audit.Record{DeviceIP: "10.0.0.1", Path: "/sys", Summary: "system"})

	st := status.NewStore()
	st.Record("10.0.0.1", "/vlan", status.OutcomeConverged, 0, nil)
	st.Record("10.0.0.2", "/ifm", status.OutcomeDrifted, 3, nil)
	// 10.0.0.1 /sys 未对账 → join 为 unknown

	return NewAuditHandler(fakeAuditMgr{audit: as, status: st})
}

func TestListLogs_JoinsLiveOutcome(t *testing.T) {
	d := decodeAudit(t, listLogsReq(seedHandler(), ""))
	assert.Equal(t, 3, d.Total)
	// newest-first：/sys, /ifm, /vlan
	assert.Equal(t, "/sys", d.Logs[0].Path)
	assert.Equal(t, "unknown", d.Logs[0].Outcome) // 未对账 → unknown
	assert.Equal(t, "/ifm", d.Logs[1].Path)
	assert.Equal(t, "drifted", d.Logs[1].Outcome)
	assert.Equal(t, 3, d.Logs[1].DiffCount)
	assert.Equal(t, "converged", d.Logs[2].Outcome)
}

func TestListLogs_FilterByDevice(t *testing.T) {
	d := decodeAudit(t, listLogsReq(seedHandler(), "?device=10.0.0.1"))
	assert.Equal(t, 2, d.Total)
	for _, e := range d.Logs {
		assert.Equal(t, "10.0.0.1", e.DeviceIP)
	}
}

func TestListLogs_FilterByOutcome(t *testing.T) {
	d := decodeAudit(t, listLogsReq(seedHandler(), "?status=drifted"))
	assert.Equal(t, 1, d.Total)
	assert.Equal(t, "/ifm", d.Logs[0].Path)
}

func TestListLogs_Pagination(t *testing.T) {
	d := decodeAudit(t, listLogsReq(seedHandler(), "?limit=1&offset=1"))
	assert.Equal(t, 3, d.Total, "total 是筛选后总数(分页前)")
	assert.Len(t, d.Logs, 1)
	assert.Equal(t, "/ifm", d.Logs[0].Path) // newest-first 第二条
}

func TestListLogs_OffsetBeyondTotalSafe(t *testing.T) {
	d := decodeAudit(t, listLogsReq(seedHandler(), "?offset=99"))
	assert.Equal(t, 3, d.Total)
	assert.Empty(t, d.Logs) // 越界不崩、返回空页(R08)
}

func TestListLogs_LimitCapped(t *testing.T) {
	assert.Equal(t, maxLogLimit, parsePositive("99999", defaultLogLimit, maxLogLimit))
	assert.Equal(t, defaultLogLimit, parsePositive("", defaultLogLimit, maxLogLimit))
	assert.Equal(t, defaultLogLimit, parsePositive("-5", defaultLogLimit, maxLogLimit))
	assert.Equal(t, defaultLogLimit, parsePositive("abc", defaultLogLimit, maxLogLimit))
}

func TestListLogs_EmptyLog(t *testing.T) {
	h := NewAuditHandler(fakeAuditMgr{audit: audit.NewStore("", 100), status: status.NewStore()})
	d := decodeAudit(t, listLogsReq(h, ""))
	assert.Equal(t, 0, d.Total)
	assert.Empty(t, d.Logs)
}
