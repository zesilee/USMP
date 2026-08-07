package api

// beego 路由等价性冒烟（replace-gin-with-beego design 风险项）：
// 在切换 handler 前先钉死 beego 路由树对本项目三类关键形态的实际行为，
// 任何一条红灯即说明 D4/D8 假设失效，禁止继续迁移。

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/beego/beego/v2/server/web"
	beecontext "github.com/beego/beego/v2/server/web/context"
)

func serveOnce(t *testing.T, reg *web.ControllerRegister, method, target string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, target, body)
	reg.ServeHTTP(w, req)
	return w
}

// 形态①：点分 IP 作路径参数段（/devices/:ip/status）
func TestBeegoRouterDottedIPParam(t *testing.T) {
	reg := web.NewControllerRegister()
	reg.Get("/api/v1/devices/:ip/status", func(ctx *beecontext.Context) {
		_ = ctx.Output.Body([]byte(ctx.Input.Param(":ip")))
	})

	w := serveOnce(t, reg, "GET", "/api/v1/devices/192.168.1.1/status", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := w.Body.String(); got != "192.168.1.1" {
		t.Fatalf(":ip = %q, want %q（点分段被路由树改写）", got, "192.168.1.1")
	}
}

// 形态②：通配尾段（/config/:ip/*）——:splat 无前导斜杠、多段、含点不被扩展名拆分
func TestBeegoRouterWildcardTail(t *testing.T) {
	cases := []struct {
		name   string
		target string
		splat  string
	}{
		{"多段尾巴", "/api/v1/config/192.168.1.1/ifm/interfaces", "ifm/interfaces"},
		{"含点尾段不拆扩展名", "/api/v1/config/1.2.3.4/vlan/vlans/vlan=1.5", "vlan/vlans/vlan=1.5"},
		{"单段尾巴", "/api/v1/config/10.0.0.1/system", "system"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reg := web.NewControllerRegister()
			reg.Get("/api/v1/config/:ip/*", func(ctx *beecontext.Context) {
				_ = ctx.Output.Body([]byte(ctx.Input.Param(":ip") + "|" + ctx.Input.Param(":splat")))
			})
			w := serveOnce(t, reg, "GET", tc.target, nil)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", w.Code)
			}
			wantIP := strings.SplitN(strings.TrimPrefix(tc.target, "/api/v1/config/"), "/", 2)[0]
			if got, want := w.Body.String(), wantIP+"|"+tc.splat; got != want {
				t.Fatalf("param = %q, want %q", got, want)
			}
		})
	}
}

// 形态③：静态段 changeset 与参数段 :ip/* 共存，静态优先（对齐 gin 1.10 现行为）
func TestBeegoRouterStaticBeatsParam(t *testing.T) {
	reg := web.NewControllerRegister()
	reg.Post("/api/v1/config/changeset/preview", func(ctx *beecontext.Context) {
		_ = ctx.Output.Body([]byte("static"))
	})
	reg.Post("/api/v1/config/:ip/*", func(ctx *beecontext.Context) {
		_ = ctx.Output.Body([]byte("param:" + ctx.Input.Param(":ip")))
	})

	if w := serveOnce(t, reg, "POST", "/api/v1/config/changeset/preview", strings.NewReader("{}")); w.Body.String() != "static" {
		t.Fatalf("changeset/preview 路由到 %q, want static（静态段被参数段吞掉）", w.Body.String())
	}
	if w := serveOnce(t, reg, "POST", "/api/v1/config/10.0.0.1/ifm/interfaces", strings.NewReader("{}")); w.Body.String() != "param:10.0.0.1" {
		t.Fatalf("参数路由到 %q, want param:10.0.0.1", w.Body.String())
	}
}

// 形态④：函数式路由下直读 Request.Body（D3 bindJSON 前提）与 query 取参
func TestBeegoRouterBodyAndQuery(t *testing.T) {
	reg := web.NewControllerRegister()
	reg.Post("/api/v1/echo", func(ctx *beecontext.Context) {
		b, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			ctx.Output.SetStatus(http.StatusInternalServerError)
			return
		}
		_ = ctx.Output.Body([]byte(ctx.Input.Query("mode") + "|" + string(b)))
	})

	w := serveOnce(t, reg, "POST", "/api/v1/echo?mode=fast", strings.NewReader(`{"a":1}`))
	if got, want := w.Body.String(), `fast|{"a":1}`; got != want {
		t.Fatalf("body/query = %q, want %q", got, want)
	}
}
