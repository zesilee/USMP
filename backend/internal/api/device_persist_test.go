package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
)

// failingStore 模拟持久化后端（CRD）写失败（DS-04/BR-13）。
type failingStore struct {
	device.Store
	failPut, failDelete bool
	puts                int
}

func (f *failingStore) Put(id string, info client.DeviceConnectionInfo) error {
	f.puts++
	if f.failPut {
		return errors.New("apiserver unreachable")
	}
	return f.Store.Put(id, info)
}

func (f *failingStore) Delete(id string) error {
	if f.failDelete {
		return errors.New("apiserver unreachable")
	}
	return f.Store.Delete(id)
}

// persistentStore 模拟集群模式 store（DS-03：种子变量忽略）。
type persistentStore struct{ device.Store }

func (persistentStore) Persistent() bool { return true }

func addDeviceReq(h *DeviceHandler, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/devices", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	h.AddDevice(c)
	return w
}

// BR-13: 集群模式注册持久化失败 → 5xx 统一信封，设备不入库。
func TestAddDevice_PersistFailureReturns5xx(t *testing.T) {
	fs := &failingStore{Store: device.NewStore(), failPut: true}
	mgr := manager.New(manager.WithDeviceStore(fs))
	h := NewDeviceHandler(mgr)

	w := addDeviceReq(h, `{"ip":"127.0.0.1","port":830,"username":"u","password":"p"}`)

	var resp Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 500, resp.Code, "持久化失败应 5xx")
	assert.False(t, resp.Success)
	if _, ok := mgr.GetDeviceStore().Get("127.0.0.1"); ok {
		t.Fatal("持久化失败设备不应可读")
	}
}

// BR-13: 删除持久化失败同样 5xx 可见。
func TestRemoveDevice_PersistFailureReturns5xx(t *testing.T) {
	fs := &failingStore{Store: device.NewStore(), failDelete: true}
	mgr := manager.New(manager.WithDeviceStore(fs))
	h := NewDeviceHandler(mgr)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "ip", Value: "10.0.0.1"}}
	c.Request = httptest.NewRequest(http.MethodDelete, "/devices/10.0.0.1", nil)
	h.RemoveDevice(c)

	var resp Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 500, resp.Code)
	assert.False(t, resp.Success)
}

// DS-03: 集群模式（持久 store）下 USMP_SEED_DEVICE 被忽略，设备集合仅来自 CR。
func TestSeedIgnoredOnPersistentStore(t *testing.T) {
	t.Setenv("USMP_SEED_DEVICE", "192.168.1.1:830,admin,admin")
	mgr := manager.New(manager.WithDeviceStore(persistentStore{device.NewStore()}))
	NewDeviceHandler(mgr)

	assert.Empty(t, mgr.GetDeviceStore().List(), "集群模式应忽略种子变量")
}
