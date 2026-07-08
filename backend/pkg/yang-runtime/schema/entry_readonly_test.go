package schema

import (
	"sync"
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ytypes"
)

// buildReadonlySchema mirrors how the ygot-unzipped IFM schema carries `config
// false` state subtrees (verified against the real embedded blob: Config is
// TSFalse on the subtree root only, descendants stay TSUnset and inherit).
// Layout:
//
//	ifm/interfaces/interface (list, key=name, config unset)
//	  ├─ name        (key)
//	  ├─ mtu         (config unset → editable)
//	  └─ oper-status (config false → readonly leaf inside an editable container)
//	ifm/statistics   (config false container → whole subtree readonly)
//	  ├─ pkt-count   (config unset → inherits readonly)
//	  ├─ detail      (list, config unset → inherits readonly)
//	  │   └─ index
//	  └─ kind        (choice, shorthand case member inherits readonly)
//	      └─ total   (leaf member)
func buildReadonlySchema() *ytypes.Schema {
	str := func() *yang.YangType { return &yang.YangType{Kind: yang.Ystring} }

	name := &yang.Entry{Name: "name", Type: str()}
	mtu := &yang.Entry{Name: "mtu", Type: str()}
	oper := &yang.Entry{Name: "oper-status", Type: str(), Config: yang.TSFalse}
	iface := &yang.Entry{
		Name: "interface", Key: "name", ListAttr: &yang.ListAttr{},
		Dir: map[string]*yang.Entry{"name": name, "mtu": mtu, "oper-status": oper},
	}
	ifaces := &yang.Entry{Name: "interfaces", Dir: map[string]*yang.Entry{"interface": iface}}

	pkt := &yang.Entry{Name: "pkt-count", Type: str()}
	index := &yang.Entry{Name: "index", Type: str()}
	detail := &yang.Entry{
		Name: "detail", Key: "index", ListAttr: &yang.ListAttr{},
		Dir: map[string]*yang.Entry{"index": index},
	}
	total := &yang.Entry{Name: "total", Type: str()}
	kind := &yang.Entry{
		Name: "kind", Kind: yang.ChoiceEntry,
		Dir: map[string]*yang.Entry{"total": total},
	}
	stats := &yang.Entry{
		Name: "statistics", Config: yang.TSFalse,
		Dir: map[string]*yang.Entry{"pkt-count": pkt, "detail": detail, "kind": kind},
	}

	ifm := &yang.Entry{Name: "ifm", Dir: map[string]*yang.Entry{"interfaces": ifaces, "statistics": stats}}
	root := &yang.Entry{Name: "Device", Dir: map[string]*yang.Entry{"ifm": ifm}}
	return &ytypes.Schema{SchemaTree: map[string]*yang.Entry{"Device": root}}
}

func readonlyTestModule(t *testing.T) Module {
	t.Helper()
	ds := NewSchema()
	AddYgotSchema(ds, buildReadonlySchema())
	mod, ok := ds.Module("ifm")
	if !ok {
		t.Fatal("ifm module not loaded")
	}
	return mod
}

func readonlyChild(t *testing.T, parent Node, name string) Node {
	t.Helper()
	type childer interface {
		Child(string) (Node, bool)
	}
	p, ok := parent.(childer)
	if !ok {
		t.Fatalf("%s cannot have children", parent.Name())
	}
	n, ok := p.Child(name)
	if !ok {
		t.Fatalf("child %s missing under %s", name, parent.Name())
	}
	return n
}

func TestReadOnlyFromConfigFalse(t *testing.T) {
	mod := readonlyTestModule(t)
	root := mod.Root()
	ifaces := readonlyChild(t, root, "interfaces")
	iface := readonlyChild(t, ifaces, "interface")
	stats := readonlyChild(t, root, "statistics")

	cases := []struct {
		desc string
		node Node
		want bool
	}{
		{"config-true container", ifaces, false},
		{"config-true list", iface, false},
		{"config-true key leaf", readonlyChild(t, iface, "name"), false},
		{"config-true plain leaf", readonlyChild(t, iface, "mtu"), false},
		{"config-false leaf in editable container", readonlyChild(t, iface, "oper-status"), true},
		{"config-false subtree root", stats, true},
		{"inherited by leaf", readonlyChild(t, stats, "pkt-count"), true},
		{"inherited by nested list", readonlyChild(t, stats, "detail"), true},
		{"inherited by nested list leaf", readonlyChild(t, readonlyChild(t, stats, "detail"), "index"), true},
	}
	for _, c := range cases {
		if got := c.node.ReadOnly(); got != c.want {
			t.Errorf("%s: ReadOnly = %v, want %v", c.desc, got, c.want)
		}
	}
}

func TestReadOnlyInheritedThroughChoice(t *testing.T) {
	mod := readonlyTestModule(t)
	stats := readonlyChild(t, mod.Root(), "statistics")
	kind := readonlyChild(t, stats, "kind")
	ch, ok := kind.(ChoiceNode)
	if !ok {
		t.Fatalf("kind is %T, want ChoiceNode", kind)
	}
	if !ch.ReadOnly() {
		t.Error("choice under config-false subtree: ReadOnly = false, want true")
	}
	cs, ok := ch.Case("total")
	if !ok {
		t.Fatal("shorthand case total missing")
	}
	member := readonlyChild(t, cs, "total")
	if !member.ReadOnly() {
		t.Error("choice member under config-false subtree: ReadOnly = false, want true")
	}
}

// TestReadOnlyConcurrentReads guards the accessor against data races: the schema
// tree is immutable after build and concurrent readers must be safe (R09).
func TestReadOnlyConcurrentReads(t *testing.T) {
	mod := readonlyTestModule(t)
	stats := readonlyChild(t, mod.Root(), "statistics")
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if !stats.ReadOnly() {
					t.Error("statistics ReadOnly = false, want true")
					return
				}
			}
		}()
	}
	wg.Wait()
}
