package drivers

import (
	"strings"

	"github.com/leezesi/usmp/backend/internal/generated/native/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/driver"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/object"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/xmlcodec"
)

// plainModule 一行数据描述一个「单容器根、走通用 XML 引擎」的华为模块（DR-06）。
// module = 根容器名 = 路由 token = ControllerToken = 控制台 /module/<module>；
// 运行时配置路径口径为根名前缀段（/<root>:<root>/...，与前端 configPathFor 一致），
// YANG prefix 只属 XML namespace 层、不作路由 token。
// namespace 须显式登记：内嵌 gzip schema 的 Entry.Namespace() 实测返回空，不可派生。
type plainModule struct {
	module string
	ns     string
	newFn  func() object.Object
}

// plainModules 是全量接入（full-yang-onboarding）的模块表：49 个新叶模块
//   - 4 个此前「已生成未注册」的漏网叶（bfd/l3vpn/routing/time-range）
//   - 4 个自手写块迁入的 2b 波次模块（tunnel-management/xpl/routing-policy/acl，
//     其中 tunnel-management/routing-policy 由 YANG prefix 锚迁为根名锚——断链修复）。
//
// 行序按根名字典序；加新模块 = 加一行 + gen.conf 加模块名（勿写手写块）。
var plainModules = []plainModule{
	{"acl", "urn:huawei:yang:huawei-acl", func() object.Object { return &huawei.HuaweiAcl_Acl{} }},
	{"analysis-collector", "urn:huawei:yang:huawei-analysis-collector", func() object.Object { return &huawei.HuaweiAnalysisCollector_AnalysisCollector{} }},
	{"anyflow", "urn:huawei:yang:huawei-anyflow", func() object.Object { return &huawei.HuaweiAnyflow_Anyflow{} }},
	{"arp", "urn:huawei:yang:huawei-arp", func() object.Object { return &huawei.HuaweiArp_Arp{} }},
	{"bd", "urn:huawei:yang:huawei-bd", func() object.Object { return &huawei.HuaweiBd_Bd{} }},
	{"bfd", "urn:huawei:yang:huawei-bfd", func() object.Object { return &huawei.HuaweiBfd_Bfd{} }},
	{"cfg", "urn:huawei:yang:huawei-cfg", func() object.Object { return &huawei.HuaweiCfg_Cfg{} }},
	{"devm", "urn:huawei:yang:huawei-devm", func() object.Object { return &huawei.HuaweiDevm_Devm{} }},
	{"driver", "urn:huawei:yang:huawei-driver", func() object.Object { return &huawei.HuaweiDriver_Driver{} }},
	{"dsa", "urn:huawei:yang:huawei-dsa", func() object.Object { return &huawei.HuaweiDsa_Dsa{} }},
	{"ecc", "urn:huawei:yang:huawei-ecc", func() object.Object { return &huawei.HuaweiEcc_Ecc{} }},
	{"evpn", "urn:huawei:yang:huawei-evpn", func() object.Object { return &huawei.HuaweiEvpn_Evpn{} }},
	{"fib", "urn:huawei:yang:huawei-fib", func() object.Object { return &huawei.HuaweiFib_Fib{} }},
	{"ftpc", "urn:huawei:yang:huawei-ftpc", func() object.Object { return &huawei.HuaweiFtpc_Ftpc{} }},
	{"grpc", "urn:huawei:yang:huawei-grpc", func() object.Object { return &huawei.HuaweiGrpc_Grpc{} }},
	{"hwtacacs", "urn:huawei:yang:huawei-hwtacacs", func() object.Object { return &huawei.HuaweiHwtacacs_Hwtacacs{} }},
	{"ifm-trunk", "urn:huawei:yang:huawei-ifm-trunk", func() object.Object { return &huawei.HuaweiIfmTrunk_IfmTrunk{} }},
	{"l3-multicast", "urn:huawei:yang:huawei-l3-multicast", func() object.Object { return &huawei.HuaweiL3Multicast_L3Multicast{} }},
	{"l3vpn", "urn:huawei:yang:huawei-l3vpn", func() object.Object { return &huawei.HuaweiL3Vpn_L3Vpn{} }},
	{"lacp", "urn:huawei:yang:huawei-lacp", func() object.Object { return &huawei.HuaweiLacp_Lacp{} }},
	{"license", "urn:huawei:yang:huawei-license", func() object.Object { return &huawei.HuaweiLicense_License{} }},
	{"lldp", "urn:huawei:yang:huawei-lldp", func() object.Object { return &huawei.HuaweiLldp_Lldp{} }},
	{"loadbalance", "urn:huawei:yang:huawei-loadbalance", func() object.Object { return &huawei.HuaweiLoadbalance_Loadbalance{} }},
	{"m-lag", "urn:huawei:yang:huawei-m-lag", func() object.Object { return &huawei.HuaweiMLag_MLag{} }},
	{"mac-flapping-detect", "urn:huawei:yang:huawei-mac-flapping-detect", func() object.Object { return &huawei.HuaweiMacFlappingDetect_MacFlappingDetect{} }},
	{"macsec", "urn:huawei:yang:huawei-macsec", func() object.Object { return &huawei.HuaweiMacsec_Macsec{} }},
	{"microsegmentation", "urn:huawei:yang:huawei-microsegmentation", func() object.Object { return &huawei.HuaweiMicrosegmentation_Microsegmentation{} }},
	{"mirror", "urn:huawei:yang:huawei-mirror", func() object.Object { return &huawei.HuaweiMirror_Mirror{} }},
	{"monitor-link", "urn:huawei:yang:huawei-monitor-link", func() object.Object { return &huawei.HuaweiMonitorLink_MonitorLink{} }},
	{"mstp", "urn:huawei:yang:huawei-mstp", func() object.Object { return &huawei.HuaweiMstp_Mstp{} }},
	{"multicast", "urn:huawei:yang:huawei-multicast", func() object.Object { return &huawei.HuaweiMulticast_Multicast{} }},
	{"mvpn", "urn:huawei:yang:huawei-mvpn", func() object.Object { return &huawei.HuaweiMvpn_Mvpn{} }},
	{"nqa", "urn:huawei:yang:huawei-nqa", func() object.Object { return &huawei.HuaweiNqa_Nqa{} }},
	{"ntp", "urn:huawei:yang:huawei-ntp", func() object.Object { return &huawei.HuaweiNtp_Ntp{} }},
	{"nvo3", "urn:huawei:yang:huawei-nvo3", func() object.Object { return &huawei.HuaweiNvo3_Nvo3{} }},
	{"openflow-agent", "urn:huawei:yang:huawei-openflow-agent", func() object.Object { return &huawei.HuaweiOpenflowAgent_OpenflowAgent{} }},
	{"ospfv2", "urn:huawei:yang:huawei-ospfv2", func() object.Object { return &huawei.HuaweiOspfv2_Ospfv2{} }},
	{"ospfv3", "urn:huawei:yang:huawei-ospfv3", func() object.Object { return &huawei.HuaweiOspfv3_Ospfv3{} }},
	{"packetevent", "urn:huawei:yang:huawei-packetevent", func() object.Object { return &huawei.HuaweiPacketevent_Packetevent{} }},
	{"qos", "urn:huawei:yang:huawei-qos", func() object.Object { return &huawei.HuaweiQos_Qos{} }},
	{"routing", "urn:huawei:yang:huawei-routing", func() object.Object { return &huawei.HuaweiRouting_Routing{} }},
	{"routing-policy", "urn:huawei:yang:huawei-routing-policy", func() object.Object { return &huawei.HuaweiRoutingPolicy_RoutingPolicy{} }},
	{"rsa", "urn:huawei:yang:huawei-rsa", func() object.Object { return &huawei.HuaweiRsa_Rsa{} }},
	{"sflow", "urn:huawei:yang:huawei-sflow", func() object.Object { return &huawei.HuaweiSflow_Sflow{} }},
	{"sm2", "urn:huawei:yang:huawei-sm2", func() object.Object { return &huawei.HuaweiSm2_Sm2{} }},
	{"snmp", "urn:huawei:yang:huawei-snmp", func() object.Object { return &huawei.HuaweiSnmp_Snmp{} }},
	{"syslog", "urn:huawei:yang:huawei-syslog", func() object.Object { return &huawei.HuaweiSyslog_Syslog{} }},
	{"system-resources-usage", "urn:huawei:yang:huawei-system-resources-usage", func() object.Object { return &huawei.HuaweiSystemResourcesUsage_SystemResourcesUsage{} }},
	{"telemetry-system", "http://openconfig.net/yang/telemetry", func() object.Object { return &huawei.OpenconfigTelemetry_TelemetrySystem{} }},
	{"time-range", "urn:huawei:yang:huawei-time-range", func() object.Object { return &huawei.HuaweiTimeRange_TimeRange{} }},
	{"tunnel-management", "urn:huawei:yang:huawei-tunnel-management", func() object.Object { return &huawei.HuaweiTunnelManagement_TunnelManagement{} }},
	{"unicast-forward", "urn:huawei:yang:huawei-unicast-forward", func() object.Object { return &huawei.HuaweiUnicastForward_UnicastForward{} }},
	{"vrrp", "urn:huawei:yang:huawei-vrrp", func() object.Object { return &huawei.HuaweiVrrp_Vrrp{} }},
	{"vty", "urn:huawei:yang:huawei-vty", func() object.Object { return &huawei.HuaweiVty_Vty{} }},
	{"vxlan-ext", "urn:huawei:yang:huawei-vxlan-ext", func() object.Object { return &huawei.HuaweiVxlanExt_VxlanExt{} }},
	{"vxlan-path-detect", "urn:huawei:yang:huawei-vxlan-path-detect", func() object.Object { return &huawei.HuaweiVxlanPathDetect_VxlanPathDetect{} }},
	{"xpl", "urn:huawei:yang:huawei-xpl", func() object.Object { return &huawei.HuaweiXpl_Xpl{} }},
}

// registerPlain 按表行派生谓词/锚点/编解码数据并注册描述符。
func registerPlain(pm plainModule) {
	anchor := "/" + pm.module + ":" + pm.module
	spec := &xmlcodec.Spec{
		Namespace: pm.ns,
		// DR-06 口径：plain 模块的根容器名 = 模块名 = 数据路径首段。
		Schema: irNode("/" + pm.module),
	}
	match := func(p string) bool { return strings.HasPrefix(p, anchor) }
	newFn := pm.newFn
	driver.Register(driver.Descriptor{
		Vendor: "huawei", Module: pm.module,
		MatchRoute:      match,
		ControllerToken: pm.module,
		MatchDecode:     match,
		DecodeXML: func(raw []byte) (object.Object, error) {
			v := newFn()
			if err := xmlcodec.Decode(spec, raw, v); err != nil {
				return nil, err
			}
			return v, nil
		},
		MatchEncode:  match,
		NewStruct:    newFn,
		EncodeAnchor: anchor,
		Unmarshal:    huawei.Unmarshal,
		XML:          spec,
	})
}

func init() {
	for _, pm := range plainModules {
		registerPlain(pm)
	}
}
