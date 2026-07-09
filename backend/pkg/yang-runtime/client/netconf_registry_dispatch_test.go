package client

import (
	"strings"
	"testing"

	_ "github.com/leezesi/usmp/backend/internal/drivers" // 注册 huawei 描述符：使本测试二进制真实走注册表分发（XC-04）
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/internal/generated/openconfig"
	"github.com/leezesi/usmp/backend/internal/testutil/hwfix"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/xmlcodec"
)

// TestMarshalChangeRegistryDispatch：marshalChange 对已注册模型（容器与
// diff 引擎产出的内层 map 两种形态）经注册表 + 通用引擎编码，输出与 legacy
// golden 等价（XC-04 场景「注册模块经通用引擎编码」）。
func TestMarshalChangeRegistryDispatch(t *testing.T) {
	c := &NETCONFClient{}
	tests := []struct {
		name   string
		value  interface{}
		golden string
	}{
		{"vlan container", hwfix.VlanFull(), "vlan_full"},
		{"vlan inner map", hwfix.VlanFull().Vlan, "vlan_full"},
		{"ifm container", hwfix.IfmFull(), "ifm_full"},
		{"ifm inner map", hwfix.IfmFull().Interface, "ifm_full"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := c.marshalChange(Change{Type: ModifyChange, Path: "irrelevant", NewValue: tt.value})
			if err != nil {
				t.Fatalf("marshalChange: %v", err)
			}
			canon, err := xmlcodec.Canonicalize([]byte(out))
			if err != nil {
				t.Fatalf("canonicalize: %v", err)
			}
			if want := hwfix.Golden(t, tt.golden); canon != want {
				t.Errorf("dispatch output != golden %s\n got: %s\nwant: %s", tt.golden, canon, want)
			}
		})
	}
}

// TestMarshalDeleteChangeRegistryDispatch：删除编码经注册表 + ΛListKeyMap
// 通用引擎，与 legacy golden 等价（DP-07 行为保持）。
func TestMarshalDeleteChangeRegistryDispatch(t *testing.T) {
	tests := []struct {
		name   string
		value  interface{}
		golden string
	}{
		{"vlan delete", hwfix.VlanDeleteSet(), "delete_vlan"},
		{"ifm delete incl nil-name entry", hwfix.IfmDeleteSet(), "delete_ifm"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := marshalDeleteChange(tt.value)
			if err != nil {
				t.Fatalf("marshalDeleteChange: %v", err)
			}
			canon, err := xmlcodec.Canonicalize([]byte(out))
			if err != nil {
				t.Fatalf("canonicalize: %v", err)
			}
			if want := hwfix.Golden(t, tt.golden); canon != want {
				t.Errorf("delete dispatch != golden %s\n got: %s\nwant: %s", tt.golden, canon, want)
			}
		})
	}
}

// TestMarshalDeleteChangeUnknownModel：未注册模型仍为明确错误，不发送裸
// delete（DP-07 负路径不变）。
func TestMarshalDeleteChangeUnknownModelViaRegistry(t *testing.T) {
	if _, err := marshalDeleteChange(struct{ X int }{1}); err == nil {
		t.Fatal("want explicit error for unsupported delete target")
	}
}

// TestMarshalChangeFallbackUnregistered：未注册类型不受注册表影响，仍走既有
// fallback 链（XC-04 场景「未注册类型降级」——此处 openconfig 遗留分支）。
func TestMarshalChangeFallbackUnregistered(t *testing.T) {
	c := &NETCONFClient{}
	name := "eth0"
	out, err := c.marshalChange(Change{Type: ModifyChange, Path: "/interfaces", NewValue: &openconfig.OpenconfigInterfaces_Interfaces{
		Interface: map[string]*openconfig.OpenconfigInterfaces_Interfaces_Interface{
			name: {Name: &name},
		},
	}})
	if err != nil {
		t.Fatalf("openconfig fallback broken: %v", err)
	}
	if !strings.Contains(out, "openconfig.net/yang/interfaces") {
		t.Errorf("openconfig legacy branch not taken: %s", out)
	}
}

// TestVlanSuppressionSingleEmission：注册表路径下 suppression 仅发一次
// （修复 legacy 重复发送 bug 的回归锚点，T07）。
func TestVlanSuppressionSingleEmission(t *testing.T) {
	c := &NETCONFClient{}
	v := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
			5: {Id: uptr16(5), Suppression: &huawei.HuaweiVlan_Vlan_Vlans_Vlan_Suppression{
				Inbound: huawei.E_HuaweiVlan_EnableStatus(1),
			}},
		},
	}
	out, err := c.marshalChange(Change{Type: ModifyChange, NewValue: v})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(out, "<suppression>"); got != 1 {
		t.Errorf("suppression emitted %d times, want exactly 1: %s", got, out)
	}
}

func uptr16(v uint16) *uint16 { return &v }
