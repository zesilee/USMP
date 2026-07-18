package netconfsim

import (
	"encoding/xml"
	"testing"
)

// nodesEqual compares two data trees for semantic equality: local name, namespace
// URI, attribute set (order-independent), leaf text, and children (in order).
func nodesEqual(a, b *dataNode) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Name.Local != b.Name.Local || a.Name.Space != b.Name.Space {
		return false
	}
	if a.leafText() != b.leafText() {
		return false
	}
	if !attrsEqual(a.Attrs, b.Attrs) {
		return false
	}
	if len(a.Children) != len(b.Children) {
		return false
	}
	for i := range a.Children {
		if !nodesEqual(a.Children[i], b.Children[i]) {
			return false
		}
	}
	return true
}

// attrsEqual compares two attribute lists order-independently by local name+value.
func attrsEqual(x, y []xml.Attr) bool {
	if len(x) != len(y) {
		return false
	}
	m := make(map[string]string, len(x))
	for _, a := range x {
		m[a.Name.Local] = a.Value
	}
	for _, a := range y {
		if v, ok := m[a.Name.Local]; !ok || v != a.Value {
			return false
		}
	}
	return true
}

func TestParseSerializeRoundtrip(t *testing.T) {
	cases := map[string]string{
		"huawei-vlan":        `<vlan xmlns="urn:huawei:vlan"><vlans><vlan><id>100</id><name>office</name></vlan><vlan><id>200</id><name>guest</name></vlan></vlans></vlan>`,
		"foreign-iface":      `<interfaces xmlns="http://example.com/yang/interfaces"><interface><name>eth0</name><config><name>eth0</name><enabled>true</enabled><mtu>1500</mtu></config></interface></interfaces>`,
		"self-closing-empty": `<system xmlns="urn:huawei:system"><info><name/></info></system>`,
		"attr-operation":     `<vlan xmlns="urn:huawei:vlan"><vlans><vlan operation="delete"><id>300</id></vlan></vlans></vlan>`,
		"multiple-top-level": `<vlan xmlns="urn:huawei:vlan"><vlans/></vlan><ifm xmlns="urn:huawei:ifm"><interfaces/></ifm>`,
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			tree1, err := parseXML([]byte(in))
			if err != nil {
				t.Fatalf("parse #1: %v", err)
			}
			out := tree1.xmlBytes()
			tree2, err := parseXML(out)
			if err != nil {
				t.Fatalf("parse #2 of %q: %v", out, err)
			}
			if !nodesEqual(tree1, tree2) {
				t.Fatalf("roundtrip not equal\n in:  %s\n out: %s", in, out)
			}
		})
	}
}

func TestParseError(t *testing.T) {
	if _, err := parseXML([]byte(`<vlan><unclosed></vlan>`)); err == nil {
		t.Fatal("expected parse error for malformed XML")
	}
}

func TestFindAndLeafText(t *testing.T) {
	tree, err := parseXML([]byte(`<vlan xmlns="urn:huawei:vlan"><vlans><vlan><id>100</id><name>office</name></vlan></vlans></vlan>`))
	if err != nil {
		t.Fatal(err)
	}
	vlan := tree.find("vlan", "vlans", "vlan")
	if vlan == nil {
		t.Fatal("expected to find vlan node")
	}
	if got := vlan.child("id").leafText(); got != "100" {
		t.Fatalf("id = %q, want 100", got)
	}
	if got := vlan.child("name").leafText(); got != "office" {
		t.Fatalf("name = %q, want office", got)
	}
	if tree.find("vlan", "nope") != nil {
		t.Fatal("expected nil for missing path")
	}
}

func TestChildrenMulti(t *testing.T) {
	tree, err := parseXML([]byte(`<vlans><vlan><id>1</id></vlan><vlan><id>2</id></vlan><vlan><id>3</id></vlan></vlans>`))
	if err != nil {
		t.Fatal(err)
	}
	vlans := tree.child("vlans")
	if n := len(vlans.children("vlan")); n != 3 {
		t.Fatalf("children(vlan) = %d, want 3", n)
	}
}

func TestCloneIndependence(t *testing.T) {
	tree, err := parseXML([]byte(`<vlan><vlans><vlan><id>100</id></vlan></vlans></vlan>`))
	if err != nil {
		t.Fatal(err)
	}
	cp := tree.clone()
	// Mutate original leaf; clone must be unaffected.
	tree.find("vlan", "vlans", "vlan", "id").Text = "999"
	if got := cp.find("vlan", "vlans", "vlan", "id").leafText(); got != "100" {
		t.Fatalf("clone leaked mutation: id = %q, want 100", got)
	}
}
