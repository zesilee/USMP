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

// DS-03: 种子设备不再硬编码，仅经 USMP_SEED_DEVICE 注入（root fix for #100/#101
// 的注册语义保持：完整连接信息进共享 DeviceStore）。
func TestDeviceHandler_SeedFromEnvWritesToStore(t *testing.T) {
	t.Setenv("USMP_SEED_DEVICE", "192.168.1.1:830,admin,admin")
	mgr := manager.New()
	NewDeviceHandler(mgr)

	info, ok := mgr.GetDeviceStore().Get("192.168.1.1")
	assert.True(t, ok, "种子设备应写入共享 DeviceStore")
	assert.Equal(t, 830, info.Port)
	assert.Equal(t, "admin", info.Username)
	assert.Equal(t, "admin", info.Password)
	assert.Equal(t, client.ProtocolAUTO, info.Protocol)
}

// DS-03: 未设种子变量则空库启动，不崩溃。
func TestDeviceHandler_NoSeedEnvEmptyStore(t *testing.T) {
	t.Setenv("USMP_SEED_DEVICE", "")
	mgr := manager.New()
	NewDeviceHandler(mgr)

	assert.Empty(t, mgr.GetDeviceStore().List(), "未设 USMP_SEED_DEVICE 应空库启动")
}

// DS-03 边界：格式错误的种子变量仅告警不入库、不崩溃（R08）。
func TestDeviceHandler_MalformedSeedEnvIgnored(t *testing.T) {
	for _, bad := range []string{"192.168.1.1", "192.168.1.1,admin", "ip:notaport,u,p", ",u,p"} {
		t.Setenv("USMP_SEED_DEVICE", bad)
		mgr := manager.New()
		NewDeviceHandler(mgr)
		assert.Empty(t, mgr.GetDeviceStore().List(), "格式错误 %q 不应入库", bad)
	}
}

// DS-03: 端口缺省 830、厂商可显式指定。
func TestDeviceHandler_SeedEnvDefaultsAndVendor(t *testing.T) {
	t.Setenv("USMP_SEED_DEVICE", "10.0.0.5,op,pw,h3c")
	mgr := manager.New()
	NewDeviceHandler(mgr)

	info, ok := mgr.GetDeviceStore().Get("10.0.0.5")
	assert.True(t, ok)
	assert.Equal(t, 830, info.Port, "未带端口应缺省 830")
	assert.Equal(t, "h3c", info.Vendor)
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
	t.Setenv("USMP_SEED_DEVICE", "192.168.1.1:830,admin,admin")
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
