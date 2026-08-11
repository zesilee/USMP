package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// TestBuildIRDecodable：产物必须可解码、非空、含既有关键模块（vlan/ifm 是最早
// 交付的两个模块，任何丢模块的回归都会在此显形）。
func TestBuildIRDecodable(t *testing.T) {
	blob, err := BuildIR()
	if err != nil {
		t.Fatalf("BuildIR: %v", err)
	}
	ds, err := schema.DecodeIR(blob)
	if err != nil {
		t.Fatalf("DecodeIR(BuildIR()): %v", err)
	}
	if len(ds.Modules()) == 0 {
		t.Fatal("decoded IR has no modules")
	}
	for _, name := range []string{"vlan", "ifm"} {
		m, ok := ds.Module(name)
		if !ok {
			t.Fatalf("module %s missing from IR", name)
		}
		if m.Vendor() != "huawei" {
			t.Fatalf("module %s vendor = %q, want huawei", name, m.Vendor())
		}
	}
}

// TestBuildIRDeterministic：两次构建字节一致（CG-01 可复现契约在 IR 产物上的体现）。
func TestBuildIRDeterministic(t *testing.T) {
	a, err := BuildIR()
	if err != nil {
		t.Fatal(err)
	}
	b, err := BuildIR()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("BuildIR is not deterministic")
	}
}

func TestRunWritesFile(t *testing.T) {
	out := filepath.Join(t.TempDir(), "schema.ir.gz")
	if err := Run(out); err != nil {
		t.Fatalf("Run: %v", err)
	}
	blob, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := schema.DecodeIR(blob); err != nil {
		t.Fatalf("written file not decodable: %v", err)
	}
}

func TestRunBadPath(t *testing.T) {
	if err := Run(filepath.Join(t.TempDir(), "no-such-dir", "x.gz")); err == nil {
		t.Fatal("Run to non-existent dir should error")
	}
}
