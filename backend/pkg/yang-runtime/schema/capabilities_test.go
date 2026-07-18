package schema

import (
	"sort"
	"testing"
)

// modsFixture builds a small module set with namespaces for narrowing tests.
func modsFixture() []Module {
	return []Module{
		NewModule("vlan", "urn:huawei:yang:huawei-vlan", "", NewContainer("vlan", "", "/vlan", nil, false)),
		NewModule("ifm", "urn:huawei:yang:huawei-ifm", "", NewContainer("ifm", "", "/ifm", nil, false)),
		NewModule("interfaces", "http://example.com/yang/interfaces", "", NewContainer("interfaces", "", "/interfaces", nil, false)),
	}
}

func names(mods []Module) []string {
	out := make([]string, 0, len(mods))
	for _, m := range mods {
		out = append(out, m.Name())
	}
	sort.Strings(out)
	return out
}

// TestNarrowFallbackWhenNoModuleCaps: only base caps → all modules (model tree authoritative).
func TestNarrowFallbackWhenNoModuleCaps(t *testing.T) {
	caps := []string{
		"urn:ietf:params:netconf:base:1.0",
		"urn:ietf:params:netconf:capability:candidate:1.0",
		"urn:ietf:params:netconf:capability:writable-running:1.0",
	}
	got := NarrowModulesByCapabilities(caps, modsFixture())
	if len(got) != 3 {
		t.Fatalf("no module caps → expected all 3 modules, got %v", names(got))
	}
}

// TestNarrowByNamespace: a module cap matching by namespace narrows to that module.
func TestNarrowByNamespace(t *testing.T) {
	caps := []string{
		"urn:ietf:params:netconf:base:1.0",
		"urn:huawei:yang:huawei-vlan?module=huawei-vlan&revision=2021-01-01",
	}
	got := names(NarrowModulesByCapabilities(caps, modsFixture()))
	if len(got) != 1 || got[0] != "vlan" {
		t.Fatalf("expected only [vlan], got %v", got)
	}
}

// TestNarrowByModuleParamAndSubstring: URL 形态命名空间 cap narrows to interfaces.
func TestNarrowByModuleParam(t *testing.T) {
	caps := []string{
		"http://example.com/yang/interfaces?module=example-interfaces&revision=2022-01-01",
	}
	got := names(NarrowModulesByCapabilities(caps, modsFixture()))
	if len(got) != 1 || got[0] != "interfaces" {
		t.Fatalf("expected only [interfaces], got %v", got)
	}
}

// TestNarrowMultipleCaps: two module caps narrow to two modules.
func TestNarrowMultipleCaps(t *testing.T) {
	caps := []string{
		"urn:huawei:yang:huawei-vlan?module=huawei-vlan",
		"urn:huawei:yang:huawei-ifm?module=huawei-ifm",
	}
	got := names(NarrowModulesByCapabilities(caps, modsFixture()))
	want := []string{"ifm", "vlan"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

// TestNarrowEmptyCaps: nil caps → all modules (fallback).
func TestNarrowEmptyCaps(t *testing.T) {
	if got := NarrowModulesByCapabilities(nil, modsFixture()); len(got) != 3 {
		t.Fatalf("nil caps → all modules, got %v", names(got))
	}
}
