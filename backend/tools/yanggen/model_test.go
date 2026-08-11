package main

import (
	"testing"
)

func buildFixtureModel(t *testing.T) *Model {
	t.Helper()
	entries, mods := loadFixture(t)
	m, err := BuildModel("testpkg", entries, mods)
	if err != nil {
		t.Fatalf("BuildModel: %v", err)
	}
	return m
}

func (m *Model) mustStruct(t *testing.T, name string) *Struct {
	t.Helper()
	s, ok := m.structIdx[name]
	if !ok {
		names := make([]string, 0, len(m.Structs))
		for _, s := range m.Structs {
			names = append(names, s.Name)
		}
		t.Fatalf("struct %s missing; have %v", name, names)
	}
	return s
}

func (s *Struct) mustField(t *testing.T, goName string) *Field {
	t.Helper()
	for _, f := range s.Fields {
		if f.GoName == goName {
			return f
		}
	}
	t.Fatalf("%s: field %s missing", s.Name, goName)
	return nil
}

func TestModelStructSetAndDevice(t *testing.T) {
	m := buildFixtureModel(t)
	dev := m.mustStruct(t, "Device")
	f := dev.mustField(t, "Box")
	if f.Type != "*UsmpTest_Box" || f.Module != "usmp-test" || f.YangName != "box" {
		t.Fatalf("Device.Box = %+v", f)
	}
	for _, want := range []string{
		"UsmpTest_Box", "UsmpTest_Box_Feature", "UsmpTest_Box_Vlans",
		"UsmpTest_Box_Vlans_Vlan", "UsmpTest_Box_Routes", "UsmpTest_Box_Routes_Route",
		"UsmpTest_Box_Extras",
	} {
		m.mustStruct(t, want)
	}
}

func TestModelFieldTypes(t *testing.T) {
	m := buildFixtureModel(t)
	box := m.mustStruct(t, "UsmpTest_Box")
	cases := map[string]string{
		"Host":     "*string",
		"Mtu":      "*uint16",
		"On":       "*bool",
		"Ratio":    "*float64",
		"Big":      "*uint64",
		"Mark":     "object.Empty",
		"Status":   "E_UsmpTest_EnableStatus",
		"Mode":     "E_UsmpTest_Box_Mode",
		"Port":     "UsmpTest_Box_Port_Union",
		"Tags":     "[]string",
		"Levels":   "[]uint32",
		"Feature":  "*UsmpTest_Box_Feature",
		"Vlans":    "*UsmpTest_Box_Vlans",
		"StaticIp": "*string", // choice/case 拍平进宿主
		"DhcpPool": "*string",
		"Extras":   "*UsmpTest_Box_Extras", // augment 并入宿主
	}
	for name, want := range cases {
		if got := box.mustField(t, name).Type; got != want {
			t.Errorf("Box.%s type = %s, want %s", name, got, want)
		}
	}
	// augment 字段 module tag = 来源模块
	if got := box.mustField(t, "Extras").Module; got != "usmp-test-ext" {
		t.Errorf("Extras module tag = %s, want usmp-test-ext", got)
	}
	if got := box.mustField(t, "Host").Module; got != "usmp-test" {
		t.Errorf("Host module tag = %s", got)
	}
	// 字段按 Go 名字典序
	for i := 1; i < len(box.Fields); i++ {
		if box.Fields[i-1].GoName >= box.Fields[i].GoName {
			t.Fatalf("fields not sorted: %s >= %s", box.Fields[i-1].GoName, box.Fields[i].GoName)
		}
	}
}

func TestModelLists(t *testing.T) {
	m := buildFixtureModel(t)
	vlans := m.mustStruct(t, "UsmpTest_Box_Vlans")
	if got := vlans.mustField(t, "Vlan").Type; got != "map[uint16]*UsmpTest_Box_Vlans_Vlan" {
		t.Errorf("Vlan list type = %s", got)
	}
	vlan := m.mustStruct(t, "UsmpTest_Box_Vlans_Vlan")
	if len(vlan.Keys) != 1 || vlan.Keys[0].GoName != "Id" || !vlan.Keys[0].Ptr || vlan.KeyName != "" {
		t.Fatalf("vlan keys = %+v keyName=%q", vlan.Keys, vlan.KeyName)
	}

	routes := m.mustStruct(t, "UsmpTest_Box_Routes")
	if got := routes.mustField(t, "Route").Type; got != "map[UsmpTest_Box_Routes_Route_Key]*UsmpTest_Box_Routes_Route" {
		t.Errorf("Route list type = %s", got)
	}
	route := m.mustStruct(t, "UsmpTest_Box_Routes_Route")
	if route.KeyName != "UsmpTest_Box_Routes_Route_Key" {
		t.Fatalf("route KeyName = %q", route.KeyName)
	}
	// Keys 按 YANG key 语句顺序（vrf prefix），非字典序
	if len(route.Keys) != 2 || route.Keys[0].YangName != "vrf" || route.Keys[1].YangName != "prefix" {
		t.Fatalf("route keys order = %+v", route.Keys)
	}
}

func TestModelEnums(t *testing.T) {
	m := buildFixtureModel(t)
	var typedefEnum, inlineEnum *Enum
	for _, e := range m.Enums {
		switch e.Name {
		case "E_UsmpTest_EnableStatus":
			typedefEnum = e
		case "E_UsmpTest_Box_Mode":
			inlineEnum = e
		}
	}
	if typedefEnum == nil || inlineEnum == nil {
		t.Fatalf("enums = %+v", m.Enums)
	}
	// value = YANG value + 1
	if typedefEnum.Values[0].ConstName != "UsmpTest_EnableStatus_enable" || typedefEnum.Values[0].Value != 2 {
		t.Errorf("typedef enum v0 = %+v (YANG value 1 → const 2)", typedefEnum.Values[0])
	}
	// 无显式 value：goyang 自动 0,1 → 常量 1,2
	if inlineEnum.Values[0].RawName != "access" || inlineEnum.Values[0].Value != 1 {
		t.Errorf("inline enum v0 = %+v", inlineEnum.Values[0])
	}
}

func TestModelUnion(t *testing.T) {
	m := buildFixtureModel(t)
	if len(m.Unions) != 1 {
		t.Fatalf("unions = %+v", m.Unions)
	}
	u := m.Unions[0]
	if u.Name != "UsmpTest_Box_Port_Union" {
		t.Fatalf("union name = %s", u.Name)
	}
	// 成员按声明序：uint16, string
	if len(u.Members) != 2 || u.Members[0] != "uint16" || u.Members[1] != "string" {
		t.Fatalf("union members = %v", u.Members)
	}
}

// TestModelDeterministic：两次构建产出的模型序列一致（结构/枚举/联合名序）。
func TestModelDeterministic(t *testing.T) {
	a := buildFixtureModel(t)
	c := buildFixtureModel(t)
	if len(a.Structs) != len(c.Structs) {
		t.Fatal("struct count differs between builds")
	}
	for i := range a.Structs {
		if a.Structs[i].Name != c.Structs[i].Name {
			t.Fatalf("struct order differs at %d: %s vs %s", i, a.Structs[i].Name, c.Structs[i].Name)
		}
	}
}
