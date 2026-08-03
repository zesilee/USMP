// Package staticweb 提供发布包内前端静态站的极简 HTTP 处理器。
//
// 行为对齐 frontend/nginx.conf 的 try_files $uri /index.html：
// 命中真实文件直接返回，其余一律回 index.html（Vue history 路由 SPA fallback），
// /healthz 返回 200 供容器健康检查。零第三方依赖（R10），发布包解压即用，
// 目标镜像无需预装 nginx。
package staticweb

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
)

// Handler 返回以 root 为站点根目录的静态文件处理器。
func Handler(root string) http.Handler {
	fileServer := http.FileServer(http.Dir(root))
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("healthy\n"))
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// path.Clean 锚定到 "/" 前缀，消解 ".."，穿越请求无法逃出 root。
		cleaned := path.Clean("/" + r.URL.Path)
		full := filepath.Join(root, filepath.FromSlash(cleaned))
		if info, err := os.Stat(full); err == nil && !info.IsDir() {
			r.URL.Path = cleaned
			fileServer.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(root, "index.html"))
	})

	return mux
}
