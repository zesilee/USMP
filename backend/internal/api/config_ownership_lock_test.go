package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/leezesi/usmp/backend/internal/intent"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
)

// BR-11 硬锁二期（矩阵 B3）：认领路径手改缺省 409 拒绝（携 intents、不下发不记审计），
// force=true 放行（附警告 + 审计 Forced 留痕），未认领/兄弟路径不受锁。

// postConfigRaw 同 postConfigReq，额外允许 query（force=true）。
func postConfigRaw(h *ConfigHandler, ip, path, rawQuery, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "ip", Value: ip}, {Key: "path", Value: path}}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.URL.RawQuery = rawQuery
	c.Request = req
	h.SetConfig(c)
	return w
}

type lockEnvelope struct {
	Code    int  `json:"code"`
	Success bool `json:"success"`
	Data    struct {
		Intents          []string          `json:"intents"`
		OwnershipWarning *OwnershipWarning `json:"ownershipWarning"`
	} `json:"data"`
}

func decodeLockEnvelope(t *testing.T, w *httptest.ResponseRecorder) lockEnvelope {
	t.Helper()
	var env lockEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v (%s)", err, w.Body.String())
	}
	return env
}

// 无 force 命中认领路径：409 + intents，不下发不记审计。
func TestSetConfig_OwnedPathRejected409(t *testing.T) {
	withOwnership(t, map[string][]intent.Claim{
		"default/biz-100": {{Device: "10.0.0.1", Module: "vlan", Path: intent.VlanPath + "/vlan[id=100]"}},
	})
	mgr := manager.New()
	h := NewConfigHandler(mgr)

	w := postConfigRaw(h, "10.0.0.1", intent.VlanPath, "", `{"vlans":[{"id":100,"name":"HAND"}]}`)
	assert.Equal(t, http.StatusOK, w.Code) // 信封恒 200
	env := decodeLockEnvelope(t, w)
	assert.Equal(t, 409, env.Code)
	assert.False(t, env.Success)
	assert.Equal(t, []string{"default/biz-100"}, env.Data.Intents)
	assert.Contains(t, w.Body.String(), "force", "message 应指引 force 逃生通道")
	assert.Empty(t, mgr.GetAuditStore().List(), "归属拒绝不应写审计")
}

// force=true 放行：正常接受 + ownershipWarning + 审计 Forced/ForcedOwners 留痕。
func TestSetConfig_ForceBypassesLock(t *testing.T) {
	withOwnership(t, map[string][]intent.Claim{
		"default/biz-100": {{Device: "10.0.0.1", Module: "vlan", Path: intent.VlanPath + "/vlan[id=100]"}},
	})
	mgr := manager.New()
	h := NewConfigHandler(mgr)

	w := postConfigRaw(h, "10.0.0.1", intent.VlanPath, "force=true", `{"vlans":[{"id":100,"name":"HAND"}]}`)
	env := decodeLockEnvelope(t, w)
	assert.Equal(t, 0, env.Code)
	assert.True(t, env.Success)
	if assert.NotNil(t, env.Data.OwnershipWarning, "force 放行仍应附归属警告") {
		assert.Equal(t, []string{"default/biz-100"}, env.Data.OwnershipWarning.Intents)
	}
	logs := mgr.GetAuditStore().List()
	if assert.Len(t, logs, 1) {
		assert.True(t, logs[0].Forced, "force 覆盖必须留痕")
		assert.Equal(t, []string{"default/biz-100"}, logs[0].ForcedOwners)
	}
}

// 未认领路径：无 force 照常接受，审计不带 Forced。
func TestSetConfig_UnownedPathUnaffected(t *testing.T) {
	mgr := manager.New()
	h := NewConfigHandler(mgr)

	w := postConfigRaw(h, "10.0.0.1", intent.VlanPath, "", `{"vlans":[{"id":10,"name":"OK"}]}`)
	env := decodeLockEnvelope(t, w)
	assert.Equal(t, 0, env.Code)
	assert.True(t, env.Success)
	assert.Nil(t, env.Data.OwnershipWarning)
	logs := mgr.GetAuditStore().List()
	if assert.Len(t, logs, 1) {
		assert.False(t, logs[0].Forced)
		assert.Empty(t, logs[0].ForcedOwners)
	}
}

// 兄弟路径不受锁（负路径）：认领只在 vlan，ifm 路径照常。
func TestSetConfig_SiblingPathNotLocked(t *testing.T) {
	withOwnership(t, map[string][]intent.Claim{
		"default/biz-100": {{Device: "10.0.0.1", Module: "vlan", Path: intent.VlanPath + "/vlan[id=100]"}},
	})
	mgr := manager.New()
	h := NewConfigHandler(mgr)

	w := postConfigRaw(h, "10.0.0.1", intent.IfmPath, "", `{"interfaces":[{"name":"GE0/0/1"}]}`)
	env := decodeLockEnvelope(t, w)
	assert.NotEqual(t, 409, env.Code, "未认领的兄弟路径不应被硬锁: %s", w.Body.String())
}

// 行删除同受硬锁：无 force → 409 携 intents，不触达设备；force → 越过门禁（后续因设备
// 不可达失败也证明门禁已放行，断言非 409 即可）。
func TestDeleteConfig_OwnedPathRejected409(t *testing.T) {
	withOwnership(t, map[string][]intent.Claim{
		"default/biz-100": {{Device: "10.0.0.1", Module: "vlan", Path: intent.VlanPath + "/vlan[id=100]"}},
	})
	mgr := manager.New()
	h := NewConfigHandler(mgr)
	pushed := false
	h.pushDelete = func(context.Context, string, interface{}) error { pushed = true; return nil }

	w := deleteConfigRaw(h, "10.0.0.1", intent.VlanPath, "key=100", "")
	env := decodeLockEnvelope(t, w)
	assert.Equal(t, 409, env.Code)
	assert.Equal(t, []string{"default/biz-100"}, env.Data.Intents)
	assert.False(t, pushed, "归属拒绝不应触达设备")
	assert.Empty(t, mgr.GetAuditStore().List(), "归属拒绝不应写审计")

	w2 := deleteConfigRaw(h, "10.0.0.1", intent.VlanPath, "key=100", "force=true")
	env2 := decodeLockEnvelope(t, w2)
	assert.Equal(t, 0, env2.Code, "force 应越过归属门禁: %s", w2.Body.String())
	assert.True(t, pushed)
	logs := mgr.GetAuditStore().List()
	if assert.Len(t, logs, 1) {
		assert.True(t, logs[0].Forced)
		assert.Equal(t, []string{"default/biz-100"}, logs[0].ForcedOwners)
	}
}

func deleteConfigRaw(h *ConfigHandler, ip, path, keyQuery, forceQuery string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	raw := keyQuery
	if forceQuery != "" {
		raw += "&" + forceQuery
	}
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/config/"+ip+path+"?"+raw, nil)
	c.Request.URL.RawQuery = raw
	c.Params = gin.Params{{Key: "ip", Value: ip}, {Key: "path", Value: path}}
	h.DeleteConfig(c)
	return w
}
