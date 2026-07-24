package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/leezesi/usmp/backend/internal/api"
	"github.com/leezesi/usmp/backend/internal/yangschema"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// loadSchema loads the framework schema once for the tests, failing fast.
func loadSchema(t *testing.T) schema.Schema {
	t.Helper()
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("load schema: %v", err)
	}
	return s
}

// loadedModuleNames returns the set of module names the exporter is expected to
// cover: every loaded module with a non-nil root (the exact set ListModules
// exposes over HTTP). Derived dynamically from the schema — no hardcoded list.
func loadedModuleNames(s schema.Schema) map[string]bool {
	want := map[string]bool{}
	for _, mod := range s.Modules() {
		if mod.Root() == nil {
			continue
		}
		want[mod.Name()] = true
	}
	return want
}

// SF-01: 导出覆盖全部已加载模块，且集合来自动态发现（模块名集合 == Modules() 全集）。
func TestExportAll_CoversEveryLoadedModule(t *testing.T) {
	s := loadSchema(t)
	want := loadedModuleNames(s)
	if len(want) == 0 {
		t.Fatal("no modules loaded — schema fixture would be vacuous")
	}

	fixtures, err := exportAll(s)
	if err != nil {
		t.Fatalf("exportAll: %v", err)
	}

	// 数量相等：导出数 == 已加载模块数（SF-01 硬不变式）。
	if len(fixtures) != len(want) {
		t.Fatalf("fixture count = %d, want %d (one per loaded module)", len(fixtures), len(want))
	}
	// 键集合逐一相等：既不漏模块、也不凭空多出（动态发现，不硬编码）。
	for name := range want {
		if _, ok := fixtures[name]; !ok {
			t.Errorf("missing fixture for loaded module %q", name)
		}
	}
	for name := range fixtures {
		if !want[name] {
			t.Errorf("fixture for module %q not in loaded set", name)
		}
	}
}

// SF-01: 每份 fixture 含 module/title/vendor 标识与嵌套 fields 树，且 module 字段
// 与文件键一致（自描述，防止串号）。
func TestExportAll_FixtureSelfConsistent(t *testing.T) {
	s := loadSchema(t)
	fixtures, err := exportAll(s)
	if err != nil {
		t.Fatalf("exportAll: %v", err)
	}

	for name, raw := range fixtures {
		var ys api.YangSchema
		if err := json.Unmarshal(raw, &ys); err != nil {
			t.Errorf("module %q: fixture is not valid YangSchema JSON: %v", name, err)
			continue
		}
		if ys.Module != name {
			t.Errorf("module %q: fixture .module = %q (should match file key)", name, ys.Module)
		}
		if ys.Title == "" {
			t.Errorf("module %q: fixture missing title", name)
		}
		if ys.Vendor == "" {
			t.Errorf("module %q: fixture missing vendor", name)
		}
	}
}

// run() 写盘管线：落盘全部 fixture、清除陈旧 *.json、文件内容 == exportAll。
func TestRun_WritesAllAndClearsStale(t *testing.T) {
	s := loadSchema(t)
	dir := t.TempDir()

	// 预置一份陈旧 fixture 与一个非 json 文件——前者应被清、后者应保留。
	stale := filepath.Join(dir, "__removed_module__.json")
	keep := filepath.Join(dir, "README.md")
	if err := os.WriteFile(stale, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keep, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}

	n, err := run(s, dir)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	want := loadedModuleNames(s)
	if n != len(want) {
		t.Fatalf("run wrote %d fixtures, want %d", n, len(want))
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale *.json not removed (dir must be a pure function of schema)")
	}
	if _, err := os.Stat(keep); err != nil {
		t.Errorf("non-json file README.md must be preserved: %v", err)
	}

	// 落盘内容与内存导出逐字节一致（run 不引入额外变换）。
	fixtures, err := exportAll(s)
	if err != nil {
		t.Fatalf("exportAll: %v", err)
	}
	for name := range want {
		got, err := os.ReadFile(filepath.Join(dir, name+".json"))
		if err != nil {
			t.Errorf("missing fixture file %s.json: %v", name, err)
			continue
		}
		if string(got) != string(fixtures[name]) {
			t.Errorf("module %q: on-disk fixture != exportAll output", name)
		}
	}
}

// run() 到不可创建的输出路径应返回错误（负路径，不 panic）。
func TestRun_MkdirError(t *testing.T) {
	s := loadSchema(t)
	// 用一个已存在的普通文件当「目录」——MkdirAll 应失败。
	f := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := run(s, filepath.Join(f, "sub")); err == nil {
		t.Error("run should error when output dir cannot be created")
	}
}

// SF-02: 确定性——同一 schema 连续两次导出逐字节相同（每模块）。
func TestExportAll_Deterministic(t *testing.T) {
	s := loadSchema(t)

	first, err := exportAll(s)
	if err != nil {
		t.Fatalf("exportAll #1: %v", err)
	}
	second, err := exportAll(s)
	if err != nil {
		t.Fatalf("exportAll #2: %v", err)
	}

	if len(first) != len(second) {
		t.Fatalf("run1 has %d fixtures, run2 has %d", len(first), len(second))
	}
	for name, a := range first {
		b, ok := second[name]
		if !ok {
			t.Errorf("module %q present in run1 but missing in run2", name)
			continue
		}
		if string(a) != string(b) {
			t.Errorf("module %q: non-deterministic export (run1 != run2, %d vs %d bytes)", name, len(a), len(b))
		}
	}
}
