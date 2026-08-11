package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
)

func loadFixture(t *testing.T) (map[string]*yang.Entry, map[string]*yang.Module) {
	t.Helper()
	entries, mods, err := LoadEntries([]string{"testdata/yang"}, []string{"usmp-test", "usmp-test-ext"})
	if err != nil {
		t.Fatalf("LoadEntries: %v", err)
	}
	return entries, mods
}

func TestLoadEntriesFixture(t *testing.T) {
	entries, mods := loadFixture(t)
	root := entries["usmp-test"]
	if root == nil {
		t.Fatal("usmp-test entry missing")
	}
	box := root.Dir["box"]
	if box == nil {
		t.Fatal("box container missing")
	}
	// augment 已并入宿主树
	if box.Dir["extras"] == nil {
		t.Fatal("augmented container extras missing from host tree")
	}
	// belonging module 判定数据可用
	if mods["usmp-test-ext"].Namespace.Name != "urn:usmp:test-ext" {
		t.Fatalf("ext namespace = %v", mods["usmp-test-ext"].Namespace)
	}
	// choice/case 结构存在（模型层拍平在 model.go 做）
	if box.Dir["source"] == nil || !box.Dir["source"].IsChoice() {
		t.Fatal("choice source missing")
	}
	if box.Dir["routes"].Dir["route"].Key != "vrf prefix" {
		t.Fatalf("composite key = %q", box.Dir["routes"].Dir["route"].Key)
	}
}

func TestParseGenConf(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "gen.conf")
	writeConf := func(s string) {
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeConf("# c\nyang_path=a,b\nmodules=m1 m2\ngenerate_fakeroot=true\ncompress_paths=false\nsplit_count=8\n")
	conf, err := ParseGenConf(p)
	if err != nil {
		t.Fatalf("ParseGenConf: %v", err)
	}
	if len(conf.YangPaths) != 2 || len(conf.Modules) != 2 || !conf.FakeRoot || conf.SplitCount != 8 {
		t.Fatalf("conf = %+v", conf)
	}

	writeConf("yang_path=a\nmodules=m\nbogus_key=1\n")
	if _, err := ParseGenConf(p); err == nil {
		t.Fatal("unknown key must error")
	}
	writeConf("yang_path=a\nmodules=m\ncompress_paths=true\n")
	if _, err := ParseGenConf(p); err == nil {
		t.Fatal("compress_paths=true must error (约定冻结非压缩)")
	}
	writeConf("modules=m\n")
	if _, err := ParseGenConf(p); err == nil {
		t.Fatal("missing yang_path must error")
	}
}

func TestLoadEntriesMissingModule(t *testing.T) {
	if _, _, err := LoadEntries([]string{"testdata/yang"}, []string{"no-such-module"}); err == nil {
		t.Fatal("missing module must error")
	}
}
