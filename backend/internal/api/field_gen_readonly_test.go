package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/leezesi/usmp/backend/internal/yangschema"
)

// TestNestedSchemaCarriesReadonlyFromRealIFM verifies `config false` state
// subtrees surface readonly=true on FieldDef — container, nested list and leaf
// alike (BR-09), so the console degrades them to read-only views instead of
// rendering editable tabs (presentation-noise debt).
func TestNestedSchemaCarriesReadonlyFromRealIFM(t *testing.T) {
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	mod, ok := s.Module("ifm")
	if !ok {
		t.Fatal("ifm module not loaded")
	}
	ys := buildYangSchemaNested(mod)

	// remote-interfaces is a real `config false` subtree in huawei-ifm.
	remote, ok := findFieldBySuffix(ys.Fields, "remote-interfaces")
	if !ok {
		t.Fatal("remote-interfaces not found")
	}
	if !remote.Readonly {
		t.Error("remote-interfaces Readonly = false, want true")
	}
	// The whole subtree inherits: nested list and its leaves.
	if len(remote.Fields) == 0 {
		t.Fatal("remote-interfaces has no children")
	}
	var assertSubtreeRO func(fs []FieldDef)
	assertSubtreeRO = func(fs []FieldDef) {
		for _, f := range fs {
			if !f.Readonly {
				t.Errorf("%s Readonly = false, want true (inherited)", f.Path)
			}
			assertSubtreeRO(f.Fields)
		}
	}
	assertSubtreeRO(remote.Fields)

	// Editable config nodes must not carry readonly.
	ifaces, ok := findFieldBySuffix(ys.Fields, "interfaces")
	if !ok {
		t.Fatal("interfaces not found")
	}
	if ifaces.Readonly {
		t.Error("interfaces Readonly = true, want false")
	}
	class, ok := findFieldBySuffix(ys.Fields, "class")
	if !ok {
		t.Fatal("class not found")
	}
	if class.Readonly {
		t.Error("class Readonly = true, want false")
	}
}

// TestFieldDefOmitsReadonlyWhenFalse verifies config-true fields serialize
// without a readonly key (omitempty — BR-09 boundary scenario).
func TestFieldDefOmitsReadonlyWhenFalse(t *testing.T) {
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	mod, _ := s.Module("ifm")
	ys := buildYangSchemaNested(mod)
	class, ok := findFieldBySuffix(ys.Fields, "class")
	if !ok {
		t.Fatal("class not found")
	}
	b, err := json.Marshal(class)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(b), `"readonly"`) {
		t.Errorf("config-true leaf serializes readonly key: %s", b)
	}
}
