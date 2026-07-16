package netconfsim

import (
	"strings"
	"testing"
)

// baseVlans seeds two Huawei VLAN list entries (keyed by <id>) into running, so
// candidate starts as a clone of it. These tests drive the T3 edit-config
// operation semantics (merge/create/delete/replace/remove) on the candidate tree.
const baseVlans = `<vlan xmlns="urn:huawei:vlan"><vlans>` +
	`<vlan><id>100</id><name>office</name></vlan>` +
	`<vlan><id>200</id><name>guest</name></vlan>` +
	`</vlans></vlan>`

// seedVlans returns a treeDatastore whose running/candidate hold baseVlans.
func seedVlans(t *testing.T) *treeDatastore {
	t.Helper()
	ds := newTreeDatastore()
	if err := ds.SetRunning([]byte(baseVlans)); err != nil {
		t.Fatalf("SetRunning: %v", err)
	}
	return ds
}

// candidateVlanByID looks up a <vlan> list entry in the candidate tree by its
// <id> key, returning nil if absent.
func candidateVlanByID(ds *treeDatastore, id string) *dataNode {
	vlans := ds.candidateTree().find("vlan", "vlans")
	if vlans == nil {
		return nil
	}
	for _, v := range vlans.children("vlan") {
		if idNode := v.child("id"); idNode != nil && idNode.leafText() == id {
			return v
		}
	}
	return nil
}

func candidateVlanCount(ds *treeDatastore) int {
	vlans := ds.candidateTree().find("vlan", "vlans")
	if vlans == nil {
		return 0
	}
	return len(vlans.children("vlan"))
}

// --- merge (default operation) ---

func TestEditConfigMergeModifiesLeaf(t *testing.T) {
	ds := seedVlans(t)
	// No operation attr => default merge. Change vlan 100's name only.
	edit := `<vlan xmlns="urn:huawei:vlan"><vlans><vlan><id>100</id><name>NEWNAME</name></vlan></vlans></vlan>`
	if err := ds.EditConfig([]byte(edit)); err != nil {
		t.Fatalf("EditConfig: %v", err)
	}
	v100 := candidateVlanByID(ds, "100")
	if v100 == nil || v100.child("name").leafText() != "NEWNAME" {
		t.Fatalf("vlan 100 name not merged: %s", ds.GetCandidate())
	}
	// vlan 200 must be untouched, and no entry duplicated.
	if v200 := candidateVlanByID(ds, "200"); v200 == nil || v200.child("name").leafText() != "guest" {
		t.Fatalf("vlan 200 clobbered: %s", ds.GetCandidate())
	}
	if got := candidateVlanCount(ds); got != 2 {
		t.Fatalf("expected 2 vlans, got %d: %s", got, ds.GetCandidate())
	}
}

func TestEditConfigMergeAddsNewListEntry(t *testing.T) {
	ds := seedVlans(t)
	// Merging a not-yet-present keyed entry creates it (implicit create).
	edit := `<vlan xmlns="urn:huawei:vlan"><vlans><vlan><id>300</id><name>eng</name></vlan></vlans></vlan>`
	if err := ds.EditConfig([]byte(edit)); err != nil {
		t.Fatalf("EditConfig: %v", err)
	}
	if candidateVlanCount(ds) != 3 {
		t.Fatalf("expected 3 vlans, got %d: %s", candidateVlanCount(ds), ds.GetCandidate())
	}
	if v := candidateVlanByID(ds, "300"); v == nil || v.child("name").leafText() != "eng" {
		t.Fatalf("vlan 300 not added: %s", ds.GetCandidate())
	}
}

func TestEditConfigMergeNestedContainerPreservesSiblings(t *testing.T) {
	ds := newTreeDatastore()
	base := `<interfaces xmlns="http://openconfig.net/yang/interfaces">` +
		`<interface><name>eth0</name><config><name>eth0</name><enabled>true</enabled><mtu>1500</mtu></config></interface>` +
		`</interfaces>`
	if err := ds.SetRunning([]byte(base)); err != nil {
		t.Fatal(err)
	}
	// Merge only mtu into the single <config> container; enabled must survive.
	edit := `<interfaces xmlns="http://openconfig.net/yang/interfaces">` +
		`<interface><name>eth0</name><config><mtu>9000</mtu></config></interface></interfaces>`
	if err := ds.EditConfig([]byte(edit)); err != nil {
		t.Fatalf("EditConfig: %v", err)
	}
	cfg := ds.candidateTree().find("interfaces", "interface", "config")
	if cfg == nil {
		t.Fatalf("config container missing: %s", ds.GetCandidate())
	}
	if cfg.child("mtu").leafText() != "9000" {
		t.Fatalf("mtu not merged: %s", ds.GetCandidate())
	}
	if cfg.child("enabled") == nil || cfg.child("enabled").leafText() != "true" {
		t.Fatalf("enabled sibling lost: %s", ds.GetCandidate())
	}
	// The single container must not be duplicated.
	if n := len(ds.candidateTree().find("interfaces", "interface").children("config")); n != 1 {
		t.Fatalf("config duplicated (%d): %s", n, ds.GetCandidate())
	}
}

// --- create ---

func TestEditConfigCreateSucceedsWhenAbsent(t *testing.T) {
	ds := seedVlans(t)
	edit := `<vlan xmlns="urn:huawei:vlan"><vlans><vlan operation="create"><id>300</id><name>eng</name></vlan></vlans></vlan>`
	if err := ds.EditConfig([]byte(edit)); err != nil {
		t.Fatalf("EditConfig: %v", err)
	}
	if candidateVlanByID(ds, "300") == nil {
		t.Fatalf("create did not add vlan 300: %s", ds.GetCandidate())
	}
	// The operation attribute must not persist in the datastore.
	if strings.Contains(string(ds.GetCandidate()), "operation=") {
		t.Fatalf("operation attr leaked into store: %s", ds.GetCandidate())
	}
}

func TestEditConfigCreateFailsWhenExists(t *testing.T) {
	ds := seedVlans(t)
	edit := `<vlan xmlns="urn:huawei:vlan"><vlans><vlan operation="create"><id>100</id><name>dup</name></vlan></vlans></vlan>`
	err := ds.EditConfig([]byte(edit))
	if err == nil {
		t.Fatal("expected data-exists error for create on existing entry")
	}
	// Candidate must be unchanged (still original name).
	if v := candidateVlanByID(ds, "100"); v == nil || v.child("name").leafText() != "office" {
		t.Fatalf("failed create must not mutate candidate: %s", ds.GetCandidate())
	}
}

// --- delete / remove ---

func TestEditConfigDeleteRemovesKeyedEntry(t *testing.T) {
	ds := seedVlans(t)
	edit := `<vlan xmlns="urn:huawei:vlan"><vlans><vlan operation="delete"><id>200</id></vlan></vlans></vlan>`
	if err := ds.EditConfig([]byte(edit)); err != nil {
		t.Fatalf("EditConfig: %v", err)
	}
	if candidateVlanByID(ds, "200") != nil {
		t.Fatalf("vlan 200 not deleted: %s", ds.GetCandidate())
	}
	if candidateVlanByID(ds, "100") == nil || candidateVlanCount(ds) != 1 {
		t.Fatalf("delete affected wrong entries: %s", ds.GetCandidate())
	}
}

func TestEditConfigDeleteMissingErrors(t *testing.T) {
	ds := seedVlans(t)
	edit := `<vlan xmlns="urn:huawei:vlan"><vlans><vlan operation="delete"><id>999</id></vlan></vlans></vlan>`
	if err := ds.EditConfig([]byte(edit)); err == nil {
		t.Fatal("expected data-missing error deleting absent entry")
	}
	if candidateVlanCount(ds) != 2 {
		t.Fatalf("failed delete must not mutate candidate: %s", ds.GetCandidate())
	}
}

func TestEditConfigRemoveMissingIsNoop(t *testing.T) {
	ds := seedVlans(t)
	// remove differs from delete: absent target is not an error.
	edit := `<vlan xmlns="urn:huawei:vlan"><vlans><vlan operation="remove"><id>999</id></vlan></vlans></vlan>`
	if err := ds.EditConfig([]byte(edit)); err != nil {
		t.Fatalf("remove of absent entry should be a no-op, got: %v", err)
	}
	if candidateVlanCount(ds) != 2 {
		t.Fatalf("remove no-op must leave candidate intact: %s", ds.GetCandidate())
	}
}

// --- replace ---

func TestEditConfigReplaceSwapsSubtree(t *testing.T) {
	ds := newTreeDatastore()
	base := `<vlan xmlns="urn:huawei:vlan"><vlans>` +
		`<vlan><id>100</id><name>office</name><desc>old</desc></vlan></vlans></vlan>`
	if err := ds.SetRunning([]byte(base)); err != nil {
		t.Fatal(err)
	}
	// replace vlan 100 with a subtree lacking <desc>: desc must disappear.
	edit := `<vlan xmlns="urn:huawei:vlan"><vlans><vlan operation="replace"><id>100</id><name>new</name></vlan></vlans></vlan>`
	if err := ds.EditConfig([]byte(edit)); err != nil {
		t.Fatalf("EditConfig: %v", err)
	}
	v := candidateVlanByID(ds, "100")
	if v == nil || v.child("name").leafText() != "new" {
		t.Fatalf("replace did not set new name: %s", ds.GetCandidate())
	}
	if v.child("desc") != nil {
		t.Fatalf("replace should drop desc: %s", ds.GetCandidate())
	}
}

// --- commit isolation: edits land on candidate only until commit ---

func TestEditConfigDoesNotTouchRunningUntilCommit(t *testing.T) {
	ds := seedVlans(t)
	edit := `<vlan xmlns="urn:huawei:vlan"><vlans><vlan operation="delete"><id>200</id></vlan></vlans></vlan>`
	if err := ds.EditConfig([]byte(edit)); err != nil {
		t.Fatalf("EditConfig: %v", err)
	}
	// running still has both until commit.
	runningVlans := ds.runningTree().find("vlan", "vlans")
	if runningVlans == nil || len(runningVlans.children("vlan")) != 2 {
		t.Fatalf("running mutated before commit: %s", ds.GetRunning())
	}
	if err := ds.Commit(); err != nil {
		t.Fatal(err)
	}
	runningVlans = ds.runningTree().find("vlan", "vlans")
	if len(runningVlans.children("vlan")) != 1 {
		t.Fatalf("commit did not apply delete: %s", ds.GetRunning())
	}
}

// --- malformed edit XML surfaces as an error ---

func TestEditConfigMalformedXMLErrors(t *testing.T) {
	ds := seedVlans(t)
	if err := ds.EditConfig([]byte(`<vlan><bad></vlan>`)); err == nil {
		t.Fatal("expected error for malformed edit-config XML")
	}
}

// 回归（config-delete-semantics 接线暴露）：存量仅 1 个列表条目时，merge 一批含新键的
// 条目不得把新条目误并进已有条目——list 判定除「存量多兄弟」外还须看「edit 内多兄弟」。
// 此前 server 走整树替换掩盖了该缺陷；切到 per-operation EditConfig 后成为主路径。
func TestEditConfigMergeAddsSecondEntryToSingletonList(t *testing.T) {
	ds := newTreeDatastore()
	single := `<vlan xmlns="urn:huawei:vlan"><vlans>` +
		`<vlan><id>10</id><name>ten</name></vlan>` +
		`</vlans></vlan>`
	if err := ds.SetRunning([]byte(single)); err != nil {
		t.Fatalf("SetRunning: %v", err)
	}
	// 全量 desired 形态（union）：既有 10 + 新增 20
	edit := `<vlan xmlns="urn:huawei:vlan"><vlans>` +
		`<vlan><id>10</id><name>ten</name></vlan>` +
		`<vlan><id>20</id><name>twenty</name></vlan>` +
		`</vlans></vlan>`
	if err := ds.EditConfig([]byte(edit)); err != nil {
		t.Fatalf("EditConfig: %v", err)
	}
	if got := candidateVlanCount(ds); got != 2 {
		t.Fatalf("vlan count = %d, want 2 (new entry must not merge into existing)", got)
	}
	v10 := candidateVlanByID(ds, "10")
	if v10 == nil || v10.child("name").leafText() != "ten" {
		t.Fatalf("vlan 10 corrupted: %s", ds.GetCandidate())
	}
	if v20 := candidateVlanByID(ds, "20"); v20 == nil || v20.child("name").leafText() != "twenty" {
		t.Fatalf("vlan 20 missing/corrupted: %s", ds.GetCandidate())
	}
}

// 容器 merge 不受 list 判定影响：单实例容器按名匹配原地并（无 key 概念）。
func TestEditConfigMergeContainerLeafUpdateStillWorks(t *testing.T) {
	ds := newTreeDatastore()
	if err := ds.SetRunning([]byte(`<ifm xmlns="urn:huawei:ifm"><global><statistic-interval>10</statistic-interval></global></ifm>`)); err != nil {
		t.Fatalf("SetRunning: %v", err)
	}
	edit := `<ifm xmlns="urn:huawei:ifm"><global><statistic-interval>5</statistic-interval></global></ifm>`
	if err := ds.EditConfig([]byte(edit)); err != nil {
		t.Fatalf("EditConfig: %v", err)
	}
	ifm := ds.candidateTree().find("ifm")
	if ifm == nil || len(ifm.children("global")) != 1 {
		t.Fatalf("global duplicated or missing: %s", ds.GetCandidate())
	}
	if got := ifm.find("global", "statistic-interval").leafText(); got != "5" {
		t.Fatalf("statistic-interval = %s, want 5", got)
	}
}

// 回归（business-network-config 2PC 集成暴露）：店侧仅一条 keyed 条目时 merge 进
// 不同 key 的新条目，必须按键追加而非就地折叠（否则 id 被改写、原条目丢失）。
// 判定依赖 wellKnownListKeys 的最小模型知识（schema-less 启发式对单-单场景无信号）。
func TestEditConfigMergeSingleEntryStoreAddsNewKeyedEntry(t *testing.T) {
	ds := newTreeDatastore()
	if err := ds.SetRunning([]byte(`<vlan xmlns="urn:huawei:vlan"><vlans><vlan><id>300</id><name>manual</name></vlan></vlans></vlan>`)); err != nil {
		t.Fatalf("SetRunning: %v", err)
	}
	edit := `<vlan xmlns="urn:huawei:vlan"><vlans><vlan><id>100</id><name>biz</name></vlan></vlans></vlan>`
	if err := ds.EditConfig([]byte(edit)); err != nil {
		t.Fatalf("EditConfig: %v", err)
	}
	if got := candidateVlanCount(ds); got != 2 {
		t.Fatalf("vlan count = %d, want 2 (single-entry store must key-match, not fold): %s", got, ds.GetCandidate())
	}
	if v := candidateVlanByID(ds, "300"); v == nil || v.child("name").leafText() != "manual" {
		t.Fatalf("manual vlan 300 corrupted: %s", ds.GetCandidate())
	}
	if v := candidateVlanByID(ds, "100"); v == nil || v.child("name").leafText() != "biz" {
		t.Fatalf("new vlan 100 missing: %s", ds.GetCandidate())
	}
}

// 同场景的 interfaces/interface（键=name）：单条店侧接口 + 新接口 merge。
func TestEditConfigMergeSingleInterfaceStoreAddsNew(t *testing.T) {
	ds := newTreeDatastore()
	if err := ds.SetRunning([]byte(`<ifm xmlns="urn:huawei:ifm"><interfaces><interface><name>GE0/0/1</name><mtu>1500</mtu></interface></interfaces></ifm>`)); err != nil {
		t.Fatalf("SetRunning: %v", err)
	}
	edit := `<ifm xmlns="urn:huawei:ifm"><interfaces><interface><name>GE0/0/2</name><mtu>9000</mtu></interface></interfaces></ifm>`
	if err := ds.EditConfig([]byte(edit)); err != nil {
		t.Fatalf("EditConfig: %v", err)
	}
	ifaces := ds.candidateTree().find("ifm", "interfaces")
	if ifaces == nil || len(ifaces.children("interface")) != 2 {
		t.Fatalf("interface count wrong (fold?): %s", ds.GetCandidate())
	}
}
