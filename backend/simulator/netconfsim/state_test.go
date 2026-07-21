package netconfsim

import (
	"strings"
	"sync"
	"testing"
)

// NS-08 状态数据 overlay：treeDatastore 第三棵树（state）与 <get> 合并语义。
// get = running+state 合并（list 条目按键并入、无配置匹配的 list 条目丢弃、
// 纯状态容器并入、同名叶状态覆盖）；get-config 不含状态；写操作不触碰状态树。

const (
	stateIfmRunning = `<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm">
  <interfaces>
    <interface><name>200GE0/1/0</name><mtu>9216</mtu><description>uplink</description></interface>
    <interface><name>200GE0/1/1</name><mtu>9216</mtu></interface>
  </interfaces>
</ifm>`

	stateIfmOverlay = `<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm">
  <interfaces>
    <interface>
      <name>200GE0/1/0</name>
      <dynamic><oper-status>1</oper-status><mac-address>00:11:22:33:44:55</mac-address></dynamic>
    </interface>
  </interfaces>
</ifm>`
)

func mustSetRunning(t *testing.T, ds *treeDatastore, xml string) {
	t.Helper()
	if err := ds.SetRunning([]byte(xml)); err != nil {
		t.Fatalf("SetRunning: %v", err)
	}
}

func mustSetState(t *testing.T, ds *treeDatastore, xml string) {
	t.Helper()
	if err := ds.SetState([]byte(xml)); err != nil {
		t.Fatalf("SetState: %v", err)
	}
}

func mustGetFiltered(t *testing.T, ds *treeDatastore, filter string) string {
	t.Helper()
	out, err := ds.GetFiltered([]byte(filter))
	if err != nil {
		t.Fatalf("GetFiltered: %v", err)
	}
	return string(out)
}

// 状态子树按 list 键并入对应配置条目；get-config 不受影响。
func TestStateOverlayMergesIntoListEntry(t *testing.T) {
	ds := newTreeDatastore()
	mustSetRunning(t, ds, stateIfmRunning)
	mustSetState(t, ds, stateIfmOverlay)

	got := treeFromXML(t, mustGetFiltered(t, ds, ""))
	ifaces := got.find("ifm", "interfaces").children("interface")
	if len(ifaces) != 2 {
		t.Fatalf("want 2 interfaces after merge, got %d", len(ifaces))
	}
	var withState *dataNode
	for _, it := range ifaces {
		if k := it.child("name"); k != nil && k.leafText() == "200GE0/1/0" {
			withState = it
		}
	}
	if withState == nil {
		t.Fatal("interface 200GE0/1/0 missing from merged get result")
	}
	dyn := withState.child("dynamic")
	if dyn == nil {
		t.Fatalf("dynamic state container not merged into its interface: %s", mustGetFiltered(t, ds, ""))
	}
	if v := dyn.child("oper-status"); v == nil || v.leafText() != "1" {
		t.Fatalf("oper-status not merged, dynamic=%v", dyn)
	}
	// 配置叶保留（合并不是替换）
	if v := withState.child("mtu"); v == nil || v.leafText() != "9216" {
		t.Fatal("config leaf mtu lost during state merge")
	}

	// get-config 恒不含状态叶
	cfg, err := ds.GetConfigFiltered(nil)
	if err != nil {
		t.Fatalf("GetConfigFiltered: %v", err)
	}
	if strings.Contains(string(cfg), "dynamic") {
		t.Fatalf("get-config leaked state data: %s", cfg)
	}
}

// 状态树中无配置匹配的 keyed list 条目丢弃——配置条目删除后状态不得复活为幽灵条目。
func TestStateOverlayDropsUnmatchedListEntry(t *testing.T) {
	ds := newTreeDatastore()
	mustSetRunning(t, ds, stateIfmRunning)
	mustSetState(t, ds, `<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm">
  <interfaces>
    <interface><name>GhostIf</name><dynamic><oper-status>2</oper-status></dynamic></interface>
  </interfaces>
</ifm>`)

	got := treeFromXML(t, mustGetFiltered(t, ds, ""))
	for _, it := range got.find("ifm", "interfaces").children("interface") {
		if k := it.child("name"); k != nil && k.leafText() == "GhostIf" {
			t.Fatalf("ghost list entry from state overlay must be dropped: %s", mustGetFiltered(t, ds, ""))
		}
	}
}

// 纯状态容器（配置树中不存在）照常并入：既有容器下的新子容器与全新顶层容器都成立。
func TestStateOverlayCreatesPureStateContainers(t *testing.T) {
	ds := newTreeDatastore()
	mustSetRunning(t, ds, stateIfmRunning)
	mustSetState(t, ds, `<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm">
  <ipv4-interface-count><total>2</total></ipv4-interface-count>
</ifm>
<sysstat xmlns="urn:example:sysstat"><cpu-usage>17</cpu-usage></sysstat>`)

	got := treeFromXML(t, mustGetFiltered(t, ds, ""))
	if n := got.find("ifm", "ipv4-interface-count", "total"); n == nil || n.leafText() != "2" {
		t.Fatalf("state-only container under existing config container missing: %s", mustGetFiltered(t, ds, ""))
	}
	if n := got.find("sysstat", "cpu-usage"); n == nil || n.leafText() != "17" {
		t.Fatalf("state-only top-level container missing: %s", mustGetFiltered(t, ds, ""))
	}
}

// 同名叶：状态树取胜（状态语义覆盖配置回显值）。
func TestStateOverlayLeafStateWins(t *testing.T) {
	ds := newTreeDatastore()
	mustSetRunning(t, ds, stateIfmRunning)
	mustSetState(t, ds, `<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm">
  <interfaces>
    <interface><name>200GE0/1/0</name><mtu>1500</mtu></interface>
  </interfaces>
</ifm>`)

	got := treeFromXML(t, mustGetFiltered(t, ds, ""))
	for _, it := range got.find("ifm", "interfaces").children("interface") {
		if k := it.child("name"); k != nil && k.leafText() == "200GE0/1/0" {
			if v := it.child("mtu"); v == nil || v.leafText() != "1500" {
				t.Fatalf("state leaf must override config leaf, got %v", it.child("mtu"))
			}
			return
		}
	}
	t.Fatal("interface 200GE0/1/0 missing")
}

// 未注入状态时 get 与 get-config 等值。
func TestStateOverlayGetEqualsGetConfigWithoutState(t *testing.T) {
	ds := newTreeDatastore()
	mustSetRunning(t, ds, stateIfmRunning)

	g := mustGetFiltered(t, ds, "")
	c, err := ds.GetConfigFiltered(nil)
	if err != nil {
		t.Fatalf("GetConfigFiltered: %v", err)
	}
	if !nodesEqual(treeFromXML(t, g), treeFromXML(t, string(c))) {
		t.Fatalf("get without state must equal get-config:\nget=%s\nget-config=%s", g, c)
	}
}

// 写操作（EditConfig/Commit/Discard）不触碰状态树，且状态并入的是最新 running。
func TestStateOverlaySurvivesConfigWrites(t *testing.T) {
	ds := newTreeDatastore()
	mustSetRunning(t, ds, stateIfmRunning)
	mustSetState(t, ds, stateIfmOverlay)

	if err := ds.EditConfig([]byte(`<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm">
  <interfaces><interface><name>200GE0/1/0</name><description>rewired</description></interface></interfaces>
</ifm>`)); err != nil {
		t.Fatalf("EditConfig: %v", err)
	}
	if err := ds.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	got := treeFromXML(t, mustGetFiltered(t, ds, ""))
	for _, it := range got.find("ifm", "interfaces").children("interface") {
		if k := it.child("name"); k != nil && k.leafText() == "200GE0/1/0" {
			if d := it.child("description"); d == nil || d.leafText() != "rewired" {
				t.Fatal("merged get must reflect updated running config")
			}
			if it.child("dynamic") == nil {
				t.Fatal("state overlay lost after edit-config+commit")
			}
		}
	}

	ds.DiscardCandidate()
	if !strings.Contains(mustGetFiltered(t, ds, ""), "dynamic") {
		t.Fatal("state overlay lost after discard")
	}
}

// 合并不得污染 running：get 之后 get-config 仍不含状态叶（clone 隔离）。
func TestStateOverlayMergeDoesNotMutateRunning(t *testing.T) {
	ds := newTreeDatastore()
	mustSetRunning(t, ds, stateIfmRunning)
	mustSetState(t, ds, stateIfmOverlay)

	_ = mustGetFiltered(t, ds, "")
	cfg, err := ds.GetConfigFiltered(nil)
	if err != nil {
		t.Fatalf("GetConfigFiltered: %v", err)
	}
	if strings.Contains(string(cfg), "dynamic") {
		t.Fatalf("merge mutated running tree: %s", cfg)
	}
}

// get 套 subtree filter：按键选单条目，合并后的状态子树随之返回。
func TestStateOverlayFilteredGet(t *testing.T) {
	ds := newTreeDatastore()
	mustSetRunning(t, ds, stateIfmRunning)
	mustSetState(t, ds, stateIfmOverlay)

	out := mustGetFiltered(t, ds, `<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm">
  <interfaces><interface><name>200GE0/1/0</name></interface></interfaces>
</ifm>`)
	got := treeFromXML(t, out)
	ifaces := got.find("ifm", "interfaces").children("interface")
	if len(ifaces) != 1 {
		t.Fatalf("filter should select exactly 1 interface, got %d: %s", len(ifaces), out)
	}
	if ifaces[0].child("dynamic") == nil {
		t.Fatalf("filtered get must include merged state subtree: %s", out)
	}
}

// running 树带 <config> 包裹（SetRunningConfig/带壳 XML 种子的历史形态）时，
// 合并必须在模块容器层进行——状态树不得被当「纯状态容器」追加成顶层兄弟，
// <get> 输出以模块容器为顶层（无 <config> 壳）。
func TestStateOverlayMergesThroughConfigWrapper(t *testing.T) {
	ds := newTreeDatastore()
	mustSetRunning(t, ds, "<config>"+stateIfmRunning+"</config>")
	mustSetState(t, ds, stateIfmOverlay)

	out := mustGetFiltered(t, ds, "")
	got := treeFromXML(t, out)
	if got.child("config") != nil {
		t.Fatalf("<get> output must unwrap the <config> wrapper: %s", out)
	}
	ifaces := got.find("ifm", "interfaces").children("interface")
	if len(ifaces) != 2 {
		t.Fatalf("want 2 interfaces after merge through wrapper, got %d: %s", len(ifaces), out)
	}
	var withState *dataNode
	for _, it := range ifaces {
		if k := it.child("name"); k != nil && k.leafText() == "200GE0/1/0" {
			withState = it
		}
	}
	if withState == nil || withState.child("dynamic") == nil {
		t.Fatalf("state must merge into the wrapped config entry (not a top-level sibling): %s", out)
	}
	// 配置叶必须还在——若状态树被当独立树返回则 mtu 丢失
	if v := withState.child("mtu"); v == nil || v.leafText() != "9216" {
		t.Fatalf("config leaf lost when merging through wrapper: %s", out)
	}

	// 带 filter 的读同样不得命中「状态树顶层兄弟」旁路
	filtered := mustGetFiltered(t, ds, `<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm"><interfaces><interface><name>200GE0/1/0</name></interface></interfaces></ifm>`)
	fgot := treeFromXML(t, filtered)
	fifaces := fgot.find("ifm", "interfaces").children("interface")
	if len(fifaces) != 1 || fifaces[0].child("dynamic") == nil || fifaces[0].child("mtu") == nil {
		t.Fatalf("filtered get through wrapper must return merged entry with config+state: %s", filtered)
	}
}

// SetState 畸形 XML 报错且不改变已有状态。
func TestStateOverlaySetStateError(t *testing.T) {
	ds := newTreeDatastore()
	mustSetRunning(t, ds, stateIfmRunning)
	mustSetState(t, ds, stateIfmOverlay)

	if err := ds.SetState([]byte(`<ifm><bad></ifm>`)); err == nil {
		t.Fatal("expected error for malformed state XML")
	}
	if !strings.Contains(mustGetFiltered(t, ds, ""), "dynamic") {
		t.Fatal("failed SetState must not clobber existing state")
	}
}

// 并发 get×写×SetState：-race 干净（R09）。
func TestStateOverlayConcurrent(t *testing.T) {
	ds := newTreeDatastore()
	mustSetRunning(t, ds, stateIfmRunning)
	mustSetState(t, ds, stateIfmOverlay)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(4)
		go func() { defer wg.Done(); _, _ = ds.GetFiltered(nil) }()
		go func() { defer wg.Done(); _ = ds.SetState([]byte(stateIfmOverlay)) }()
		go func() { defer wg.Done(); _ = ds.SetCandidate([]byte(stateIfmRunning)) }()
		go func() { defer wg.Done(); _ = ds.Commit() }()
	}
	wg.Wait()
}
