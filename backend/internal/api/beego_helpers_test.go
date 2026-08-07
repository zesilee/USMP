package api

import (
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	beecontext "github.com/beego/beego/v2/server/web/context"
)

// newTestContext 造一个可直调 handler 的 beego 上下文（替代 gin.CreateTestContext）。
// params 为 key/value 平铺对，key 不带冒号（如 "ip", "1.2.3.4", "module", "vlan"）。
func newTestContext(method, target string, body io.Reader, params ...string) (*beecontext.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, target, body)
	ctx := beecontext.NewContext()
	ctx.Reset(w, req)
	for i := 0; i+1 < len(params); i += 2 {
		ctx.Input.SetParam(":"+params[i], params[i+1])
	}
	return ctx, w
}

func TestWildcardPath(t *testing.T) {
	cases := []struct {
		name  string
		splat string
		want  string
	}{
		{"多段", "ifm/interfaces", "/ifm/interfaces"},
		{"单段", "system", "/system"},
		{"空尾巴", "", "/"},
		{"含点与键值", "vlan/vlans/vlan=1.5", "/vlan/vlans/vlan=1.5"},
		{"已带斜杠不重复加", "/already/rooted", "/already/rooted"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := newTestContext("GET", "/api/v1/config/1.2.3.4/x", nil, "splat", tc.splat)
			if got := wildcardPath(ctx); got != tc.want {
				t.Fatalf("wildcardPath(%q) = %q, want %q", tc.splat, got, tc.want)
			}
		})
	}
}

func TestBindJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}
	cases := []struct {
		name    string
		body    string
		nilBody bool
		wantErr bool
		want    payload
	}{
		{name: "正常对象", body: `{"name":"vlan10","n":10}`, want: payload{Name: "vlan10", N: 10}},
		{name: "空体报错", body: "", wantErr: true},
		{name: "坏JSON报错", body: `{"name":`, wantErr: true},
		{name: "nil体报错", nilBody: true, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var body io.Reader
			if !tc.nilBody {
				body = strings.NewReader(tc.body)
			}
			ctx, _ := newTestContext("POST", "/api/v1/echo", body)
			if tc.nilBody {
				ctx.Request.Body = nil
			}
			var got payload
			err := bindJSON(ctx, &got)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("bindJSON(%q) err = nil, want error", tc.body)
				}
				return
			}
			if err != nil {
				t.Fatalf("bindJSON(%q) err = %v", tc.body, err)
			}
			if got != tc.want {
				t.Fatalf("bindJSON(%q) = %+v, want %+v", tc.body, got, tc.want)
			}
		})
	}
}

// 并发安全：各 goroutine 各自上下文互不串扰（对齐 B1 层 race 要求）
func TestHelpersConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			splat := "mod/path" + string(rune('a'+n%26))
			ctx, _ := newTestContext("GET", "/x", nil, "splat", splat)
			if got := wildcardPath(ctx); got != "/"+splat {
				t.Errorf("goroutine %d: wildcardPath = %q, want %q", n, got, "/"+splat)
			}
		}(i)
	}
	wg.Wait()
}
