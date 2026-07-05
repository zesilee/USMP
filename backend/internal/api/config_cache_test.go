package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/stretchr/testify/assert"
)

// newConfigHandlerWithFetch builds a ConfigHandler backed by a real manager
// (for the running cache) but with a counting fake device fetch injected, so
// the cache behaviour is deterministic without touching a device/sim.
func newConfigHandlerWithFetch(fetch func(ctx context.Context, ip, path string) (interface{}, error)) *ConfigHandler {
	h := NewConfigHandler(manager.New())
	h.fetch = fetch
	return h
}

func getConfigReq(h *ConfigHandler, ip, path string, force bool) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "ip", Value: ip}, {Key: "path", Value: path}}
	url := "/"
	if force {
		url = "/?force_refresh=true"
	}
	c.Request = httptest.NewRequest(http.MethodGet, url, nil)
	h.GetConfig(c)
	return w
}

func decodeConfigData(t *testing.T, w *httptest.ResponseRecorder) ConfigGetData {
	t.Helper()
	assert.Equal(t, http.StatusOK, w.Code)
	var env struct {
		Data ConfigGetData `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return env.Data
}

func TestGetConfig_CacheMissThenHit(t *testing.T) {
	calls := 0
	h := newConfigHandlerWithFetch(func(ctx context.Context, ip, path string) (interface{}, error) {
		calls++
		return map[string]interface{}{"v": calls}, nil
	})

	// First GET: miss -> fetch once, fresh.
	d1 := decodeConfigData(t, getConfigReq(h, "10.0.0.1", "/vlans", false))
	assert.Equal(t, 1, calls)
	assert.False(t, d1.Cached)
	assert.Equal(t, "device", d1.Source)
	assert.Equal(t, 30, d1.TTLSeconds)

	// Second GET within TTL: hit -> fetch NOT called again.
	d2 := decodeConfigData(t, getConfigReq(h, "10.0.0.1", "/vlans", false))
	assert.Equal(t, 1, calls, "cache hit must not re-fetch from device")
	assert.True(t, d2.Cached)
	assert.Equal(t, "cache", d2.Source)
	assert.GreaterOrEqual(t, d2.CacheAgeSeconds, 0)
}

func TestGetConfig_ForceRefreshBypasses(t *testing.T) {
	calls := 0
	h := newConfigHandlerWithFetch(func(ctx context.Context, ip, path string) (interface{}, error) {
		calls++
		return "v", nil
	})
	getConfigReq(h, "10.0.0.1", "/vlans", false) // prime cache, calls=1
	d := decodeConfigData(t, getConfigReq(h, "10.0.0.1", "/vlans", true))
	assert.Equal(t, 2, calls, "force_refresh must bypass cache and re-fetch")
	assert.False(t, d.Cached)
	assert.Equal(t, "device", d.Source)
}

func TestGetConfig_TrailingSlashHitsSameEntry(t *testing.T) {
	calls := 0
	h := newConfigHandlerWithFetch(func(ctx context.Context, ip, path string) (interface{}, error) {
		calls++
		return "v", nil
	})
	getConfigReq(h, "10.0.0.1", "/vlans", false)  // calls=1
	getConfigReq(h, "10.0.0.1", "/vlans/", false) // normalized -> same key, hit
	assert.Equal(t, 1, calls, "trailing-slash variant must hit the same cache entry")
}

func TestGetConfig_NotConnected503(t *testing.T) {
	h := newConfigHandlerWithFetch(func(ctx context.Context, ip, path string) (interface{}, error) {
		return nil, errDeviceNotConnected
	})
	w := getConfigReq(h, "10.0.0.1", "/vlans", false)
	var env struct {
		Code    int  `json:"code"`
		Success bool `json:"success"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	assert.False(t, env.Success)
	assert.Equal(t, 503, env.Code)
}

func TestGetConfig_FetchError500(t *testing.T) {
	h := newConfigHandlerWithFetch(func(ctx context.Context, ip, path string) (interface{}, error) {
		return nil, errors.New("boom")
	})
	w := getConfigReq(h, "10.0.0.1", "/vlans", false)
	var env struct {
		Code int `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	assert.Equal(t, 500, env.Code)
}

func postConfigReq(h *ConfigHandler, ip, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "ip", Value: ip}, {Key: "path", Value: path}}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	h.SetConfig(c)
	return w
}

func TestSetConfig_InvalidatesRunningCache(t *testing.T) {
	calls := 0
	h := newConfigHandlerWithFetch(func(ctx context.Context, ip, path string) (interface{}, error) {
		calls++
		return "v", nil
	})
	getConfigReq(h, "10.0.0.1", "/vlan:vlan/vlan:vlans", false) // calls=1, cached
	getConfigReq(h, "10.0.0.1", "/vlan:vlan/vlan:vlans", false) // hit
	assert.Equal(t, 1, calls)

	w := postConfigReq(h, "10.0.0.1", "/vlan:vlan/vlan:vlans", `{"vlans":[{"id":10,"name":"VLAN10"}]}`)
	assert.Equal(t, http.StatusOK, w.Code)

	getConfigReq(h, "10.0.0.1", "/vlan:vlan/vlan:vlans", false) // must re-fetch
	assert.Equal(t, 2, calls, "SetConfig must invalidate the running cache")
}

func TestSetConfig_InvalidatesSubPaths(t *testing.T) {
	calls := 0
	h := newConfigHandlerWithFetch(func(ctx context.Context, ip, path string) (interface{}, error) {
		calls++
		return "v", nil
	})
	// cache a finer sub-path read
	getConfigReq(h, "10.0.0.1", "/vlan:vlan/vlan:vlans/detail", false) // calls=1
	getConfigReq(h, "10.0.0.1", "/vlan:vlan/vlan:vlans/detail", false) // hit
	assert.Equal(t, 1, calls)

	// push at the container path -> prefix invalidation clears the sub-path too
	postConfigReq(h, "10.0.0.1", "/vlan:vlan/vlan:vlans", `{"vlans":[{"id":10,"name":"VLAN10"}]}`)

	getConfigReq(h, "10.0.0.1", "/vlan:vlan/vlan:vlans/detail", false)
	assert.Equal(t, 2, calls, "push must invalidate all cached sub-paths of the device")
}

func TestSetConfig_StoreFailDoesNotInvalidate(t *testing.T) {
	calls := 0
	h := newConfigHandlerWithFetch(func(ctx context.Context, ip, path string) (interface{}, error) {
		calls++
		return "v", nil
	})
	getConfigReq(h, "10.0.0.1", "/vlan:vlan/vlan:vlans", false) // calls=1, cached

	// invalid VLAN ID -> validateConfig rejects (400) before any store/invalidate
	w := postConfigReq(h, "10.0.0.1", "/vlan:vlan/vlan:vlans", `{"vlans":[{"id":9999,"name":"BAD"}]}`)
	assert.Equal(t, http.StatusOK, w.Code) // envelope is 200 with error code in body
	var env struct {
		Code int `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	assert.Equal(t, 400, env.Code, "invalid config must be rejected")

	getConfigReq(h, "10.0.0.1", "/vlan:vlan/vlan:vlans", false) // still cached
	assert.Equal(t, 1, calls, "a rejected push must not invalidate the cache")
}
