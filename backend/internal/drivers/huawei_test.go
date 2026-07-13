package drivers

import (
	"strings"
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/driver"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/xmlcodec"
)

// 对拍表：现网全部真实路径形态 × 三处 dispatch 的既有行为。
// route/decode/encode 三列分别对应改造前 manager if-链、decodeRunningConfig
// switch、ygotRegistry 的命中结果（"" = 不命中）。行为等价是本期硬约束（DR-02/03）。
var dispatchEquivalence = []struct {
	path   string
	route  string // 命中的 ControllerToken
	decode string // 命中的解码模块
	encode string // 命中的编码模块
}{
	{"/vlan:vlan/vlan:vlans", "vlan", "vlan", "vlan"},
	{"/ifm:ifm/ifm:interfaces", "ifm", "ifm", "ifm"},
	{"/system:system", "system", "", "system"},
	// 别名形态（原 manager 额外接受的裸词）
	{"/foo/vlans", "vlan", "", ""},
	{"/foo/interfaces", "ifm", "", ""},
	// BGP 公网根：三处均命中 bgp（HasPrefix "/bgp:bgp"）
	{"/bgp:bgp", "bgp", "bgp", "bgp"},
	{"/bgp:bgp/base-process", "bgp", "bgp", "bgp"},
	// BGP 负路径（谓词精确锚定，禁止误命中，BGP-03）：
	{"/bgp-flow:flow", "", "", ""}, // feature 模块前缀含 "bgp" 但非 "/bgp:bgp"
	{"/bgp-evpn:evpn", "", "", ""}, // 同上
	{"/ni:network-instance/ni:instances/ni:instance/bgp:bgp/base-process", "", "", ""}, // per-VPN（三期，本期不接）——不以 "/bgp:bgp" 起头
	// 未覆盖路径：三处均降级
	{"/route:routing", "", "", ""},
	{"", "", "", ""},
}

func TestHuaweiDescriptors_DispatchEquivalence(t *testing.T) {
	for _, tc := range dispatchEquivalence {
		t.Run(tc.path, func(t *testing.T) {
			if d, ok := driver.Route(tc.path); ok != (tc.route != "") || (ok && d.ControllerToken != tc.route) {
				t.Fatalf("Route(%q) = (%v,%v)，期望 token=%q", tc.path, d.ControllerToken, ok, tc.route)
			}
			if d, ok := driver.DecoderFor(tc.path); ok != (tc.decode != "") || (ok && d.Module != tc.decode) {
				t.Fatalf("DecoderFor(%q) = (%v,%v)，期望 module=%q", tc.path, d.Module, ok, tc.decode)
			}
			if d, ok := driver.EncoderFor(tc.path); ok != (tc.encode != "") || (ok && d.Module != tc.encode) {
				t.Fatalf("EncoderFor(%q) = (%v,%v)，期望 module=%q", tc.path, d.Module, ok, tc.encode)
			}
		})
	}
}

// BGP 描述符全链路真值往返（写路径 RFC7951 解码 → XML 编码下发 → XML 回读解码），
// 断言实际字段值而非仅非空——防"装配了描述符但字段/namespace 静默错"的基本缺陷。
func TestHuaweiDescriptors_BgpEncodeDecodeRoundtrip(t *testing.T) {
	// 写路径：RFC7951 JSON → ygot struct（EncoderFor / DR-03 写）
	enc, ok := driver.EncoderFor("/bgp:bgp")
	if !ok {
		t.Fatal("bgp 编码描述符应命中")
	}
	dest := enc.NewStruct()
	if err := enc.Unmarshal([]byte(`{"base-process":{"enable":true,"as":"100"}}`), dest); err != nil {
		t.Fatalf("RFC7951 解码失败: %v", err)
	}
	bgp, ok := dest.(*huawei.HuaweiBgp_Bgp)
	if !ok || bgp.BaseProcess == nil {
		t.Fatalf("NewStruct/Unmarshal 装配错误: %#v", dest)
	}
	if bgp.BaseProcess.Enable == nil || !*bgp.BaseProcess.Enable {
		t.Fatal("enable 未正确解码为 true")
	}
	if bgp.BaseProcess.As == nil || *bgp.BaseProcess.As != "100" {
		t.Fatalf("as 未正确解码, got %v", bgp.BaseProcess.As)
	}

	// 下发路径：struct → XML（xmlcodec via 描述符 XML Spec），断言 namespace + 真值
	if enc.XML == nil {
		t.Fatal("bgp 描述符缺 XML Spec（下发通道未装配）")
	}
	xml, err := xmlcodec.Encode(enc.XML, bgp)
	if err != nil {
		t.Fatalf("XML 编码失败: %v", err)
	}
	for _, want := range []string{`xmlns="urn:huawei:yang:huawei-bgp"`, "<enable>true</enable>", "<as>100</as>"} {
		if !strings.Contains(xml, want) {
			t.Errorf("下发 XML 缺 %q\n实际: %s", want, xml)
		}
	}

	// 回读路径：XML → struct（DecoderFor / DR-03 读），真值等价
	dec, ok := driver.DecoderFor("/bgp:bgp")
	if !ok {
		t.Fatal("bgp 解码描述符应命中")
	}
	parsed, err := dec.DecodeXML([]byte(xml))
	if err != nil {
		t.Fatalf("XML 解码失败: %v", err)
	}
	rt := parsed.(*huawei.HuaweiBgp_Bgp)
	if rt.BaseProcess == nil || rt.BaseProcess.Enable == nil || !*rt.BaseProcess.Enable ||
		rt.BaseProcess.As == nil || *rt.BaseProcess.As != "100" {
		t.Fatalf("回读真值不等价: %#v", rt.BaseProcess)
	}
}

// 编解码闭包可实际工作（非空指针装配）：vlan XML 回读解码冒烟。
func TestHuaweiDescriptors_VlanDecodeSmoke(t *testing.T) {
	d, ok := driver.DecoderFor("/vlan:vlan/vlan:vlans")
	if !ok {
		t.Fatal("vlan 解码描述符应命中")
	}
	xml := []byte(`<vlans xmlns="urn:huawei:params:xml:ns:yang:huawei-vlan"><vlan><id>10</id><name>t</name></vlan></vlans>`)
	parsed, err := d.DecodeXML(xml)
	if err != nil || parsed == nil {
		t.Fatalf("vlan XML 解码应成功: %v", err)
	}
}
