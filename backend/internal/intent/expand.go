// Package intent implements the business-network-config orchestration layer:
// intent CR specs expand into per-device native config fragments that ride the
// existing Stack B declarative pipeline (BIO-02). The expansion is a pure
// function of the spec — no local state, no history — so any replica replays
// the same result from the CR alone (BIO-08 多实例就绪).
package intent

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/leezesi/usmp/backend/internal/generated/business"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/openconfig/ygot/ygot"
)

// Native desired paths — identical to the vlan/ifm controllers and config-api
// so the declarative reconcile, cache invalidation and DELETE command channel
// work on intent-expanded config unchanged (BVS-02).
const (
	VlanPath = "/vlan:vlan/vlan:vlans"
	IfmPath  = "/ifm:ifm/ifm:interfaces"
)

// Fragment is one expanded native-config unit: what the orchestrator writes
// into the desired ConfigStore for one (device, native module) pair after the
// cross-device transaction succeeds (BIO-03).
type Fragment struct {
	Device string        // device IP (DeviceStore / ConfigStore key)
	Module string        // native module root name ("vlan" / "ifm")
	Path   string        // native desired path
	Config ygot.GoStruct // typed native config (huawei generated, R04)
}

// Claim records soft ownership of one native entry on one device (BIO-07).
// Paths are entry-level (list key qualified) so the ownership index can flag
// manual edits and detect cross-intent conflicts.
type Claim struct {
	Device string `json:"device"`
	Module string `json:"module"`
	Path   string `json:"path"`
}

// ExpandBusinessVlan expands a cross-device VLAN intent into per-device
// huawei-vlan + huawei-ifm fragments (BVS-02):
//   - every device gets the VLAN entry {id, name} (name defaults to VLAN<id>);
//   - access ports get link-type=access + pvid=<id>;
//   - trunk ports get link-type=trunk + trunk-vlans=<id>.
//
// Deterministic: devices ascending by IP, vlan fragment before ifm per device.
// Blank port names are skipped (R08 — a sloppy CR must not panic the expander).
func ExpandBusinessVlan(spec *business.UsmpBusinessVlan_BusinessVlanService) ([]Fragment, []Claim, error) {
	if spec == nil {
		return nil, nil, fmt.Errorf("expand business-vlan: nil spec")
	}
	if spec.VlanId == nil {
		return nil, nil, fmt.Errorf("expand business-vlan: missing vlan-id")
	}
	if len(spec.Devices) == 0 {
		return nil, nil, fmt.Errorf("expand business-vlan: devices is empty (min-elements 1)")
	}
	vlanID := *spec.VlanId
	name := fmt.Sprintf("VLAN%d", vlanID)
	if spec.Name != nil && *spec.Name != "" {
		name = *spec.Name
	}

	ips := make([]string, 0, len(spec.Devices))
	for ip := range spec.Devices {
		ips = append(ips, ip)
	}
	sort.Strings(ips)

	var frags []Fragment
	var claims []Claim
	for _, ip := range ips {
		dev := spec.Devices[ip]
		device := ip
		if dev != nil && dev.Ip != nil && *dev.Ip != "" {
			device = *dev.Ip
		}

		id := vlanID
		vlanName := name
		frags = append(frags, Fragment{
			Device: device,
			Module: "vlan",
			Path:   VlanPath,
			Config: &huawei.HuaweiVlan_Vlan_Vlans{
				Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
					id: {Id: &id, Name: &vlanName},
				},
			},
		})
		claims = append(claims, Claim{
			Device: device,
			Module: "vlan",
			Path:   fmt.Sprintf("%s/vlan[id=%d]", VlanPath, vlanID),
		})

		ifaces := map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{}
		var portNames []string
		if dev != nil {
			for _, port := range dev.AccessPorts {
				port = strings.TrimSpace(port)
				if port == "" {
					continue
				}
				pvid := vlanID
				ifaces[port] = ifmPortEntry(port, &huawei.HuaweiIfm_Ifm_Interfaces_Interface_Ethernet_MainInterface_L2Attribute{
					LinkType: huawei.HuaweiEthernet_LinkType_access,
					Pvid:     &pvid,
				})
				portNames = append(portNames, port)
			}
			for _, port := range dev.TrunkPorts {
				port = strings.TrimSpace(port)
				if port == "" {
					continue
				}
				trunk := strconv.Itoa(int(vlanID))
				ifaces[port] = ifmPortEntry(port, &huawei.HuaweiIfm_Ifm_Interfaces_Interface_Ethernet_MainInterface_L2Attribute{
					LinkType:   huawei.HuaweiEthernet_LinkType_trunk,
					TrunkVlans: &trunk,
				})
				portNames = append(portNames, port)
			}
		}
		if len(ifaces) > 0 {
			frags = append(frags, Fragment{
				Device: device,
				Module: "ifm",
				Path:   IfmPath,
				Config: &huawei.HuaweiIfm_Ifm_Interfaces{Interface: ifaces},
			})
			sort.Strings(portNames)
			for _, port := range portNames {
				claims = append(claims, Claim{
					Device: device,
					Module: "ifm",
					Path:   fmt.Sprintf("%s/interface[name=%s]", IfmPath, port),
				})
			}
		}
	}
	return frags, claims, nil
}

func ifmPortEntry(name string, l2 *huawei.HuaweiIfm_Ifm_Interfaces_Interface_Ethernet_MainInterface_L2Attribute) *huawei.HuaweiIfm_Ifm_Interfaces_Interface {
	n := name
	return &huawei.HuaweiIfm_Ifm_Interfaces_Interface{
		Name: &n,
		Ethernet: &huawei.HuaweiIfm_Ifm_Interfaces_Interface_Ethernet{
			MainInterface: &huawei.HuaweiIfm_Ifm_Interfaces_Interface_Ethernet_MainInterface{
				L2Attribute: l2,
			},
		},
	}
}
