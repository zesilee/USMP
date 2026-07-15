package netconfsim

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/openconfig/ygot/ygot"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
)

// enumInt resolves a YANG enumeration leaf's on-wire text to its ygot int value.
// 真机/通用引擎 encode 发值域名（如 "up"），本模拟器读回后经 sample 枚举的 ΛMap 反查
// 名→int（XC-08）。兼容历史整数形态（回退 Atoi）。sample 是该叶对应 ygot 枚举的零值，
// 提供 ΛMap 与类型名。text 为空/未知返回 0（UNSET）。
func enumInt(text string, sample ygot.GoEnum) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}
	if n, err := strconv.Atoi(text); err == nil {
		return n
	}
	for v, def := range sample.ΛMap()[reflect.TypeOf(sample).Name()] {
		if def.Name == text {
			return int(v)
		}
	}
	return 0
}

// Structured, tree-based read queries over the running config, replacing the
// legacy blob Datastore's XML token-walking Extract* helpers. testsupport (and a
// few actor integration tests) call these to assert device state.
//
// Element matching is case-insensitive and accepts both the kebab-case YANG form
// produced by the NETCONF client and the PascalCase Go-marshal form produced by
// seeding from a ygot Device — mirroring the dual-case matching the old Extract*
// relied on.

// runningRoot parses the running config into a fresh, isolated tree so queries
// never race with concurrent writes.
func (s *Simulator) runningRoot() *dataNode {
	root, err := parseXML(s.store.GetRunning())
	if err != nil {
		return &dataNode{}
	}
	return root
}

// childFold returns the first direct child whose local name case-insensitively
// equals any of the given names, or nil.
func (n *dataNode) childFold(names ...string) *dataNode {
	for _, c := range n.Children {
		for _, name := range names {
			if strings.EqualFold(c.Name.Local, name) {
				return c
			}
		}
	}
	return nil
}

// leaf returns the trimmed text of the first direct child matching any name.
func (n *dataNode) leaf(names ...string) string {
	if c := n.childFold(names...); c != nil {
		return c.leafText()
	}
	return ""
}

// descendants returns every node in the subtree (excluding the root) for which
// match reports true.
func (n *dataNode) descendants(match func(*dataNode) bool) []*dataNode {
	var out []*dataNode
	var walk func(*dataNode)
	walk = func(cur *dataNode) {
		for _, c := range cur.Children {
			if match(c) {
				out = append(out, c)
			}
			walk(c)
		}
	}
	walk(n)
	return out
}

func toInt(s string) int { v, _ := strconv.Atoi(strings.TrimSpace(s)); return v }
func toU16(s string) uint16 {
	v, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 16)
	return uint16(v)
}
func toU32(s string) uint32 {
	v, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 32)
	return uint32(v)
}
func toBool(s string) bool { v, _ := strconv.ParseBool(strings.TrimSpace(s)); return v }
func nameHas(n *dataNode, sub string) bool {
	return strings.Contains(strings.ToLower(n.Name.Local), sub)
}

// isHuaweiVlanEntry matches a <vlan> (or PascalCase HuaweiVlan…Vlan) list entry —
// i.e. one carrying an <id>/<vlan-id> key, excluding the outer <vlan> container.
func isHuaweiVlanEntry(n *dataNode) bool {
	if !strings.EqualFold(n.Name.Local, "vlan") &&
		!(strings.Contains(n.Name.Local, "HuaweiVlan") && strings.Contains(n.Name.Local, "Vlan")) {
		return false
	}
	return n.childFold("id", "vlan-id", "vlanid") != nil
}

// RunningHuaweiVLANs returns id→name for Huawei VLANs in the running config.
func (s *Simulator) RunningHuaweiVLANs() map[uint16]string {
	out := make(map[uint16]string)
	for _, v := range s.runningRoot().descendants(isHuaweiVlanEntry) {
		id := toU16(v.leaf("id", "vlan-id", "vlanid"))
		if id == 0 {
			continue
		}
		out[id] = v.leaf("name")
	}
	return out
}

// RunningHuaweiVLANsFull returns id→full attributes for Huawei VLANs.
func (s *Simulator) RunningHuaweiVLANsFull() map[uint16]*HuaweiVlanTestData {
	out := make(map[uint16]*HuaweiVlanTestData)
	for _, v := range s.runningRoot().descendants(isHuaweiVlanEntry) {
		id := toU16(v.leaf("id", "vlan-id", "vlanid"))
		if id == 0 {
			continue
		}
		d := &HuaweiVlanTestData{
			ID:                      id,
			Name:                    v.leaf("name"),
			Description:             v.leaf("description"),
			Type:                    enumInt(v.leaf("type"), huawei.HuaweiVlan_VlanType_UNSET),
			AdminStatus:             enumInt(v.leaf("admin-status", "adminstatus"), huawei.HuaweiVlan_AdminStatus_UNSET),
			BroadcastDiscard:        enumInt(v.leaf("broadcast-discard", "broadcastdiscard"), huawei.HuaweiVlan_EnableStatus_UNSET),
			UnknownMulticastDiscard: enumInt(v.leaf("unknown-multicast-discard", "unknownmulticastdiscard"), huawei.HuaweiVlan_EnableStatus_UNSET),
			MacLearning:             enumInt(v.leaf("mac-learning", "maclearning"), huawei.HuaweiVlan_EnableStatus_UNSET),
			MacAgingTime:            toU32(v.leaf("mac-aging-time", "macagingtime")),
			StatisticEnable:         enumInt(v.leaf("statistic-enable", "statisticenable"), huawei.HuaweiVlan_EnableStatus_UNSET),
			StatisticDiscard:        enumInt(v.leaf("statistic-discard", "statisticdiscard"), huawei.HuaweiVlan_EnableStatus_UNSET),
		}
		if sv := v.childFold("super-vlan", "supervlan"); sv != nil {
			x := toU16(sv.leafText())
			d.SuperVlan = &x
		}
		if u := v.childFold("unkown-unicast-discard", "unknown-unicast-discard", "unkownunicastdiscard"); u != nil {
			d.UnkownUnicastDiscard.Discard = enumInt(u.leaf("discard"), huawei.HuaweiVlan_EnableStatus_UNSET)
			d.UnkownUnicastDiscard.MacLearningEnable = enumInt(u.leaf("mac-learning-enable", "maclearningenable"), huawei.HuaweiVlan_EnableStatus_UNSET)
		}
		if sup := v.childFold("suppression"); sup != nil {
			d.Suppression.Inbound = enumInt(sup.leaf("inbound"), huawei.HuaweiVlan_EnableStatus_UNSET)
			d.Suppression.Outbound = enumInt(sup.leaf("outbound"), huawei.HuaweiVlan_EnableStatus_UNSET)
		}
		if mp := v.childFold("member-ports", "memberports"); mp != nil {
			for _, c := range mp.Children {
				if strings.EqualFold(c.Name.Local, "member-port") || strings.EqualFold(c.Name.Local, "memberport") {
					d.MemberPorts = append(d.MemberPorts, HuaweiVlanMemberPort{
						InterfaceName: c.leaf("interface-name", "interfacename"),
						AccessType:    enumInt(c.leaf("access-type", "accesstype"), huawei.HuaweiVlan_AccessType_UNSET),
						TagMode:       enumInt(c.leaf("tag-mode", "tagmode"), huawei.HuaweiVlan_TagMode_UNSET),
					})
				}
			}
		}
		out[id] = d
	}
	return out
}

// isHuaweiInterfaceEntry matches a Huawei IFM <interface> list entry (carrying a
// <name> key), excluding the outer <interfaces> container.
func isHuaweiInterfaceEntry(n *dataNode) bool {
	return nameHas(n, "interface") && n.childFold("name") != nil
}

// RunningHuaweiInterfaces returns name→full attributes for Huawei IFM interfaces.
func (s *Simulator) RunningHuaweiInterfaces() map[string]*HuaweiInterfaceTestData {
	out := make(map[string]*HuaweiInterfaceTestData)
	for _, e := range s.runningRoot().descendants(isHuaweiInterfaceEntry) {
		name := e.leaf("name")
		if name == "" {
			continue
		}
		d := &HuaweiInterfaceTestData{
			Name:                 name,
			Description:          e.leaf("description"),
			Index:                toU32(e.leaf("index")),
			Number:               e.leaf("number"),
			Position:             e.leaf("position"),
			ParentName:           e.leaf("parent-name", "parentname"),
			AdminStatus:          enumInt(e.leaf("admin-status", "adminstatus"), huawei.HuaweiIfm_PortStatus_UNSET),
			Type:                 enumInt(e.leaf("type"), huawei.HuaweiIfm_PortType_UNSET),
			Class:                enumInt(e.leaf("class"), huawei.HuaweiIfm_ClassType_UNSET),
			LinkProtocol:         enumInt(e.leaf("link-protocol", "linkprotocol"), huawei.HuaweiIfm_LinkProtocol_UNSET),
			RouterType:           enumInt(e.leaf("router-type", "routertype"), huawei.HuaweiIfm_RouterType_UNSET),
			ServiceType:          enumInt(e.leaf("service-type", "servicetype"), huawei.HuaweiIfm_ServiceType_UNSET),
			Mtu:                  toU32(e.leaf("mtu")),
			MacAddress:           e.leaf("mac-address", "macaddress"),
			Bandwidth:            toU32(e.leaf("bandwidth")),
			BandwidthKbps:        toU32(e.leaf("bandwidth-kbps", "bandwidthkbps")),
			VrfName:              e.leaf("vrf-name", "vrfname"),
			VsName:               e.leaf("vs-name", "vsname"),
			AggregationName:      e.leaf("aggregation-name", "aggregationname"),
			DownDelayTime:        toU32(e.leaf("down-delay-time", "downdelaytime")),
			ProtocolUpDelayTime:  toU32(e.leaf("protocol-up-delay-time", "protocolupdelaytime")),
			ClearIpDf:            toBool(e.leaf("clear-ip-df", "clearipdf")),
			IsL2Switch:           toBool(e.leaf("is-l2-switch", "isl2switch")),
			L2ModeEnable:         toBool(e.leaf("l2-mode-enable", "l2modeenable")),
			LinkUpDownTrapEnable: toBool(e.leaf("link-up-down-trap-enable", "linkupdowntrapenable")),
			StatisticEnable:      toBool(e.leaf("statistic-enable", "statisticenable")),
			SpreadMtuFlag:        toBool(e.leaf("spread-mtu-flag", "spreadmtuflag")),
			StatisticInterval:    toU32(e.leaf("statistic-interval", "statisticinterval")),
			StatisticMode:        enumInt(e.leaf("statistic-mode", "statisticmode"), huawei.HuaweiIfm_StatisticMode_UNSET),
		}
		if cf := e.childFold("control-flap", "controlflap"); cf != nil {
			d.ControlFlap.Ceiling = toU32(cf.leaf("ceiling"))
			d.ControlFlap.ControlFlapCount = toU32(cf.leaf("control-flap-count", "controlflapcount"))
			d.ControlFlap.DecayNg = toU32(cf.leaf("decay-ng", "decayng"))
			d.ControlFlap.DecayOk = toU32(cf.leaf("decay-ok", "decayok"))
			d.ControlFlap.Reuse = toU32(cf.leaf("reuse"))
			d.ControlFlap.Suppress = toU32(cf.leaf("suppress"))
		}
		if eth := e.childFold("ethernet"); eth != nil {
			if main := eth.childFold("main-interface", "maininterface"); main != nil {
				if l2 := main.childFold("l2-attribute", "l2attribute"); l2 != nil {
					d.L2.LinkType = enumInt(l2.leaf("link-type", "linktype"), huawei.HuaweiEthernet_LinkType_UNSET)
					d.L2.Pvid = toU16(l2.leaf("pvid"))
					d.L2.TrunkVlans = l2.leaf("trunk-vlans", "trunkvlans")
				}
			}
		}
		if damp := e.childFold("damp"); damp != nil {
			d.Damp.TxOff = toBool(damp.leaf("tx-off", "txoff"))
			if auto := damp.childFold("auto"); auto != nil {
				d.Damp.Auto.Level = toInt(auto.leaf("level"))
			}
			if manual := damp.childFold("manual"); manual != nil {
				d.Damp.Manual.HalfLifePeriod = toU16(manual.leaf("half-life-period", "halflifeperiod"))
				d.Damp.Manual.MaxSuppressTime = toU16(manual.leaf("max-suppress-time", "maxsuppresstime"))
				d.Damp.Manual.Reuse = toU32(manual.leaf("reuse"))
				d.Damp.Manual.Suppress = toU32(manual.leaf("suppress"))
			}
		}
		out[name] = d
	}
	return out
}

// RunningHuaweiSystem returns Huawei system info gathered from anywhere in the tree.
func (s *Simulator) RunningHuaweiSystem() *HuaweiSystemTestData {
	sys := &HuaweiSystemTestData{}
	for _, n := range s.runningRoot().descendants(func(*dataNode) bool { return true }) {
		switch {
		case strings.EqualFold(n.Name.Local, "sys-name"), strings.EqualFold(n.Name.Local, "sysname"):
			sys.SysName = n.leafText()
		case strings.EqualFold(n.Name.Local, "sys-contact"), strings.EqualFold(n.Name.Local, "syscontact"):
			sys.SysContact = n.leafText()
		case strings.EqualFold(n.Name.Local, "sys-location"), strings.EqualFold(n.Name.Local, "syslocation"):
			sys.SysLocation = n.leafText()
		}
	}
	return sys
}

// OCInterfaceView holds the OpenConfig interface attributes asserted by tests.
type OCInterfaceView struct {
	Name        string
	Enabled     *bool
	Mtu         *uint16
	Description string
}

// isOCInterfaceEntry matches an OpenConfig <interface> list entry.
func isOCInterfaceEntry(n *dataNode) bool {
	if !strings.EqualFold(n.Name.Local, "interface") &&
		n.Name.Local != "OpenconfigInterfaces_Interfaces_Interface" {
		return false
	}
	return n.childFold("name") != nil || n.childFold("config") != nil
}

// RunningOCInterfaces returns name→OpenConfig interface attributes. Values may sit
// directly on the interface or nested under <config>; the nested form wins.
func (s *Simulator) RunningOCInterfaces() map[string]*OCInterfaceView {
	out := make(map[string]*OCInterfaceView)
	for _, e := range s.runningRoot().descendants(isOCInterfaceEntry) {
		v := &OCInterfaceView{Name: e.leaf("name")}
		read := func(container *dataNode) {
			if container == nil {
				return
			}
			if nm := container.childFold("name"); nm != nil && v.Name == "" {
				v.Name = nm.leafText()
			}
			if en := container.childFold("enabled"); en != nil {
				b := toBool(en.leafText())
				v.Enabled = &b
			}
			if mt := container.childFold("mtu"); mt != nil {
				x := toU16(mt.leafText())
				v.Mtu = &x
			}
			if d := container.childFold("description"); d != nil {
				v.Description = d.leafText()
			}
		}
		read(e)
		read(e.childFold("config"))
		if v.Name != "" {
			out[v.Name] = v
		}
	}
	return out
}

// View structs returned by the Running* queries. Moved here from the deleted
// blob datastore.go; field shapes are unchanged so assertion call sites stay put.

// HuaweiVlanTestData contains parsed VLAN data for testing assertions.
type HuaweiVlanTestData struct {
	ID                      uint16
	Name                    string
	Description             string
	Type                    int
	AdminStatus             int
	BroadcastDiscard        int
	UnknownMulticastDiscard int
	MacLearning             int
	MacAgingTime            uint32
	StatisticEnable         int
	StatisticDiscard        int
	SuperVlan               *uint16
	// Nested containers
	UnkownUnicastDiscard struct {
		Discard           int
		MacLearningEnable int
	}
	Suppression struct {
		Inbound  int
		Outbound int
	}
	// 端口成员（VLAN 最核心功能：哪些口属于该 VLAN + 端口模式）
	MemberPorts []HuaweiVlanMemberPort
}

// HuaweiVlanMemberPort is a parsed member-port entry for test assertions.
type HuaweiVlanMemberPort struct {
	InterfaceName string
	AccessType    int
	TagMode       int
}

// HuaweiInterfaceTestData represents interface test data from Huawei IFM model.
type HuaweiInterfaceTestData struct {
	Name                 string
	Description          string
	Index                uint32
	Number               string
	Position             string
	ParentName           string
	AdminStatus          int
	Type                 int
	Class                int
	LinkProtocol         int
	RouterType           int
	ServiceType          int
	Mtu                  uint32
	MacAddress           string
	Bandwidth            uint32
	BandwidthKbps        uint32
	VrfName              string
	VsName               string
	AggregationName      string
	DownDelayTime        uint32
	ProtocolUpDelayTime  uint32
	ClearIpDf            bool
	IsL2Switch           bool
	L2ModeEnable         bool
	LinkUpDownTrapEnable bool
	StatisticEnable      bool
	SpreadMtuFlag        bool
	StatisticInterval    uint32
	StatisticMode        int
	// Nested containers
	ControlFlap struct {
		Ceiling          uint32
		ControlFlapCount uint32
		DecayNg          uint32
		DecayOk          uint32
		Reuse            uint32
		Suppress         uint32
	}
	Damp struct {
		TxOff bool
		Auto  struct {
			Level int
		}
		Manual struct {
			HalfLifePeriod  uint16
			MaxSuppressTime uint16
			Reuse           uint32
			Suppress        uint32
		}
	}
	// L2 attributes（ethernet/main-interface/l2-attribute，huawei-ethernet
	// augment）——业务 VLAN 打通展开的端口断言面（矩阵 A1/A2）。
	L2 struct {
		LinkType   int
		Pvid       uint16
		TrunkVlans string
	}
}

// HuaweiSystemTestData represents system configuration test data.
type HuaweiSystemTestData struct {
	SysName     string
	SysContact  string
	SysLocation string
}
