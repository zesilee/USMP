package api

import (
	"strings"
	"testing"

	"github.com/leezesi/usmp/backend/internal/yangschema"
)

// TestNestedSchemaCarriesPatternAndRangeFromRealIFM verifies the schema endpoint
// surfaces real IFM `pattern` and *explicit* integer `range` onto FieldDef —
// data-driven, and NOT the type's full default bounds.
//   - interface `number` leaf: string with an inline pattern.
//   - interface `statistic-interval` leaf: uint32 range "10..600" (unique name).
func TestNestedSchemaCarriesPatternAndRangeFromRealIFM(t *testing.T) {
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	mod, ok := s.Module("ifm")
	if !ok {
		t.Fatal("ifm module not loaded")
	}
	ys := buildYangSchemaNested(mod)

	num, ok := findFieldBySuffix(ys.Fields, "number")
	if !ok {
		t.Fatal("number field not found in nested IFM schema")
	}
	if num.Pattern == "" || !strings.Contains(num.Pattern, `\d`) {
		t.Errorf("number Pattern = %q, want a non-empty regex containing \\d", num.Pattern)
	}

	si, ok := findFieldBySuffix(ys.Fields, "statistic-interval")
	if !ok {
		t.Fatal("statistic-interval field not found in nested IFM schema")
	}
	if si.Minimum != 10 || si.Maximum != 600 {
		t.Errorf("statistic-interval range = [%d,%d], want [10,600]", si.Minimum, si.Maximum)
	}
}
