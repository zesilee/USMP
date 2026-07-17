package api

import (
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/internal/yangschema"
)

// findField 按 label 在 FieldDef 列表里查找（浅层）。
func findField(fs []FieldDef, label string) *FieldDef {
	for i := range fs {
		if fs[i].Label == label {
			return &fs[i]
		}
	}
	return nil
}

// 嵌套 schema：vlans → vlan(list) 应含扁平枚举叶子 + member-ports(group) → member-port(list)。
func TestBuildYangSchemaNested_VlanTree(t *testing.T) {
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	mod, ok := s.Module("vlan")
	if !ok {
		t.Fatal("vlan module not loaded")
	}
	ys := buildYangSchemaNested(mod)

	// container vlans { list vlan {...} } → vlans 是 group，其内 vlan 是 list
	vlans := findField(ys.Fields, "vlans")
	if vlans == nil || vlans.Type != "group" {
		t.Fatalf("vlans should be a group container, got %+v", vlans)
	}
	vlan := findField(vlans.Fields, "vlan")
	if vlan == nil || vlan.Type != "list" {
		t.Fatalf("vlan should be a list field, got %+v", vlan)
	}

	admin := findField(vlan.Fields, "admin-status")
	if admin == nil || admin.Type != "enum" || len(admin.Options) == 0 {
		t.Fatalf("admin-status should be enum with options, got %+v", admin)
	}

	mports := findField(vlan.Fields, "member-ports")
	if mports == nil || mports.Type != "group" {
		t.Fatalf("member-ports should be a group, got %+v", mports)
	}
	mport := findField(mports.Fields, "member-port")
	if mport == nil || mport.Type != "list" {
		t.Fatalf("member-port should be a list, got %+v", mport)
	}
	at := findField(mport.Fields, "access-type")
	if at == nil || at.Type != "enum" || len(at.Options) == 0 {
		t.Fatalf("access-type should be enum with options, got %+v", at)
	}
}

// 提交含 member-ports + 枚举字符串的 VLAN，转换应落到强类型结构。
func TestConvertVlan_MemberPortsAndEnums(t *testing.T) {
	data := map[string]interface{}{
		"vlan": []interface{}{
			map[string]interface{}{
				"id":           float64(100),
				"name":         "VLAN-100",
				"admin-status": "up",
				"member-ports": map[string]interface{}{
					"member-port": []interface{}{
						map[string]interface{}{
							"interface-name": "GigabitEthernet0/0/1",
							"access-type":    "trunk",
						},
					},
				},
			},
		},
	}
	typed, err := convertConfig("/vlan:vlan/vlan:vlans", data)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	res := typed.(*huawei.HuaweiVlan_Vlan_Vlans)
	v := res.Vlan[100]
	if v == nil {
		t.Fatal("vlan 100 not converted")
	}
	if v.AdminStatus != huawei.HuaweiVlan_AdminStatus_up {
		t.Errorf("AdminStatus = %v, want up", v.AdminStatus)
	}
	if v.MemberPorts == nil {
		t.Fatal("member-ports not converted")
	}
	p := v.MemberPorts.MemberPort["GigabitEthernet0/0/1"]
	if p == nil {
		t.Fatal("member-port GigabitEthernet0/0/1 not converted")
	}
	if p.AccessType != huawei.HuaweiVlan_AccessType_trunk {
		t.Errorf("AccessType = %v, want trunk", p.AccessType)
	}
}
