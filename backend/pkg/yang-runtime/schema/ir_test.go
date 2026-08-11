package schema

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"strings"
	"testing"
)

// buildFullSchema hand-builds a DefaultSchema exercising every node kind and
// every serialized field: container (presence/when/must/op-excludes/read-only),
// list (keys/user-ordered), leaf (all scalar kinds, enum, pattern, range,
// length, units, default, mandatory, leaf-list flag, ext flags), choice/case.
func buildFullSchema(t *testing.T) *DefaultSchema {
	t.Helper()

	root := NewContainer("vlan", "vlan root", "/vlan", nil, false).(*defaultContainer)
	root.opExcludes = []string{"delete"}

	vlans := NewContainer("vlans", "vlan table holder", "/vlan/vlans", root, false).(*defaultContainer)
	root.AddChild(vlans)

	id := NewLeaf("id", "vlan id", "/vlan/vlans/vlan/id", nil, LeafTypeUint16, true, true).(*defaultLeaf)
	id.rangeMin, id.hasMin = 1, true
	id.rangeMax, id.hasMax = 4094, true

	entry := NewList("vlan", "one vlan", "/vlan/vlans/vlan", vlans, nil, true).(*defaultList)
	entry.whenExpr = "../enabled = 'true'"
	entry.mustExprs = []string{"count(.) < 4094", "id != 1"}
	entry.opExcludes = []string{"update", "delete"}
	id.parent = entry
	entry.AddChild(id)
	entry.keys = []LeafNode{id}
	vlans.AddChild(entry)

	name := NewLeaf("name", "vlan name", "/vlan/vlans/vlan/name", entry, LeafTypeString, false, false).(*defaultLeaf)
	name.pattern = `[a-zA-Z0-9_-]+`
	name.lengthMin, name.hasLenMin = 1, true
	name.lengthMax, name.hasLenMax = 31, true
	name.defaultValue = "VLAN"
	name.supportFilter = true
	name.dynamicDefault = true
	name.whenExpr = "../id > 1"
	name.mustExprs = []string{"string-length(.) > 0", ". != 'default'"}
	name.opExcludes = []string{"update", "delete"}
	entry.AddChild(name)

	mode := NewLeaf("mode", "vlan mode", "/vlan/vlans/vlan/mode", entry, LeafTypeEnum, false, false).(*defaultLeaf)
	mode.enumValues = []string{"access", "trunk", "hybrid"}
	mode.units = "mode"
	entry.AddChild(mode)

	mtu := NewLeaf("mtu", "mtu", "/vlan/vlans/vlan/mtu", entry, LeafTypeInt32, false, false).(*defaultLeaf)
	mtu.rangeMin, mtu.hasMin = 64, true // 单侧界：只有 min
	entry.AddChild(mtu)

	tags := NewLeaf("tags", "leaf-list of tags", "/vlan/vlans/vlan/tags", entry, LeafTypeString, false, false).(*defaultLeaf)
	tags.leafList = true
	entry.AddChild(tags)

	// presence 容器 + when/must
	feat := NewContainer("suppression", "presence feature", "/vlan/vlans/vlan/suppression", entry, true).(*defaultContainer)
	feat.whenExpr = "../mode = 'access'"
	feat.mustExprs = []string{"enable = 'true'"}
	entry.AddChild(feat)

	rate := NewLeaf("rate", "suppress rate", "/vlan/vlans/vlan/suppression/rate", feat, LeafTypeUint64, false, false).(*defaultLeaf)
	feat.AddChild(rate)

	// config-false 子树（继承只读）
	state := NewContainer("state", "state subtree", "/vlan/vlans/vlan/state", entry, false).(*defaultContainer)
	state.readOnly = true
	entry.AddChild(state)
	pkts := NewLeaf("packets", "counter", "/vlan/vlans/vlan/state/packets", state, LeafTypeUint64, false, false).(*defaultLeaf)
	pkts.readOnly = true
	state.AddChild(pkts)

	// choice/case（数据路径拍平在宿主容器上）
	ch := NewChoice("assign", "assign choice", "/vlan/vlans/vlan/assign", entry).(*defaultChoice)
	ch.defaultCase = "static"
	caseA := NewCase("static", "static case", "/vlan/vlans/vlan/static", ch).(*defaultCase)
	caseALeaf := NewLeaf("static-id", "static id", "/vlan/vlans/vlan/static-id", caseA, LeafTypeUint32, false, false)
	caseA.AddChild(caseALeaf)
	// case 内嵌套 choice（entry.go caseMember 支持的形状，IR 必须存活）
	inner := NewChoice("inner", "nested choice", "/vlan/vlans/vlan/inner", caseA).(*defaultChoice)
	innerCase := NewCase("only", "inner case", "/vlan/vlans/vlan/only", inner).(*defaultCase)
	innerCase.AddChild(NewLeaf("deep", "deep leaf", "/vlan/vlans/vlan/deep", innerCase, LeafTypeBoolean, false, false))
	inner.AddCase(innerCase)
	caseA.AddChild(inner)
	ch.AddCase(caseA)
	caseB := NewCase("dynamic", "dynamic case", "/vlan/vlans/vlan/dynamic", ch).(*defaultCase)
	caseB.AddChild(NewLeaf("pool", "pool name", "/vlan/vlans/vlan/pool", caseB, LeafTypeString, false, false))
	ch.AddCase(caseB)
	entry.AddChild(ch)

	// 其余标量类型覆盖
	for _, tc := range []struct {
		n  string
		lt LeafType
	}{
		{"b", LeafTypeBoolean}, {"i8", LeafTypeInt8}, {"i16", LeafTypeInt16},
		{"i64", LeafTypeInt64}, {"u8", LeafTypeUint8}, {"u32", LeafTypeUint32},
		{"d64", LeafTypeDecimal64}, {"bits", LeafTypeBits}, {"e", LeafTypeEmpty},
	} {
		root.AddChild(NewLeaf(tc.n, "", "/vlan/"+tc.n, root, tc.lt, false, false))
	}

	m := NewModule("vlan", "urn:huawei:yang:huawei-vlan", "2023-01-01", root).(*defaultModule)
	m.vendor = "huawei"

	// 第二个模块：验证多模块排序确定性
	root2 := NewContainer("ifm", "", "/ifm", nil, false).(*defaultContainer)
	m2 := NewModule("ifm", "urn:huawei:yang:huawei-ifm", "", root2).(*defaultModule)
	m2.vendor = "huawei"

	ds := NewSchema()
	ds.AddModule(m)
	ds.AddModule(m2)
	return ds
}

func TestIRRoundTrip(t *testing.T) {
	src := buildFullSchema(t)
	blob, err := EncodeIR(src)
	if err != nil {
		t.Fatalf("EncodeIR: %v", err)
	}
	got, err := DecodeIR(blob)
	if err != nil {
		t.Fatalf("DecodeIR: %v", err)
	}
	compareSchemas(t, src, got)
}

func TestIRDeterministic(t *testing.T) {
	src := buildFullSchema(t)
	a, err := EncodeIR(src)
	if err != nil {
		t.Fatalf("EncodeIR: %v", err)
	}
	b, err := EncodeIR(src)
	if err != nil {
		t.Fatalf("EncodeIR twice: %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("EncodeIR is not deterministic: two encodings differ")
	}
}

func TestIRVersionMismatch(t *testing.T) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if err := json.NewEncoder(zw).Encode(map[string]interface{}{
		"version": 9999, "modules": []interface{}{},
	}); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	_, err := DecodeIR(buf.Bytes())
	if err == nil {
		t.Fatal("DecodeIR accepted mismatched version; want explicit error")
	}
	if !strings.Contains(err.Error(), "version") {
		t.Fatalf("version mismatch error should name the version problem, got: %v", err)
	}
}

// TestIRMalformedDecode：解码侧 fail-fast 分支逐一触达（R08——宁可报错不半解析）。
func TestIRMalformedDecode(t *testing.T) {
	gz := func(t *testing.T, v interface{}) []byte {
		t.Helper()
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		if err := json.NewEncoder(zw).Encode(v); err != nil {
			t.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}
	mod := func(root map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"version": 1,
			"modules": []interface{}{map[string]interface{}{"name": "m", "root": root}},
		}
	}
	cases := []struct {
		name    string
		payload map[string]interface{}
		wantSub string
	}{
		{"missing root", map[string]interface{}{
			"version": 1, "modules": []interface{}{map[string]interface{}{"name": "m"}},
		}, "missing root"},
		{"root not container", mod(map[string]interface{}{
			"kind": "leaf", "name": "x", "path": "/x", "leafType": "string",
		}), "want container"},
		{"unknown kind", mod(map[string]interface{}{
			"kind": "container", "name": "c", "path": "/c",
			"children": []interface{}{map[string]interface{}{"kind": "wat", "name": "x", "path": "/c/x"}},
		}), "unknown node kind"},
		{"unknown leaf type", mod(map[string]interface{}{
			"kind": "container", "name": "c", "path": "/c",
			"children": []interface{}{map[string]interface{}{"kind": "leaf", "name": "x", "path": "/c/x", "leafType": "wat"}},
		}), "unknown leaf type"},
		{"key not among children", mod(map[string]interface{}{
			"kind": "container", "name": "c", "path": "/c",
			"children": []interface{}{map[string]interface{}{
				"kind": "list", "name": "l", "path": "/c/l", "keys": []string{"nope"},
			}},
		}), "not among children"},
		{"key not a leaf", mod(map[string]interface{}{
			"kind": "container", "name": "c", "path": "/c",
			"children": []interface{}{map[string]interface{}{
				"kind": "list", "name": "l", "path": "/c/l", "keys": []string{"sub"},
				"children": []interface{}{map[string]interface{}{"kind": "container", "name": "sub", "path": "/c/l/sub"}},
			}},
		}), "want leaf"},
		{"choice member not case", mod(map[string]interface{}{
			"kind": "container", "name": "c", "path": "/c",
			"children": []interface{}{map[string]interface{}{
				"kind": "choice", "name": "ch", "path": "/c/ch",
				"cases": []interface{}{map[string]interface{}{"kind": "leaf", "name": "x", "path": "/c/x", "leafType": "string"}},
			}},
		}), "want case"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeIR(gz(t, tc.payload))
			if err == nil {
				t.Fatalf("DecodeIR accepted malformed input (%s); want error", tc.name)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantSub)
			}
		})
	}
}

func TestIRGarbageInput(t *testing.T) {
	for _, in := range [][]byte{nil, {}, []byte("junk not gzip")} {
		if _, err := DecodeIR(in); err == nil {
			t.Fatalf("DecodeIR(%q) accepted garbage; want error", in)
		}
	}
}

// TestIRKeyIdentity: 反序列化后 list 的 Keys() 与同名 Child() 必须是同一对象
// （entry.go 构建的树即如此，消费方依赖指针同一性判 key）。
func TestIRKeyIdentity(t *testing.T) {
	src := buildFullSchema(t)
	blob, err := EncodeIR(src)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeIR(blob)
	if err != nil {
		t.Fatal(err)
	}
	n, ok := got.Path("/vlan/vlans/vlan")
	if !ok {
		t.Fatal("path cache: /vlan/vlans/vlan not found after decode")
	}
	l := n.(ListNode)
	if len(l.Keys()) != 1 {
		t.Fatalf("keys = %d, want 1", len(l.Keys()))
	}
	child, ok := l.Child("id")
	if !ok {
		t.Fatal("list child id missing")
	}
	if l.Keys()[0] != child {
		t.Fatal("Keys()[0] and Child(\"id\") are different objects; want identical pointer")
	}
	if !l.Keys()[0].IsKey() {
		t.Fatal("decoded key leaf lost IsKey")
	}
}

func TestIRParentPointers(t *testing.T) {
	src := buildFullSchema(t)
	blob, err := EncodeIR(src)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeIR(blob)
	if err != nil {
		t.Fatal(err)
	}
	n, ok := got.Path("/vlan/vlans/vlan/suppression/rate")
	if !ok {
		t.Fatal("deep path not found after decode")
	}
	if n.Parent() == nil || n.Parent().Name() != "suppression" {
		t.Fatalf("parent = %v, want suppression", n.Parent())
	}
	if n.Parent().Parent() == nil || n.Parent().Parent().Path() != "/vlan/vlans/vlan" {
		t.Fatal("grandparent chain broken after decode")
	}
}

// ---- deep comparison helpers ----

func compareSchemas(t *testing.T, want, got *DefaultSchema) {
	t.Helper()
	wm, gm := want.Modules(), got.Modules()
	if len(wm) != len(gm) {
		t.Fatalf("module count: got %d want %d", len(gm), len(wm))
	}
	for _, w := range wm {
		g, ok := got.Module(w.Name())
		if !ok {
			t.Fatalf("module %s missing after decode", w.Name())
		}
		if g.Namespace() != w.Namespace() || g.Revision() != w.Revision() || g.Vendor() != w.Vendor() {
			t.Fatalf("module %s meta mismatch: got (%s,%s,%s) want (%s,%s,%s)", w.Name(),
				g.Namespace(), g.Revision(), g.Vendor(), w.Namespace(), w.Revision(), w.Vendor())
		}
		compareNodes(t, w.Root(), g.Root())
	}
}

func compareNodes(t *testing.T, want, got Node) {
	t.Helper()
	if want.Name() != got.Name() || want.Path() != got.Path() || want.Type() != got.Type() ||
		want.Description() != got.Description() || want.ReadOnly() != got.ReadOnly() {
		t.Fatalf("node mismatch at %s: got (%s,%s,%d,ro=%v) want (%s,%s,%d,ro=%v)",
			want.Path(), got.Name(), got.Path(), got.Type(), got.ReadOnly(),
			want.Name(), want.Path(), want.Type(), want.ReadOnly())
	}
	switch w := want.(type) {
	case ListNode:
		g := got.(ListNode)
		if w.IsUserOrdered() != g.IsUserOrdered() {
			t.Fatalf("%s: userOrdered mismatch", w.Path())
		}
		if len(w.Keys()) != len(g.Keys()) {
			t.Fatalf("%s: key count %d != %d", w.Path(), len(g.Keys()), len(w.Keys()))
		}
		for i := range w.Keys() {
			if w.Keys()[i].Name() != g.Keys()[i].Name() {
				t.Fatalf("%s: key[%d] %s != %s", w.Path(), i, g.Keys()[i].Name(), w.Keys()[i].Name())
			}
		}
		compareContainerish(t, w.Path(), wListMeta(w), gListMeta(g))
		compareChildren(t, w.Children(), g.Children(), w.Path())
	case ContainerNode:
		g := got.(ContainerNode)
		if w.IsPresence() != g.IsPresence() {
			t.Fatalf("%s: presence mismatch", w.Path())
		}
		compareContainerish(t, w.Path(),
			nodeMeta{w.WhenExpr(), w.MustExprs(), w.OperationExcludes()},
			nodeMeta{g.WhenExpr(), g.MustExprs(), g.OperationExcludes()})
		compareChildren(t, w.Children(), g.Children(), w.Path())
	case ChoiceNode:
		g := got.(ChoiceNode)
		if w.DefaultCase() != g.DefaultCase() {
			t.Fatalf("%s: defaultCase %q != %q", w.Path(), g.DefaultCase(), w.DefaultCase())
		}
		wc, gc := w.Cases(), g.Cases()
		if len(wc) != len(gc) {
			t.Fatalf("%s: case count %d != %d", w.Path(), len(gc), len(wc))
		}
		for i := range wc {
			compareNodes(t, wc[i], gc[i])
		}
	case CaseNode:
		g := got.(CaseNode)
		compareChildren(t, w.Children(), g.Children(), w.Path())
	case LeafNode:
		g := got.(LeafNode)
		compareLeaf(t, w, g)
	}
}

type nodeMeta struct {
	when       string
	must       []string
	opExcludes []string
}

func wListMeta(l ListNode) nodeMeta {
	c := l.(interface {
		WhenExpr() string
		MustExprs() []string
	})
	return nodeMeta{c.WhenExpr(), c.MustExprs(), l.OperationExcludes()}
}

func gListMeta(l ListNode) nodeMeta { return wListMeta(l) }

func compareContainerish(t *testing.T, path string, w, g nodeMeta) {
	t.Helper()
	if w.when != g.when {
		t.Fatalf("%s: when %q != %q", path, g.when, w.when)
	}
	if strings.Join(w.must, "\x00") != strings.Join(g.must, "\x00") {
		t.Fatalf("%s: must %v != %v", path, g.must, w.must)
	}
	if strings.Join(w.opExcludes, "\x00") != strings.Join(g.opExcludes, "\x00") {
		t.Fatalf("%s: opExcludes %v != %v", path, g.opExcludes, w.opExcludes)
	}
}

func compareChildren(t *testing.T, want, got []Node, path string) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("%s: child count %d != %d", path, len(got), len(want))
	}
	for i := range want {
		compareNodes(t, want[i], got[i])
	}
}

func compareLeaf(t *testing.T, w, g LeafNode) {
	t.Helper()
	p := w.Path()
	if w.LeafType() != g.LeafType() || w.IsKey() != g.IsKey() || w.Mandatory() != g.Mandatory() ||
		w.Units() != g.Units() || w.Pattern() != g.Pattern() || w.WhenExpr() != g.WhenExpr() ||
		w.IsLeafList() != g.IsLeafList() || w.SupportFilter() != g.SupportFilter() ||
		w.DynamicDefault() != g.DynamicDefault() {
		t.Fatalf("%s: leaf scalar meta mismatch", p)
	}
	if w.DefaultValue() != g.DefaultValue() {
		t.Fatalf("%s: default %v != %v", p, g.DefaultValue(), w.DefaultValue())
	}
	if strings.Join(w.EnumValues(), "\x00") != strings.Join(g.EnumValues(), "\x00") {
		t.Fatalf("%s: enums %v != %v", p, g.EnumValues(), w.EnumValues())
	}
	if strings.Join(w.MustExprs(), "\x00") != strings.Join(g.MustExprs(), "\x00") {
		t.Fatalf("%s: must %v != %v", p, g.MustExprs(), w.MustExprs())
	}
	if strings.Join(w.OperationExcludes(), "\x00") != strings.Join(g.OperationExcludes(), "\x00") {
		t.Fatalf("%s: opExcludes %v != %v", p, g.OperationExcludes(), w.OperationExcludes())
	}
	cmpBound := func(label string, wf, gf func() (int, bool)) {
		wv, wok := wf()
		gv, gok := gf()
		if wv != gv || wok != gok {
			t.Fatalf("%s: %s (%d,%v) != (%d,%v)", p, label, gv, gok, wv, wok)
		}
	}
	cmpBound("rangeMin", w.RangeMin, g.RangeMin)
	cmpBound("rangeMax", w.RangeMax, g.RangeMax)
	wd, gd := w.(*defaultLeaf), g.(*defaultLeaf)
	cmpBound("lengthMin", wd.LengthMin, gd.LengthMin)
	cmpBound("lengthMax", wd.LengthMax, gd.LengthMax)
}
