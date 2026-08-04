package client

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 交付红线守护（NC-01）：版本交付编译不得依赖 scrapligo（2026-08-04 交付要求，
// 自研 netconfcore 已全量替代）。本测试断言 go.mod 无 scrapligo 残留——
// 任何人重新引入（含间接依赖）在本地与 CI 都会红。回退方案不存在：
// 旧引擎已删除，问题只能在 netconfcore 修（openspec/tasks/netconf-client-selfdev.md）。
func TestNoScrapligoDependencyGuard(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"go.mod", "go.sum"} {
		data, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			t.Fatalf("读 %s: %v", f, err)
		}
		if strings.Contains(string(data), "scrapli") {
			t.Fatalf("%s 含 scrapligo 依赖——违反交付红线 NC-01（编译不得依赖 scrapligo），"+
				"请改用 pkg/yang-runtime/client/netconfcore", f)
		}
	}
}
