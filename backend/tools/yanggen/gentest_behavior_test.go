package main

import (
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/object"
	"github.com/leezesi/usmp/backend/tools/yanggen/gentest"
)

// 生成物行为断言：golden 包（gentest）作为 object 接口族实现真实可用——
// 这是 emit 层正确性的端到端证明（不止字节对拍）。

func TestGentestImplementsObject(t *testing.T) {
	var _ object.Object = (*gentest.Device)(nil)
	var _ object.Object = (*gentest.UsmpTest_Box)(nil)
	var _ object.KeyedObject = (*gentest.UsmpTest_Box_Vlans_Vlan)(nil)
	var _ object.KeyedObject = (*gentest.UsmpTest_Box_Routes_Route)(nil)
	var _ object.Enum = gentest.E_UsmpTest_EnableStatus(0)
	var _ gentest.UsmpTest_Box_Port_Union = (*gentest.UsmpTest_Box_Port_Union_Uint16)(nil)
}

func TestGentestListKeyMap(t *testing.T) {
	v := &gentest.UsmpTest_Box_Vlans_Vlan{Id: object.Uint16(10)}
	m, err := v.ListKeyMap()
	if err != nil {
		t.Fatalf("ListKeyMap: %v", err)
	}
	if got := m["id"].(uint16); got != 10 {
		t.Fatalf("key map = %#v", m)
	}
	if _, err := (&gentest.UsmpTest_Box_Vlans_Vlan{}).ListKeyMap(); err == nil {
		t.Fatal("nil key must error（冻结文案 nil value for key）")
	}

	r := &gentest.UsmpTest_Box_Routes_Route{Vrf: object.String("v1"), Prefix: object.String("10.0.0.0/8")}
	rm, err := r.ListKeyMap()
	if err != nil {
		t.Fatal(err)
	}
	if rm["vrf"] != "v1" || rm["prefix"] != "10.0.0.0/8" {
		t.Fatalf("composite key map = %#v", rm)
	}
	k := gentest.UsmpTest_Box_Routes_Route_Key{Vrf: "v1", Prefix: "10.0.0.0/8"}
	km, err := k.ListKeyMap()
	if err != nil || km["vrf"] != "v1" {
		t.Fatalf("key struct map = %#v err=%v", km, err)
	}
}

func TestGentestEnum(t *testing.T) {
	e := gentest.UsmpTest_EnableStatus_enable
	if e != 2 {
		t.Fatalf("enable const = %d, want 2（YANG value 1 + 1）", e)
	}
	if e.String() != "enable" {
		t.Fatalf("String = %q", e.String())
	}
	name, err := object.EnumName(e)
	if err != nil || name != "enable" {
		t.Fatalf("EnumName = %q, %v", name, err)
	}
	if name, err := object.EnumName(gentest.UsmpTest_EnableStatus_UNSET); err != nil || name != "" {
		t.Fatalf("UNSET EnumName = %q, %v", name, err)
	}
	if _, err := object.EnumName(gentest.E_UsmpTest_EnableStatus(99)); err == nil {
		t.Fatal("out-of-range must error")
	}
}

func TestGentestUnionAssignment(t *testing.T) {
	b := &gentest.UsmpTest_Box{Port: &gentest.UsmpTest_Box_Port_Union_String{String: "eth0"}}
	if _, ok := b.Port.(*gentest.UsmpTest_Box_Port_Union_String); !ok {
		t.Fatal("union member assignment failed")
	}
	b.Port = &gentest.UsmpTest_Box_Port_Union_Uint16{Uint16: 8080}
	if v, ok := b.Port.(*gentest.UsmpTest_Box_Port_Union_Uint16); !ok || v.Uint16 != 8080 {
		t.Fatal("union uint16 member failed")
	}
}
