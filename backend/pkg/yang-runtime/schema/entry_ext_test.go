package schema

import (
	"reflect"
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ytypes"
)

// buildExtSchema mirrors how the ygot-unzipped IFM schema carries huawei
// presentation extensions: Entry.Exts holds *yang.Statement values whose Keyword
// is the prefixed extension name (e.g. "ext:support-filter") and Argument the
// parameter — verified against the real embedded blob. Layout:
//
//	ifm/interfaces/interface (list, key=name, ext:operation-exclude on list)
//	  ├─ name           (key)
//	  ├─ class          (ext:support-filter true + ext:operation-exclude update|delete)
//	  ├─ type           (hw-ext:support-filter TRUE — alternate prefix/case)
//	  ├─ number         (ext:operation-exclude "Update, delete" — mixed separators/case)
//	  └─ description    (no extensions)
//	ifm/global
//	  ├─ ipv4-ignore-primary-sub (leaf)
//	  └─ ipv4-conflict-enable    (presence container + must)
func buildExtSchema() *ytypes.Schema {
	str := func() *yang.YangType { return &yang.YangType{Kind: yang.Ystring} }
	ext := func(kw, arg string) *yang.Statement {
		return &yang.Statement{Keyword: kw, HasArgument: arg != "", Argument: arg}
	}

	name := &yang.Entry{Name: "name", Type: str()}
	class := &yang.Entry{Name: "class", Type: str(), Exts: []*yang.Statement{
		ext("ext:support-filter", "true"),
		ext("ext:operation-exclude", "update|delete"),
	}}
	typ := &yang.Entry{Name: "type", Type: str(), Exts: []*yang.Statement{
		ext("hw-ext:support-filter", "TRUE"),
	}}
	number := &yang.Entry{Name: "number", Type: str(), Exts: []*yang.Statement{
		ext("ext:operation-exclude", "Update, delete"),
	}}
	desc := &yang.Entry{Name: "description", Type: str()}
	noArg := &yang.Entry{Name: "no-arg", Type: str(), Exts: []*yang.Statement{
		ext("ext:support-filter", ""),
		ext("ext:operation-exclude", ""),
	}}

	iface := &yang.Entry{
		Name: "interface", Key: "name", ListAttr: &yang.ListAttr{},
		Dir: map[string]*yang.Entry{
			"name": name, "class": class, "type": typ, "number": number,
			"description": desc, "no-arg": noArg,
		},
		Exts: []*yang.Statement{ext("ext:operation-exclude", "delete")},
	}
	ifaces := &yang.Entry{Name: "interfaces", Dir: map[string]*yang.Entry{"interface": iface}}

	ignore := &yang.Entry{Name: "ipv4-ignore-primary-sub", Type: &yang.YangType{Kind: yang.Ybool}}
	conflict := &yang.Entry{
		Name: "ipv4-conflict-enable",
		Dir:  map[string]*yang.Entry{},
		Extra: map[string][]interface{}{
			"presence": {map[string]interface{}{"Name": "Enable/disable IPv4 conflict function."}},
			"must":     {map[string]interface{}{"Name": "../ipv4-ignore-primary-sub='false'"}},
			"when":     {map[string]interface{}{"Name": "../enabled='true'"}},
		},
	}
	global := &yang.Entry{Name: "global", Dir: map[string]*yang.Entry{
		"ipv4-ignore-primary-sub": ignore, "ipv4-conflict-enable": conflict,
	}}

	ifm := &yang.Entry{Name: "ifm", Dir: map[string]*yang.Entry{"interfaces": ifaces, "global": global}}
	root := &yang.Entry{Name: "Device", Dir: map[string]*yang.Entry{"ifm": ifm}}
	return &ytypes.Schema{SchemaTree: map[string]*yang.Entry{"Device": root}}
}

func extTestList(t *testing.T) ListNode {
	t.Helper()
	ds := NewSchema()
	AddYgotSchema(ds, buildExtSchema())
	mod, ok := ds.Module("ifm")
	if !ok {
		t.Fatal("ifm module not loaded")
	}
	ifaces, _ := mod.Root().Child("interfaces")
	iface, _ := ifaces.(ContainerNode).Child("interface")
	return iface.(ListNode)
}

func extTestLeaf(t *testing.T, name string) LeafNode {
	t.Helper()
	n, ok := extTestList(t).Child(name)
	if !ok {
		t.Fatalf("leaf %s missing", name)
	}
	return n.(LeafNode)
}

func TestLeafSupportFilterFromExts(t *testing.T) {
	cases := []struct {
		leaf string
		want bool
	}{
		{"class", true},        // canonical ext:support-filter true
		{"type", true},         // alternate prefix + upper-case argument
		{"description", false}, // no extensions
		{"no-arg", false},      // extension present but argument missing → degrade to absent
	}
	for _, c := range cases {
		if got := extTestLeaf(t, c.leaf).SupportFilter(); got != c.want {
			t.Errorf("%s SupportFilter = %v, want %v", c.leaf, got, c.want)
		}
	}
}

func TestLeafOperationExcludesFromExts(t *testing.T) {
	cases := []struct {
		leaf string
		want []string
	}{
		{"class", []string{"update", "delete"}},  // pipe-separated
		{"number", []string{"update", "delete"}}, // comma + spaces + mixed case → normalized
		{"description", nil},                     // absent
		{"no-arg", nil},                          // argument missing → absent
	}
	for _, c := range cases {
		if got := extTestLeaf(t, c.leaf).OperationExcludes(); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s OperationExcludes = %v, want %v", c.leaf, got, c.want)
		}
	}
}

func TestListOperationExcludesFromExts(t *testing.T) {
	if got := extTestList(t).OperationExcludes(); !reflect.DeepEqual(got, []string{"delete"}) {
		t.Errorf("list OperationExcludes = %v, want [delete]", got)
	}
}

func TestPresenceContainerWithConstraints(t *testing.T) {
	ds := NewSchema()
	AddYgotSchema(ds, buildExtSchema())
	mod, _ := ds.Module("ifm")
	g, _ := mod.Root().Child("global")
	global := g.(ContainerNode)

	n, ok := global.Child("ipv4-conflict-enable")
	if !ok {
		t.Fatal("ipv4-conflict-enable missing")
	}
	c := n.(ContainerNode)
	if !c.IsPresence() {
		t.Error("ipv4-conflict-enable IsPresence = false, want true")
	}
	if got := c.MustExprs(); !reflect.DeepEqual(got, []string{"../ipv4-ignore-primary-sub='false'"}) {
		t.Errorf("MustExprs = %v", got)
	}
	if got := c.WhenExpr(); got != "../enabled='true'" {
		t.Errorf("WhenExpr = %q", got)
	}

	// A plain container carries neither presence nor constraints.
	if global.IsPresence() {
		t.Error("global IsPresence = true, want false")
	}
	if global.WhenExpr() != "" || len(global.MustExprs()) != 0 {
		t.Error("global should carry no when/must")
	}
}
