package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/stretchr/testify/assert"
)

// BR-13/BR-14 B3 契约：GET /config 分页参数（仅 list 生效、无参形状不变）
// 与 include_state 状态快照缓存（命中不打设备、force_refresh 直打、写不失效）。

// makeIfaceRows 造 n 行 RFC7951 接口条目。
func makeIfaceRows(n int) []interface{} {
	rows := make([]interface{}, 0, n)
	for i := 0; i < n; i++ {
		rows = append(rows, map[string]interface{}{
			"name":         fmt.Sprintf("GE1/0/%d", i),
			"admin-status": []string{"up", "down"}[i%2],
			"mtu":          float64(1000 + i),
		})
	}
	return rows
}

// getConfigPageReq 发起带任意 query 的 GET /config 请求。
func getConfigPageReq(h *ConfigHandler, ip, path, rawQuery string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "ip", Value: ip}, {Key: "path", Value: path}}
	url := "/"
	if rawQuery != "" {
		url = "/?" + rawQuery
	}
	c.Request = httptest.NewRequest(http.MethodGet, url, nil)
	h.GetConfig(c)
	return w
}

// pageEnvelope 解出分页模式响应（data.data 为 ListPage 形状）。
type pageEnvelope struct {
	Code    int  `json:"code"`
	Success bool `json:"success"`
	Data    struct {
		Data struct {
			Rows   []map[string]interface{} `json:"rows"`
			Total  int                      `json:"total"`
			Limit  int                      `json:"limit"`
			Offset int                      `json:"offset"`
		} `json:"data"`
		Cached          bool   `json:"cached"`
		CacheAgeSeconds int    `json:"cache_age_seconds"`
		TTLSeconds      int    `json:"ttl_seconds"`
		Source          string `json:"source"`
	} `json:"data"`
	Message string `json:"message"`
}

func decodePage(t *testing.T, w *httptest.ResponseRecorder) pageEnvelope {
	t.Helper()
	assert.Equal(t, http.StatusOK, w.Code)
	var env pageEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return env
}

// Scenario: 分页读取大 list——rows 为指定窗口、total 全量、携新鲜度字段。
func TestGetConfig_PaginatedList(t *testing.T) {
	h := newConfigHandlerWithFetch(func(ctx context.Context, ip, path string) (interface{}, error) {
		return map[string]interface{}{"interface": makeIfaceRows(66)}, nil
	})
	env := decodePage(t, getConfigPageReq(h, "10.0.0.1", "/ifm:ifm/ifm:interfaces/ifm:interface", "limit=10&offset=20"))
	assert.True(t, env.Success)
	assert.Equal(t, 66, env.Data.Data.Total)
	assert.Len(t, env.Data.Data.Rows, 10)
	assert.Equal(t, "GE1/0/20", env.Data.Data.Rows[0]["name"], "offset=20 应从第 21 行开始")
	assert.Equal(t, 10, env.Data.Data.Limit)
	assert.Equal(t, 20, env.Data.Data.Offset)
	assert.Equal(t, "device", env.Data.Source, "新鲜度字段应保留")
}

// Scenario: 过滤与排序组合——total 为过滤后总数。
func TestGetConfig_PaginatedFilterSort(t *testing.T) {
	h := newConfigHandlerWithFetch(func(ctx context.Context, ip, path string) (interface{}, error) {
		return map[string]interface{}{"interface": makeIfaceRows(66)}, nil
	})
	env := decodePage(t, getConfigPageReq(h, "10.0.0.1", "/p",
		"limit=5&filter=admin-status%3D%3Dup&sort=mtu&sort_dir=desc"))
	assert.Equal(t, 33, env.Data.Data.Total, "66 行中 admin-status=up 的应为 33")
	assert.Len(t, env.Data.Data.Rows, 5)
	assert.Equal(t, float64(1064), env.Data.Data.Rows[0]["mtu"], "mtu 降序首行应为最大偶数行 1064")
}

// Scenario: 无参数形状不变（回归锚点）——data.data 为整树，无 rows/total。
func TestGetConfig_NoParamsShapeUnchanged(t *testing.T) {
	h := newConfigHandlerWithFetch(func(ctx context.Context, ip, path string) (interface{}, error) {
		return map[string]interface{}{"interface": makeIfaceRows(3)}, nil
	})
	w := getConfigPageReq(h, "10.0.0.1", "/p", "")
	var env struct {
		Data struct {
			Data map[string]interface{} `json:"data"`
		} `json:"data"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	arr, ok := env.Data.Data["interface"].([]interface{})
	assert.True(t, ok, "无参响应必须保持整树形状（interface 数组直挂）")
	assert.Len(t, arr, 3)
	_, hasRows := env.Data.Data["rows"]
	assert.False(t, hasRows, "无参响应不得出现 rows 字段")
}

// Scenario: 非 list 路径携 limit 拒绝（信封 400，不静默忽略）。
func TestGetConfig_PaginationOnNonList400(t *testing.T) {
	h := newConfigHandlerWithFetch(func(ctx context.Context, ip, path string) (interface{}, error) {
		return map[string]interface{}{"statistic-interval": float64(300)}, nil
	})
	env := decodePage(t, getConfigPageReq(h, "10.0.0.1", "/ifm:ifm/ifm:global", "limit=10"))
	assert.False(t, env.Success)
	assert.Equal(t, 400, env.Code)
	assert.Contains(t, env.Message, "/ifm:ifm/ifm:global", "错误信息应含路径")
}

// Scenario: 参数非法拒绝（limit 越界 / filter 语法错）。
func TestGetConfig_BadPaginationParams400(t *testing.T) {
	h := newConfigHandlerWithFetch(func(ctx context.Context, ip, path string) (interface{}, error) {
		t.Fatal("参数非法必须在触达 fetch 前拒绝")
		return nil, nil
	})
	for _, q := range []string{"limit=1001", "limit=0", "limit=10&filter=name", "limit=10&sort=x&sort_dir=zig"} {
		env := decodePage(t, getConfigPageReq(h, "10.0.0.1", "/p", q))
		assert.False(t, env.Success, "query %q 应拒绝", q)
		assert.Equal(t, 400, env.Code, "query %q 应 400", q)
	}
}

// Scenario: offset 越界返回空页——rows=[]、total 保留、信封 200。
func TestGetConfig_OffsetBeyondTotalEmptyPage(t *testing.T) {
	h := newConfigHandlerWithFetch(func(ctx context.Context, ip, path string) (interface{}, error) {
		return map[string]interface{}{"interface": makeIfaceRows(66)}, nil
	})
	env := decodePage(t, getConfigPageReq(h, "10.0.0.1", "/p", "limit=10&offset=100"))
	assert.True(t, env.Success)
	assert.Equal(t, 66, env.Data.Data.Total)
	assert.NotNil(t, env.Data.Data.Rows)
	assert.Len(t, env.Data.Data.Rows, 0)
}

// Scenario: 分页切片来自缓存快照——第二页命中缓存不再 fetch。
func TestGetConfig_PaginationServedFromCache(t *testing.T) {
	calls := 0
	h := newConfigHandlerWithFetch(func(ctx context.Context, ip, path string) (interface{}, error) {
		calls++
		return map[string]interface{}{"interface": makeIfaceRows(66)}, nil
	})
	decodePage(t, getConfigPageReq(h, "10.0.0.1", "/p", "limit=10&offset=0"))
	env := decodePage(t, getConfigPageReq(h, "10.0.0.1", "/p", "limit=10&offset=10"))
	assert.Equal(t, 1, calls, "翻页必须切自缓存整树，不得重复打设备")
	assert.True(t, env.Data.Cached)
	assert.Equal(t, "cache", env.Data.Source)
	assert.Equal(t, "GE1/0/10", env.Data.Data.Rows[0]["name"])
}

// newStateHandler 注入计数 fetchState 的 handler。
func newStateHandler(fetchCalls *int, rows int) *ConfigHandler {
	h := NewConfigHandler(manager.New())
	h.fetchState = func(ctx context.Context, ip, path string) (interface{}, error) {
		*fetchCalls++
		return map[string]interface{}{"route": makeIfaceRows(rows)}, nil
	}
	return h
}

// BR-14 Scenario: 快照命中翻页不打设备 + 分页组合。
func TestGetConfig_StateSnapshotHitOnPaging(t *testing.T) {
	calls := 0
	h := newStateHandler(&calls, 30)
	d1 := decodePage(t, getConfigPageReq(h, "10.0.0.1", "/fib", "include_state=true&limit=10&offset=0"))
	assert.Equal(t, 1, calls)
	assert.False(t, d1.Data.Cached)
	assert.Equal(t, "device", d1.Data.Source)

	d2 := decodePage(t, getConfigPageReq(h, "10.0.0.1", "/fib", "include_state=true&limit=10&offset=10"))
	assert.Equal(t, 1, calls, "快照命中翻页不得再发 <get>")
	assert.True(t, d2.Data.Cached)
	assert.Equal(t, "cache", d2.Data.Source)
	assert.Equal(t, 30, d2.Data.Data.Total)
}

// BR-14 Scenario: force_refresh 绕快照直打设备并覆盖回填。
func TestGetConfig_StateForceRefreshBypassesSnapshot(t *testing.T) {
	calls := 0
	h := newStateHandler(&calls, 5)
	decodePage(t, getConfigPageReq(h, "10.0.0.1", "/fib", "include_state=true&limit=5"))
	d := decodePage(t, getConfigPageReq(h, "10.0.0.1", "/fib", "include_state=true&limit=5&force_refresh=true"))
	assert.Equal(t, 2, calls, "force_refresh 必须绕过快照")
	assert.False(t, d.Data.Cached)
	assert.Equal(t, "device", d.Data.Source)

	// 覆盖回填：force 后再次读取应命中新快照。
	decodePage(t, getConfigPageReq(h, "10.0.0.1", "/fib", "include_state=true&limit=5"))
	assert.Equal(t, 2, calls, "force_refresh 结果应回填快照供后续命中")
}

// BR-14 Scenario: 快照过期自动重拉（CC-07 独立 TTL 经环境变量配置）。
func TestGetConfig_StateSnapshotExpiryRefetch(t *testing.T) {
	t.Setenv("USMP_STATE_SNAPSHOT_TTL", "50ms")
	calls := 0
	h := newStateHandler(&calls, 5)
	decodePage(t, getConfigPageReq(h, "10.0.0.1", "/fib", "include_state=true&limit=5"))
	assert.Equal(t, 1, calls)
	time.Sleep(120 * time.Millisecond)
	decodePage(t, getConfigPageReq(h, "10.0.0.1", "/fib", "include_state=true&limit=5"))
	assert.Equal(t, 2, calls, "快照过期后必须自动重拉")
}

// BR-14 Scenario: 写操作失效运行配置缓存但不触及状态快照。
func TestGetConfig_WriteInvalidationSparesStateSnapshot(t *testing.T) {
	calls := 0
	h := newStateHandler(&calls, 5)
	decodePage(t, getConfigPageReq(h, "10.0.0.1", "/fib", "include_state=true&limit=5"))
	assert.Equal(t, 1, calls)

	// 模拟下发/删除后的运行缓存整设备失效（DeleteConfig/SetConfig 同款调用）。
	h.manager.GetRunningCache().InvalidatePrefix("10.0.0.1|")

	decodePage(t, getConfigPageReq(h, "10.0.0.1", "/fib", "include_state=true&limit=5"))
	assert.Equal(t, 1, calls, "运行缓存失效不得连坐状态快照（CC-07 隔离）")
}

// BR-14 无分页参数的 include_state 读也走快照（整树只读 Tab 同受益），
// 且响应形状保持整树。
func TestGetConfig_StateSnapshotWholeTreeRead(t *testing.T) {
	calls := 0
	h := newStateHandler(&calls, 3)
	getConfigPageReq(h, "10.0.0.1", "/fib", "include_state=true")
	w := getConfigPageReq(h, "10.0.0.1", "/fib", "include_state=true")
	assert.Equal(t, 1, calls, "整树状态读第二次应命中快照")
	var env struct {
		Data struct {
			Data   map[string]interface{} `json:"data"`
			Cached bool                   `json:"cached"`
		} `json:"data"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.True(t, env.Data.Cached)
	_, ok := env.Data.Data["route"].([]interface{})
	assert.True(t, ok, "无参状态读形状保持整树")
}
