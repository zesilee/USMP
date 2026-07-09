package client

import (
	"fmt"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	yangdriver "github.com/leezesi/usmp/backend/pkg/yang-runtime/driver"
)

// 本文件是 test-only shim：手写 parser（snd-xml-codec 退役中）的历史回归
// 用例（T07 锚点，如「member-port 嵌套解析」「回读恒空漂移」）原文保留，
// 经同名 shim 改由注册表 + 通用引擎执行——原行为断言不改一字仍全绿，即为
// 新旧路径等价的持续证明。生产代码不得引用本文件符号。

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
