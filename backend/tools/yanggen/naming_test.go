package main

import "testing"

// 例证全部取自 codegen-conventions.md §1/§4（huawei 生成物实测），
// 命名与 ygot 产物逐字对齐是 2.4 结构对拍的前提。
func TestStructName(t *testing.T) {
	cases := []struct {
		module string
		segs   []string
		want   string
	}{
		{"huawei-vlan", []string{"vlan", "vlans", "vlan"}, "HuaweiVlan_Vlan_Vlans_Vlan"},
		{"huawei-ifm", []string{"ifm", "interfaces", "interface"}, "HuaweiIfm_Ifm_Interfaces_Interface"},
		{"usmp-business-vlan", []string{"business-vlan-service"}, "UsmpBusinessVlan_BusinessVlanService"},
		{"huawei-m-lag", []string{"m-lag"}, "HuaweiMLag_MLag"},
	}
	for _, tc := range cases {
		if got := StructName(tc.module, tc.segs); got != tc.want {
			t.Errorf("StructName(%s,%v) = %s, want %s", tc.module, tc.segs, got, tc.want)
		}
	}
}

func TestFieldName(t *testing.T) {
	cases := map[string]string{
		"ce-vlan-value-8021p": "CeVlanValue_8021P",
		"clock-8k-port":       "Clock_8KPort",
		"dot1q-vid":           "Dot1QVid",
		"group6s":             "Group6S",
		"arp-l2proxys":        "ArpL2Proxys",
		"ip-pool6s":           "IpPool6S",
		"l2vpn":               "L2Vpn",
		"m-lag":               "MLag",
		"cfcard2-size":        "Cfcard2Size",
		"type":                "Type",
		"interface":           "Interface",
		"range":               "Range",
		"name":                "Name",
	}
	for in, want := range cases {
		if got := FieldName(in); got != want {
			t.Errorf("FieldName(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestSafeEnumValue：值名净化（§4，genfix 的 |→_OR_ 内建）。
func TestSafeEnumValue(t *testing.T) {
	cases := map[string]string{
		"fragment-subseq": "fragment_subseq",
		"802.3":           "802_3",
		"50|100GE":        "50_OR_100GE",
		// 替换 token 无尾下划线——冻结自 ygot gogen/helpers.go 实测源码
		"a+b":           "a_PLUSb",
		"x,y":           "x_COMMAy",
		"u@h":           "u_ATh",
		"c$d":           "c_DOLLARd",
		"e*f":           "e_ASTERISKf",
		"g:h":           "g_COLONh",
		"a b":           "a_b",
		"a/b":           "a_b",
		"ARP":           "ARP", // 大小写原样保留
		"ENC_JSON_IETF": "ENC_JSON_IETF",
	}
	for in, want := range cases {
		if got := SafeEnumValue(in); got != want {
			t.Errorf("SafeEnumValue(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestEnumConstName：常量名 = 去 E_ 的类型名 + "_" + safe(值名)。
func TestEnumConstName(t *testing.T) {
	got := EnumConstName("E_HuaweiAcl_FragmentType", "fragment-subseq")
	if got != "HuaweiAcl_FragmentType_fragment_subseq" {
		t.Errorf("EnumConstName = %q", got)
	}
	if got := EnumConstName("E_HuaweiIfm_PortType", "50|100GE"); got != "HuaweiIfm_PortType_50_OR_100GE" {
		t.Errorf("EnumConstName pipe = %q", got)
	}
}
