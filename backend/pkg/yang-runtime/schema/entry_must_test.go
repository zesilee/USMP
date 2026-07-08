package schema

import (
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ytypes"
)

// TestLeafMustExprsFromExtra verifies all `must` XPath expressions on a leaf are
// surfaced (order-preserved), tolerating both runtime element shapes.
func TestLeafMustExprsFromExtra(t *testing.T) {
	leaf := &yang.Entry{
		Name: "x", Type: &yang.YangType{Kind: yang.Yuint32},
		Extra: map[string][]interface{}{
			"must": {
				map[string]interface{}{"Name": "(../x) mod 10 = 0"},
				&yang.Value{Name: "../x>../y"},
			},
		},
	}
	root := &yang.Entry{Name: "Device", Dir: map[string]*yang.Entry{
		"m": {Name: "m", Dir: map[string]*yang.Entry{"x": leaf}},
	}}
	ds := NewSchema()
	AddYgotSchema(ds, &ytypes.Schema{SchemaTree: map[string]*yang.Entry{"Device": root}})

	mod, _ := ds.Module("m")
	x, _ := mod.Root().Child("x")
	got := x.(LeafNode).MustExprs()
	want := []string{"(../x) mod 10 = 0", "../x>../y"}
	if len(got) != len(want) {
		t.Fatalf("MustExprs() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("MustExprs()[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	// A leaf without `must` returns empty (not nil-deref).
	empty := &yang.Entry{Name: "y", Type: &yang.YangType{Kind: yang.Ystring}}
	root.Dir["m"].Dir["y"] = empty
	ds2 := NewSchema()
	AddYgotSchema(ds2, &ytypes.Schema{SchemaTree: map[string]*yang.Entry{"Device": root}})
	m2, _ := ds2.Module("m")
	yNode, _ := m2.Root().Child("y")
	if len(yNode.(LeafNode).MustExprs()) != 0 {
		t.Errorf("MustExprs() for no-must leaf = %v, want empty", yNode.(LeafNode).MustExprs())
	}
}
