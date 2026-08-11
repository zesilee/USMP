package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// emitFixture renders the fixture model（单文件模式——gentest golden 包）。
func emitFixture(t *testing.T) map[string][]byte {
	t.Helper()
	m := buildFixtureModel(t)
	m.Package = "gentest"
	files, err := EmitPackage(m, []string{"usmp-test", "usmp-test-ext"}, 0)
	if err != nil {
		t.Fatalf("EmitPackage: %v", err)
	}
	return files
}

// TestEmitMatchesGentestGolden：入库 golden 包（tools/yanggen/gentest，随仓库
// 编译）与 emit 输出逐字节一致——兼作「改生成器忘了重跑」新鲜度门禁。
// 重新生成：UPDATE_GENTEST=1 go test ./tools/yanggen/ -run TestEmitMatchesGentestGolden
func TestEmitMatchesGentestGolden(t *testing.T) {
	files := emitFixture(t)
	if os.Getenv("UPDATE_GENTEST") == "1" {
		if err := os.MkdirAll("gentest", 0o755); err != nil {
			t.Fatal(err)
		}
		for name, src := range files {
			if err := os.WriteFile(filepath.Join("gentest", name), src, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		t.Log("gentest golden updated")
	}
	for name, want := range files {
		got, err := os.ReadFile(filepath.Join("gentest", name))
		if err != nil {
			t.Fatalf("golden %s: %v（生成器变更后运行 UPDATE_GENTEST=1 go test 刷新）", name, err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("golden %s 与 emit 输出漂移（UPDATE_GENTEST=1 刷新并 review diff）", name)
		}
	}
}

// TestEmitSplitDeterministic：拆分模式两次渲染字节一致、分桶均匀。
func TestEmitSplitDeterministic(t *testing.T) {
	m := buildFixtureModel(t)
	a, err := EmitPackage(m, []string{"usmp-test"}, 3)
	if err != nil {
		t.Fatal(err)
	}
	b, err := EmitPackage(m, []string{"usmp-test"}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != len(b) {
		t.Fatalf("file sets differ: %d vs %d", len(a), len(b))
	}
	for name := range a {
		if !bytes.Equal(a[name], b[name]) {
			t.Fatalf("%s not deterministic", name)
		}
	}
	for _, name := range []string{"structs-0.go", "structs-1.go", "structs-2.go", "registry.go", "enum.go", "enum_map.go", "union.go"} {
		if _, ok := a[name]; !ok {
			t.Fatalf("split layout missing %s (have %v)", name, keysOf(a))
		}
	}
}

func keysOf(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
