package staticweb

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// 构造一个模拟前端 dist 的目录：index.html + 嵌套静态资源 + 根外的“越权”文件。
func setupWebRoot(t *testing.T) string {
	t.Helper()
	parent := t.TempDir()
	root := filepath.Join(parent, "web")
	if err := os.MkdirAll(filepath.Join(root, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(root, "index.html"):       "<html>usmp-index</html>",
		filepath.Join(root, "assets", "app.js"): "console.log('usmp')",
		filepath.Join(parent, "secret.txt"):     "top-secret",
	}
	for p, content := range files {
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestHandler(t *testing.T) {
	root := setupWebRoot(t)
	h := Handler(root)

	tests := []struct {
		name     string
		path     string
		wantCode int
		wantBody string // 子串匹配
	}{
		{"根路径回 index", "/", http.StatusOK, "usmp-index"},
		{"存在的静态资源直接返回", "/assets/app.js", http.StatusOK, "console.log('usmp')"},
		{"SPA 路由 fallback 到 index", "/module/huawei-vlan", http.StatusOK, "usmp-index"},
		{"深层 SPA 路由 fallback", "/module/huawei-ifm/rpc/reset", http.StatusOK, "usmp-index"},
		{"不存在的资源也 fallback（对齐 nginx try_files）", "/assets/gone.js", http.StatusOK, "usmp-index"},
		{"健康检查", "/healthz", http.StatusOK, "healthy"},
		// ServeMux 会先把 /../secret.txt 307 到清洗后的 /secret.txt（这就是防穿越），
		// 跟随重定向后应落到 SPA fallback，绝不能拿到根目录外的文件。
		{"路径穿越不能逃出根目录", "/../secret.txt", http.StatusOK, "usmp-index"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if loc := rec.Header().Get("Location"); rec.Code >= 300 && rec.Code < 400 && loc != "" {
				req = httptest.NewRequest(http.MethodGet, loc, nil)
				rec = httptest.NewRecorder()
				h.ServeHTTP(rec, req)
			}
			if rec.Code != tt.wantCode {
				t.Fatalf("状态码 = %d, want %d", rec.Code, tt.wantCode)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Fatalf("响应体 %q 不含 %q", rec.Body.String(), tt.wantBody)
			}
			if strings.Contains(rec.Body.String(), "top-secret") {
				t.Fatalf("路径穿越泄漏了根目录外的文件")
			}
		})
	}
}

// 并发请求安全（-race 下验证无数据竞态，R09）。
func TestHandlerConcurrent(t *testing.T) {
	root := setupWebRoot(t)
	h := Handler(root)

	var wg sync.WaitGroup
	paths := []string{"/", "/assets/app.js", "/module/x", "/healthz"}
	for i := 0; i < 20; i++ {
		for _, p := range paths {
			wg.Add(1)
			go func(p string) {
				defer wg.Done()
				req := httptest.NewRequest(http.MethodGet, p, nil)
				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, req)
				if rec.Code != http.StatusOK {
					t.Errorf("%s 状态码 = %d", p, rec.Code)
				}
			}(p)
		}
	}
	wg.Wait()
}
