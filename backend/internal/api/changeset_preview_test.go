package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/leezesi/usmp/backend/internal/generated/native/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/object"
)

const vlanAnchor = "/vlan:vlan/vlan:vlans"

// newChangesetHandlerForTest：真实 manager（ConfigStore/缓存/审计）+ 注入的
// 设备读闭包（无设备/模拟器）。
func newChangesetHandlerForTest(fetch func(ctx context.Context, ip, path string) (interface{}, error)) (*ChangesetHandler, manager.Manager) {
	mgr := manager.New()
	h := NewChangesetHandler(mgr)
	if fetch != nil {
		h.fetch = fetch
	}
	return h, mgr
}

func previewReq(h *ChangesetHandler, body string) *httptest.ResponseRecorder {
	c, w := newTestContext(http.MethodPost, "/", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Preview(c)
	return w
}

func decodePreview(t *testing.T, w *httptest.ResponseRecorder) ChangesetPreviewData {
	t.Helper()
	var env struct {
		Code    int                  `json:"code"`
		Success bool                 `json:"success"`
		Data    ChangesetPreviewData `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !env.Success || env.Code != 0 {
		t.Fatalf("preview failed: %s", w.Body.String())
	}
	return env.Data
}

func seedDesiredVlan(mgr manager.Manager, ip string, desc string) {
	_ = mgr.GetConfigStore().Set(ip, vlanAnchor, &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
			10: {Id: object.Uint16(10), Name: object.String("mgmt"), Description: object.String(desc)},
		},
	})
}

// CS-01：update 条目——正向报文含新值、回滚报文含基线旧值、diff 为 MODIFY、
// 基线来源=desired、纯计算无副作用（desired/缓存/审计零变化）、幂等。
func TestChangesetPreview_UpdateForwardRollbackDiff(t *testing.T) {
	fetchCalls := 0
	h, mgr := newChangesetHandlerForTest(func(ctx context.Context, ip, path string) (interface{}, error) {
		fetchCalls++
		return nil, errors.New("must not reach device when desired exists")
	})
	seedDesiredVlan(mgr, "10.0.0.1", "old-desc")

	body := `{"device":"10.0.0.1","entries":[{"op":"update","path":"/vlan:vlan/vlan:vlans","payload":{"vlan":[{"id":10,"description":"new-desc"}]}}]}`
	d := decodePreview(t, previewReq(h, body))

	assert.Equal(t, "10.0.0.1", d.Device)
	if !assert.Len(t, d.Entries, 1) {
		return
	}
	e := d.Entries[0]
	assert.Equal(t, "desired", e.BaselineSource)
	assert.False(t, e.Unsupported)
	assert.Contains(t, e.ForwardXML, "<description>new-desc</description>")
	assert.Contains(t, e.ForwardXML, "<id>10</id>")
	assert.Contains(t, e.RollbackXML, "<description>old-desc</description>")
	assert.GreaterOrEqual(t, d.Summary.Modifies, 1)
	assert.Equal(t, 0, fetchCalls, "desired 命中时不得回读设备")

	// 无副作用：desired 原值保留、审计零记录
	stored, err := mgr.GetConfigStore().Get("10.0.0.1", vlanAnchor)
	assert.NoError(t, err)
	v := stored.(*huawei.HuaweiVlan_Vlan_Vlans)
	assert.Equal(t, "old-desc", *v.Vlan[10].Description, "preview must not mutate desired")
	assert.Empty(t, mgr.GetAuditStore().List(), "preview must not write audit")

	// 幂等：二次调用结果一致
	d2 := decodePreview(t, previewReq(h, body))
	assert.Equal(t, d.Entries[0].ForwardXML, d2.Entries[0].ForwardXML)
	assert.Equal(t, d.Entries[0].RollbackXML, d2.Entries[0].RollbackXML)
}

// CS-01/02：create 条目（基线无此模块）——正向为整条目、回滚为键定位删除、
// diff 计 ADD、基线来源=none（设备读失败如实降级，R08）。
func TestChangesetPreview_CreateRollbackIsDelete(t *testing.T) {
	h, _ := newChangesetHandlerForTest(func(ctx context.Context, ip, path string) (interface{}, error) {
		return nil, errors.New("device unreachable")
	})
	body := `{"device":"10.0.0.2","entries":[{"op":"create","path":"/vlan:vlan/vlan:vlans","payload":{"vlan":[{"id":20,"name":"v20"}]}}]}`
	d := decodePreview(t, previewReq(h, body))
	if !assert.Len(t, d.Entries, 1) {
		return
	}
	e := d.Entries[0]
	assert.Equal(t, "none", e.BaselineSource)
	assert.Contains(t, e.ForwardXML, "<id>20</id>")
	assert.Contains(t, e.RollbackXML, `nc:operation="delete"`)
	assert.Contains(t, e.RollbackXML, "<id>20</id>")
	assert.GreaterOrEqual(t, d.Summary.Adds, 1)
}

// CS-02：delete 条目——正向为键定位删除、回滚按基线值重建条目。
func TestChangesetPreview_DeleteRollbackRebuilds(t *testing.T) {
	h, mgr := newChangesetHandlerForTest(nil)
	seedDesiredVlan(mgr, "10.0.0.3", "keep-me")

	body := `{"device":"10.0.0.3","entries":[{"op":"delete","path":"/vlan:vlan/vlan:vlans","key":"10"}]}`
	d := decodePreview(t, previewReq(h, body))
	if !assert.Len(t, d.Entries, 1) {
		return
	}
	e := d.Entries[0]
	assert.Contains(t, e.ForwardXML, `nc:operation="delete"`)
	assert.Contains(t, e.ForwardXML, "<id>10</id>")
	assert.Contains(t, e.RollbackXML, "<description>keep-me</description>", "回滚必须按基线重建条目")
	assert.NotContains(t, e.RollbackXML, `nc:operation="delete"`)
	assert.GreaterOrEqual(t, d.Summary.Deletes, 1)
}

// CS-05：cleared 叶——正向报文含叶级 delete，diff 含该叶 DELETE 变更（旧值），
// 回滚报文含基线旧值重建。
func TestChangesetPreview_ClearedLeaves(t *testing.T) {
	h, mgr := newChangesetHandlerForTest(nil)
	seedDesiredVlan(mgr, "10.0.0.4", "will-clear")

	body := `{"device":"10.0.0.4","entries":[{"op":"update","path":"/vlan:vlan/vlan:vlans","payload":{"vlan":[{"id":10}]},"cleared":["description"]}]}`
	d := decodePreview(t, previewReq(h, body))
	if !assert.Len(t, d.Entries, 1) {
		return
	}
	e := d.Entries[0]
	assert.Contains(t, e.ForwardXML, `<description nc:operation="delete"`)
	assert.Contains(t, e.RollbackXML, "<description>will-clear</description>")
	foundLeafDelete := false
	for _, ch := range e.Diff {
		if ch.Type == "DELETE" && strings.HasSuffix(ch.Path, "description") {
			foundLeafDelete = true
			assert.Equal(t, "will-clear", ch.Old)
		}
	}
	assert.True(t, foundLeafDelete, "diff 必须含 cleared 叶的 DELETE 变更: %+v", e.Diff)
}

// CS-03：无 XML 通道模块（system）如实降级——unsupported 标记 + 原因，
// 不伪造报文；同请求中有 XML 通道的条目正常出报文。
func TestChangesetPreview_UnsupportedModuleDegrades(t *testing.T) {
	h, mgr := newChangesetHandlerForTest(func(ctx context.Context, ip, path string) (interface{}, error) {
		return nil, errors.New("no device")
	})
	seedDesiredVlan(mgr, "10.0.0.5", "x")

	body := `{"device":"10.0.0.5","entries":[
		{"op":"update","path":"/system:system","payload":{"system-info":{"sys-name":"sw1"}}},
		{"op":"update","path":"/vlan:vlan/vlan:vlans","payload":{"vlan":[{"id":10,"description":"y"}]}}
	]}`
	d := decodePreview(t, previewReq(h, body))
	if !assert.Len(t, d.Entries, 2) {
		return
	}
	sys, vlan := d.Entries[0], d.Entries[1]
	assert.True(t, sys.Unsupported)
	assert.NotEmpty(t, sys.UnsupportedReason)
	assert.Empty(t, sys.ForwardXML, "不支持报文预览就不得伪造报文")
	assert.False(t, vlan.Unsupported)
	assert.NotEmpty(t, vlan.ForwardXML)
}

// CS-01 负路径：坏路径/坏 payload/空条目/缺 device → 400，且不返回部分结果。
func TestChangesetPreview_BadRequests(t *testing.T) {
	h, _ := newChangesetHandlerForTest(nil)
	for name, body := range map[string]string{
		"unknown path":  `{"device":"10.0.0.9","entries":[{"op":"update","path":"/nope:nope","payload":{"x":1}}]}`,
		"bad payload":   `{"device":"10.0.0.9","entries":[{"op":"update","path":"/vlan:vlan/vlan:vlans","payload":{"vlan":[{"id":"not-a-number"}]}}]}`,
		"empty entries": `{"device":"10.0.0.9","entries":[]}`,
		"no device":     `{"entries":[{"op":"update","path":"/vlan:vlan/vlan:vlans","payload":{"vlan":[{"id":10}]}}]}`,
		"delete no key": `{"device":"10.0.0.9","entries":[{"op":"delete","path":"/vlan:vlan/vlan:vlans"}]}`,
		"bad op":        `{"device":"10.0.0.9","entries":[{"op":"replace","path":"/vlan:vlan/vlan:vlans","payload":{"vlan":[{"id":10}]}}]}`,
		"not json":      `{`,
	} {
		t.Run(name, func(t *testing.T) {
			w := previewReq(h, body)
			assert.Equal(t, http.StatusBadRequest, envelopeCode(t, w), "body: %s", w.Body.String())
		})
	}
}

// 基线链第二级：desired 缺失时命中 running cache（不回读设备）。
func TestChangesetPreview_BaselineFromCache(t *testing.T) {
	fetchCalls := 0
	h, mgr := newChangesetHandlerForTest(func(ctx context.Context, ip, path string) (interface{}, error) {
		fetchCalls++
		return nil, errors.New("must not fetch")
	})
	mgr.GetRunningCache().Set(runKey("10.0.0.6", vlanAnchor), map[string]interface{}{
		"vlan": []interface{}{map[string]interface{}{"id": 10, "description": "from-cache"}},
	})

	body := `{"device":"10.0.0.6","entries":[{"op":"update","path":"/vlan:vlan/vlan:vlans","payload":{"vlan":[{"id":10,"description":"newer"}]}}]}`
	d := decodePreview(t, previewReq(h, body))
	if !assert.Len(t, d.Entries, 1) {
		return
	}
	assert.Equal(t, "cache", d.Entries[0].BaselineSource)
	assert.Contains(t, d.Entries[0].RollbackXML, "<description>from-cache</description>")
	assert.Equal(t, 0, fetchCalls)
}

// 基线链第三级：desired 与缓存都缺失时实时回读设备（source=device）。
func TestChangesetPreview_BaselineFromDevice(t *testing.T) {
	h, _ := newChangesetHandlerForTest(func(ctx context.Context, ip, path string) (interface{}, error) {
		return map[string]interface{}{
			"vlan": []interface{}{map[string]interface{}{"id": 10, "description": "live"}},
		}, nil
	})
	body := `{"device":"10.0.0.7","entries":[{"op":"update","path":"/vlan:vlan/vlan:vlans","payload":{"vlan":[{"id":10,"description":"newer"}]}}]}`
	d := decodePreview(t, previewReq(h, body))
	if !assert.Len(t, d.Entries, 1) {
		return
	}
	assert.Equal(t, "device", d.Entries[0].BaselineSource)
	assert.Contains(t, d.Entries[0].RollbackXML, "<description>live</description>")
}
