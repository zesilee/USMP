// Package hwfix provides shared Huawei-model test fixtures and golden-file
// helpers for the snd-xml-codec golden tests (D3): the client package freezes
// the legacy hand-written builders' output as canonical goldens, and the
// xmlcodec engine tests replay the same fixtures against the same goldens.
// Test-only: production code must never import this package.
package hwfix

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
)

// Update rewrites golden files instead of comparing when set:
//
//	go test ./pkg/yang-runtime/client/ -run Golden -args -update-golden
var Update = flag.Bool("update-golden", false, "rewrite golden files instead of comparing")

func ptr[T any](v T) *T { return &v }

// goldenDir resolves the golden directory next to this source file so both
// consuming test binaries (client, xmlcodec) share one copy.
func goldenDir() string {
	_, self, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(self), "golden")
}

// Golden reads the named canonical golden file.
func Golden(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(goldenDir(), name+".canon.txt"))
	if err != nil {
		t.Fatalf("read golden %s (run client golden test with -args -update-golden first): %v", name, err)
	}
	return string(b)
}

// WriteGolden writes the canonical form for name.
func WriteGolden(t *testing.T, name, canonical string) {
	t.Helper()
	if err := os.MkdirAll(goldenDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goldenDir(), name+".canon.txt"), []byte(canonical), 0o644); err != nil {
		t.Fatal(err)
	}
}

// VlanFull exercises every field the legacy buildHuaweiVlanVlansXML emits,
// on two entries (map iteration order nondeterminism), with nested
// member-ports / suppression / unkown-unicast-discard containers.
func VlanFull() *huawei.HuaweiVlan_Vlan_Vlans {
	return &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
			10: {
				Id:                      ptr(uint16(10)),
				Name:                    ptr("mgmt"),
				Description:             ptr("management vlan"),
				AdminStatus:             huawei.E_HuaweiVlan_AdminStatus(1),
				Type:                    huawei.E_HuaweiVlan_VlanType(1),
				BroadcastDiscard:        huawei.E_HuaweiVlan_EnableStatus(2),
				MacLearning:             huawei.E_HuaweiVlan_EnableStatus(1),
				StatisticEnable:         huawei.E_HuaweiVlan_EnableStatus(1),
				StatisticDiscard:        huawei.E_HuaweiVlan_EnableStatus(2),
				UnknownMulticastDiscard: huawei.E_HuaweiVlan_EnableStatus(1),
				MacAgingTime:            ptr(uint32(300)),
				SuperVlan:               ptr(uint16(100)),
				UnkownUnicastDiscard: &huawei.HuaweiVlan_Vlan_Vlans_Vlan_UnkownUnicastDiscard{
					Discard:           huawei.E_HuaweiVlan_EnableStatus(1),
					MacLearningEnable: huawei.E_HuaweiVlan_EnableStatus(2),
				},
				Suppression: &huawei.HuaweiVlan_Vlan_Vlans_Vlan_Suppression{
					Inbound:  huawei.E_HuaweiVlan_EnableStatus(1),
					Outbound: huawei.E_HuaweiVlan_EnableStatus(2),
				},
				MemberPorts: &huawei.HuaweiVlan_Vlan_Vlans_Vlan_MemberPorts{
					MemberPort: map[string]*huawei.HuaweiVlan_Vlan_Vlans_Vlan_MemberPorts_MemberPort{
						"GE0/0/1": {
							InterfaceName: ptr("GE0/0/1"),
							AccessType:    huawei.E_HuaweiVlan_AccessType(1),
							TagMode:       huawei.E_HuaweiVlan_TagMode(2),
						},
						"GE0/0/2": {
							InterfaceName: ptr("GE0/0/2"),
							AccessType:    huawei.E_HuaweiVlan_AccessType(2),
							TagMode:       huawei.E_HuaweiVlan_TagMode(1),
						},
					},
				},
			},
			20: {
				Id:          ptr(uint16(20)),
				Name:        ptr("users"),
				AdminStatus: huawei.E_HuaweiVlan_AdminStatus(2),
			},
		},
	}
}

// VlanMinimal is a single entry with only the key set.
func VlanMinimal() *huawei.HuaweiVlan_Vlan_Vlans {
	return &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
			30: {Id: ptr(uint16(30))},
		},
	}
}

// VlanEmpty is an initialized container with no entries.
func VlanEmpty() *huawei.HuaweiVlan_Vlan_Vlans {
	return &huawei.HuaweiVlan_Vlan_Vlans{Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{}}
}

// VlanEscape carries XML special characters through string leaves.
func VlanEscape() *huawei.HuaweiVlan_Vlan_Vlans {
	return &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
			40: {
				Id:          ptr(uint16(40)),
				Name:        ptr(`a<b&c"d'e>f`),
				Description: ptr("desc <&> quoted"),
			},
		},
	}
}

// IfmFull exercises every field the legacy buildHuaweiIfmInterfacesXML emits,
// on two entries, with nested damp/error-down/control-flap containers.
func IfmFull() *huawei.HuaweiIfm_Ifm_Interfaces {
	return &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{
			"GE0/0/1": {
				Name:                 ptr("GE0/0/1"),
				Description:          ptr("uplink"),
				AdminStatus:          huawei.E_HuaweiIfm_PortStatus(2),
				Mtu:                  ptr(uint32(1500)),
				Type:                 huawei.E_HuaweiIfm_PortType(1),
				Bandwidth:            ptr(uint32(1000)),
				BandwidthKbps:        ptr(uint32(1000000)),
				MacAddress:           ptr("00-11-22-33-44-55"),
				Class:                huawei.E_HuaweiIfm_ClassType(1),
				ServiceType:          huawei.E_HuaweiIfm_ServiceType(1),
				LinkProtocol:         huawei.E_HuaweiIfm_LinkProtocol(1),
				EncapsulationType:    huawei.E_HuaweiIfm_EncapsulationType(1),
				RouterType:           huawei.E_HuaweiIfm_RouterType(1),
				NetworkLayerStatus:   huawei.E_HuaweiIfm_NetworkLayerState(1),
				Index:                ptr(uint32(7)),
				Number:               ptr("0/0/1"),
				ParentName:           ptr("GE0/0"),
				Position:             ptr("0/0"),
				VrfName:              ptr("vrf-a"),
				VsName:               ptr("vs-a"),
				AggregationName:      ptr("Eth-Trunk1"),
				ClearIpDf:            ptr(true),
				IsL2Switch:           ptr(false),
				L2ModeEnable:         ptr(true),
				LinkUpDownTrapEnable: ptr(false),
				SpreadMtuFlag:        ptr(true),
				StatisticEnable:      ptr(true),
				StatisticInterval:    ptr(uint32(30)),
				StatisticMode:        huawei.E_HuaweiIfm_StatisticMode(1),
				L2SwitchPortIndex:    ptr(uint32(3)),
				DownDelayTime:        ptr(uint32(5)),
				ProtocolUpDelayTime:  ptr(uint32(10)),
				Damp: &huawei.HuaweiIfm_Ifm_Interfaces_Interface_Damp{
					Auto: &huawei.HuaweiIfm_Ifm_Interfaces_Interface_Damp_Auto{
						Level: huawei.E_HuaweiIfm_DampLevelType(1),
					},
					Manual: &huawei.HuaweiIfm_Ifm_Interfaces_Interface_Damp_Manual{
						HalfLifePeriod:  ptr(uint16(60)),
						MaxSuppressTime: ptr(uint16(120)),
						Reuse:           ptr(uint32(750)),
						Suppress:        ptr(uint32(2000)),
					},
				},
				ErrorDown: &huawei.HuaweiIfm_Ifm_Interfaces_Interface_ErrorDown{
					Cause:         huawei.E_HuaweiIfm_ErrorDownType(1),
					RecoveryTime:  ptr(uint32(30)),
					RemainderTime: ptr(uint32(15)),
				},
				ControlFlap: &huawei.HuaweiIfm_Ifm_Interfaces_Interface_ControlFlap{
					Ceiling:          ptr(uint32(16000)),
					Reuse:            ptr(uint32(750)),
					Suppress:         ptr(uint32(2000)),
					DecayOk:          ptr(uint32(600)),
					DecayNg:          ptr(uint32(900)),
					ControlFlapCount: ptr(uint32(4)),
				},
			},
			"GE0/0/2": {
				Name:        ptr("GE0/0/2"),
				AdminStatus: huawei.E_HuaweiIfm_PortStatus(1),
				Mtu:         ptr(uint32(9000)),
			},
		},
	}
}

// IfmMinimal is a single entry with only the key set.
func IfmMinimal() *huawei.HuaweiIfm_Ifm_Interfaces {
	return &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{
			"LoopBack0": {Name: ptr("LoopBack0")},
		},
	}
}

// IfmEmpty is an initialized container with no entries.
func IfmEmpty() *huawei.HuaweiIfm_Ifm_Interfaces {
	return &huawei.HuaweiIfm_Ifm_Interfaces{Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{}}
}

// VlanDeleteSet is a key-only VLAN set for delete encoding, two entries.
func VlanDeleteSet() *huawei.HuaweiVlan_Vlan_Vlans {
	return &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
			10: {Id: ptr(uint16(10))},
			20: {Id: ptr(uint16(20))},
		},
	}
}

// IfmDeleteSet is a key-only interface set for delete encoding, including the
// map-key fallback shape (nil Name) the legacy encoder tolerated.
func IfmDeleteSet() *huawei.HuaweiIfm_Ifm_Interfaces {
	return &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{
			"GE0/0/1": {Name: ptr("GE0/0/1")},
			"GE0/0/2": {},
		},
	}
}
