package translator

import (
	"testing"

	bizv1 "github.com/leezesi/usmp/backend/api/biz/v1"
)

// TestHuaweiVlanTranslate_BizV1: biz/v1 VLAN 字段映射到 huawei ygot（含此前被忽略的
// MacLearning/BroadcastDiscard/UnknownMulticastDiscard）。
func TestHuaweiVlanTranslate_BizV1(t *testing.T) {
	tr := NewHuaweiVlanTranslator()
	vlans, err := tr.Translate(bizv1.BusinessVlanSpec{
		VlanID:                  100,
		Name:                    "office",
		Description:             "d",
		AdminStatus:             bizv1.VlanAdminStatusDown,
		MacLearning:             bizv1.MacLearningDisabled,
		BroadcastDiscard:        true,
		UnknownMulticastDiscard: true,
	})
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	v := vlans.Vlan[100]
	if v == nil || v.Name == nil {
		t.Fatal("vlan 100 / name missing")
	}
	if *v.Name != "office" || v.AdminStatus != 2 {
		t.Fatalf("name/admin = %s/%d, want office/2(down)", *v.Name, v.AdminStatus)
	}
	if v.MacLearning != huaweiDisable || v.BroadcastDiscard != huaweiEnable || v.UnknownMulticastDiscard != huaweiEnable {
		t.Fatalf("mac/bcast/mcast = %d/%d/%d, want %d/%d/%d",
			v.MacLearning, v.BroadcastDiscard, v.UnknownMulticastDiscard, huaweiDisable, huaweiEnable, huaweiEnable)
	}
}

// TestHuaweiVlanTranslate_InvalidID: 越界 VLAN ID 报错（R08）。
func TestHuaweiVlanTranslate_InvalidID(t *testing.T) {
	if _, err := NewHuaweiVlanTranslator().Translate(bizv1.BusinessVlanSpec{VlanID: 5000}); err == nil {
		t.Fatal("out-of-range vlanID should error")
	}
}

// TestHuaweiInterfaceTranslate_BizV1: biz/v1 接口（IfName/MTU/access）映射到 L2 ygot。
func TestHuaweiInterfaceTranslate_BizV1(t *testing.T) {
	tr := NewHuaweiInterfaceTranslator()
	ifaces, err := tr.Translate(bizv1.BusinessInterfaceSpec{
		IfName:      "GE0/0/1",
		Description: "uplink",
		AdminStatus: bizv1.InterfaceAdminStatusUp,
		Mode:        bizv1.InterfaceModeAccess,
		MTU:         1500,
	})
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	i := ifaces.Interface["GE0/0/1"]
	if i == nil || i.Name == nil || i.Mtu == nil || i.IsL2Switch == nil {
		t.Fatal("interface / fields missing")
	}
	if *i.Name != "GE0/0/1" || *i.Mtu != 1500 || i.ServiceType != 2 || *i.IsL2Switch != true {
		t.Fatalf("iface = name %s mtu %d svc %d l2 %v; want GE0/0/1/1500/2/true",
			*i.Name, *i.Mtu, i.ServiceType, *i.IsL2Switch)
	}
}

// TestHuaweiInterfaceTranslate_EmptyName: 空接口名报错。
func TestHuaweiInterfaceTranslate_EmptyName(t *testing.T) {
	if _, err := NewHuaweiInterfaceTranslator().Translate(bizv1.BusinessInterfaceSpec{IfName: ""}); err == nil {
		t.Fatal("empty ifName should error")
	}
}
