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
	// HuaweiTunnelManagementNS 取 8.20.10 huawei-tunnel-management.yang 声明的 module
	// namespace（BGP 2b tunnel-policy leafref 目标，越序禁令要求先接成可配模型）。
	HuaweiTunnelManagementNS = "urn:huawei:yang:huawei-tunnel-management"
	// HuaweiXplNS 取 8.20.10 huawei-xpl.yang 声明的 module namespace（BGP 2b route-filter
	// leafref 目标 /xpl:xpl/route-filters/route-filter，越序禁令要求先接成可配模型）。
	HuaweiXplNS = "urn:huawei:yang:huawei-xpl"
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
	tnlmXML := &xmlcodec.Spec{
		Namespace: HuaweiTunnelManagementNS,
		Schema:    func() *yang.Entry { return huawei.SchemaTree["HuaweiTunnelManagement_TunnelManagement"] },
	}
	xplXML := &xmlcodec.Spec{
		Namespace: HuaweiXplNS,
		Schema:    func() *yang.Entry { return huawei.SchemaTree["HuaweiXpl_Xpl"] },
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
		MatchEncode: func(p string) bool { return strings.HasPrefix(p, "/bgp:bgp") },
		NewStruct:   func() ygot.GoStruct { return &huawei.HuaweiBgp_Bgp{} },
		Unmarshal:   huawei.Unmarshal,
		XML:         bgpXML,
	})
	// network-instance（/ni:network-instance）——BGP 二期 peering 唯一硬前置（NI-03）。
	// 单描述符覆盖整棵子树：因 augment 合并，peers（huawei-bgp）/afs（huawei-l3vpn）结构
	// 上同属此根，未来 peering 分期扩展本描述符驱动面、不另立描述符（design D1）。谓词用
	// HasPrefix "/ni:network-instance" 精确锚定，避免裸子串误命中。
	driver.Register(driver.Descriptor{
		Vendor: "huawei", Module: "network-instance",
		MatchRoute:      func(p string) bool { return strings.HasPrefix(p, "/ni:network-instance") },
		ControllerToken: "network-instance",
		MatchDecode:     func(p string) bool { return strings.HasPrefix(p, "/ni:network-instance") },
		DecodeXML: func(raw []byte) (ygot.GoStruct, error) {
			v := &huawei.HuaweiNetworkInstance_NetworkInstance{}
			if err := xmlcodec.Decode(niXML, raw, v); err != nil {
				return nil, err
			}
			return v, nil
		},
		MatchEncode: func(p string) bool { return strings.HasPrefix(p, "/ni:network-instance") },
		NewStruct:   func() ygot.GoStruct { return &huawei.HuaweiNetworkInstance_NetworkInstance{} },
		Unmarshal:   huawei.Unmarshal,
		XML:         niXML,
	})
	// tunnel-management（/tnlm:tunnel-management）——BGP 2b tunnel-policy leafref 前置
	// （越序禁令：目标模型须先可配，TNLM-01/03）。容器根模块（非 list 根），与 /bgp:bgp
	// 同构，走通用引擎 plain-container（XC-05）。谓词用 HasPrefix "/tnlm:tunnel-management"
	// 精确锚定：tnlm-ext 数据 augment 入本树不独立成根，裸前缀不误命中其他模块。
	driver.Register(driver.Descriptor{
		Vendor: "huawei", Module: "tunnel-management",
		MatchRoute:      func(p string) bool { return strings.HasPrefix(p, "/tnlm:tunnel-management") },
		ControllerToken: "tunnel-management",
		MatchDecode:     func(p string) bool { return strings.HasPrefix(p, "/tnlm:tunnel-management") },
		DecodeXML: func(raw []byte) (ygot.GoStruct, error) {
			v := &huawei.HuaweiTunnelManagement_TunnelManagement{}
			if err := xmlcodec.Decode(tnlmXML, raw, v); err != nil {
				return nil, err
			}
			return v, nil
		},
		MatchEncode: func(p string) bool { return strings.HasPrefix(p, "/tnlm:tunnel-management") },
		NewStruct:   func() ygot.GoStruct { return &huawei.HuaweiTunnelManagement_TunnelManagement{} },
		Unmarshal:   huawei.Unmarshal,
		XML:         tnlmXML,
	})
	// xpl（/xpl:xpl）——BGP 2b route-filter leafref 前置（越序禁令：目标模型须先可配，
	// XPL-01/03）。容器根模块（非 list 根），与 /bgp:bgp、/tnlm:tunnel-management 同构，
	// 走通用引擎 plain-container（XC-05）。谓词 HasPrefix "/xpl:xpl" 精确锚定。本波次功能
	// 面仅 route-filters/route-filter；xpl 其他策略 list 仍 generated-but-not-integrated。
	driver.Register(driver.Descriptor{
		Vendor: "huawei", Module: "xpl",
		MatchRoute:      func(p string) bool { return strings.HasPrefix(p, "/xpl:xpl") },
		ControllerToken: "xpl",
		MatchDecode:     func(p string) bool { return strings.HasPrefix(p, "/xpl:xpl") },
		DecodeXML: func(raw []byte) (ygot.GoStruct, error) {
			v := &huawei.HuaweiXpl_Xpl{}
			if err := xmlcodec.Decode(xplXML, raw, v); err != nil {
				return nil, err
			}
			return v, nil
		},
		MatchEncode: func(p string) bool { return strings.HasPrefix(p, "/xpl:xpl") },
		NewStruct:   func() ygot.GoStruct { return &huawei.HuaweiXpl_Xpl{} },
		Unmarshal:   huawei.Unmarshal,
		XML:         xplXML,
	})
}
