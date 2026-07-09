// Package drivers wires the huawei device-driver descriptors into the driver
// registry (DR-01)：把此前散落在 manager.TriggerReconcile、config_codec 的
// decodeRunningConfig/ygotRegistry 三处路径 dispatch 收敛为每模块一条描述符。
// 谓词逐字保留原匹配语义（行为等价是本期硬约束）；消费方以空白导入触发注册。
//
// P5-2（snd-xml-codec）：描述符携带 XML 编解码数据（namespace + SchemaTree
// 入口），编解码本体是通用引擎 pkg/yang-runtime/xmlcodec——加新模块只登记
// 数据，零 XML 代码（XC-01~04）。namespace 是显式数据：内嵌 gzip schema 的
// Entry.Namespace() 实测为空，无法派生（design D3b）。
package drivers

import (
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/driver"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/xmlcodec"
)

// Huawei module XML namespaces（驱动数据，随描述符注册）。
const (
	HuaweiVlanNS = "urn:huawei:params:xml:ns:yang:huawei-vlan"
	HuaweiIfmNS  = "urn:huawei:params:xml:ns:yang:huawei-ifm"
)

func init() {
	vlanXML := &xmlcodec.Spec{
		Namespace: HuaweiVlanNS,
		Schema:    func() *yang.Entry { return huawei.SchemaTree["HuaweiVlan_Vlan_Vlans"] },
	}
	ifmXML := &xmlcodec.Spec{
		Namespace: HuaweiIfmNS,
		Schema:    func() *yang.Entry { return huawei.SchemaTree["HuaweiIfm_Ifm_Interfaces"] },
	}

	// 注册序 = 原 manager if-链检查序（system → vlan → ifm），先注册先匹配。
	driver.Register(driver.Descriptor{
		Vendor: "huawei", Module: "system",
		// 原 manager.go: strings.Contains(path, "system:")
		MatchRoute:      func(p string) bool { return strings.Contains(p, "system:") },
		ControllerToken: "system",
		// system 无 XML 回读解码（原 decodeRunningConfig 不含 system 分支），
		// 亦无 XML 下发通道 → XML 数据缺省 nil，调用方走既有降级（XC-04）。
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
		DecodeXML: func(raw []byte) (ygot.GoStruct, error) {
			v := &huawei.HuaweiVlan_Vlan_Vlans{}
			if err := xmlcodec.Decode(vlanXML, raw, v); err != nil {
				return nil, err
			}
			return v, nil
		},
		// 原 ygotRegistry: vlan: 且 vlan
		MatchEncode: func(p string) bool { return strings.Contains(p, "vlan:") && strings.Contains(p, "vlan") },
		NewStruct:   func() ygot.GoStruct { return &huawei.HuaweiVlan_Vlan_Vlans{} },
		Unmarshal:   huawei.Unmarshal,
		XML:         vlanXML,
	})
	driver.Register(driver.Descriptor{
		Vendor: "huawei", Module: "ifm",
		// 原 manager.go: ifm: 或 interfaces
		MatchRoute:      func(p string) bool { return strings.Contains(p, "ifm:") || strings.Contains(p, "interfaces") },
		ControllerToken: "ifm",
		// 原 decodeRunningConfig: ifm:interfaces
		MatchDecode: func(p string) bool { return strings.Contains(p, "ifm:interfaces") },
		DecodeXML: func(raw []byte) (ygot.GoStruct, error) {
			v := &huawei.HuaweiIfm_Ifm_Interfaces{}
			if err := xmlcodec.Decode(ifmXML, raw, v); err != nil {
				return nil, err
			}
			return v, nil
		},
		// 原 ygotRegistry: ifm:ifm 且 interfaces
		MatchEncode: func(p string) bool { return strings.Contains(p, "ifm:ifm") && strings.Contains(p, "interfaces") },
		NewStruct:   func() ygot.GoStruct { return &huawei.HuaweiIfm_Ifm_Interfaces{} },
		Unmarshal:   huawei.Unmarshal,
		XML:         ifmXML,
	})
}
