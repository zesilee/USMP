package api

import (
	"reflect"
	"testing"

	"github.com/leezesi/usmp/backend/internal/yangschema"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ytypes"
)

// TestNestedSchemaCarriesExtensionsFromRealIFM verifies the schema endpoint
// surfaces the real IFM presentation extensions onto FieldDef: `support-filter`
// marks advanced-search fields and `operation-exclude` marks create-only identity
// fields (BR-07) — 100% metadata-driven, no hardcoding.
func TestNestedSchemaCarriesExtensionsFromRealIFM(t *testing.T) {
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	mod, ok := s.Module("ifm")
	if !ok {
		t.Fatal("ifm module not loaded")
	}
	ys := buildYangSchemaNested(mod)

	class, ok := findFieldBySuffix(ys.Fields, "class")
	if !ok {
		t.Fatal("class field not found")
	}
	if !class.SupportFilter {
		t.Error("class SupportFilter = false, want true")
	}
	if want := []string{"update", "delete"}; !reflect.DeepEqual(class.OperationExclude, want) {
		t.Errorf("class OperationExclude = %v, want %v", class.OperationExclude, want)
	}

	// A leaf without extensions must omit both (zero values → omitempty in JSON).
	desc, ok := findFieldBySuffix(ys.Fields, "description")
	if ok && (desc.SupportFilter || len(desc.OperationExclude) != 0) {
		t.Errorf("description carries unexpected extensions: %+v", desc)
	}
}

// TestNestedSchemaCarriesPresenceAndContainerMustFromRealIFM verifies presence
// containers surface presence=true plus their container-level `must` (BR-08):
// IFM global/ipv4-conflict-enable exists only when ipv4-ignore-primary-sub=false.
func TestNestedSchemaCarriesPresenceAndContainerMustFromRealIFM(t *testing.T) {
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	mod, _ := s.Module("ifm")
	ys := buildYangSchemaNested(mod)

	conflict, ok := findFieldBySuffix(ys.Fields, "ipv4-conflict-enable")
	if !ok {
		t.Fatal("ipv4-conflict-enable group not found")
	}
	if conflict.Type != "group" {
		t.Fatalf("ipv4-conflict-enable Type = %q, want group", conflict.Type)
	}
	if !conflict.Presence {
		t.Error("ipv4-conflict-enable Presence = false, want true")
	}
	found := false
	for _, m := range conflict.Must {
		if m.Expr == "../ipv4-ignore-primary-sub='false'" {
			found = true
		}
	}
	if !found {
		t.Errorf("ipv4-conflict-enable Must = %+v, want expr ../ipv4-ignore-primary-sub='false'", conflict.Must)
	}

	// A plain (non-presence) container must omit presence.
	global, ok := findFieldBySuffix(ys.Fields, "global")
	if ok && global.Presence {
		t.Error("global Presence = true, want false")
	}
}

// TestNestedSchemaCarriesListOperationExclude verifies a list-level
// `operation-exclude` lands on the list FieldDef (real IFM has none on the
// interface list, so a synthetic tree exercises this path).
func TestNestedSchemaCarriesListOperationExclude(t *testing.T) {
	item := &yang.Entry{
		Name: "item", Key: "id", ListAttr: &yang.ListAttr{},
		Dir: map[string]*yang.Entry{
			"id": {Name: "id", Type: &yang.YangType{Kind: yang.Ystring}},
		},
		Exts: []*yang.Statement{{Keyword: "ext:operation-exclude", HasArgument: true, Argument: "update|delete"}},
	}
	mod := &yang.Entry{Name: "demo", Dir: map[string]*yang.Entry{"item": item}}
	root := &yang.Entry{Name: "Device", Dir: map[string]*yang.Entry{"demo": mod}}

	ds := schema.NewSchema()
	schema.AddYgotSchema(ds, &ytypes.Schema{SchemaTree: map[string]*yang.Entry{"Device": root}})
	m, ok := ds.Module("demo")
	if !ok {
		t.Fatal("demo module not loaded")
	}
	ys := buildYangSchemaNested(m)

	list, ok := findFieldBySuffix(ys.Fields, "item")
	if !ok || list.Type != "list" {
		t.Fatalf("item list not found: %+v", ys.Fields)
	}
	if want := []string{"update", "delete"}; !reflect.DeepEqual(list.OperationExclude, want) {
		t.Errorf("list OperationExclude = %v, want %v", list.OperationExclude, want)
	}
}

// TestNestedSchemaMarksListKey verifies list key leaves carry isKey=true so the
// generic console can derive keyField without per-module route props (BR-07).
func TestNestedSchemaMarksListKey(t *testing.T) {
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	mod, _ := s.Module("ifm")
	ys := buildYangSchemaNested(mod)

	name, ok := findFieldBySuffix(ys.Fields, "interface/name")
	if !ok {
		t.Fatal("interface/name not found")
	}
	if !name.IsKey {
		t.Error("interface name IsKey = false, want true")
	}
	desc, ok := findFieldBySuffix(ys.Fields, "interface/description")
	if ok && desc.IsKey {
		t.Error("description IsKey = true, want false")
	}
}
