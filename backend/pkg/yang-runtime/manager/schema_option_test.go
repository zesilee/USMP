package manager

import (
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ytypes"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// TestManagerWithSchema verifies an injected pre-built schema is mounted and
// takes precedence over the empty default (fixes D4 at the manager level).
func TestManagerWithSchema(t *testing.T) {
	ds := schema.NewSchema()
	root := &yang.Entry{Name: "Device", Dir: map[string]*yang.Entry{
		"sys": {Name: "sys", Dir: map[string]*yang.Entry{
			"host": {Name: "host", Type: &yang.YangType{Kind: yang.Ystring}},
		}},
	}}
	schema.AddYgotSchema(ds, &ytypes.Schema{SchemaTree: map[string]*yang.Entry{"Device": root}})

	m := New(WithSchema(ds))
	got := m.GetSchema()
	if got == nil {
		t.Fatal("GetSchema() is nil after WithSchema")
	}
	if len(got.Modules()) == 0 {
		t.Fatal("injected schema has no modules after mounting")
	}
	if _, ok := got.Module("sys"); !ok {
		t.Fatal("injected module 'sys' not mounted on manager")
	}
}

// TestManagerDefaultSchemaEmpty verifies that without WithSchema, the manager's
// schema starts empty (documents the pre-fix behavior the option replaces).
func TestManagerDefaultSchemaEmpty(t *testing.T) {
	m := New()
	if m.GetSchema() == nil {
		t.Fatal("default schema should be non-nil (empty), got nil")
	}
	if len(m.GetSchema().Modules()) != 0 {
		t.Fatal("default schema should start empty")
	}
}
