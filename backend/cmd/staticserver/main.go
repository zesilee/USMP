// staticserver — 发布包内的前端静态站服务进程。
//
// 由发布包 start.sh 拉起，托管 web/（前端 dist），行为对齐 nginx try_files。
// 配置走环境变量：USMP_WEB_ROOT（站点根目录，默认 ./web）、USMP_WEB_PORT（默认 3002）。
package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/leezesi/usmp/backend/internal/staticweb"
)

func main() {
	root := os.Getenv("USMP_WEB_ROOT")
	if root == "" {
		root = "./web"
	}
	port := os.Getenv("USMP_WEB_PORT")
	if port == "" {
		port = "3002"
	}

	// 启动即校验站点根目录，配置错误立即报错退出，不带病监听（R08 快速失败）。
	if _, err := os.Stat(filepath.Join(root, "index.html")); err != nil {
		log.Fatalf("staticserver: 站点根目录不可用（缺 index.html）: %s: %v", root, err)
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           staticweb.Handler(root),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("staticserver: 前端静态站启动 :%s（root=%s）", port, root)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("staticserver: %v", err)
	}
}
