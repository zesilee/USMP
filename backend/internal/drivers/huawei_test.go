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
	// network-instance 根：三处均命中（HasPrefix "/ni:network-instance"，NI-03）
	{"/ni:network-instance", "network-instance", "network-instance", "network-instance"},
	{"/ni:network-instance/instances/instance", "network-instance", "network-instance", "network-instance"},
	// per-VPN BGP（/ni:.../bgp:bgp/…）结构上在 ni 根下（单一 ygot 根，design D1）——
	// 由 network-instance 描述符路由，而非 /bgp:bgp 公网描述符（后者 HasPrefix 不命中）。
	{"/ni:network-instance/instances/instance/bgp:bgp/base-process", "network-instance", "network-instance", "network-instance"},
	// network-instance 负路径：前缀含 "network-instance" 但非 "/ni:network-instance" 起头
	{"/ni-feature:network-instance-x", "", "", ""},
	// tunnel-management 根（BGP 2b tunnel-policy leafref 前置，TNLM-03）：三处均命中
	{"/tnlm:tunnel-management", "tunnel-management", "tunnel-management", "tunnel-management"},
	{"/tnlm:tunnel-management/tunnel-policys/tunnel-policy", "tunnel-management", "tunnel-management", "tunnel-management"},
	// tunnel-management 负路径：ext 模块（tnlm-ext）数据 augment 入 tnlm 树、不独立成根，
	// 裸前缀不得误命中（谓词精确锚定 "/tnlm:tunnel-management"）
	{"/tnlm-ext:foo", "", "", ""},
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

// network-instance 描述符全链路真值往返（RFC7951 写 → XML 编码下发 → XML 回读），
// 覆盖容器根 global 标量 + 嵌套 instance list（NI-01/NI-02/NI-03）。
func TestHuaweiDescriptors_NetworkInstanceEncodeDecodeRoundtrip(t *testing.T) {
	enc, ok := driver.EncoderFor("/ni:network-instance")
	if !ok {
		t.Fatal("network-instance 编码描述符应命中")
	}
	dest := enc.NewStruct()
	if err := enc.Unmarshal([]byte(`{"global":{"cfg-router-id":"1.1.1.1"},"instances":{"instance":[{"name":"vpn-a","description":"d1"}]}}`), dest); err != nil {
		t.Fatalf("RFC7951 解码失败: %v", err)
	}
	ni, ok := dest.(*huawei.HuaweiNetworkInstance_NetworkInstance)
	if !ok || ni.Global == nil || ni.Instances == nil {
		t.Fatalf("NewStruct/Unmarshal 装配错误: %#v", dest)
	}
	if ni.Global.CfgRouterId == nil || *ni.Global.CfgRouterId != "1.1.1.1" {
		t.Fatalf("cfg-router-id 未正确解码: %v", ni.Global.CfgRouterId)
	}

	if enc.XML == nil {
		t.Fatal("network-instance 描述符缺 XML Spec（下发通道未装配）")
	}
	xml, err := xmlcodec.Encode(enc.XML, ni)
	if err != nil {
		t.Fatalf("XML 编码失败: %v", err)
	}
	for _, want := range []string{
		`xmlns="urn:huawei:yang:huawei-network-instance"`,
		"<cfg-router-id>1.1.1.1</cfg-router-id>", "<name>vpn-a</name>", "<description>d1</description>",
	} {
		if !strings.Contains(xml, want) {
			t.Errorf("下发 XML 缺 %q\n实际: %s", want, xml)
		}
	}

	dec, ok := driver.DecoderFor("/ni:network-instance")
	if !ok {
		t.Fatal("network-instance 解码描述符应命中")
	}
	parsed, err := dec.DecodeXML([]byte(xml))
	if err != nil {
		t.Fatalf("XML 解码失败: %v", err)
	}
	rt := parsed.(*huawei.HuaweiNetworkInstance_NetworkInstance)
	if rt.Global == nil || rt.Global.CfgRouterId == nil || *rt.Global.CfgRouterId != "1.1.1.1" {
		t.Fatalf("回读 global 真值不等价: %#v", rt.Global)
	}
	inst := rt.Instances.Instance["vpn-a"]
	if inst == nil || inst.Description == nil || *inst.Description != "d1" {
		t.Fatalf("回读嵌套 list 真值不等价: %#v", rt.Instances)
	}
}

// XC-06 真机正确性：经 ni 描述符编码含 huawei-bgp augment 子树的 network-instance，
// <bgp> 元素须带 huawei-bgp namespace（真机按 namespace 校验，否则拒绝）；ni 原生字段
// 不另发 xmlns。sim/decode namespace-宽容测不出，故断言 encode 输出真值。
func TestHuaweiDescriptors_NetworkInstancePerNodeNamespace(t *testing.T) {
	enc, ok := driver.EncoderFor("/ni:network-instance")
	if !ok || enc.XML == nil {
		t.Fatal("network-instance 编码描述符/XML Spec 应命中")
	}
	dest := enc.NewStruct()
	// instance[_public_] + huawei-bgp augment 子树（base-process/router-id）
	if err := enc.Unmarshal([]byte(`{"instances":{"instance":[{"name":"_public_","bgp":{"base-process":{"router-id":"1.1.1.1"}}}]}}`), dest); err != nil {
		t.Fatalf("RFC7951 解码失败: %v", err)
	}
	xml, err := xmlcodec.Encode(enc.XML, dest)
	if err != nil {
		t.Fatalf("XML 编码失败: %v", err)
	}
	if !strings.Contains(xml, `<bgp xmlns="urn:huawei:yang:huawei-bgp">`) {
		t.Errorf("bgp augment 子树缺 huawei-bgp namespace（真机会拒绝）\n实际: %s", xml)
	}
	if !strings.HasPrefix(xml, `<network-instance xmlns="urn:huawei:yang:huawei-network-instance">`) {
		t.Errorf("根 namespace 错\n实际: %s", xml)
	}
	if strings.Contains(xml, `<name xmlns=`) || strings.Contains(xml, `<router-id xmlns=`) {
		t.Errorf("原生/继承节点不应另发 xmlns\n实际: %s", xml)
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

// tunnel-management 描述符全链路真值往返（RFC7951 写 → XML 编码下发 → XML 回读），
// 覆盖容器根标量边界：tunnel-policy(name+description) 嵌套 list + tunnel-down-switch
// enable 标量（TNLM-01/02）。BGP 2b tunnel-policy leafref 的目标实例经此路径可配。
func TestHuaweiDescriptors_TunnelManagementEncodeDecodeRoundtrip(t *testing.T) {
	enc, ok := driver.EncoderFor("/tnlm:tunnel-management")
	if !ok {
		t.Fatal("tunnel-management 编码描述符应命中")
	}
	dest := enc.NewStruct()
	if err := enc.Unmarshal([]byte(`{"tunnel-policys":{"tunnel-policy":[{"name":"P1","description":"policy one"}]},"tunnel-down-switch":{"enable":true}}`), dest); err != nil {
		t.Fatalf("RFC7951 解码失败: %v", err)
	}
	tm, ok := dest.(*huawei.HuaweiTunnelManagement_TunnelManagement)
	if !ok || tm.TunnelPolicys == nil || tm.TunnelDownSwitch == nil {
		t.Fatalf("NewStruct/Unmarshal 装配错误: %#v", dest)
	}
	p := tm.TunnelPolicys.TunnelPolicy["P1"]
	if p == nil || p.Description == nil || *p.Description != "policy one" {
		t.Fatalf("tunnel-policy 未正确解码: %#v", tm.TunnelPolicys)
	}
	if tm.TunnelDownSwitch.Enable == nil || !*tm.TunnelDownSwitch.Enable {
		t.Fatalf("tunnel-down-switch/enable 未正确解码: %#v", tm.TunnelDownSwitch)
	}

	if enc.XML == nil {
		t.Fatal("tunnel-management 描述符缺 XML Spec（下发通道未装配）")
	}
	xml, err := xmlcodec.Encode(enc.XML, tm)
	if err != nil {
		t.Fatalf("XML 编码失败: %v", err)
	}
	for _, want := range []string{
		`xmlns="urn:huawei:yang:huawei-tunnel-management"`,
		"<name>P1</name>", "<description>policy one</description>", "<enable>true</enable>",
	} {
		if !strings.Contains(xml, want) {
			t.Errorf("下发 XML 缺 %q\n实际: %s", want, xml)
		}
	}

	dec, ok := driver.DecoderFor("/tnlm:tunnel-management")
	if !ok {
		t.Fatal("tunnel-management 解码描述符应命中")
	}
	parsed, err := dec.DecodeXML([]byte(xml))
	if err != nil {
		t.Fatalf("XML 解码失败: %v", err)
	}
	rt := parsed.(*huawei.HuaweiTunnelManagement_TunnelManagement)
	rp := rt.TunnelPolicys.TunnelPolicy["P1"]
	if rp == nil || rp.Description == nil || *rp.Description != "policy one" {
		t.Fatalf("回读嵌套 list 真值不等价: %#v", rt.TunnelPolicys)
	}
	if rt.TunnelDownSwitch == nil || rt.TunnelDownSwitch.Enable == nil || !*rt.TunnelDownSwitch.Enable {
		t.Fatalf("回读 tunnel-down-switch 真值不等价: %#v", rt.TunnelDownSwitch)
	}
}
