package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/internal/intent"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
)

// fakePusher 记录 Push 的片段并返回预设结局（CS-04 B3：不触设备）。
type fakePusher struct {
	frags   []intent.Fragment
	results map[string]intent.TxResult
	calls   int
}

func (f *fakePusher) Push(ctx context.Context, frags []intent.Fragment) map[string]intent.TxResult {
	f.calls++
	f.frags = append(f.frags, frags...)
	return f.results
}

func newCommitHandlerForTest(push intent.Pusher) (*ChangesetHandler, manager.Manager) {
	mgr := manager.New()
	h := NewChangesetHandler(mgr)
	h.push = push
	return h, mgr
}

func commitReq(h *ChangesetHandler, body, query string) *httptest.ResponseRecorder {
	c, w := newTestContext(http.MethodPost, "/"+query, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Commit(c)
	return w
}

// CS-04：提交成功 → 片段映射正确、desired 落地、缓存失效、审计逐条目、
// 响应携对账触发状态。
func TestChangesetCommit_SuccessWritesDesiredAndAudit(t *testing.T) {
	push := &fakePusher{results: map[string]intent.TxResult{"10.0.0.1": {Device: "10.0.0.1"}}}
	h, mgr := newCommitHandlerForTest(push)
	seedDesiredVlan(mgr, "10.0.0.1", "old-desc")
	mgr.GetRunningCache().Set(runKey("10.0.0.1", vlanAnchor), map[string]interface{}{"stale": true})

	body := `{"device":"10.0.0.1","entries":[{"op":"update","path":"/vlan:vlan/vlan:vlans","payload":{"vlan":[{"id":10,"description":"new-desc"}]},"cleared":["name"]}]}`
	w := commitReq(h, body, "")
	assert.Equal(t, 0, envelopeCode(t, w), "body: %s", w.Body.String())

	// 片段映射：merge 主片段 + cleared 叶的 RawXML 片段
	assert.Equal(t, 1, push.calls)
	if assert.Len(t, push.frags, 2) {
		assert.Equal(t, intent.FragmentOpMerge, push.frags[0].Op)
		assert.NotNil(t, push.frags[0].Config)
		assert.Contains(t, push.frags[1].RawXML, `<name nc:operation="delete"`)
	}

	// desired 落地（同键覆盖）
	stored, err := mgr.GetConfigStore().Get("10.0.0.1", vlanAnchor)
	assert.NoError(t, err)
	v := stored.(*huawei.HuaweiVlan_Vlan_Vlans)
	if assert.Contains(t, v.Vlan, uint16(10)) {
		assert.Equal(t, "new-desc", *v.Vlan[10].Description)
	}

	// 缓存已失效
	_, _, hit := mgr.GetRunningCache().GetWithAge(runKey("10.0.0.1", vlanAnchor))
	assert.False(t, hit, "commit 成功后必须失效该设备缓存")

	// 审计逐条目
	recs := mgr.GetAuditStore().List()
	if assert.Len(t, recs, 1) {
		assert.Equal(t, "10.0.0.1", recs[0].DeviceIP)
	}

	var env struct {
		Data ChangesetCommitData `json:"data"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, "COMMITTED", env.Data.Status)
	assert.Equal(t, 1, env.Data.Entries)
}

// CS-04：delete 条目提交成功 → desired 移除该键、片段为 delete 语义。
func TestChangesetCommit_DeleteEntryRemovesDesired(t *testing.T) {
	push := &fakePusher{results: map[string]intent.TxResult{"10.0.0.2": {Device: "10.0.0.2"}}}
	h, mgr := newCommitHandlerForTest(push)
	seedDesiredVlan(mgr, "10.0.0.2", "keep")

	body := `{"device":"10.0.0.2","entries":[{"op":"delete","path":"/vlan:vlan/vlan:vlans","key":"10"}]}`
	w := commitReq(h, body, "")
	assert.Equal(t, 0, envelopeCode(t, w), "body: %s", w.Body.String())

	if assert.Len(t, push.frags, 1) {
		assert.Equal(t, intent.FragmentOpDelete, push.frags[0].Op)
	}
	stored, err := mgr.GetConfigStore().Get("10.0.0.2", vlanAnchor)
	assert.NoError(t, err)
	v := stored.(*huawei.HuaweiVlan_Vlan_Vlans)
	assert.NotContains(t, v.Vlan, uint16(10), "delete 条目提交后 desired 必须移除该键")
}

// CS-04 负路径：设备下发失败 → 502 信封、desired 原样、审计零记录、缓存保留
// （整体回退承诺：失败不得留任何控制器侧痕迹）。
func TestChangesetCommit_PushFailureNoSideEffects(t *testing.T) {
	push := &fakePusher{results: map[string]intent.TxResult{"10.0.0.3": {Device: "10.0.0.3", Err: assert.AnError}}}
	h, mgr := newCommitHandlerForTest(push)
	seedDesiredVlan(mgr, "10.0.0.3", "old")
	mgr.GetRunningCache().Set(runKey("10.0.0.3", vlanAnchor), map[string]interface{}{"v": 1})

	body := `{"device":"10.0.0.3","entries":[{"op":"update","path":"/vlan:vlan/vlan:vlans","payload":{"vlan":[{"id":10,"description":"newer"}]}}]}`
	w := commitReq(h, body, "")
	assert.Equal(t, 502, envelopeCode(t, w), "body: %s", w.Body.String())

	stored, _ := mgr.GetConfigStore().Get("10.0.0.3", vlanAnchor)
	v := stored.(*huawei.HuaweiVlan_Vlan_Vlans)
	assert.Equal(t, "old", *v.Vlan[10].Description, "失败后 desired 必须保持提交前状态")
	assert.Empty(t, mgr.GetAuditStore().List(), "失败不得写成功审计")
	_, _, hit := mgr.GetRunningCache().GetWithAge(runKey("10.0.0.3", vlanAnchor))
	assert.True(t, hit, "失败不得失效缓存")
}

// CS-04：归属硬锁（BR-11 口径）——认领路径无 force 409 拒绝且零下发；
// force=true 放行并审计留痕。
func TestChangesetCommit_OwnershipLock(t *testing.T) {
	const key = "default/test-intent"
	intent.DefaultOwnership.Replace(key, []intent.Claim{{Device: "10.0.0.4", Module: "vlan", Path: vlanAnchor}})
	t.Cleanup(func() { intent.DefaultOwnership.Remove(key) })

	push := &fakePusher{results: map[string]intent.TxResult{"10.0.0.4": {Device: "10.0.0.4"}}}
	h, mgr := newCommitHandlerForTest(push)

	body := `{"device":"10.0.0.4","entries":[{"op":"update","path":"/vlan:vlan/vlan:vlans","payload":{"vlan":[{"id":10,"description":"x"}]}}]}`

	w := commitReq(h, body, "")
	assert.Equal(t, 409, envelopeCode(t, w), "body: %s", w.Body.String())
	assert.Equal(t, 0, push.calls, "无 force 时不得触达下发通道")

	w = commitReq(h, body, "?force=true")
	assert.Equal(t, 0, envelopeCode(t, w), "body: %s", w.Body.String())
	recs := mgr.GetAuditStore().List()
	if assert.Len(t, recs, 1) {
		assert.True(t, recs[0].Forced, "force 覆盖必须审计留痕")
		assert.NotEmpty(t, recs[0].ForcedOwners)
	}
}

// CS-04 负路径：解码失败整体 400（与 preview 同口径），零下发。
func TestChangesetCommit_BadRequest(t *testing.T) {
	push := &fakePusher{}
	h, _ := newCommitHandlerForTest(push)
	for name, body := range map[string]string{
		"empty":        `{"device":"10.0.0.5","entries":[]}`,
		"unknown path": `{"device":"10.0.0.5","entries":[{"op":"update","path":"/nope:x","payload":{"a":1}}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			w := commitReq(h, body, "")
			assert.Equal(t, http.StatusBadRequest, envelopeCode(t, w))
			assert.Equal(t, 0, push.calls)
		})
	}
}
