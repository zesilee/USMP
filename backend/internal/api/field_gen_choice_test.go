package api

import (
	"strings"
	"testing"

	"github.com/leezesi/usmp/backend/internal/yangschema"
)

// findChoiceField DFS-walks a nested FieldDef tree (descending into both .Fields
// and .Cases[].Fields) for the first type=="choice" field with the given label.
func findChoiceField(fields []FieldDef, label string) (FieldDef, bool) {
	for _, f := range fields {
		if f.Type == "choice" && f.Label == label {
			return f, true
		}
		if got, ok := findChoiceField(f.Fields, label); ok {
			return got, true
		}
		for _, c := range f.Cases {
			if got, ok := findChoiceField(c.Fields, label); ok {
				return got, true
			}
		}
	}
	return FieldDef{}, false
}

// TestNestedSchemaSurfacesChoiceFromRealIFM verifies the nested schema endpoint
// exposes the real IFM `choice bandwidth-type` as type="choice" with its two cases,
// and that member leaf paths are FLAT (no choice/case segment) so the NETCONF write
// path is unaffected (BR-06). 100% metadata-driven, no hardcoding.
func TestNestedSchemaSurfacesChoiceFromRealIFM(t *testing.T) {
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	mod, ok := s.Module("ifm")
	if !ok {
		t.Fatal("ifm module not loaded")
	}
	ys := buildYangSchemaNested(mod)

	ch, ok := findChoiceField(ys.Fields, "bandwidth-type")
	if !ok {
		t.Fatal("bandwidth-type choice field not found in nested IFM schema")
	}
	if ch.Type != "choice" {
		t.Errorf("bandwidth-type Type = %q, want choice", ch.Type)
	}
	if len(ch.Cases) != 2 {
		t.Fatalf("bandwidth-type has %d cases, want 2", len(ch.Cases))
	}

	// Locate case bandwidth-mbps and assert its leaf has a flat data path.
	var mbps *CaseDef
	for i := range ch.Cases {
		if ch.Cases[i].Name == "bandwidth-mbps" {
			mbps = &ch.Cases[i]
		}
	}
	if mbps == nil {
		t.Fatal("case bandwidth-mbps missing")
	}
	if len(mbps.Fields) != 1 {
		t.Fatalf("case bandwidth-mbps has %d fields, want 1", len(mbps.Fields))
	}
	bw := mbps.Fields[0]
	if bw.Path != "/ifm/interfaces/interface/bandwidth" {
		t.Errorf("bandwidth Path = %q, want flat /ifm/interfaces/interface/bandwidth", bw.Path)
	}
	if strings.Contains(bw.Path, "bandwidth-type") || strings.Contains(bw.Path, "bandwidth-mbps") {
		t.Errorf("bandwidth Path %q must not contain choice/case segments", bw.Path)
	}
}

// TestFlatSchemaRecursesChoiceMembers verifies the flat (collectFields) form still
// lists choice member leaves so consumers of the flat schema don't lose fields.
func TestFlatSchemaRecursesChoiceMembers(t *testing.T) {
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	mod, _ := s.Module("ifm")
	ys := buildYangSchema(mod) // flat form

	found := false
	for _, f := range ys.Fields {
		if strings.HasSuffix(f.Path, "/bandwidth") {
			found = true
		}
	}
	if !found {
		t.Error("flat schema lost choice member leaf `bandwidth`")
	}
}
