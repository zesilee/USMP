package schema

import (
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ytypes"
)

// buildChoiceSchema mirrors how the ygot-embedded IFM schema keeps `choice`/`case`
// entries intact (huawei.Schema() rebuilds them from the compiled gzip blob — they
// are NOT flattened). Shape under interface{name}:
//
//	choice bandwidth-type       (two single-leaf cases → RadioGroup material)
//	  case bandwidth-mbps { leaf bandwidth }
//	  case bandwidth-kbps { leaf bandwidth-kbps }
//	choice damping              (nested: case→container→choice→case→container→leaf)
//	  case damp { container damp { leaf tx-off; choice level { case manual { container manual { leaf suppress } } } } }
//
// The framework MUST expose these as Choice/Case nodes while flattening member data
// paths (choice/case contribute no path segment; real containers do), so the NETCONF
// write path stays flat: /ifm/interfaces/interface/bandwidth , …/damp/manual/suppress.
func choiceLeaf(name string) *yang.Entry {
	return &yang.Entry{Name: name, Type: &yang.YangType{Kind: yang.Ystring}}
}

func buildChoiceSchema() *ytypes.Schema {
	bwType := &yang.Entry{
		Name: "bandwidth-type", Kind: yang.ChoiceEntry,
		Dir: map[string]*yang.Entry{
			"bandwidth-mbps": {Name: "bandwidth-mbps", Kind: yang.CaseEntry,
				Dir: map[string]*yang.Entry{"bandwidth": choiceLeaf("bandwidth")}},
			"bandwidth-kbps": {Name: "bandwidth-kbps", Kind: yang.CaseEntry,
				Dir: map[string]*yang.Entry{"bandwidth-kbps": choiceLeaf("bandwidth-kbps")}},
		},
	}
	levelChoice := &yang.Entry{
		Name: "level", Kind: yang.ChoiceEntry,
		Dir: map[string]*yang.Entry{
			"manual": {Name: "manual", Kind: yang.CaseEntry,
				Dir: map[string]*yang.Entry{
					"manual": {Name: "manual", Dir: map[string]*yang.Entry{"suppress": choiceLeaf("suppress")}},
				}},
		},
	}
	damping := &yang.Entry{
		Name: "damping", Kind: yang.ChoiceEntry,
		Dir: map[string]*yang.Entry{
			"damp": {Name: "damp", Kind: yang.CaseEntry,
				Dir: map[string]*yang.Entry{
					"damp": {Name: "damp", Dir: map[string]*yang.Entry{
						"tx-off": choiceLeaf("tx-off"),
						"level":  levelChoice,
					}},
				}},
		},
	}
	iface := &yang.Entry{
		Name: "interface", Key: "name", ListAttr: &yang.ListAttr{},
		Dir: map[string]*yang.Entry{
			"name":           choiceLeaf("name"),
			"bandwidth-type": bwType,
			"damping":        damping,
		},
	}
	ifaces := &yang.Entry{Name: "interfaces", Dir: map[string]*yang.Entry{"interface": iface}}
	ifm := &yang.Entry{Name: "ifm", Dir: map[string]*yang.Entry{"interfaces": ifaces}}
	root := &yang.Entry{Name: "Device", Dir: map[string]*yang.Entry{"ifm": ifm}}
	return &ytypes.Schema{SchemaTree: map[string]*yang.Entry{"Device": root}}
}

func ifaceList(t *testing.T, ds *DefaultSchema) ListNode {
	t.Helper()
	mod, ok := ds.Module("ifm")
	if !ok {
		t.Fatal("ifm module not loaded")
	}
	ifaces, _ := mod.Root().Child("interfaces")
	iface, _ := ifaces.(ContainerNode).Child("interface")
	return iface.(ListNode)
}

func TestChoiceNodeBuiltFromEntry(t *testing.T) {
	ds := NewSchema()
	AddYgotSchema(ds, buildChoiceSchema())
	list := ifaceList(t, ds)

	// bandwidth-type surfaces as a Choice node (not a container).
	child, ok := list.Child("bandwidth-type")
	if !ok {
		t.Fatal("bandwidth-type child missing")
	}
	ch, ok := child.(ChoiceNode)
	if !ok {
		t.Fatalf("bandwidth-type is %T, want ChoiceNode", child)
	}
	if child.Type() != ChoiceNodeType {
		t.Errorf("bandwidth-type NodeType = %v, want ChoiceNodeType", child.Type())
	}
	if len(ch.Cases()) != 2 {
		t.Fatalf("bandwidth-type has %d cases, want 2", len(ch.Cases()))
	}

	// Case member leaf carries a FLAT data path (no choice/case segment).
	mbps, ok := ch.Case("bandwidth-mbps")
	if !ok {
		t.Fatal("case bandwidth-mbps missing")
	}
	bw, ok := mbps.Child("bandwidth")
	if !ok {
		t.Fatal("bandwidth leaf missing under case")
	}
	if got, want := bw.Path(), "/ifm/interfaces/interface/bandwidth"; got != want {
		t.Errorf("bandwidth Path = %q, want flat %q", got, want)
	}
	if _, isLeaf := bw.(LeafNode); !isLeaf {
		t.Errorf("bandwidth is %T, want LeafNode", bw)
	}
}

func TestChoiceMemberResolvableByFlatPath(t *testing.T) {
	ds := NewSchema()
	AddYgotSchema(ds, buildChoiceSchema())

	// Flat data paths must resolve through the schema path cache (write path relies on it).
	for _, p := range []string{
		"/ifm/interfaces/interface/bandwidth",
		"/ifm/interfaces/interface/bandwidth-kbps",
		"/ifm/interfaces/interface/damp/manual/suppress",
	} {
		if _, ok := ds.Path(p); !ok {
			t.Errorf("Path(%q) not resolvable", p)
		}
	}
	// The choice/case names must NOT appear as data-tree segments.
	for _, p := range []string{
		"/ifm/interfaces/interface/bandwidth-type/bandwidth-mbps/bandwidth",
		"/ifm/interfaces/interface/damping/damp/damp/level/manual/manual/suppress",
	} {
		if _, ok := ds.Path(p); ok {
			t.Errorf("Path(%q) resolved but choice/case segments must be flattened out", p)
		}
	}
}

func TestNestedChoiceInsideCaseContainer(t *testing.T) {
	ds := NewSchema()
	AddYgotSchema(ds, buildChoiceSchema())
	list := ifaceList(t, ds)

	damping, _ := list.Child("damping")
	dch, ok := damping.(ChoiceNode)
	if !ok {
		t.Fatalf("damping is %T, want ChoiceNode", damping)
	}
	dampCase, ok := dch.Case("damp")
	if !ok {
		t.Fatal("case damp missing")
	}
	// case damp holds a real container `damp` (kept in the data path).
	dampC, ok := dampCase.Child("damp")
	if !ok {
		t.Fatal("container damp missing under case")
	}
	cont, ok := dampC.(ContainerNode)
	if !ok {
		t.Fatalf("damp is %T, want ContainerNode", dampC)
	}
	if got, want := cont.Path(), "/ifm/interfaces/interface/damp"; got != want {
		t.Errorf("damp container Path = %q, want %q", got, want)
	}
	// nested choice `level` lives inside the container and is a Choice node.
	level, ok := cont.Child("level")
	if !ok {
		t.Fatal("nested choice level missing")
	}
	lch, ok := level.(ChoiceNode)
	if !ok {
		t.Fatalf("level is %T, want ChoiceNode", level)
	}
	manual, _ := lch.Case("manual")
	if manual == nil {
		t.Fatal("case manual missing")
	}
	manC, _ := manual.Child("manual")
	suppress, ok := manC.(ContainerNode).Child("suppress")
	if !ok {
		t.Fatal("suppress leaf missing")
	}
	if got, want := suppress.Path(), "/ifm/interfaces/interface/damp/manual/suppress"; got != want {
		t.Errorf("suppress Path = %q, want flat %q", got, want)
	}
}
