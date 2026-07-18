package yangschema

import (
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// TestLoadHasVendorModules verifies Load builds a non-empty schema tree with the
// expected huawei modules (BR-11: huawei + usmp business only).
func TestLoadHasVendorModules(t *testing.T) {
	s, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(s.Modules()) == 0 {
		t.Fatal("schema tree is empty; expected huawei+business modules")
	}
	for _, name := range []string{"ifm", "system", "vlan"} {
		if _, ok := s.Module(name); !ok {
			t.Errorf("expected module %q to be loaded", name)
		}
	}
}

// TestLoadVendorBoundary asserts BR-11: the runtime schema tree carries only
// huawei vendor models and the usmp business intent model — openconfig/ietf
// models must not be loaded.
func TestLoadVendorBoundary(t *testing.T) {
	s, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, m := range s.Modules() {
		if v := m.Vendor(); v != "huawei" && v != "usmp" {
			t.Errorf("module %q vendor = %q, want huawei or usmp (BR-11)", m.Name(), v)
		}
	}
	for _, name := range []string{"interfaces", "vlans"} {
		if _, ok := s.Module(name); ok {
			t.Errorf("openconfig root container %q must not be loaded (BR-11)", name)
		}
	}
}

// TestLoadModulesHaveConfigurableAttributes verifies the loaded modules expose
// real configurable attributes (leaves), i.e. the tree carries YANG structure,
// not just empty roots.
func TestLoadModulesHaveConfigurableAttributes(t *testing.T) {
	s, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	total := 0
	for _, m := range s.Modules() {
		total += countLeaves(m.Root())
	}
	if total == 0 {
		t.Fatal("no leaf attributes found across modules; schema tree lacks structure")
	}
}

// countLeaves recursively counts LeafNode descendants of a node.
func countLeaves(n schema.Node) int {
	switch node := n.(type) {
	case schema.LeafNode:
		return 1
	case schema.ContainerNode:
		c := 0
		for _, ch := range node.Children() {
			c += countLeaves(ch)
		}
		return c
	case schema.ListNode:
		c := 0
		for _, ch := range node.Children() {
			c += countLeaves(ch)
		}
		return c
	default:
		return 0
	}
}
