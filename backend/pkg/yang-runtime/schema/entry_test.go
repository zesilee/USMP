package schema

import (
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ytypes"
)

// buildSyntheticSchema builds a small resolved yang.Entry tree standing in for an
// ygot-generated schema:  root → module "net" → { host (leaf string, mandatory),
// mtu (leaf uint16), enabled (leaf bool), vlans (list keyed by id) }.
func buildSyntheticSchema() *ytypes.Schema {
	host := &yang.Entry{Name: "host", Type: &yang.YangType{Kind: yang.Ystring}, Mandatory: yang.TSTrue}
	mtu := &yang.Entry{Name: "mtu", Type: &yang.YangType{Kind: yang.Yuint16}}
	enabled := &yang.Entry{Name: "enabled", Type: &yang.YangType{Kind: yang.Ybool}}

	vlanID := &yang.Entry{Name: "id", Type: &yang.YangType{Kind: yang.Yuint16}}
	vlanName := &yang.Entry{Name: "name", Type: &yang.YangType{Kind: yang.Ystring}}
	vlan := &yang.Entry{
		Name:     "vlan",
		Key:      "id",
		ListAttr: &yang.ListAttr{},
		Dir:      map[string]*yang.Entry{"id": vlanID, "name": vlanName},
	}
	vlans := &yang.Entry{Name: "vlans", Dir: map[string]*yang.Entry{"vlan": vlan}}

	net := &yang.Entry{Name: "net", Dir: map[string]*yang.Entry{
		"host": host, "mtu": mtu, "enabled": enabled, "vlans": vlans,
	}}
	root := &yang.Entry{Name: "Device", Dir: map[string]*yang.Entry{"net": net}}

	// Root left nil → AddYgotSchema uses the parentless-with-children fallback.
	return &ytypes.Schema{SchemaTree: map[string]*yang.Entry{"Device": root}}
}

func TestAddYgotSchemaModules(t *testing.T) {
	ds := NewSchema()
	AddYgotSchema(ds, buildSyntheticSchema())

	mods := ds.Modules()
	if len(mods) != 1 {
		t.Fatalf("Modules() = %d, want 1", len(mods))
	}
	if _, ok := ds.Module("net"); !ok {
		t.Fatalf("module 'net' not found; modules=%v", mods)
	}
}

func TestAddYgotSchemaLeafTypesAndTree(t *testing.T) {
	ds := NewSchema()
	AddYgotSchema(ds, buildSyntheticSchema())

	mod, _ := ds.Module("net")
	root := mod.Root()

	host, ok := root.Child("host")
	if !ok {
		t.Fatal("net/host missing")
	}
	hl := host.(LeafNode)
	if hl.LeafType() != LeafTypeString {
		t.Errorf("host leaf type = %v, want string", hl.LeafType())
	}
	if !hl.Mandatory() {
		t.Error("host should be mandatory")
	}

	if mtu, _ := root.Child("mtu"); mtu.(LeafNode).LeafType() != LeafTypeUint16 {
		t.Errorf("mtu leaf type = %v, want uint16", mtu.(LeafNode).LeafType())
	}
	if en, _ := root.Child("enabled"); en.(LeafNode).LeafType() != LeafTypeBoolean {
		t.Errorf("enabled leaf type = %v, want bool", en.(LeafNode).LeafType())
	}
}

func TestAddYgotSchemaListKeys(t *testing.T) {
	ds := NewSchema()
	AddYgotSchema(ds, buildSyntheticSchema())

	mod, _ := ds.Module("net")
	vlans, ok := mod.Root().Child("vlans")
	if !ok {
		t.Fatal("net/vlans missing")
	}
	vlan, ok := vlans.(ContainerNode).Child("vlan")
	if !ok {
		t.Fatal("net/vlans/vlan missing")
	}
	list, ok := vlan.(ListNode)
	if !ok {
		t.Fatalf("vlan should be a ListNode, got %T", vlan)
	}
	if len(list.Keys()) != 1 || list.Keys()[0].Name() != "id" {
		t.Fatalf("vlan keys = %v, want [id]", list.Keys())
	}
	if !list.Keys()[0].IsKey() {
		t.Error("id leaf should be marked IsKey")
	}
}

func TestAddYgotSchemaPathResolves(t *testing.T) {
	ds := NewSchema()
	AddYgotSchema(ds, buildSyntheticSchema())
	if _, ok := ds.Path("/net/vlans/vlan"); !ok {
		t.Fatal("Path(/net/vlans/vlan) should resolve")
	}
}

func TestAddYgotSchemaEmptyAndNil(t *testing.T) {
	ds := NewSchema()
	AddYgotSchema(ds, nil) // must not panic
	AddYgotSchema(nil, buildSyntheticSchema())
	AddYgotSchema(ds, &ytypes.Schema{}) // empty tree → no modules
	if len(ds.Modules()) != 0 {
		t.Fatalf("expected no modules, got %d", len(ds.Modules()))
	}
}
