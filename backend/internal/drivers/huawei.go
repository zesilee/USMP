// Package drivers wires the huawei device-driver descriptors into the driver
// registry (DR-01)：把此前散落在 manager.TriggerReconcile、config_codec 的
// decodeRunningConfig/ygotRegistry 三处路径 dispatch 收敛为每模块一条描述符。
// 谓词逐字保留原匹配语义（行为等价是本期硬约束）；消费方以空白导入触发注册。
package drivers

import (
	"strings"

	"github.com/openconfig/ygot/ygot"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/driver"
)

func init() {
	// 注册序 = 原 manager if-链检查序（system → vlan → ifm），先注册先匹配。
	driver.Register(driver.Descriptor{
		Vendor: "huawei", Module: "system",
		// 原 manager.go: strings.Contains(path, "system:")
		MatchRoute:      func(p string) bool { return strings.Contains(p, "system:") },
		ControllerToken: "system",
		// system 无 XML 回读解码（原 decodeRunningConfig 不含 system 分支）。
		// 原 ygotRegistry: strings.Contains(p, "system:")
		MatchEncode: func(p string) bool { return strings.Contains(p, "system:") },
		NewStruct:   func() ygot.GoStruct { return &huawei.HuaweiSystem_System{} },
		Unmarshal:   huawei.Unmarshal,
	})
	driver.Register(driver.Descriptor{
		Vendor: "huawei", Module: "vlan",
		// 原 manager.go: vlan: 或 vlans
		MatchRoute:      func(p string) bool { return strings.Contains(p, "vlan:") || strings.Contains(p, "vlans") },
		ControllerToken: "vlan",
		// 原 decodeRunningConfig: vlan:vlans
		MatchDecode: func(p string) bool { return strings.Contains(p, "vlan:vlans") },
		DecodeXML:   func(raw []byte) (ygot.GoStruct, error) { return client.ParseHuaweiVlanVlansXML(raw) },
		// 原 ygotRegistry: vlan: 且 vlan
		MatchEncode: func(p string) bool { return strings.Contains(p, "vlan:") && strings.Contains(p, "vlan") },
		NewStruct:   func() ygot.GoStruct { return &huawei.HuaweiVlan_Vlan_Vlans{} },
		Unmarshal:   huawei.Unmarshal,
	})
	driver.Register(driver.Descriptor{
		Vendor: "huawei", Module: "ifm",
		// 原 manager.go: ifm: 或 interfaces
		MatchRoute:      func(p string) bool { return strings.Contains(p, "ifm:") || strings.Contains(p, "interfaces") },
		ControllerToken: "ifm",
		// 原 decodeRunningConfig: ifm:interfaces
		MatchDecode: func(p string) bool { return strings.Contains(p, "ifm:interfaces") },
		DecodeXML:   func(raw []byte) (ygot.GoStruct, error) { return client.ParseHuaweiIfmInterfacesXML(raw) },
		// 原 ygotRegistry: ifm:ifm 且 interfaces
		MatchEncode: func(p string) bool { return strings.Contains(p, "ifm:ifm") && strings.Contains(p, "interfaces") },
		NewStruct:   func() ygot.GoStruct { return &huawei.HuaweiIfm_Ifm_Interfaces{} },
		Unmarshal:   huawei.Unmarshal,
	})
}
