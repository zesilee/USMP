package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/backend/internal/intent"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client/netconfcore"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/stretchr/testify/assert"
)

// BR-12（tasks 3.1）：节点不支持快速失败 + 结构化 reason + force 逃生 + 写门禁。

// fakeSupportView 记录标记/清除动作的假不支持集。
type fakeSupportView struct {
	set     map[string]bool
	marked  []string
	cleared []string
}

func (f *fakeSupportView) IsUnsupportedPath(p string) bool { return f.set[p] }
func (f *fakeSupportView) MarkUnsupportedPath(p string) {
	f.marked = append(f.marked, p)
	f.set[p] = true
}
func (f *fakeSupportView) ClearUnsupportedPath(p string) {
	f.cleared = append(f.cleared, p)
	delete(f.set, p)
}
func (f *fakeSupportView) UnsupportedPathsUnder(prefix string) []string {
	var out []string
	for p := range f.set {
		if p == prefix || strings.HasPrefix(p, prefix+"/") {
			out = append(out, p)
		}
	}
	return out
}

func newNodeUnsupportedHandler(view *fakeSupportView, fetch func(ctx context.Context, ip, path string) (interface{}, error)) *ConfigHandler {
	h := NewConfigHandler(manager.New())
	h.fetch = fetch
	h.fetchState = fetch
	h.support = func(ip string) nodeSupportView {
		if view == nil {
			return nil
		}
		return view
	}
	return h
}

func getConfigReqQS(h *ConfigHandler, ip, path, qs string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "ip", Value: ip}, {Key: "path", Value: path}}
	c.Request = httptest.NewRequest(http.MethodGet, "/?"+qs, nil)
	h.GetConfig(c)
	return w
}

func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) (code int, message string, data map[string]interface{}) {
	t.Helper()
	var env struct {
		Code    int                    `json:"code"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
		Success bool                   `json:"success"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	return env.Code, env.Message, env.Data
}

func unknownElemErr(bad string) error {
	return &netconfcore.RPCReplyError{Errors: []netconfcore.RPCError{{
		Tag: "unknown-element", Severity: "error", BadElement: bad,
		Message: "Unexpected element: " + bad + ".",
	}}}
}

// 已标记路径：不打设备，reason=node-unsupported。
func TestGetConfig_NodeUnsupported_FastFail(t *testing.T) {
	calls := 0
	view := &fakeSupportView{set: map[string]bool{"/devm:devm/devm:cards": true}}
	h := newNodeUnsupportedHandler(view, func(ctx context.Context, ip, path string) (interface{}, error) {
		calls++
		return "data", nil
	})
	w := getConfigReqQS(h, "10.0.0.1", "/devm:devm/devm:cards", "")
	code, _, data := decodeEnvelope(t, w)
	assert.Equal(t, 0, calls, "已标记路径不得打设备")
	assert.NotEqual(t, 0, code)
	assert.Equal(t, "node-unsupported", data["reason"])
}

// 首次学习：设备回 unknown-element → 本次响应即带 reason + 入集。
func TestGetConfig_NodeUnsupported_LearnOnFirstHit(t *testing.T) {
	view := &fakeSupportView{set: map[string]bool{}}
	h := newNodeUnsupportedHandler(view, func(ctx context.Context, ip, path string) (interface{}, error) {
		return nil, unknownElemErr("cards")
	})
	w := getConfigReqQS(h, "10.0.0.1", "/devm:devm/devm:cards", "")
	code, _, data := decodeEnvelope(t, w)
	assert.NotEqual(t, 0, code)
	assert.Equal(t, "node-unsupported", data["reason"])
	assert.Contains(t, view.marked, "/devm:devm/devm:cards")
}

// force_refresh 绕过快速失败：成功 → 清标记 + 返回数据。
func TestGetConfig_NodeUnsupported_ForceRetryRecovers(t *testing.T) {
	calls := 0
	view := &fakeSupportView{set: map[string]bool{"/devm:devm/devm:cards": true}}
	h := newNodeUnsupportedHandler(view, func(ctx context.Context, ip, path string) (interface{}, error) {
		calls++
		return "fresh", nil
	})
	w := getConfigReqQS(h, "10.0.0.1", "/devm:devm/devm:cards", "force_refresh=true")
	code, _, _ := decodeEnvelope(t, w)
	assert.Equal(t, 1, calls, "force 必须真打设备")
	assert.Equal(t, 0, code)
	assert.Contains(t, view.cleared, "/devm:devm/devm:cards")
}

// force 重试仍失败：标记保留。
func TestGetConfig_NodeUnsupported_ForceRetryStillFails(t *testing.T) {
	view := &fakeSupportView{set: map[string]bool{"/devm:devm/devm:cards": true}}
	h := newNodeUnsupportedHandler(view, func(ctx context.Context, ip, path string) (interface{}, error) {
		return nil, unknownElemErr("cards")
	})
	w := getConfigReqQS(h, "10.0.0.1", "/devm:devm/devm:cards", "force_refresh=true")
	code, _, data := decodeEnvelope(t, w)
	assert.NotEqual(t, 0, code)
	assert.Equal(t, "node-unsupported", data["reason"])
	assert.True(t, view.set["/devm:devm/devm:cards"], "失败后标记保留")
	assert.Empty(t, view.cleared)
}

// 普通设备错误不误标、无 reason（负路径）。
func TestGetConfig_NodeUnsupported_PlainErrorNotLearned(t *testing.T) {
	view := &fakeSupportView{set: map[string]bool{}}
	h := newNodeUnsupportedHandler(view, func(ctx context.Context, ip, path string) (interface{}, error) {
		return nil, errors.New("device timeout")
	})
	w := getConfigReqQS(h, "10.0.0.1", "/devm:devm/devm:cards", "")
	code, msg, data := decodeEnvelope(t, w)
	assert.NotEqual(t, 0, code)
	assert.NotEqual(t, "node-unsupported", data["reason"])
	assert.Empty(t, view.marked)
	assert.Contains(t, msg, "device timeout")
}

// 状态通道（include_state，只读 Tab）同受快速失败与学习约束。
func TestGetConfig_NodeUnsupported_StateChannel(t *testing.T) {
	calls := 0
	view := &fakeSupportView{set: map[string]bool{"/devm:devm/devm:schedule-reboot": true}}
	h := newNodeUnsupportedHandler(view, func(ctx context.Context, ip, path string) (interface{}, error) {
		calls++
		return "state", nil
	})
	w := getConfigReqQS(h, "10.0.0.1", "/devm:devm/devm:schedule-reboot", "include_state=true")
	code, _, data := decodeEnvelope(t, w)
	assert.Equal(t, 0, calls)
	assert.NotEqual(t, 0, code)
	assert.Equal(t, "node-unsupported", data["reason"])
}

// 状态通道学习：include_state 读遇 unknown-element 同样入集带 reason。
func TestGetConfig_NodeUnsupported_StateChannelLearns(t *testing.T) {
	view := &fakeSupportView{set: map[string]bool{}}
	h := newNodeUnsupportedHandler(view, func(ctx context.Context, ip, path string) (interface{}, error) {
		return nil, unknownElemErr("schedule-reboot")
	})
	w := getConfigReqQS(h, "10.0.0.1", "/devm:devm/devm:schedule-reboot", "include_state=true")
	_, _, data := decodeEnvelope(t, w)
	assert.Equal(t, "node-unsupported", data["reason"])
	assert.Contains(t, view.marked, "/devm:devm/devm:schedule-reboot")
}

// 无 support 视图（设备无连接实现）：行为与既有链路完全一致。
func TestGetConfig_NodeUnsupported_NilViewUnchanged(t *testing.T) {
	h := newNodeUnsupportedHandler(nil, func(ctx context.Context, ip, path string) (interface{}, error) {
		return "data", nil
	})
	w := getConfigReqQS(h, "10.0.0.1", "/devm:devm/devm:cards", "")
	code, _, _ := decodeEnvelope(t, w)
	assert.Equal(t, 0, code)
}

// 变更集提交门禁：任一条目命中不支持路径 → 整体拒绝（2PC 全有全无）、
// 不打设备、带 reason（BR-12）。
func TestChangesetCommit_NodeUnsupported_Rejected(t *testing.T) {
	push := &fakePusher{results: map[string]intent.TxResult{"10.0.0.1": {Device: "10.0.0.1"}}}
	h, _ := newCommitHandlerForTest(push)
	view := &fakeSupportView{set: map[string]bool{"/vlan:vlan/vlan:vlans": true}}
	h.support = func(ip string) nodeSupportView { return view }

	body := `{"device":"10.0.0.1","entries":[{"op":"update","path":"/vlan:vlan/vlan:vlans","payload":{"vlan":[{"id":10}]}}]}`
	w := commitReq(h, body, "")
	code, _, data := decodeEnvelope(t, w)
	assert.NotEqual(t, 0, code)
	assert.Equal(t, "node-unsupported", data["reason"])
	assert.Equal(t, 0, push.calls, "不得向设备下发")
}

// 快速失败子树语义（BR-12「落在其下」）：标记父路径后，后代谓词路径（如
// 详情面板单行状态读）同样被拒。集合层子树匹配须用真实集合（零值可用），
// 假视图的 exact-map 测不到。
func TestGetConfig_NodeUnsupported_DescendantFastFail(t *testing.T) {
	calls := 0
	real := &client.NETCONFClient{}
	real.MarkUnsupportedPath("vlan:vlan/vlan:vlans")
	h := newNodeUnsupportedHandler(nil, func(ctx context.Context, ip, path string) (interface{}, error) {
		calls++
		return "data", nil
	})
	h.support = func(ip string) nodeSupportView { return real }

	w := getConfigReqQS(h, "10.0.0.1", "/vlan:vlan/vlan:vlans/vlan:vlan[id='10']", "include_state=true")
	code, _, data := decodeEnvelope(t, w)
	assert.Equal(t, 0, calls, "后代路径同样不得打设备")
	assert.NotEqual(t, 0, code)
	assert.Equal(t, "node-unsupported", data["reason"])
}

// 写通道学习（CN-04）：2PC 下发返回可归因 unknown-element → 该条目路径入集、
// 响应带 reason（下次同路径提交快速失败）。
func TestChangesetCommit_NodeUnsupported_LearnsFromPushError(t *testing.T) {
	push := &fakePusher{results: map[string]intent.TxResult{"10.0.0.1": {
		Device: "10.0.0.1", Err: unknownElemErr("vlans"),
	}}}
	h, _ := newCommitHandlerForTest(push)
	view := &fakeSupportView{set: map[string]bool{}}
	h.support = func(ip string) nodeSupportView { return view }

	body := `{"device":"10.0.0.1","entries":[{"op":"update","path":"/vlan:vlan/vlan:vlans","payload":{"vlan":[{"id":10}]}}]}`
	w := commitReq(h, body, "")
	code, _, data := decodeEnvelope(t, w)
	assert.NotEqual(t, 0, code)
	assert.Equal(t, "node-unsupported", data["reason"])
	assert.Contains(t, view.marked, "/vlan:vlan/vlan:vlans")
}

// 写门禁：POST /config 命中不支持路径 → 拒绝、不入 desired、带 reason。
func TestSetConfig_NodeUnsupported_Rejected(t *testing.T) {
	view := &fakeSupportView{set: map[string]bool{"/devm:devm/devm:cards": true}}
	h := newNodeUnsupportedHandler(view, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "ip", Value: "10.0.0.1"}, {Key: "path", Value: "/devm:devm/devm:cards"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"card":[]}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.SetConfig(c)
	code, _, data := decodeEnvelope(t, w)
	assert.NotEqual(t, 0, code)
	assert.Equal(t, "node-unsupported", data["reason"])
}
