package schema

import (
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ytypes"
)

// buildWhenSchema mirrors how an ygot-unzipped schema stores a leaf `when`:
// under Entry.Extra["when"] as a []interface{} whose element is a generic JSON
// map carrying the XPath under key "Name" (verified against the real IFM blob).
// interface list: { name (key), parent-name (when ../class='sub-interface') }.
func buildWhenSchema() *ytypes.Schema {
	name := &yang.Entry{Name: "name", Type: &yang.YangType{Kind: yang.Ystring}}
	parent := &yang.Entry{
		Name: "parent-name",
		Type: &yang.YangType{Kind: yang.Ystring},
		Extra: map[string][]interface{}{
			"when": {map[string]interface{}{"Name": "../class='sub-interface'"}},
		},
	}
	iface := &yang.Entry{
		Name: "interface", Key: "name", ListAttr: &yang.ListAttr{},
		Dir: map[string]*yang.Entry{"name": name, "parent-name": parent},
	}
	ifaces := &yang.Entry{Name: "interfaces", Dir: map[string]*yang.Entry{"interface": iface}}
	ifm := &yang.Entry{Name: "ifm", Dir: map[string]*yang.Entry{"interfaces": ifaces}}
	root := &yang.Entry{Name: "Device", Dir: map[string]*yang.Entry{"ifm": ifm}}
	return &ytypes.Schema{SchemaTree: map[string]*yang.Entry{"Device": root}}
}

func TestLeafWhenExprFromExtra(t *testing.T) {
	ds := NewSchema()
	AddYgotSchema(ds, buildWhenSchema())

	mod, ok := ds.Module("ifm")
	if !ok {
		t.Fatal("ifm module not loaded")
	}
	ifaces, _ := mod.Root().Child("interfaces")
	iface, _ := ifaces.(ContainerNode).Child("interface")
	list := iface.(ListNode)

	parent, ok := list.Child("parent-name")
	if !ok {
		t.Fatal("parent-name leaf missing")
	}
	if got := parent.(LeafNode).WhenExpr(); got != "../class='sub-interface'" {
		t.Errorf("parent-name WhenExpr = %q, want %q", got, "../class='sub-interface'")
	}

	// A leaf without `when` must report an empty expression (not panic).
	name, _ := list.Child("name")
	if got := name.(LeafNode).WhenExpr(); got != "" {
		t.Errorf("name WhenExpr = %q, want empty", got)
	}
}

// TestLeafWhenExprGoyangValueShape guards the alternate element shape produced by
// goyang's own parser (a *yang.Value with an exported Name), not just the
// ygot-unzipped JSON map — the extractor must handle both.
func TestLeafWhenExprGoyangValueShape(t *testing.T) {
	leafEntry := &yang.Entry{
		Name: "x", Type: &yang.YangType{Kind: yang.Ystring},
		Extra: map[string][]interface{}{
			"when": {&yang.Value{Name: "../y='z'"}},
		},
	}
	root := &yang.Entry{Name: "Device", Dir: map[string]*yang.Entry{
		"m": {Name: "m", Dir: map[string]*yang.Entry{"x": leafEntry}},
	}}
	ds := NewSchema()
	AddYgotSchema(ds, &ytypes.Schema{SchemaTree: map[string]*yang.Entry{"Device": root}})
	mod, _ := ds.Module("m")
	x, _ := mod.Root().Child("x")
	if got := x.(LeafNode).WhenExpr(); got != "../y='z'" {
		t.Errorf("WhenExpr = %q, want %q", got, "../y='z'")
	}
}
