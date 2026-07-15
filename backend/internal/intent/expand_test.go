package intent

import (
	"reflect"
	"strings"
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/business"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
)

// BIO-02/BVS-02（矩阵 C）—— expand 纯函数：意图 spec → N×(设备,模块,路径,片段)
// 确定性展开 + 认领集合。展开片段按 huawei 原拼写与原生控制器同路径，保证
// 声明式对账与删除通道直接可用。

func u16(v uint16) *uint16 { return &v }
func s(v string) *string   { return &v }

func fullSpec() *business.UsmpBusinessVlan_BusinessVlanService {
	return &business.UsmpBusinessVlan_BusinessVlanService{
		VlanId: u16(100),
		Name:   s("biz-100"),
		Devices: map[string]*business.UsmpBusinessVlan_BusinessVlanService_Devices{
			"10.0.0.2": {
				Ip:         s("10.0.0.2"),
				TrunkPorts: []string{"GE0/0/3"},
			},
			"10.0.0.1": {
				Ip:          s("10.0.0.1"),
				AccessPorts: []string{"GE0/0/1"},
				TrunkPorts:  []string{"GE0/0/2"},
			},
		},
	}
}

// 矩阵 C1：双设备全属性展开——片段形状逐字段断言（LinkType/Pvid/TrunkVlans 精确叶子）。
func TestExpandDualDeviceFullAttributes(t *testing.T) {
	frags, claims, err := ExpandBusinessVlan(fullSpec())
	if err != nil {
		t.Fatalf("ExpandBusinessVlan: %v", err)
	}

	// 确定性顺序：设备 IP 升序，设备内 vlan 先于 ifm。
	wantOrder := []struct{ device, module string }{
		{"10.0.0.1", "vlan"}, {"10.0.0.1", "ifm"},
		{"10.0.0.2", "vlan"}, {"10.0.0.2", "ifm"},
	}
	if len(frags) != len(wantOrder) {
		t.Fatalf("fragments = %d, want %d: %+v", len(frags), len(wantOrder), frags)
	}
	for i, w := range wantOrder {
		if frags[i].Device != w.device || frags[i].Module != w.module {
			t.Errorf("frag[%d] = (%s,%s), want (%s,%s)", i, frags[i].Device, frags[i].Module, w.device, w.module)
		}
	}

	// vlan 片段：与原生 vlan 控制器同路径，条目含 Id+Name。
	v, ok := frags[0].Config.(*huawei.HuaweiVlan_Vlan_Vlans)
	if !ok {
		t.Fatalf("frag[0].Config type = %T, want *HuaweiVlan_Vlan_Vlans", frags[0].Config)
	}
	if frags[0].Path != "/vlan:vlan/vlan:vlans" {
		t.Errorf("vlan path = %s", frags[0].Path)
	}
	entry := v.Vlan[100]
	if entry == nil || entry.Id == nil || *entry.Id != 100 || entry.Name == nil || *entry.Name != "biz-100" {
		t.Fatalf("vlan entry = %+v, want Id=100 Name=biz-100", entry)
	}

	// ifm 片段：access 口 LinkType=access+Pvid，trunk 口 LinkType=trunk+TrunkVlans。
	ifm1, ok := frags[1].Config.(*huawei.HuaweiIfm_Ifm_Interfaces)
	if !ok {
		t.Fatalf("frag[1].Config type = %T, want *HuaweiIfm_Ifm_Interfaces", frags[1].Config)
	}
	if frags[1].Path != "/ifm:ifm/ifm:interfaces" {
		t.Errorf("ifm path = %s", frags[1].Path)
	}
	acc := ifm1.Interface["GE0/0/1"]
	if acc == nil || acc.Ethernet == nil || acc.Ethernet.MainInterface == nil || acc.Ethernet.MainInterface.L2Attribute == nil {
		t.Fatalf("access interface GE0/0/1 missing l2-attribute chain: %+v", acc)
	}
	accL2 := acc.Ethernet.MainInterface.L2Attribute
	if accL2.LinkType != huawei.HuaweiEthernet_LinkType_access || accL2.Pvid == nil || *accL2.Pvid != 100 {
		t.Errorf("access l2 = %+v, want LinkType=access Pvid=100", accL2)
	}
	trk := ifm1.Interface["GE0/0/2"]
	if trk == nil || trk.Ethernet == nil || trk.Ethernet.MainInterface == nil || trk.Ethernet.MainInterface.L2Attribute == nil {
		t.Fatalf("trunk interface GE0/0/2 missing l2-attribute chain")
	}
	trkL2 := trk.Ethernet.MainInterface.L2Attribute
	if trkL2.LinkType != huawei.HuaweiEthernet_LinkType_trunk || trkL2.TrunkVlans == nil || *trkL2.TrunkVlans != "100" {
		t.Errorf("trunk l2 = %+v, want LinkType=trunk TrunkVlans=100", trkL2)
	}

	// 认领集合：每设备 vlan 条目级 + 每端口 interface 条目级。
	wantClaims := map[string]bool{
		"10.0.0.1|vlan|/vlan:vlan/vlan:vlans/vlan[id=100]":             true,
		"10.0.0.1|ifm|/ifm:ifm/ifm:interfaces/interface[name=GE0/0/1]": true,
		"10.0.0.1|ifm|/ifm:ifm/ifm:interfaces/interface[name=GE0/0/2]": true,
		"10.0.0.2|vlan|/vlan:vlan/vlan:vlans/vlan[id=100]":             true,
		"10.0.0.2|ifm|/ifm:ifm/ifm:interfaces/interface[name=GE0/0/3]": true,
	}
	if len(claims) != len(wantClaims) {
		t.Fatalf("claims = %d, want %d: %+v", len(claims), len(wantClaims), claims)
	}
	for _, c := range claims {
		key := c.Device + "|" + c.Module + "|" + c.Path
		if !wantClaims[key] {
			t.Errorf("unexpected claim %s", key)
		}
	}
}

// 矩阵 C2：幂等重入——重复展开逐字节一致（含认领）。
func TestExpandDeterministic(t *testing.T) {
	f1, c1, err1 := ExpandBusinessVlan(fullSpec())
	f2, c2, err2 := ExpandBusinessVlan(fullSpec())
	if err1 != nil || err2 != nil {
		t.Fatalf("expand: %v / %v", err1, err2)
	}
	if !reflect.DeepEqual(f1, f2) || !reflect.DeepEqual(c1, c2) {
		t.Error("ExpandBusinessVlan is not deterministic across invocations")
	}
}

// 矩阵 C4：name 缺省 → VLAN<id>。
func TestExpandDefaultName(t *testing.T) {
	spec := fullSpec()
	spec.Name = nil
	frags, _, err := ExpandBusinessVlan(spec)
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	v := frags[0].Config.(*huawei.HuaweiVlan_Vlan_Vlans)
	if got := v.Vlan[100].Name; got == nil || *got != "VLAN100" {
		t.Errorf("default name = %v, want VLAN100", got)
	}
}

// 矩阵 C3：边界——nil spec / 缺 vlan-id / 空 devices 明确报错；无端口设备只出 vlan 片段；
// 空端口名跳过（R08 不 panic）。
func TestExpandEdges(t *testing.T) {
	if _, _, err := ExpandBusinessVlan(nil); err == nil {
		t.Error("nil spec should error")
	}

	noID := fullSpec()
	noID.VlanId = nil
	if _, _, err := ExpandBusinessVlan(noID); err == nil {
		t.Error("missing vlan-id should error")
	}

	noDev := fullSpec()
	noDev.Devices = nil
	if _, _, err := ExpandBusinessVlan(noDev); err == nil || !strings.Contains(err.Error(), "devices") {
		t.Errorf("empty devices should error mentioning devices, got %v", err)
	}

	bare := fullSpec()
	bare.Devices = map[string]*business.UsmpBusinessVlan_BusinessVlanService_Devices{
		"10.0.0.9": {Ip: s("10.0.0.9")},
	}
	frags, claims, err := ExpandBusinessVlan(bare)
	if err != nil {
		t.Fatalf("expand bare device: %v", err)
	}
	if len(frags) != 1 || frags[0].Module != "vlan" {
		t.Fatalf("bare device should expand to vlan fragment only, got %+v", frags)
	}
	if len(claims) != 1 {
		t.Fatalf("bare device claims = %+v, want 1 vlan claim", claims)
	}

	empties := fullSpec()
	empties.Devices = map[string]*business.UsmpBusinessVlan_BusinessVlanService_Devices{
		"10.0.0.9": {Ip: s("10.0.0.9"), AccessPorts: []string{""}, TrunkPorts: []string{" "}},
	}
	frags, _, err = ExpandBusinessVlan(empties)
	if err != nil {
		t.Fatalf("expand with empty port names: %v", err)
	}
	if len(frags) != 1 {
		t.Fatalf("empty port names should be skipped, got %+v", frags)
	}
}
