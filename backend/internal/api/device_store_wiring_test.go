package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/stretchr/testify/assert"
)

// TestDeviceHandler_SeedWritesToStore: the seeded device must land in the shared
// DeviceStore with complete connection info, so reconcile/config/periodic can
// resolve its credentials (root fix for #100/#101).
func TestDeviceHandler_SeedWritesToStore(t *testing.T) {
	mgr := manager.New()
	NewDeviceHandler(mgr)

	info, ok := mgr.GetDeviceStore().Get("192.168.1.1")
	assert.True(t, ok, "种子设备应写入共享 DeviceStore")
	assert.Equal(t, 830, info.Port)
	assert.Equal(t, "admin", info.Username)
	assert.Equal(t, "admin", info.Password)
	assert.Equal(t, client.ProtocolAUTO, info.Protocol)
}

// TestDeviceHandler_AddDeviceWritesToStore: AddDevice must also register the
// device (with credentials) in the shared store. The immediate connect may fail
// for an unreachable IP, but registration happens before the connect attempt.
func TestDeviceHandler_AddDeviceWritesToStore(t *testing.T) {
	mgr := manager.New()
	h := NewDeviceHandler(mgr)

	body := `{"ip":"127.0.0.1","port":830,"username":"u","password":"p"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/devices", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	h.AddDevice(c)

	info, ok := mgr.GetDeviceStore().Get("127.0.0.1")
	assert.True(t, ok, "AddDevice 应写入 DeviceStore")
	assert.Equal(t, "u", info.Username)
	assert.Equal(t, "p", info.Password)
	assert.Equal(t, client.ProtocolAUTO, info.Protocol)
}

// TestDeviceHandler_RemoveDeviceDeletesFromStore: RemoveDevice must drop the
// device from the shared store too, keeping it as the single source of truth.
func TestDeviceHandler_RemoveDeviceDeletesFromStore(t *testing.T) {
	mgr := manager.New()
	h := NewDeviceHandler(mgr)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "ip", Value: "192.168.1.1"}}
	c.Request = httptest.NewRequest(http.MethodDelete, "/devices/192.168.1.1", nil)
	h.RemoveDevice(c)

	if _, ok := mgr.GetDeviceStore().Get("192.168.1.1"); ok {
		t.Fatal("RemoveDevice 应从 DeviceStore 删除设备")
	}
}
