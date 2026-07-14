package xmlcodec

import (
	"reflect"
	"strings"
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/internal/testutil/hwfix"
)

// TestDecodeRoundTrip：Encode→Decode 往返恒等（XC-02 核心不变量：可下发字段
// 必可回读，消除 build/parse 字段不对称的字段级永久漂移）。
func TestDecodeRoundTrip(t *testing.T) {
	t.Run("vlan full", func(t *testing.T) {
		src := hwfix.VlanFull()
		out, err := Encode(vlanSpec(), src)
		if err != nil {
			t.Fatal(err)
		}
		got := &huawei.HuaweiVlan_Vlan_Vlans{}
		if err := Decode(vlanSpec(), []byte(out), got); err != nil {
			t.Fatalf("Decode: %v", err)
		}
		if !reflect.DeepEqual(src, got) {
			t.Errorf("round trip diverged\n src: %+v\n got: %+v\nxml: %s", src, got, out)
		}
	})
	t.Run("ifm full", func(t *testing.T) {
		src := hwfix.IfmFull()
		out, err := Encode(ifmSpec(), src)
		if err != nil {
			t.Fatal(err)
		}
		got := &huawei.HuaweiIfm_Ifm_Interfaces{}
		if err := Decode(ifmSpec(), []byte(out), got); err != nil {
			t.Fatalf("Decode: %v", err)
		}
		if !reflect.DeepEqual(src, got) {
			t.Errorf("round trip diverged\n src: %+v\n got: %+v\nxml: %s", src, got, out)
		}
	})
}

// TestDecodeWrappedForms：rpc-reply/data 包裹、namespace 前缀、裸容器三形态
// 解析结果一致（legacy token 扫描的鲁棒性保持）。
func TestDecodeWrappedForms(t *testing.T) {
	inner := `<vlan><id>10</id><name>mgmt</name><admin-status>1</admin-status></vlan>`
	forms := map[string]string{
		"bare":     `<vlans>` + inner + `</vlans>`,
		"wrapped":  `<rpc-reply><data><vlans xmlns="` + vlanNS + `">` + inner + `</vlans></data></rpc-reply>`,
		"prefixed": `<h:vlans xmlns:h="` + vlanNS + `"><h:vlan><h:id>10</h:id><h:name>mgmt</h:name><h:admin-status>1</h:admin-status></h:vlan></h:vlans>`,
	}
	var want *huawei.HuaweiVlan_Vlan_Vlans
	for name, doc := range forms {
		got := &huawei.HuaweiVlan_Vlan_Vlans{}
		if err := Decode(vlanSpec(), []byte(doc), got); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(got.Vlan) != 1 || got.Vlan[10] == nil || got.Vlan[10].Name == nil || *got.Vlan[10].Name != "mgmt" {
			t.Fatalf("%s: bad decode result: %+v", name, got)
		}
		if want == nil {
			want = got
		} else if !reflect.DeepEqual(want, got) {
			t.Errorf("%s: differs from other forms", name)
		}
	}
}

func TestDecodeEdgeCases(t *testing.T) {
	t.Run("empty input non-nil container", func(t *testing.T) {
		got := &huawei.HuaweiVlan_Vlan_Vlans{}
		if err := Decode(vlanSpec(), nil, got); err != nil {
			t.Fatal(err)
		}
		if got.Vlan == nil || len(got.Vlan) != 0 {
			t.Errorf("want initialized empty map, got %+v", got.Vlan)
		}
	})
	t.Run("no entries", func(t *testing.T) {
		got := &huawei.HuaweiVlan_Vlan_Vlans{}
		if err := Decode(vlanSpec(), []byte(`<vlans xmlns="`+vlanNS+`"/>`), got); err != nil {
			t.Fatal(err)
		}
		if len(got.Vlan) != 0 {
			t.Errorf("want empty, got %+v", got.Vlan)
		}
	})
	t.Run("unknown elements skipped", func(t *testing.T) {
		doc := `<vlans><vlan><id>7</id><future-leaf>x</future-leaf><future-tree><a>1</a></future-tree></vlan></vlans>`
		got := &huawei.HuaweiVlan_Vlan_Vlans{}
		if err := Decode(vlanSpec(), []byte(doc), got); err != nil {
			t.Fatal(err)
		}
		if got.Vlan[7] == nil {
			t.Errorf("entry lost when unknown siblings present: %+v", got)
		}
	})
	t.Run("missing key synthesized", func(t *testing.T) {
		doc := `<vlans><vlan><name>orphan</name></vlan></vlans>`
		got := &huawei.HuaweiVlan_Vlan_Vlans{}
		if err := Decode(vlanSpec(), []byte(doc), got); err != nil {
			t.Fatal(err)
		}
		if len(got.Vlan) != 1 {
			t.Errorf("keyless entry dropped: %+v", got.Vlan)
		}
	})
	t.Run("invalid xml", func(t *testing.T) {
		got := &huawei.HuaweiVlan_Vlan_Vlans{}
		if err := Decode(vlanSpec(), []byte(`<vlans><vlan>`), got); err == nil {
			t.Error("want error for truncated XML")
		}
	})
	t.Run("enum decodes by YANG name", func(t *testing.T) {
		// XC-08：枚举按值域名解码（真机/本引擎 encode 均发名），"up" → AdminStatus_up。
		got := &huawei.HuaweiVlan_Vlan_Vlans{}
		if err := Decode(vlanSpec(), []byte(`<vlans><vlan><id>1</id><admin-status>up</admin-status></vlan></vlans>`), got); err != nil {
			t.Fatalf("枚举名解码不应报错: %v", err)
		}
		if got.Vlan[1] == nil || got.Vlan[1].AdminStatus != huawei.HuaweiVlan_AdminStatus_up {
			t.Errorf("admin-status=up 未解码为枚举常量: %#v", got.Vlan[1])
		}
	})
	t.Run("unknown enum value errors", func(t *testing.T) {
		// 真正非法的枚举值（非名非整数）SHALL 报错命名该叶（R08，不静默）。
		got := &huawei.HuaweiVlan_Vlan_Vlans{}
		err := Decode(vlanSpec(), []byte(`<vlans><vlan><id>1</id><admin-status>bogus-xyz</admin-status></vlan></vlans>`), got)
		if err == nil || !strings.Contains(err.Error(), "admin-status") {
			t.Errorf("want parse error naming the leaf, got %v", err)
		}
	})
	t.Run("nil destination", func(t *testing.T) {
		if err := Decode(vlanSpec(), []byte(`<vlans/>`), (*huawei.HuaweiVlan_Vlan_Vlans)(nil)); err == nil {
			t.Error("want error for nil destination")
		}
	})
}

// TestDecodePresentationLeaves：呈现叶（class/parent-name/number 等通用控制台
// 表格列）必须透出（D3b：decode 不做 config-false 过滤）。
func TestDecodePresentationLeaves(t *testing.T) {
	doc := `<interfaces><interface><name>GE0/0/1</name><class>1</class><parent-name>GE0/0</parent-name><number>0/0/1</number><link-protocol>1</link-protocol><router-type>1</router-type></interface></interfaces>`
	got := &huawei.HuaweiIfm_Ifm_Interfaces{}
	if err := Decode(ifmSpec(), []byte(doc), got); err != nil {
		t.Fatal(err)
	}
	e := got.Interface["GE0/0/1"]
	if e == nil {
		t.Fatal("entry missing")
	}
	if e.Class == 0 || e.ParentName == nil || e.Number == nil || e.LinkProtocol == 0 || e.RouterType == 0 {
		t.Errorf("presentation leaves lost: %+v", e)
	}
}

func TestDecodeConcurrent(t *testing.T) {
	out, err := Encode(vlanSpec(), hwfix.VlanFull())
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 8)
	for i := 0; i < 8; i++ {
		go func() {
			got := &huawei.HuaweiVlan_Vlan_Vlans{}
			done <- Decode(vlanSpec(), []byte(out), got)
		}()
	}
	for i := 0; i < 8; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
}
