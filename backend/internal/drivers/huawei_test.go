package drivers

import (
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/driver"
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
