package api

import (
	"strings"
	"testing"

	"github.com/leezesi/usmp/backend/internal/yangschema"
)

// findFieldBySuffix DFS-walks a nested FieldDef tree and returns the first field
// whose Path ends with the given leaf-name suffix.
func findFieldBySuffix(fields []FieldDef, suffix string) (FieldDef, bool) {
	for _, f := range fields {
		if strings.HasSuffix(f.Path, "/"+suffix) || f.Path == suffix {
			return f, true
		}
		if len(f.Fields) > 0 {
			if got, ok := findFieldBySuffix(f.Fields, suffix); ok {
				return got, true
			}
		}
	}
	return FieldDef{}, false
}

// TestNestedSchemaCarriesWhenFromRealIFM verifies the schema endpoint surfaces the
// real IFM `when` XPath onto the generated FieldDef — 100% metadata-driven, no
// hardcoding. parent-name is gated by `when "../class='sub-interface'"`.
func TestNestedSchemaCarriesWhenFromRealIFM(t *testing.T) {
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	mod, ok := s.Module("ifm")
	if !ok {
		t.Fatal("ifm module not loaded")
	}
	ys := buildYangSchemaNested(mod)

	parent, ok := findFieldBySuffix(ys.Fields, "parent-name")
	if !ok {
		t.Fatal("parent-name field not found in nested IFM schema")
	}
	if parent.When != "../class='sub-interface'" {
		t.Errorf("parent-name When = %q, want %q", parent.When, "../class='sub-interface'")
	}

	// A leaf with no `when` (the key) must omit the field entirely (empty).
	name, ok := findFieldBySuffix(ys.Fields, "name")
	if ok && name.When != "" {
		t.Errorf("name When = %q, want empty", name.When)
	}
}
