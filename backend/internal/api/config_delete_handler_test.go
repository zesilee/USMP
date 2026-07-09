package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/openconfig/ygot/ygot"
	"github.com/stretchr/testify/assert"
)

// envelopeCode 解析统一信封的业务码（项目约定：HTTP 恒 200，错误在 body.code）。
func envelopeCode(t *testing.T, w *httptest.ResponseRecorder) int {
	t.Helper()
	var env struct {
		Code    int  `json:"code"`
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v (%s)", err, w.Body.String())
	}
	return env.Code
}

func deleteConfigReq(h *ConfigHandler, ip, path, key string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/config/"+ip+path+"?key="+key, nil)
	if key != "" {
		c.Request.URL.RawQuery = "key=" + key
	}
	c.Params = gin.Params{{Key: "ip", Value: ip}, {Key: "path", Value: path}}
	h.DeleteConfig(c)
	return w
}

// BR-09：删除成功——同步下发、desired 移除、缓存失效、审计、触发对账。
func TestDeleteConfig_Success(t *testing.T) {
	mgr := manager.New()
	h := NewConfigHandler(mgr)
	var pushed interface{}
	h.pushDelete = func(_ context.Context, ip string, target interface{}) error {
		pushed = target
		return nil
	}

	// 预置 desired（两条）与运行缓存
	seed := &huawei.HuaweiVlan_Vlan_Vlans{Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
		10: {Id: ygot.Uint16(10)}, 20: {Id: ygot.Uint16(20)},
	}}
	assert.NoError(t, mgr.GetConfigStore().Set("10.0.0.1", "/vlan:vlan/vlan:vlans", seed))
	mgr.GetRunningCache().Set(runKey("10.0.0.1", "/vlan:vlan/vlan:vlans"), "cached")

	w := deleteConfigReq(h, "10.0.0.1", "/vlan:vlan/vlan:vlans", "10")
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 0, envelopeCode(t, w), w.Body.String())

	// 下发目标是仅含键 10 的模型对象
	vlans, ok := pushed.(*huawei.HuaweiVlan_Vlan_Vlans)
	if assert.True(t, ok, "pushed %T", pushed) {
		assert.Len(t, vlans.Vlan, 1)
		assert.NotNil(t, vlans.Vlan[10])
	}
	// desired 移除键 10、保留 20
	got, _ := mgr.GetConfigStore().Get("10.0.0.1", "/vlan:vlan/vlan:vlans")
	stored := got.(*huawei.HuaweiVlan_Vlan_Vlans)
	assert.NotContains(t, stored.Vlan, uint16(10))
	assert.Contains(t, stored.Vlan, uint16(20))
	// 运行缓存已失效
	_, hit := mgr.GetRunningCache().Get(runKey("10.0.0.1", "/vlan:vlan/vlan:vlans"))
	assert.False(t, hit, "running cache must be invalidated after delete")
	// 审计一条删除记录
	logs := mgr.GetAuditStore().List()
	if assert.Len(t, logs, 1) {
		assert.Equal(t, "10.0.0.1", logs[0].DeviceIP)
		assert.Contains(t, logs[0].Summary, "10")
	}
	// 响应携带对账触发信息
	var env struct {
		Data ConfigDeleteData `json:"data"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, "DELETED", env.Data.Status)
	assert.Equal(t, "10", env.Data.Key)
}

// BR-09 负路径：key 缺失/非法/未知路径 → 400，不触达设备、无审计。
func TestDeleteConfig_BadRequest(t *testing.T) {
	cases := []struct{ desc, path, key string }{
		{"缺 key", "/vlan:vlan/vlan:vlans", ""},
		{"vlan key 非整数", "/vlan:vlan/vlan:vlans", "abc"},
		{"vlan key 超范围", "/vlan:vlan/vlan:vlans", "5000"},
		{"未知路径", "/route:route/tables", "1"},
	}
	for _, cse := range cases {
		t.Run(cse.desc, func(t *testing.T) {
			mgr := manager.New()
			h := NewConfigHandler(mgr)
			called := false
			h.pushDelete = func(context.Context, string, interface{}) error { called = true; return nil }
			w := deleteConfigReq(h, "10.0.0.1", cse.path, cse.key)
			assert.Equal(t, 400, envelopeCode(t, w), w.Body.String())
			assert.False(t, called, "must not reach device")
			assert.Empty(t, mgr.GetAuditStore().List())
		})
	}
}

// BR-09 负路径：下发失败（如设备 data-missing）→ 错误透出，缓存不失效、无审计。
func TestDeleteConfig_PushFailure(t *testing.T) {
	mgr := manager.New()
	h := NewConfigHandler(mgr)
	h.pushDelete = func(context.Context, string, interface{}) error {
		return errors.New(`edit-config delete: target "vlan" not found (data-missing)`)
	}
	mgr.GetRunningCache().Set(runKey("10.0.0.1", "/vlan:vlan/vlan:vlans"), "cached")

	w := deleteConfigReq(h, "10.0.0.1", "/vlan:vlan/vlan:vlans", "10")
	assert.Equal(t, 502, envelopeCode(t, w), w.Body.String())
	assert.Contains(t, w.Body.String(), "data-missing")
	_, hit := mgr.GetRunningCache().Get(runKey("10.0.0.1", "/vlan:vlan/vlan:vlans"))
	assert.True(t, hit, "cache must survive failed delete")
	assert.Empty(t, mgr.GetAuditStore().List())
}

// BR-10：模型门禁经 handler 生效——operation-exclude∋delete 的路径 400（先于模型分支解析）。
func TestDeleteConfig_GateRejects(t *testing.T) {
	mgr := manager.New(manager.WithSchema(buildGateSchema(t)))
	h := NewConfigHandler(mgr)
	called := false
	h.pushDelete = func(context.Context, string, interface{}) error { called = true; return nil }

	w := deleteConfigReq(h, "10.0.0.1", "/demo:demo/demo:locked/demo:entry", "x")
	assert.Equal(t, 400, envelopeCode(t, w), w.Body.String())
	assert.Contains(t, w.Body.String(), "operation-exclude")
	assert.False(t, called)
}
