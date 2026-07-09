package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/stretchr/testify/assert"
)

func postDevice(t *testing.T, h *DeviceHandler, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/devices", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	h.AddDevice(c)
	return w
}

// BR-04/DS-03: 注册请求未带 vendor → 缺省 huawei 写入（存量行为零破坏）。
func TestAddDevice_VendorDefaultsHuawei(t *testing.T) {
	mgr := manager.New()
	h := NewDeviceHandler(mgr)

	postDevice(t, h, `{"ip":"127.0.0.1","port":830,"username":"u","password":"p"}`)

	info, ok := mgr.GetDeviceStore().Get("127.0.0.1")
	assert.True(t, ok)
	assert.Equal(t, "huawei", info.Vendor, "缺省厂商应为 huawei")
}

// BR-04: 显式携带受支持厂商 → 原样写入。
func TestAddDevice_ExplicitVendorStored(t *testing.T) {
	mgr := manager.New()
	h := NewDeviceHandler(mgr)

	postDevice(t, h, `{"ip":"127.0.0.2","port":830,"username":"u","password":"p","vendor":"huawei"}`)

	info, ok := mgr.GetDeviceStore().Get("127.0.0.2")
	assert.True(t, ok)
	assert.Equal(t, "huawei", info.Vendor)
}

// BR-04 负路径: 无已注册驱动的厂商 → code=400，不写 store（早失败）。
func TestAddDevice_UnknownVendorRejected(t *testing.T) {
	mgr := manager.New()
	h := NewDeviceHandler(mgr)

	w := postDevice(t, h, `{"ip":"127.0.0.3","port":830,"username":"u","password":"p","vendor":"nokia"}`)

	var resp Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 400, resp.Code, "未知厂商应 code=400")
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Message, "nokia", "错误应含厂商名")
	_, ok := mgr.GetDeviceStore().Get("127.0.0.3")
	assert.False(t, ok, "未知厂商不应写入 DeviceStore")
}

// DS-03: 种子设备携带缺省厂商。
func TestSeedDevice_VendorHuawei(t *testing.T) {
	mgr := manager.New()
	NewDeviceHandler(mgr)

	info, ok := mgr.GetDeviceStore().Get("192.168.1.1")
	assert.True(t, ok)
	assert.Equal(t, "huawei", info.Vendor, "种子设备 Vendor 应为 huawei")
}
