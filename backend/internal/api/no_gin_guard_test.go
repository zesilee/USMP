package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 依赖守护（replace-gin-with-beego）：开源选型拍板后端 Web 框架统一为
// beego/v2，go.mod/go.sum 不得再出现 gin（含 gin-contrib 中间件与间接依赖）。
// 任何人重新引入在本地与 CI 都会红。对齐 no_scrapligo_guard_test 模式。
func TestNoGinDependencyGuard(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"go.mod", "go.sum"} {
		data, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			t.Fatalf("读 %s: %v", f, err)
		}
		for _, banned := range []string{"github.com/gin-gonic/gin", "github.com/gin-contrib/"} {
			if strings.Contains(string(data), banned) {
				t.Fatalf("%s 含 %s——后端 Web 框架已拍板 beego/v2（replace-gin-with-beego），"+
					"HTTP 层请用 web.ControllerRegister + beecontext", f, banned)
			}
		}
	}
}
