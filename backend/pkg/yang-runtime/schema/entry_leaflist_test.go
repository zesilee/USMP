package schema

import (
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ytypes"
)

// TestLeafListModeledAsLeafListNode verifies a YANG leaf-list is surfaced as a
// LeafNode flagged IsLeafList()==true (repeatable scalar), distinct from a scalar
// leaf. The framework already lacks a distinct leaf-list impl; we mark the leaf.
func TestLeafListModeledAsLeafListNode(t *testing.T) {
	// leaf-list: has a Type (scalar element) + ListAttr, no Dir.
	tags := &yang.Entry{Name: "tags", Type: &yang.YangType{Kind: yang.Ystring}, ListAttr: &yang.ListAttr{}}
	scalar := &yang.Entry{Name: "name", Type: &yang.YangType{Kind: yang.Ystring}}
	root := &yang.Entry{Name: "Device", Dir: map[string]*yang.Entry{
		"m": {Name: "m", Dir: map[string]*yang.Entry{"tags": tags, "name": scalar}},
	}}
	ds := NewSchema()
	AddYgotSchema(ds, &ytypes.Schema{SchemaTree: map[string]*yang.Entry{"Device": root}})

	mod, _ := ds.Module("m")
	tagsNode, ok := mod.Root().Child("tags")
	if !ok {
		t.Fatal("tags leaf-list missing")
	}
	leaf, ok := tagsNode.(LeafNode)
	if !ok {
		t.Fatalf("tags should be a LeafNode, got %T", tagsNode)
	}
	if !leaf.IsLeafList() {
		t.Error("tags should report IsLeafList() == true")
	}

	nameNode, _ := mod.Root().Child("name")
	if nameNode.(LeafNode).IsLeafList() {
		t.Error("scalar leaf name should report IsLeafList() == false")
	}
}
