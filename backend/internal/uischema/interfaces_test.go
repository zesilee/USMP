package uischema

import (
	"errors"
	"testing"
)

func TestValidateApplyAcceptsValidInterfaces(t *testing.T) {
	g := NewInterfacesGenerator()
	req := ApplyRequest{
		SchemaVersion: "interfaces:v1",
		Values: map[string]interface{}{
			InterfacesWidgetID: []interface{}{
				map[string]interface{}{
					"name":         "GigabitEthernet0/0/1",
					"description":  "uplink",
					"mtu":          float64(1500),
					"admin-status": float64(1),
				},
			},
		},
	}

	err := g.ValidateApply(req)
	if err != nil {
		t.Fatalf("ValidateApply() error = %v, want nil", err)
	}
}

func TestValidateApplyRejectsSchemaVersionMismatch(t *testing.T) {
	g := NewInterfacesGenerator()
	req := ApplyRequest{
		SchemaVersion: "interfaces:old",
		Values:        map[string]interface{}{},
	}

	err := g.ValidateApply(req)
	if err == nil {
		t.Fatalf("ValidateApply() error = nil, want non-nil")
	}

	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("ValidateApply() error = %T, want *ValidationError", err)
	}

	if vErr.Code != "SCHEMA_VERSION_MISMATCH" {
		t.Errorf("ValidationError.Code = %q, want SCHEMA_VERSION_MISMATCH", vErr.Code)
	}
}

func TestValidateApplyRejectsInvalidMTU(t *testing.T) {
	g := NewInterfacesGenerator()
	req := ApplyRequest{
		SchemaVersion: "interfaces:v1",
		Values: map[string]interface{}{
			InterfacesWidgetID: []interface{}{
				map[string]interface{}{
					"name": "GigabitEthernet0/0/1",
					"mtu":  float64(42),
				},
			},
		},
	}

	err := g.ValidateApply(req)
	if err == nil {
		t.Fatalf("ValidateApply() error = nil, want non-nil")
	}

	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("ValidateApply() error = %T, want *ValidationError", err)
	}

	fieldKey := "interfaces-table:row:GigabitEthernet0/0/1:mtu"
	if len(vErr.FieldErrors[fieldKey]) == 0 {
		t.Errorf("FieldErrors[%q] is empty, want non-empty", fieldKey)
	}
}

func TestInterfacesSchemaShape(t *testing.T) {
	schema := NewInterfacesGenerator().BuildSchema("192.168.1.1")

	if schema.SchemaVersion == "" {
		t.Fatalf("expected schema version")
	}
	if schema.Module != "huawei-ifm" {
		t.Fatalf("module = %q, want huawei-ifm", schema.Module)
	}
	if schema.TargetPath != "/ifm:ifm/ifm:interfaces" {
		t.Fatalf("target path = %q", schema.TargetPath)
	}
	if schema.Layout.Type != "grid" || schema.Layout.Columns != 12 {
		t.Fatalf("unexpected layout: %+v", schema.Layout)
	}
	if len(schema.Sections) != 1 {
		t.Fatalf("sections = %d, want 1", len(schema.Sections))
	}
	if len(schema.Widgets) != 1 {
		t.Fatalf("widgets = %d, want 1", len(schema.Widgets))
	}

	widget := schema.Widgets[0]
	if widget.ID != "interfaces-table" || widget.Type != WidgetTable {
		t.Fatalf("unexpected widget: %+v", widget)
	}
	if widget.RowKey != "name" {
		t.Fatalf("row key = %q, want name", widget.RowKey)
	}
	if len(widget.Columns) != 4 {
		t.Fatalf("columns = %d, want 4", len(widget.Columns))
	}

	// Check column configuration
	columns := make(map[string]GridColumn)
	for _, col := range widget.Columns {
		columns[col.ID] = col
	}

	// Check name column
	nameCol, ok := columns["name"]
	if !ok {
		t.Fatalf("missing name column")
	}
	if !nameCol.Readonly {
		t.Fatalf("name column should be readonly")
	}
	if !nameCol.Validation.Required {
		t.Fatalf("name column should be required")
	}

	// Check description column
	descCol, ok := columns["description"]
	if !ok {
		t.Fatalf("missing description column")
	}
	if descCol.Validation.MaxLength == nil || *descCol.Validation.MaxLength != 80 {
		t.Fatalf("description column should have max length 80")
	}

	// Check mtu column
	mtuCol, ok := columns["mtu"]
	if !ok {
		t.Fatalf("missing mtu column")
	}
	if mtuCol.Validation.Min == nil || *mtuCol.Validation.Min != 1280 {
		t.Fatalf("mtu column should have min 1280")
	}
	if mtuCol.Validation.Max == nil || *mtuCol.Validation.Max != 9216 {
		t.Fatalf("mtu column should have max 9216")
	}

	// Check admin-status column
	adminCol, ok := columns["admin-status"]
	if !ok {
		t.Fatalf("missing admin-status column")
	}
	if len(adminCol.Options) != 2 {
		t.Fatalf("admin-status should have 2 options")
	}

	// Check values
	if schema.Values == nil {
		t.Fatalf("missing values map")
	}
	if _, ok := schema.Values[InterfacesWidgetID]; !ok {
		t.Fatalf("missing widget values")
	}
}
