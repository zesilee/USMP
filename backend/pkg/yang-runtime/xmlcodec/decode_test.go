package xmlcodec

import (
	"reflect"
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/native/huawei"
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
	t.Run("unknown enum value tolerated", func(t *testing.T) {
		// 语义升级（真机回归）：非名非整数的枚举值=设备取值超出本地模型，
		// 叶级容错跳过（留日志），条目与后续叶不受影响——不再整树报错。
		got := &huawei.HuaweiVlan_Vlan_Vlans{}
		err := Decode(vlanSpec(), []byte(`<vlans><vlan><id>1</id><admin-status>bogus-xyz</admin-status><name>n1</name></vlan></vlans>`), got)
		if err != nil {
			t.Fatalf("叶级异常不得整树失败: %v", err)
		}
		v := got.Vlan[1]
		if v == nil || v.Name == nil || *v.Name != "n1" {
			t.Errorf("异常叶之后的正常叶应继续解析: %#v", v)
		}
		if v != nil && v.AdminStatus != 0 {
			t.Errorf("未知枚举叶应保持未设置")
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

// 真机回归（T07，CE9866 界面接口全空终局）：设备回读值超出本地模型时（新款
// 设备的枚举值/异形标量），叶级解析失败必须**跳过该叶继续**，不得毒死整棵
// 树——1MB 回读因一个叶失败整体降级原始透传，前端零行可渲染且无任何日志。
// 展示路径哲学与 EmitJSON SkipValidation 同源（R08）；结构性 XML 错误仍致命。
func TestDecodeToleratesUnknownLeafValues(t *testing.T) {
	spec := ifmSpec()
	xmlIn := `<ifm xmlns="urn:huawei:yang:huawei-ifm"><interfaces>` +
		`<interface><name>GE0/0/1</name><class>1</class><mtu>1500</mtu></interface>` +
		// 未知枚举值（本地模型没有的新款取值）+ 非法数字标量
		`<interface><name>XGE9/9/9</name><class>enum-from-newer-device</class><mtu>not-a-number</mtu><description>ok</description></interface>` +
		`</interfaces></ifm>`
	v := &huawei.HuaweiIfm_Ifm_Interfaces{}
	if err := Decode(spec, []byte(xmlIn), v); err != nil {
		t.Fatalf("叶级异常不得整树失败: %v", err)
	}
	if len(v.Interface) != 2 {
		t.Fatalf("两个接口都应解出，got %d", len(v.Interface))
	}
	good := v.Interface["GE0/0/1"]
	if good == nil || good.Mtu == nil || *good.Mtu != 1500 {
		t.Errorf("正常接口不受影响")
	}
	bad := v.Interface["XGE9/9/9"]
	if bad == nil {
		t.Fatal("异常叶所在接口仍应存在")
	}
	if bad.Description == nil || *bad.Description != "ok" {
		t.Errorf("异常叶之后的正常叶应继续解析")
	}
	if bad.Mtu != nil {
		t.Errorf("非法标量叶应保持未设置，got %v", *bad.Mtu)
	}
	if bad.Class != 0 {
		t.Errorf("未知枚举叶应保持未设置，got %v", bad.Class)
	}
}

// 结构性错误（残缺 XML）仍必须致命——容错只限叶级取值。
func TestDecodeStructuralErrorStillFatal(t *testing.T) {
	spec := ifmSpec()
	v := &huawei.HuaweiIfm_Ifm_Interfaces{}
	if err := Decode(spec, []byte(`<ifm><interfaces><interface><name>x`), v); err == nil {
		t.Fatal("残缺 XML 应报错")
	}
}
