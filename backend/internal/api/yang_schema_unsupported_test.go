package api

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
)

// clientConnInfoUnreachable 指向不可达地址（TEST-NET-3，CN-06 离线降级用）。
// 不能用 127.0.0.1：连接池按 info.IP 作键，会与 sim 的既有连接串键。
func clientConnInfoUnreachable() client.DeviceConnectionInfo {
	return client.DeviceConnectionInfo{
		IP: "203.0.113.1", Port: 830, Protocol: client.ProtocolNETCONF, Timeout: 500 * time.Millisecond,
	}
}

// CN-05/CN-06（tasks 4.1/4.3）：schema ?device= 透出已学习不支持子路径；
// /devices/:ip/capabilities 透出 hello 原文。

func getSchemaReq(h *YangHandler, module, query string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "module", Value: module}}
	c.Request = httptest.NewRequest("GET", "/api/v1/yang/schema/"+module+query, nil)
	h.GetSchema(c)
	return w
}

// ?device= 且已学习：unsupported 返回相对模块根的首段名。
func TestGetSchema_DeviceUnsupportedExposed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	gin.SetMode(gin.TestMode)
	h, deviceID, cleanup := newDeviceYangHarness(t, nil)
	defer cleanup()

	// 在真实连接上种学习结果（学习链路已由 BR-12 测试覆盖，此处测透出）。
	view := supportViewFromPool(h.manager, deviceID)
	if view == nil {
		// 无既有连接时 Peek 为 nil：先经能力查询建连再取视图。
		info, _ := h.manager.GetDeviceStore().Get(deviceID)
		_, _ = h.manager.GetClientPool().Get(info)
		view = supportViewFromPool(h.manager, deviceID)
	}
	if view == nil {
		t.Fatal("无法取得节点支持视图")
	}
	view.MarkUnsupportedPath("vlan:vlan/vlan:vlans")
	view.MarkUnsupportedPath("ifm:ifm/ifm:interfaces") // 其他模块不得混入

	var schema struct {
		Module      string   `json:"module"`
		Unsupported []string `json:"unsupported"`
	}
	w := getSchemaReq(h, "vlan", "?device="+deviceID)
	decodeData(t, w.Body.Bytes(), &schema)
	assert.Equal(t, []string{"vlans"}, schema.Unsupported)
}

// 无 device 参数：响应不含 unsupported 键（向后兼容）。
func TestGetSchema_NoDeviceOmitsUnsupported(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	gin.SetMode(gin.TestMode)
	h, _, cleanup := newDeviceYangHarness(t, nil)
	defer cleanup()
	w := getSchemaReq(h, "vlan", "")
	assert.NotContains(t, w.Body.String(), `"unsupported"`)
}

// ?device= 但零学习：省略键（空集不输出）。
func TestGetSchema_DeviceNoLearningOmitsKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	gin.SetMode(gin.TestMode)
	h, deviceID, cleanup := newDeviceYangHarness(t, nil)
	defer cleanup()
	w := getSchemaReq(h, "vlan", "?device="+deviceID)
	assert.NotContains(t, w.Body.String(), `"unsupported"`)
}

// CN-06：在线设备 hello 原文透出。
func TestDeviceCapabilities_Online(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	gin.SetMode(gin.TestMode)
	yh, deviceID, cleanup := newDeviceYangHarness(t, []string{
		"urn:ietf:params:netconf:base:1.0",
		"urn:huawei:yang:huawei-vlan?module=huawei-vlan&revision=2020-02-07&deviations=huawei-vlan-deviations",
	})
	defer cleanup()
	h := NewDeviceHandler(yh.manager)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "ip", Value: deviceID}}
	c.Request = httptest.NewRequest("GET", "/", nil)
	h.GetCapabilities(c)

	var data struct {
		Capabilities []string `json:"capabilities"`
		Negotiated   bool     `json:"negotiated"`
	}
	decodeData(t, w.Body.Bytes(), &data)
	assert.True(t, data.Negotiated)
	assert.Contains(t, data.Capabilities, "urn:huawei:yang:huawei-vlan?module=huawei-vlan&revision=2020-02-07&deviations=huawei-vlan-deviations")
}

// CN-06 负路径：未注册 404；离线空列表+negotiated:false 不 5xx。
func TestDeviceCapabilities_NegativePaths(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	gin.SetMode(gin.TestMode)
	yh, _, cleanup := newDeviceYangHarness(t, nil)
	defer cleanup()
	h := NewDeviceHandler(yh.manager)

	// 未注册 → 404（信封 code）
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "ip", Value: "10.9.9.9"}}
	c.Request = httptest.NewRequest("GET", "/", nil)
	h.GetCapabilities(c)
	code, _, _ := decodeEnvelope(t, w)
	assert.Equal(t, 404, code)

	// 已注册但建连失败（离线）→ 空列表 + negotiated:false，不 5xx（CN-06 降级）
	assert.NoError(t, yh.manager.GetDeviceStore().Put("dead-dev", clientConnInfoUnreachable()))
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Params = gin.Params{{Key: "ip", Value: "dead-dev"}}
	c2.Request = httptest.NewRequest("GET", "/", nil)
	h.GetCapabilities(c2)
	var data struct {
		Capabilities []string `json:"capabilities"`
		Negotiated   bool     `json:"negotiated"`
	}
	decodeData(t, w2.Body.Bytes(), &data)
	assert.False(t, data.Negotiated)
	assert.NotNil(t, data.Capabilities)
	assert.Empty(t, data.Capabilities)
}
