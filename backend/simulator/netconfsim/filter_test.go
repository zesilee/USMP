package netconfsim

import "testing"

// filterData is a two-root running config (Huawei vlan list + system container)
// used to drive subtree-filter selection/pruning tests. <desc> exists so tests
// can prove non-selected siblings are pruned.
const filterData = `<vlan xmlns="urn:huawei:vlan"><vlans>` +
	`<vlan><id>100</id><name>office</name><desc>d1</desc></vlan>` +
	`<vlan><id>200</id><name>guest</name><desc>d2</desc></vlan>` +
	`</vlans></vlan>` +
	`<system xmlns="urn:huawei:system"><info><name>sw1</name></info></system>`

func filterResult(t *testing.T, data, filter string) *dataNode {
	t.Helper()
	d, err := parseXML([]byte(data))
	if err != nil {
		t.Fatalf("parse data: %v", err)
	}
	f, err := parseXML([]byte(filter))
	if err != nil {
		t.Fatalf("parse filter: %v", err)
	}
	return filterTree(d, f)
}

// A selection node (empty element) selects the whole matching subtree and
// excludes non-matching top-level roots.
func TestFilterSelectionNodeSelectsWholeSubtree(t *testing.T) {
	res := filterResult(t, filterData, `<vlan xmlns="urn:huawei:vlan"/>`)
	if res.child("system") != nil {
		t.Fatalf("system should be filtered out: %s", res.xmlBytes())
	}
	vlans := res.find("vlan", "vlans")
	if vlans == nil || len(vlans.children("vlan")) != 2 {
		t.Fatalf("whole vlan subtree expected: %s", res.xmlBytes())
	}
	// full subtree => desc preserved
	if vlans.children("vlan")[0].child("desc") == nil {
		t.Fatalf("selection node should keep desc: %s", res.xmlBytes())
	}
}

// A content-match leaf (e.g. <id>100</id>) selects only the matching list entry
// and returns its whole subtree.
func TestFilterContentMatchSelectsEntry(t *testing.T) {
	filter := `<vlan xmlns="urn:huawei:vlan"><vlans><vlan><id>100</id></vlan></vlans></vlan>`
	res := filterResult(t, filterData, filter)
	vlans := res.find("vlan", "vlans")
	if vlans == nil || len(vlans.children("vlan")) != 1 {
		t.Fatalf("expected exactly vlan 100: %s", res.xmlBytes())
	}
	v := vlans.children("vlan")[0]
	if v.child("id").leafText() != "100" || v.child("name").leafText() != "office" {
		t.Fatalf("wrong entry selected: %s", res.xmlBytes())
	}
	// content match with no sibling selectors => whole entry (desc kept)
	if v.child("desc") == nil {
		t.Fatalf("content match should return whole entry: %s", res.xmlBytes())
	}
}

// A content-match key plus a selection leaf returns the key and the selected
// leaf only; other siblings (desc) are pruned.
func TestFilterSelectionLeafPrunesSiblings(t *testing.T) {
	filter := `<vlan xmlns="urn:huawei:vlan"><vlans><vlan><id>100</id><name/></vlan></vlans></vlan>`
	res := filterResult(t, filterData, filter)
	v := res.find("vlan", "vlans", "vlan")
	if v == nil {
		t.Fatalf("vlan 100 missing: %s", res.xmlBytes())
	}
	if v.child("id").leafText() != "100" || v.child("name").leafText() != "office" {
		t.Fatalf("id/name expected: %s", res.xmlBytes())
	}
	if v.child("desc") != nil {
		t.Fatalf("desc should be pruned by selection: %s", res.xmlBytes())
	}
}

// A content match that matches nothing yields an empty result.
func TestFilterNoMatchYieldsEmpty(t *testing.T) {
	filter := `<vlan xmlns="urn:huawei:vlan"><vlans><vlan><id>999</id></vlan></vlans></vlan>`
	res := filterResult(t, filterData, filter)
	if len(res.Children) != 0 {
		t.Fatalf("expected empty result, got: %s", res.xmlBytes())
	}
}

// A namespace-less filter still matches namespaced data (client XPath/subtree
// filters routinely omit namespaces).
func TestFilterNamespaceLessMatchesNamespacedData(t *testing.T) {
	data := `<interfaces xmlns="http://openconfig.net/yang/interfaces">` +
		`<interface><name>eth0</name><config><mtu>1500</mtu></config></interface>` +
		`<interface><name>eth1</name><config><mtu>9000</mtu></config></interface>` +
		`</interfaces>`
	filter := `<interfaces><interface><name>eth0</name></interface></interfaces>`
	res := filterResult(t, data, filter)
	ifs := res.find("interfaces")
	if ifs == nil || len(ifs.children("interface")) != 1 {
		t.Fatalf("expected exactly eth0: %s", res.xmlBytes())
	}
	if ifs.children("interface")[0].child("name").leafText() != "eth0" {
		t.Fatalf("wrong interface: %s", res.xmlBytes())
	}
}

// The datastore wrapper filters the running tree and serializes the result.
func TestGetConfigFilteredRunning(t *testing.T) {
	ds := newTreeDatastore()
	if err := ds.SetRunning([]byte(filterData)); err != nil {
		t.Fatal(err)
	}
	out, err := ds.GetConfigFiltered([]byte(`<vlan xmlns="urn:huawei:vlan"><vlans><vlan><id>200</id></vlan></vlans></vlan>`))
	if err != nil {
		t.Fatalf("GetConfigFiltered: %v", err)
	}
	got := treeFromXML(t, string(out))
	vlans := got.find("vlan", "vlans")
	if vlans == nil || len(vlans.children("vlan")) != 1 || vlans.children("vlan")[0].child("id").leafText() != "200" {
		t.Fatalf("wrapper should return only vlan 200: %s", out)
	}
}

// An empty/absent filter returns the whole running config unchanged.
func TestGetConfigFilteredEmptyReturnsAll(t *testing.T) {
	ds := newTreeDatastore()
	if err := ds.SetRunning([]byte(filterData)); err != nil {
		t.Fatal(err)
	}
	out, err := ds.GetConfigFiltered(nil)
	if err != nil {
		t.Fatalf("GetConfigFiltered(nil): %v", err)
	}
	got := treeFromXML(t, string(out))
	if got.child("vlan") == nil || got.child("system") == nil {
		t.Fatalf("empty filter should return whole config: %s", out)
	}
}
