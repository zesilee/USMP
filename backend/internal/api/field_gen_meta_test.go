package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/leezesi/usmp/backend/internal/yangschema"
)

// TestNestedSchemaCarriesDynamicDefaultFromRealIFM verifies the vendor
// `dynamic-default` extension surfaces as dynamicDefault=true (BR-10): the form
// then treats an empty value as "device decides", not "missing configuration".
func TestNestedSchemaCarriesDynamicDefaultFromRealIFM(t *testing.T) {
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	mod, ok := s.Module("ifm")
	if !ok {
		t.Fatal("ifm module not loaded")
	}
	ys := buildYangSchemaNested(mod)

	// admin-status carries a bare ext:dynamic-default in real huawei-ifm.
	admin, ok := findFieldBySuffix(ys.Fields, "interface/admin-status")
	if !ok {
		t.Fatal("interface/admin-status not found")
	}
	if !admin.DynamicDefault {
		t.Error("admin-status DynamicDefault = false, want true")
	}

	// description carries no extensions at all → no false positive.
	desc, ok := findFieldBySuffix(ys.Fields, "interface/description")
	if !ok {
		t.Fatal("interface/description not found")
	}
	if desc.DynamicDefault {
		t.Error("description DynamicDefault = true, want false")
	}
}

// TestNestedSchemaCarriesUnitsFromRealIFM verifies YANG `units` surfaces onto
// FieldDef (BR-09) so inputs can render a unit suffix.
func TestNestedSchemaCarriesUnitsFromRealIFM(t *testing.T) {
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	mod, _ := s.Module("ifm")
	ys := buildYangSchemaNested(mod)

	// dynamic/bandwidth is the one real units carrier in huawei-ifm ("bit/s").
	bw, ok := findFieldBySuffix(ys.Fields, "dynamic/bandwidth")
	if !ok {
		t.Fatal("dynamic/bandwidth not found")
	}
	if bw.Units != "bit/s" {
		t.Errorf("bandwidth Units = %q, want bit/s", bw.Units)
	}
}

// TestFieldDefOmitsMetaWhenAbsent verifies fields without dynamic-default/units
// serialize without those keys (omitempty — boundary scenario).
func TestFieldDefOmitsMetaWhenAbsent(t *testing.T) {
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	mod, _ := s.Module("ifm")
	ys := buildYangSchemaNested(mod)
	desc, ok := findFieldBySuffix(ys.Fields, "interface/description")
	if !ok {
		t.Fatal("interface/description not found")
	}
	b, err := json.Marshal(desc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, key := range []string{`"dynamicDefault"`, `"units"`} {
		if strings.Contains(string(b), key) {
			t.Errorf("description serializes %s unexpectedly: %s", key, b)
		}
	}
}
