package api

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
)

func addDeviceRaw(t *testing.T, h *DeviceHandler, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/v1/devices", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	h.AddDevice(c)
	return w
}

// TestAddDeviceWithRole（BR-14 正路径）：注册带 role → store 落库、列表透传。
func TestAddDeviceWithRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mgr := manager.New()
	h := NewDeviceHandler(mgr)

	w := addDeviceRaw(t, h, `{"ip":"10.0.0.9","port":830,"username":"u","password":"p","role":"DCGW"}`)
	if w.Code != 200 {
		t.Fatalf("add: %d %s", w.Code, w.Body.String())
	}
	info, ok := mgr.GetDeviceStore().Get("10.0.0.9")
	if !ok || info.Role != "DCGW" {
		t.Fatalf("store role = %q ok=%v, want DCGW", info.Role, ok)
	}

	lw := httptest.NewRecorder()
	lc, _ := gin.CreateTestContext(lw)
	h.ListDevices(lc)
	var data DeviceListData
	decodeData(t, lw.Body.Bytes(), &data)
	found := false
	for _, d := range data.Devices {
		if d.IP == "10.0.0.9" {
			found = true
			if d.Role != "DCGW" {
				t.Errorf("list role = %q, want DCGW", d.Role)
			}
		}
	}
	if !found {
		t.Fatal("device not in list")
	}
}

// TestAddDeviceRoleInvalid（BR-14 负路径）：非法 role → 400 不落库。
func TestAddDeviceRoleInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mgr := manager.New()
	h := NewDeviceHandler(mgr)

	for _, bad := range []string{
		`"role":"` + strings.Repeat("A", 33) + `"`, // 超长
		`"role":"DC GW"`, // 空格
		`"role":"DC/GW"`, // 非法字符
	} {
		w := addDeviceRaw(t, h, `{"ip":"10.0.0.10","username":"u","password":"p",`+bad+`}`)
		var env struct {
			Code    int  `json:"code"`
			Success bool `json:"success"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if env.Success || env.Code != 400 {
			t.Errorf("bad role %s: envelope code = %d success=%v, want 400/false", bad, env.Code, env.Success)
		}
	}
	if _, ok := mgr.GetDeviceStore().Get("10.0.0.10"); ok {
		t.Error("非法 role 不应落库")
	}
}

// TestAddDeviceRoleOmitted（BR-14 缺省）：不带 role → 正常注册且响应无 role 键。
func TestAddDeviceRoleOmitted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mgr := manager.New()
	h := NewDeviceHandler(mgr)

	w := addDeviceRaw(t, h, `{"ip":"10.0.0.11","username":"u","password":"p"}`)
	if w.Code != 200 {
		t.Fatalf("add: %d %s", w.Code, w.Body.String())
	}
	lw := httptest.NewRecorder()
	lc, _ := gin.CreateTestContext(lw)
	h.ListDevices(lc)
	var env struct {
		Data struct {
			Devices []map[string]json.RawMessage `json:"devices"`
		} `json:"data"`
	}
	if err := json.Unmarshal(lw.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, d := range env.Data.Devices {
		if _, has := d["role"]; has {
			t.Errorf("缺省 role 不应序列化 role 键: %v", d)
		}
	}
}
