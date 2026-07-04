package api

import (
	"testing"

	"github.com/leezesi/usmp/backend/internal/yangschema"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

func TestFieldTypeMapping(t *testing.T) {
	cases := map[schema.LeafType]string{
		schema.LeafTypeBoolean:   "boolean",
		schema.LeafTypeUint16:    "number",
		schema.LeafTypeInt32:     "number",
		schema.LeafTypeDecimal64: "number",
		schema.LeafTypeEnum:      "enum",
		schema.LeafTypeString:    "string",
		schema.LeafTypeBits:      "string",
	}
	for in, want := range cases {
		if got := fieldType(in); got != want {
			t.Errorf("fieldType(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestVendorForNamespace(t *testing.T) {
	cases := map[string]string{
		"urn:huawei:yang:huawei-vlan":           "huawei",
		"http://openconfig.net/yang/interfaces": "openconfig",
		"urn:example:other":                     "unknown",
	}
	for ns, want := range cases {
		if got := vendorForNamespace(ns); got != want {
			t.Errorf("vendorForNamespace(%q) = %q, want %q", ns, got, want)
		}
	}
}

// TestBuildYangSchemaFromRealModule verifies the generator produces a non-stub,
// attribute-covering schema from a real loaded YANG module (huawei vlan).
func TestBuildYangSchemaFromRealModule(t *testing.T) {
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	mod, ok := s.Module("vlan")
	if !ok {
		t.Fatal("vlan module not loaded")
	}

	ys := buildYangSchema(mod)
	if ys.Module != "vlan" {
		t.Errorf("Module = %q, want vlan", ys.Module)
	}
	if ys.Vendor != "huawei" {
		t.Errorf("Vendor = %q, want huawei", ys.Vendor)
	}
	if len(ys.Fields) == 0 {
		t.Fatal("generated schema has no fields (stub, not dynamic)")
	}
	// vlan list is keyed by a numeric id → expect a number field and a list column.
	hasNumber := false
	for _, f := range ys.Fields {
		if f.Type == "number" {
			hasNumber = true
		}
		if f.Label == "" || f.Path == "" {
			t.Errorf("field missing label/path: %+v", f)
		}
	}
	if !hasNumber {
		t.Error("expected at least one numeric field (vlan id)")
	}
	if len(ys.ListCols) == 0 {
		t.Error("expected list columns from vlan list keys")
	}
}
