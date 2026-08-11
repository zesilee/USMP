package main

import (
	"testing"

	ygothuawei "github.com/leezesi/usmp/backend/internal/generated/huawei"
	native "github.com/leezesi/usmp/backend/internal/generated/native/huawei"
	"github.com/leezesi/usmp/backend/internal/yangschema"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/object"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/xmlcodec"
	"github.com/openconfig/ygot/ygot"
)

// XML 通道双族对拍（任务4.2，S2）：同值的 ygot 结构体与 native 结构体过**同一
// 引擎**（S2 双族兼容层）→ Encode/EncodeDelete 输出逐字节相等；Decode 双族
// 结果再编码亦相等（往返对称）。

func xmlIRSpec(t *testing.T, path, ns string) *xmlcodec.Spec {
	t.Helper()
	return &xmlcodec.Spec{
		Namespace: ns,
		Schema: func() schema.Node {
			s, err := yangschema.Load()
			if err != nil {
				return nil
			}
			n, _ := s.Path(path)
			return n
		},
	}
}

const vlanNS = "urn:huawei:yang:huawei-vlan"

func ygotVlans() *ygothuawei.HuaweiVlan_Vlan_Vlans {
	return &ygothuawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*ygothuawei.HuaweiVlan_Vlan_Vlans_Vlan{
			10: {Id: ygot.Uint16(10), Name: ygot.String("v10"), Description: ygot.String("ten")},
			20: {Id: ygot.Uint16(20), Name: ygot.String("v20")},
		},
	}
}

func nativeVlans() *native.HuaweiVlan_Vlan_Vlans {
	return &native.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*native.HuaweiVlan_Vlan_Vlans_Vlan{
			10: {Id: object.Uint16(10), Name: object.String("v10"), Description: object.String("ten")},
			20: {Id: object.Uint16(20), Name: object.String("v20")},
		},
	}
}

func TestXMLParityEncode(t *testing.T) {
	spec := xmlIRSpec(t, "/vlan/vlans", vlanNS)
	yx, err := xmlcodec.Encode(spec, ygotVlans())
	if err != nil {
		t.Fatalf("ygot encode: %v", err)
	}
	nx, err := xmlcodec.Encode(spec, nativeVlans())
	if err != nil {
		t.Fatalf("native encode: %v", err)
	}
	if yx != nx {
		t.Fatalf("XML 编码不等:\nygot:   %s\nnative: %s", yx, nx)
	}
}

func TestXMLParityEnumLeaf(t *testing.T) {
	// 枚举叶值域名编码（XC-08）双族一致——ifm interface 带 enum admin-status。
	spec := xmlIRSpec(t, "/ifm/interfaces", "urn:huawei:yang:huawei-ifm")
	name := "GE0/0/1"
	yv := &ygothuawei.HuaweiIfm_Ifm_Interfaces{Interface: map[string]*ygothuawei.HuaweiIfm_Ifm_Interfaces_Interface{
		name: {Name: ygot.String(name), AdminStatus: 2},
	}}
	nv := &native.HuaweiIfm_Ifm_Interfaces{Interface: map[string]*native.HuaweiIfm_Ifm_Interfaces_Interface{
		name: {Name: object.String(name), AdminStatus: 2},
	}}
	yx, err := xmlcodec.Encode(spec, yv)
	if err != nil {
		t.Fatal(err)
	}
	nx, err := xmlcodec.Encode(spec, nv)
	if err != nil {
		t.Fatal(err)
	}
	if yx != nx {
		t.Fatalf("枚举叶编码不等:\nygot:   %s\nnative: %s", yx, nx)
	}
}

func TestXMLParityDelete(t *testing.T) {
	spec := xmlIRSpec(t, "/vlan/vlans", vlanNS)
	yx, err := xmlcodec.EncodeDelete(spec, ygotVlans())
	if err != nil {
		t.Fatal(err)
	}
	nx, err := xmlcodec.EncodeDelete(spec, nativeVlans())
	if err != nil {
		t.Fatal(err)
	}
	if yx != nx {
		t.Fatalf("删除编码不等:\nygot:   %s\nnative: %s", yx, nx)
	}
}

func TestXMLParityDecodeRoundTrip(t *testing.T) {
	spec := xmlIRSpec(t, "/vlan/vlans", vlanNS)
	raw, err := xmlcodec.Encode(spec, ygotVlans())
	if err != nil {
		t.Fatal(err)
	}
	yd := &ygothuawei.HuaweiVlan_Vlan_Vlans{}
	if err := xmlcodec.Decode(spec, []byte(raw), yd); err != nil {
		t.Fatalf("ygot decode: %v", err)
	}
	nd := &native.HuaweiVlan_Vlan_Vlans{}
	if err := xmlcodec.Decode(spec, []byte(raw), nd); err != nil {
		t.Fatalf("native decode: %v", err)
	}
	// 双族解码结果再编码回 XML，逐字节相等即语义相等。
	yx, err := xmlcodec.Encode(spec, yd)
	if err != nil {
		t.Fatal(err)
	}
	nx, err := xmlcodec.Encode(spec, nd)
	if err != nil {
		t.Fatal(err)
	}
	if yx != nx {
		t.Fatalf("解码往返不等:\nygot:   %s\nnative: %s", yx, nx)
	}
	if nd.Vlan[10] == nil || *nd.Vlan[10].Name != "v10" {
		t.Fatal("native 解码内容缺失")
	}
}
