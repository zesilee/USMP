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
	// HuaweiBgpNS 取 8.20.10 huawei-bgp.yang 声明的 module namespace（RFC7950：
	// 模块 namespace 即其数据节点的 XML namespace，真实 8.20.10 设备据此校验）。
	// 注意与 VLAN/IFM 上面两条的旧式 params:xml:ns 形态不同——各模块用自身声明值；
	// namespace 须显式（内嵌 gzip schema 的 Entry.Namespace() 实测返回空，不可派生）。
	HuaweiBgpNS = "urn:huawei:yang:huawei-bgp"
	// HuaweiNetworkInstanceNS 取 8.20.10 huawei-network-instance.yang 声明的 module
	// namespace（BGP peering 的唯一硬前置，peers/afs/peer-groups 均 augment 于此根下）。
	HuaweiNetworkInstanceNS = "urn:huawei:yang:huawei-network-instance"
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
	bgpXML := &xmlcodec.Spec{
		Namespace: HuaweiBgpNS,
		Schema:    func() *yang.Entry { return huawei.SchemaTree["HuaweiBgp_Bgp"] },
	}
	niXML := &xmlcodec.Spec{
		Namespace: HuaweiNetworkInstanceNS,
		Schema:    func() *yang.Entry { return huawei.SchemaTree["HuaweiNetworkInstance_NetworkInstance"] },
		// per-node namespace（XC-06）：network-instance 根下的 huawei-bgp augment
		// 子树（peers/afs，2a）须带自身 namespace，真机才接受。单模块字段（ni 原生）
		// 解析为根 namespace → 不另发 xmlns。加新 augment 模块（l3vpn 等）时在此登记。
		Namespaces: map[string]string{
			"huawei-network-instance": HuaweiNetworkInstanceNS,
			"huawei-bgp":              HuaweiBgpNS,
		},
	}

	// 注册序 = 原 manager if-链检查序（system → vlan → ifm），先注册先匹配。
	driver.Register(driver.Descriptor{
		Vendor: "huawei", Module: "system",
		// 根名前缀锚（DR-06）：原 Contains "system:" 会误吞 /telemetry-system: 等
		// 含 "-system:" 的新模块路径；真实调用方（controller/source/前端派生）均为
		// /system:system 前缀。
		MatchRoute:      func(p string) bool { return strings.HasPrefix(p, "/system:system") },
		ControllerToken: "system",
		// system 无 XML 回读解码（原 decodeRunningConfig 不含 system 分支），
		// 亦无 XML 下发通道 → XML 数据缺省 nil，调用方走既有降级（XC-04）。
		MatchEncode:  func(p string) bool { return strings.HasPrefix(p, "/system:system") },
		NewStruct:    func() ygot.GoStruct { return &huawei.HuaweiSystem_System{} },
		EncodeAnchor: "/system:system",
		Unmarshal:    huawei.Unmarshal,
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
		MatchEncode:  func(p string) bool { return strings.Contains(p, "vlan:") && strings.Contains(p, "vlan") },
		NewStruct:    func() ygot.GoStruct { return &huawei.HuaweiVlan_Vlan_Vlans{} },
		EncodeAnchor: "/vlan:vlan/vlan:vlans",
		Unmarshal:    huawei.Unmarshal,
		XML:          vlanXML,
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
		MatchEncode:  func(p string) bool { return strings.Contains(p, "ifm:ifm") && strings.Contains(p, "interfaces") },
		NewStruct:    func() ygot.GoStruct { return &huawei.HuaweiIfm_Ifm_Interfaces{} },
		EncodeAnchor: "/ifm:ifm/ifm:interfaces",
		Unmarshal:    huawei.Unmarshal,
		XML:          ifmXML,
	})
	// BGP 公网进程（/bgp:bgp）——本期接入面（design D1/D3）。谓词用 HasPrefix
	// "/bgp:bgp" 精确锚定公网根：排除 feature 模块前缀（/bgp-flow:、/bgp-evpn:）
	// 与 per-VPN 增强（/ni:.../bgp:bgp，三期）；避免裸子串 "bgp" 误命中（BGP-03）。
	driver.Register(driver.Descriptor{
		Vendor: "huawei", Module: "bgp",
		MatchRoute:      func(p string) bool { return strings.HasPrefix(p, "/bgp:bgp") },
		ControllerToken: "bgp",
		MatchDecode:     func(p string) bool { return strings.HasPrefix(p, "/bgp:bgp") },
		DecodeXML: func(raw []byte) (ygot.GoStruct, error) {
			v := &huawei.HuaweiBgp_Bgp{}
			if err := xmlcodec.Decode(bgpXML, raw, v); err != nil {
				return nil, err
			}
			return v, nil
		},
		MatchEncode:  func(p string) bool { return strings.HasPrefix(p, "/bgp:bgp") },
		NewStruct:    func() ygot.GoStruct { return &huawei.HuaweiBgp_Bgp{} },
		EncodeAnchor: "/bgp:bgp",
		Unmarshal:    huawei.Unmarshal,
		XML:          bgpXML,
	})
	// network-instance（/ni:network-instance）——BGP 二期 peering 唯一硬前置（NI-03）。
	// 单描述符覆盖整棵子树：因 augment 合并，peers（huawei-bgp）/afs（huawei-l3vpn）结构
	// 上同属此根，未来 peering 分期扩展本描述符驱动面、不另立描述符（design D1）。
	// 双口径谓词（DR-06）：业务意图编排层以 /ni: 调用（历史口径），控制台前端按根名派生
	// /network-instance:network-instance——两者都必须命中；编排层迁根名口径后可收敛。
	niMatch := func(p string) bool {
		return strings.HasPrefix(p, "/ni:network-instance") ||
			strings.HasPrefix(p, "/network-instance:network-instance")
	}
	driver.Register(driver.Descriptor{
		Vendor: "huawei", Module: "network-instance",
		MatchRoute:      niMatch,
		ControllerToken: "network-instance",
		MatchDecode:     niMatch,
		DecodeXML: func(raw []byte) (ygot.GoStruct, error) {
			v := &huawei.HuaweiNetworkInstance_NetworkInstance{}
			if err := xmlcodec.Decode(niXML, raw, v); err != nil {
				return nil, err
			}
			return v, nil
		},
		MatchEncode:  niMatch,
		NewStruct:    func() ygot.GoStruct { return &huawei.HuaweiNetworkInstance_NetworkInstance{} },
		EncodeAnchor: "/ni:network-instance",
		Unmarshal:    huawei.Unmarshal,
		XML:          niXML,
	})
}
