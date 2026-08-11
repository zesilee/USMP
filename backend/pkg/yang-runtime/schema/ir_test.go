package schema

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"strings"
	"testing"
)

// 冒烟：最小树 round-trip + 版本门禁。完整测试矩阵（全字段深比较/确定性/
// 畸形表格/指针同一性）在后续 commit 补齐（≤500 行提交拆分）。
func TestIRSmokeRoundTrip(t *testing.T) {
	root := NewContainer("vlan", "root", "/vlan", nil, false).(*defaultContainer)
	id := NewLeaf("id", "vlan id", "/vlan/vlan/id", nil, LeafTypeUint16, true, true).(*defaultLeaf)
	l := NewList("vlan", "entry", "/vlan/vlan", root, nil, false).(*defaultList)
	id.parent = l
	l.AddChild(id)
	l.keys = []LeafNode{id}
	root.AddChild(l)
	m := NewModule("vlan", "urn:x", "", root).(*defaultModule)
	m.vendor = "huawei"
	ds := NewSchema()
	ds.AddModule(m)

	blob, err := EncodeIR(ds)
	if err != nil {
		t.Fatalf("EncodeIR: %v", err)
	}
	got, err := DecodeIR(blob)
	if err != nil {
		t.Fatalf("DecodeIR: %v", err)
	}
	gm, ok := got.Module("vlan")
	if !ok || gm.Vendor() != "huawei" {
		t.Fatalf("module lost in round-trip: %+v", gm)
	}
	n, ok := got.Path("/vlan/vlan")
	if !ok {
		t.Fatal("path cache missing /vlan/vlan")
	}
	gl := n.(ListNode)
	if len(gl.Keys()) != 1 || !gl.Keys()[0].IsKey() || gl.Keys()[0].LeafType() != LeafTypeUint16 {
		t.Fatalf("key leaf mismatch: %+v", gl.Keys())
	}
}

func TestIRSmokeVersionMismatch(t *testing.T) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if err := json.NewEncoder(zw).Encode(map[string]interface{}{"version": 9999}); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	_, err := DecodeIR(buf.Bytes())
	if err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("want version error, got %v", err)
	}
}
