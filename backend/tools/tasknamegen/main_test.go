package main

import (
	"strings"
	"testing"
)

// extractTaskNames maps each top-level data container of the listed modules to
// the module-level vendor `task-name` extension value (BR-01 category). Keys are
// root container names — the same keys /yang/modules serves as module `name`.
func TestExtractTaskNames(t *testing.T) {
	got, err := extractTaskNames("testdata", []string{"demo-a", "demo-b"})
	if err != nil {
		t.Fatalf("extractTaskNames: %v", err)
	}
	want := map[string]string{"widgets": "group-a", "gadgets": "group-a"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
	// demo-b has no task-name → its containers must be absent, not empty-valued.
	if _, ok := got["things"]; ok {
		t.Error("things present, want absent (module without task-name)")
	}
	// RPC nodes are not data containers → never mapped (BR-01 keys mirror /yang/modules).
	if _, ok := got["reset-widgets"]; ok {
		t.Error("reset-widgets present, want absent (rpc, not a data container)")
	}
}

func TestExtractTaskNamesUnknownModule(t *testing.T) {
	if _, err := extractTaskNames("testdata", []string{"no-such-module"}); err == nil {
		t.Error("err = nil, want parse error for unknown module")
	}
}

func TestRenderSource(t *testing.T) {
	src, err := renderSource("huawei", "TaskNames", map[string]string{"ifm": "interface-mgr", "vlan": "vlan"})
	if err != nil {
		t.Fatalf("renderSource: %v", err)
	}
	for _, want := range []string{"package huawei", `"ifm":`, `"interface-mgr"`, "TaskNames"} {
		if !strings.Contains(src, want) {
			t.Errorf("rendered source missing %q:\n%s", want, src)
		}
	}
	// Deterministic output: map iteration order must not leak into the file.
	src2, _ := renderSource("huawei", "TaskNames", map[string]string{"vlan": "vlan", "ifm": "interface-mgr"})
	if src != src2 {
		t.Error("renderSource is not deterministic across map orderings")
	}
}
