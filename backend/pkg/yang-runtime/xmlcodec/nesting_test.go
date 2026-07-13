package xmlcodec

import (
	"strings"
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/openconfig/ygot/ygot"
)

// Regression（内置接口删不掉 bug）：edit-config 报文必须把 list 容器包进其 YANG 模块
// 顶层容器（<ifm>/<vlan>），与真机及模拟器种子 DemoSeedConfig 的嵌套结构一致。此前编码器
// 只发扁平 <interfaces>/<vlans>，导致针对嵌套存量条目的删除在设备树里匹配不到而落空。
// namespace 落在顶层容器上（对齐种子 <ifm xmlns=NS>），内层容器继承、不再重复声明。
func TestEncodeWrapsInModuleContainer(t *testing.T) {
	t.Run("ifm nests under <ifm>", func(t *testing.T) {
		out, err := Encode(ifmSpec(), &huawei.HuaweiIfm_Ifm_Interfaces{
			Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{
				"GE0/0/9": {Name: ygot.String("GE0/0/9")},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		want := `<ifm xmlns="` + ifmNS + `"><interfaces><interface><name>GE0/0/9</name></interface></interfaces></ifm>`
		if out != want {
			t.Errorf("got  %s\nwant %s", out, want)
		}
	})

	t.Run("vlan nests under <vlan>", func(t *testing.T) {
		out, err := Encode(vlanSpec(), &huawei.HuaweiVlan_Vlan_Vlans{
			Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
				30: {Id: ygot.Uint16(30)},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		want := `<vlan xmlns="` + vlanNS + `"><vlans><vlan><id>30</id></vlan></vlans></vlan>`
		if out != want {
			t.Errorf("got  %s\nwant %s", out, want)
		}
	})

	t.Run("empty list still wrapped and self-closing inner", func(t *testing.T) {
		out, err := Encode(ifmSpec(), &huawei.HuaweiIfm_Ifm_Interfaces{})
		if err != nil {
			t.Fatal(err)
		}
		want := `<ifm xmlns="` + ifmNS + `"><interfaces/></ifm>`
		if out != want {
			t.Errorf("got  %s\nwant %s", out, want)
		}
	})
}

func TestEncodeDeleteWrapsInModuleContainer(t *testing.T) {
	out, err := EncodeDelete(ifmSpec(), &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{
			"200GE0/1/2": {Name: ygot.String("200GE0/1/2")},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	// 顶层 <ifm xmlns=NS> 包裹 + 内层 <interfaces> + 条目携带 nc:operation="delete"
	for _, want := range []string{
		`<ifm xmlns="` + ifmNS + `">`,
		`<interfaces>`,
		`<interface nc:operation="delete" xmlns:nc="` + NetconfBaseNS + `"><name>200GE0/1/2</name></interface>`,
		`</interfaces></ifm>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("delete XML missing %q:\n%s", want, out)
		}
	}
	// 内层 <interfaces> 不得再声明 xmlns（继承顶层）
	if strings.Contains(out, `<interfaces xmlns=`) {
		t.Errorf("inner <interfaces> must not redeclare xmlns:\n%s", out)
	}
}
