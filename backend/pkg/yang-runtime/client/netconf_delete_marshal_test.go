package client

import (
	"strings"
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/openconfig/ygot/ygot"
)

// DP-07：DeleteChange 编码——外层模型容器 + 条目元素带 NETCONF base 命名空间的
// operation="delete" 属性 + 仅序列化 key 叶（key 为首个子元素）。
func TestMarshalDeleteChangeVlan(t *testing.T) {
	vlans := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
			10: {Id: ygot.Uint16(10)},
		},
	}
	c := &NETCONFClient{}
	xml, err := c.marshalChange(Change{Type: DeleteChange, OldValue: vlans, NewValue: nil})
	if err != nil {
		t.Fatalf("marshalChange: %v", err)
	}
	for _, want := range []string{
		`<vlan xmlns="` + HuaweiVlanNS + `"><vlans>`,
		`nc:operation="delete"`,
		`xmlns:nc="urn:ietf:params:xml:ns:netconf:base:1.0"`,
		`<id>10</id>`,
	} {
		if !strings.Contains(xml, want) {
			t.Errorf("delete XML missing %q:\n%s", want, xml)
		}
	}
	// 仅 key 叶：不得序列化其他字段元素
	for _, forbid := range []string{"<name>", "<description>", "<admin-status>"} {
		if strings.Contains(xml, forbid) {
			t.Errorf("delete XML must carry key leaf only, found %q:\n%s", forbid, xml)
		}
	}
}

func TestMarshalDeleteChangeVlanKeyFromMapKey(t *testing.T) {
	// 条目 Id 指针为空时以 map 键兜底（与 merge builder 同规，R08）。
	vlans := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{20: {}},
	}
	c := &NETCONFClient{}
	xml, err := c.marshalChange(Change{Type: DeleteChange, OldValue: vlans})
	if err != nil {
		t.Fatalf("marshalChange: %v", err)
	}
	if !strings.Contains(xml, "<id>20</id>") {
		t.Errorf("map-key fallback missing: %s", xml)
	}
}

func TestMarshalDeleteChangeIfm(t *testing.T) {
	ifaces := &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{
			"GigabitEthernet0/0/1": {Name: ygot.String("GigabitEthernet0/0/1")},
		},
	}
	c := &NETCONFClient{}
	xml, err := c.marshalChange(Change{Type: DeleteChange, OldValue: ifaces})
	if err != nil {
		t.Fatalf("marshalChange: %v", err)
	}
	for _, want := range []string{
		`<ifm xmlns="` + HuaweiIfmNS + `"><interfaces>`,
		`nc:operation="delete"`,
		`<name>GigabitEthernet0/0/1</name>`,
	} {
		if !strings.Contains(xml, want) {
			t.Errorf("delete XML missing %q:\n%s", want, xml)
		}
	}
	for _, forbid := range []string{"<description>", "<mtu>", "<admin-status>"} {
		if strings.Contains(xml, forbid) {
			t.Errorf("delete XML must carry key leaf only, found %q:\n%s", forbid, xml)
		}
	}
}

func TestMarshalDeleteChangeUnknownModel(t *testing.T) {
	c := &NETCONFClient{}
	if _, err := c.marshalChange(Change{Type: DeleteChange, OldValue: struct{ X int }{1}}); err == nil {
		t.Error("err = nil, want explicit encoding error for unknown model (R08)")
	}
	// OldValue 也为空：无法定位删除目标，必须报错而非发裸 delete 元素。
	if _, err := c.marshalChange(Change{Type: DeleteChange}); err == nil {
		t.Error("err = nil, want error when DeleteChange carries no target")
	}
}
