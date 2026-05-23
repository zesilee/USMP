package uischema

import "testing"

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
    if len(widget.Columns) < 4 {
        t.Fatalf("columns = %d, want at least 4", len(widget.Columns))
    }
}