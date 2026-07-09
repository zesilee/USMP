package client

import (
	"fmt"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	yangdriver "github.com/leezesi/usmp/backend/pkg/yang-runtime/driver"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/xmlcodec"
)

// 本文件是 test-only shim：手写 builder/parser（snd-xml-codec 已退役）的历史
// 回归用例（T07 锚点，如「IFM 内层 map 漏发」「member-port 嵌套解析」）原文
// 保留，经同名 shim 改由注册表 + 通用引擎执行——原行为断言不改一字仍全绿，
// 即为新旧路径等价的持续证明。生产代码不得引用本文件符号。

func ParseHuaweiVlanVlansXML(data []byte) (*huawei.HuaweiVlan_Vlan_Vlans, error) {
	d, ok := yangdriver.DecoderFor("vlan:vlans")
	if !ok {
		return nil, fmt.Errorf("shim: vlan decoder not registered")
	}
	gs, err := d.DecodeXML(data)
	if err != nil {
		return nil, err
	}
	return gs.(*huawei.HuaweiVlan_Vlan_Vlans), nil
}

func ParseHuaweiIfmInterfacesXML(data []byte) (*huawei.HuaweiIfm_Ifm_Interfaces, error) {
	d, ok := yangdriver.DecoderFor("ifm:interfaces")
	if !ok {
		return nil, fmt.Errorf("shim: ifm decoder not registered")
	}
	gs, err := d.DecodeXML(data)
	if err != nil {
		return nil, err
	}
	return gs.(*huawei.HuaweiIfm_Ifm_Interfaces), nil
}

func buildHuaweiVlanVlansXML(vlans *huawei.HuaweiVlan_Vlan_Vlans) (string, error) {
	if vlans == nil {
		vlans = &huawei.HuaweiVlan_Vlan_Vlans{}
	}
	d, ok := yangdriver.XMLEncoderForValue(vlans)
	if !ok {
		return "", fmt.Errorf("shim: vlan encoder not registered")
	}
	return xmlcodec.Encode(d.XML, vlans)
}

func buildHuaweiIfmInterfacesXML(ifaces *huawei.HuaweiIfm_Ifm_Interfaces) (string, error) {
	if ifaces == nil {
		ifaces = &huawei.HuaweiIfm_Ifm_Interfaces{}
	}
	d, ok := yangdriver.XMLEncoderForValue(ifaces)
	if !ok {
		return "", fmt.Errorf("shim: ifm encoder not registered")
	}
	return xmlcodec.Encode(d.XML, ifaces)
}

// NS 常量 shim：生产常量已随手写 builder 退役（namespace 成为驱动注册数据，
// 见 internal/drivers），历史断言原文保留。
const (
	HuaweiVlanNS = "urn:huawei:params:xml:ns:yang:huawei-vlan"
	HuaweiIfmNS  = "urn:huawei:params:xml:ns:yang:huawei-ifm"
)
