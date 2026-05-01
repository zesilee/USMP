package netconfsim

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/leezesi/usmp/backend/internal/generated/openconfig"
)

// Datastore represents NETCONF datastore with running and candidate configurations.
type Datastore struct {
	mu        sync.RWMutex
	running   []byte
	candidate []byte // candidate is copy of running until edited
}

// NewDatastore creates a new empty datastore.
func NewDatastore() *Datastore {
	emptyConfig := []byte(`<config/>`)
	return &Datastore{
		running:   emptyConfig,
		candidate: emptyConfig,
	}
}

// SetRunningFromDevice sets the running configuration from a Device struct.
	// This also updates candidate to match. Accepts any device struct type.
	func (d *Datastore) SetRunningFromDevice(dev interface{}) {
		d.mu.Lock()
		defer d.mu.Unlock()

		// Marshal to XML directly
		buf, err := xml.Marshal(dev)
		if err != nil {
			// Fallback to empty on error
			d.running = []byte(`<config/>`)
			d.candidate = []byte(`<config/>`)
			return
		}

		// Wrap in <config> tag if not already
		if !bytes.Contains(buf, []byte(`<config`)) {
			buf = []byte(fmt.Sprintf(`<config>%s</config>`, buf))
		}

		d.running = buf
	d.candidate = make([]byte, len(buf))
	copy(d.candidate, buf)
}

	// SetRunningFromXML sets the running configuration directly from XML bytes.
	// This also updates candidate to match.
	func (d *Datastore) SetRunningFromXML(xmlBytes []byte) {
		d.mu.Lock()
		defer d.mu.Unlock()

		d.running = xmlBytes
		d.candidate = make([]byte, len(xmlBytes))
		copy(d.candidate, xmlBytes)
	}


// GetRunning returns the current running configuration as XML.
func (d *Datastore) GetRunning() []byte {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.running
}

// GetCandidate returns the current candidate configuration as XML.
func (d *Datastore) GetCandidate() []byte {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.candidate
}

// SetCandidate updates the candidate configuration with merged XML content.
func (d *Datastore) SetCandidate(newConfig []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// For simplicity in simulator: just replace candidate
	// Full XPath merge would be more complex but for testing we keep it simple
	d.candidate = make([]byte, len(newConfig))
	copy(d.candidate, newConfig)
	return nil
}

// Commit copies candidate to running.
func (d *Datastore) Commit() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.running = make([]byte, len(d.candidate))
	copy(d.running, d.candidate)
	return nil
}

// DiscardCandidate resets candidate to match running.
func (d *Datastore) DiscardCandidate() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.candidate = make([]byte, len(d.running))
	copy(d.candidate, d.running)
}

// ExtractVLANs extracts VLANs from running configuration for testing assertions.
// Supports both legacy format and standard OpenConfig XML with namespaces.
func (d *Datastore) ExtractVLANs() (*openconfig.OpenconfigVlan_Vlans, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Normalize XML: remove namespaces, convert tag names to match Go struct fields
	// Supports both legacy format and standard OpenConfig XML format
	xmlStr := string(d.running)
	xmlStr = normalizeOpenConfigXML(xmlStr)

	// Since the OpenconfigVlan_Vlans struct contains a map field and xml.Unmarshal doesn't support maps,
	// we need to manually parse the VLAN entries from XML and construct the struct ourselves.
	vlans := &openconfig.OpenconfigVlan_Vlans{}
	vlans.Vlan = make(map[uint16]*openconfig.OpenconfigVlan_Vlans_Vlan)

	// Use xml decoder to walk the XML and manually collect each VLAN
	decoder := xml.NewDecoder(bytes.NewReader([]byte(xmlStr)))
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		// We're looking for <Vlan> (already converted from <vlan>)
		if start.Name.Local == "Vlan" {
			vlan := &openconfig.OpenconfigVlan_Vlans_Vlan{}
			// Manually parse inside Vlan because ygot structs don't have xml tags
			// and encoding/xml doesn't match <config> to Config field without xml tag
			vlan.Config = &openconfig.OpenconfigVlan_Vlans_Vlan_Config{}
			for {
				token, err := decoder.Token()
				if err != nil {
					break
				}

				// Check for closing </Vlan>
				if _, ok := token.(xml.EndElement); ok {
					break
				}

				innerStart, ok := token.(xml.StartElement)
				if !ok {
					continue
				}

				switch innerStart.Name.Local {
				case "VlanId":
					var vid uint16
					if err := decoder.DecodeElement(&vid, &innerStart); err == nil {
						vlan.VlanId = &vid
					}
				case "config":
					// Parse inside config manually - ygot doesn't have xml tags
					for {
						token, err := decoder.Token()
						if err != nil {
							break
						}
						if _, ok := token.(xml.EndElement); ok {
							break
						}
						configStart, ok := token.(xml.StartElement)
						if !ok {
							continue
						}
						switch configStart.Name.Local {
						case "Name":
							var name string
							if err := decoder.DecodeElement(&name, &configStart); err == nil {
								vlan.Config.Name = &name
							}
						case "Status":
							// Status is an enum, but we don't need to parse it for assertion
							_ = decoder.Skip()
						case "VlanId":
							var vid uint16
							if err := decoder.DecodeElement(&vid, &configStart); err == nil {
								vlan.Config.VlanId = &vid
							}
						default:
							_ = decoder.Skip()
						}
					}
				default:
					_ = decoder.Skip()
				}
			}

			// Get VLAN ID: check top-level VlanId first, then check Config.VlanId
			var vlanID *uint16
			if vlan.VlanId != nil {
				vlanID = vlan.VlanId
			} else if vlan.Config != nil && vlan.Config.VlanId != nil {
				vlanID = vlan.Config.VlanId
			}
			if vlanID != nil {
				vlans.Vlan[*vlanID] = vlan
			}
		}
	}

	// Always return the vlans struct (even if empty) so assertions can work with it
	// Fallback only if we didn't find any VLANs through manual parsing and vlans is empty
	if len(vlans.Vlan) > 0 || len(vlans.Vlan) == 0 {
		return vlans, nil
	}

	// Fallback to trying standard xml.Unmarshal in case structure is different
	var direct struct {
		VLans *openconfig.OpenconfigVlan_Vlans `xml:"vlans"`
	}
	err := xml.Unmarshal([]byte(xmlStr), &direct)
	if err == nil && direct.VLans != nil {
		return direct.VLans, nil
	}

	// Structure 2: <config><vlans> ... </vlans></config> (this is what we get from edit-config)
	var configDirect struct {
		Config struct {
			VLans *openconfig.OpenconfigVlan_Vlans `xml:"vlans"`
		} `xml:"config"`
	}
	err = xml.Unmarshal([]byte(xmlStr), &configDirect)
	if err == nil && configDirect.Config.VLans != nil {
		return configDirect.Config.VLans, nil
	}

	// Structure 3: <config><device><vlans> ... </vlans></device></config> (initial set from test)
	var configDevice struct {
		Config struct {
			Device struct {
				VLans *openconfig.OpenconfigVlan_Vlans `xml:"vlans"`
			} `xml:"device"`
		} `xml:"config"`
	}
	err = xml.Unmarshal([]byte(xmlStr), &configDevice)
	if err == nil && configDevice.Config.Device.VLans != nil {
		return configDevice.Config.Device.VLans, nil
	}

	// Structure 4: <device><vlans> ... </vlans></device> at top level
	var deviceOnly struct {
		Device struct {
			VLans *openconfig.OpenconfigVlan_Vlans `xml:"vlans"`
		} `xml:"device"`
	}
	err = xml.Unmarshal([]byte(xmlStr), &deviceOnly)
	if err == nil && deviceOnly.Device.VLans != nil {
		return deviceOnly.Device.VLans, nil
	}

	return nil, fmt.Errorf("failed to extract VLANs from XML after trying all structures")
}

// ExtractInterfaces extracts Interfaces from running configuration for testing assertions.
// Supports both legacy format and standard OpenConfig XML with namespaces.
func (d *Datastore) ExtractInterfaces() (*openconfig.OpenconfigInterfaces_Interfaces, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Normalize XML: remove namespaces, convert tag names to match Go struct fields
	// Supports both legacy format and standard OpenConfig XML format
	xmlStr := string(d.running)
	xmlStr = normalizeOpenConfigXML(xmlStr)

	interfaces := &openconfig.OpenconfigInterfaces_Interfaces{}
	interfaces.Interface = make(map[string]*openconfig.OpenconfigInterfaces_Interfaces_Interface)

	// Use xml decoder to manually parse Interface entries (ygot structs don't have xml tags)
	decoder := xml.NewDecoder(bytes.NewReader([]byte(xmlStr)))
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		// Look for <Interface> or <OpenconfigInterfaces_Interfaces_Interface>
		if start.Name.Local == "Interface" || start.Name.Local == "OpenconfigInterfaces_Interfaces_Interface" {
			iface := &openconfig.OpenconfigInterfaces_Interfaces_Interface{}
			iface.Config = &openconfig.OpenconfigInterfaces_Interfaces_Interface_Config{}

			for {
				token, err := decoder.Token()
				if err != nil {
					break
				}

				// Check for closing tag
				if _, ok := token.(xml.EndElement); ok {
					break
				}

				innerStart, ok := token.(xml.StartElement)
				if !ok {
					continue
				}

				switch innerStart.Name.Local {
				case "Name":
					var name string
					if err := decoder.DecodeElement(&name, &innerStart); err == nil {
						iface.Name = &name
						iface.Config.Name = &name
					}
				case "Type":
					// Type parsing skipped for assertions - not needed for basic existence checks
					_ = decoder.Skip()
				case "Enabled":
					var enabled bool
					if err := decoder.DecodeElement(&enabled, &innerStart); err == nil {
						iface.Config.Enabled = &enabled
					}
				case "Mtu":
					var mtu uint16
					if err := decoder.DecodeElement(&mtu, &innerStart); err == nil {
						iface.Config.Mtu = &mtu
					}
				case "Description":
					var desc string
					if err := decoder.DecodeElement(&desc, &innerStart); err == nil {
						iface.Config.Description = &desc
					}
				case "config":
					// Parse inside config manually
					for {
						token, err := decoder.Token()
						if err != nil {
							break
						}
						if _, ok := token.(xml.EndElement); ok {
							break
						}
						configStart, ok := token.(xml.StartElement)
						if !ok {
							continue
						}
						switch configStart.Name.Local {
						case "Name":
							var name string
							if err := decoder.DecodeElement(&name, &configStart); err == nil {
								iface.Config.Name = &name
							}
						case "Type":
							// Type parsing skipped
							_ = decoder.Skip()
						case "Enabled":
							var enabled bool
							if err := decoder.DecodeElement(&enabled, &configStart); err == nil {
								iface.Config.Enabled = &enabled
							}
						case "Mtu":
							var mtu uint16
							if err := decoder.DecodeElement(&mtu, &configStart); err == nil {
								iface.Config.Mtu = &mtu
							}
						case "Description":
							var desc string
							if err := decoder.DecodeElement(&desc, &configStart); err == nil {
								iface.Config.Description = &desc
							}
						default:
							_ = decoder.Skip()
						}
					}
				default:
					_ = decoder.Skip()
				}
			}

			// Add to map if we found a name
			if iface.Name != nil {
				interfaces.Interface[*iface.Name] = iface
			} else if iface.Config != nil && iface.Config.Name != nil {
				interfaces.Interface[*iface.Config.Name] = iface
			}
		}
	}

	// Always return interfaces struct
	return interfaces, nil
}

// ExtractHuaweiVLANs extracts Huawei model VLANs from running configuration for testing assertions.
func (d *Datastore) ExtractHuaweiVLANs() (map[uint16]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	xmlStr := string(d.running)
	vlans := make(map[uint16]string)

	decoder := xml.NewDecoder(bytes.NewReader([]byte(xmlStr)))
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		// We're looking for <HuaweiVlan_Vlan_Vlans_Vlan> which is the Go XML serialization format
		if strings.Contains(start.Name.Local, "HuaweiVlan") && strings.Contains(start.Name.Local, "Vlan") {
			var vlanID uint16
			var name string

			for {
				token, err := decoder.Token()
				if err != nil {
					break
				}

				if _, ok := token.(xml.EndElement); ok {
					break
				}

				innerStart, ok := token.(xml.StartElement)
				if !ok {
					continue
				}

				switch innerStart.Name.Local {
				case "Id", "id", "VlanId":
					if err := decoder.DecodeElement(&vlanID, &innerStart); err == nil {
					}
				case "Name", "name":
					if err := decoder.DecodeElement(&name, &innerStart); err == nil {
					}
				default:
					_ = decoder.Skip()
				}
			}

			if vlanID > 0 {
				vlans[vlanID] = name
			}
		}
	}

	return vlans, nil
}

// cleanNamespaces removes XML namespace declarations and prefixes from tags.
// Example: <if:interface xmlns:if="http://openconfig.net/yang/interfaces"> → <interface>
func cleanNamespaces(xmlStr string) string {
	// Remove xmlns attributes
	re1 := regexp.MustCompile(`\s+xmlns(:[a-zA-Z0-9_-]+)?="[^"]*"`)
	xmlStr = re1.ReplaceAllString(xmlStr, "")

	// Remove tag prefixes (e.g., <if:interface> → <interface>)
	re2 := regexp.MustCompile(`<(/?)([a-zA-Z0-9_-]+):([a-zA-Z0-9_-]+)`)
	xmlStr = re2.ReplaceAllString(xmlStr, "<$1$3")

	return xmlStr
}

// normalizeOpenConfigXML transforms standard OpenConfig XML into a format
// that can be parsed by Go's xml.Unmarshal using ygot-generated structs.
// Steps:
//  1. Remove namespace declarations and prefixes
//  2. Convert lowercase YANG tag names to PascalCase to match Go struct fields
//  3. Convert kebab-case to PascalCase (e.g., vlan-id → VlanId)
func normalizeOpenConfigXML(xmlStr string) string {
	// Step 1: Clean namespaces
	xmlStr = cleanNamespaces(xmlStr)

	// Step 2: Convert tag names from YANG conventions to Go struct conventions
	repl := strings.NewReplacer(
		// VLAN related
		"<vlans>", "<Vlans>",
		"</vlans>", "</Vlans>",
		"<vlan>", "<Vlan>",
		"</vlan>", "</Vlan>",
		"<vlan-id>", "<VlanId>",
		"</vlan-id>", "</VlanId>",
		"<name>", "<Name>",
		"</name>", "</Name>",
		"<status>", "<Status>",
		"</status>", "</Status>",
		// Interface related
		"<interfaces>", "<Interfaces>",
		"</interfaces>", "</Interfaces>",
		"<interface>", "<Interface>",
		"</interface>", "</Interface>",
		"<type>", "<Type>",
		"</type>", "</Type>",
		"<enabled>", "<Enabled>",
		"</enabled>", "</Enabled>",
		"<mtu>", "<Mtu>",
		"</mtu>", "</Mtu>",
		"<description>", "<Description>",
		"</description>", "</Description>",
	)
	return repl.Replace(xmlStr)
}

// fixXMLTagNames is DEPRECATED - use normalizeOpenConfigXML instead
// Kept for backward compatibility during transition
func fixXMLTagNames(xml string) string {
	return normalizeOpenConfigXML(xml)
}

// GetXML returns the XML for the requested datastore.
func (d *Datastore) GetXML(source string) []byte {
	switch source {
	case "running":
		return d.GetRunning()
	case "candidate":
		return d.GetCandidate()
	default:
		return d.GetRunning()
	}
}

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
}

// ExtractHuaweiVLANsFull extracts complete Huawei model VLAN data including all fields.
func (d *Datastore) ExtractHuaweiVLANsFull() (map[uint16]*HuaweiVlanTestData, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	xmlStr := string(d.running)
	vlans := make(map[uint16]*HuaweiVlanTestData)

	decoder := xml.NewDecoder(bytes.NewReader([]byte(xmlStr)))
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		// We're looking for <HuaweiVlan_Vlan_Vlans_Vlan> which is the Go XML serialization format
		if strings.Contains(start.Name.Local, "HuaweiVlan") && strings.Contains(start.Name.Local, "Vlan") {
			vlan := &HuaweiVlanTestData{}

			for {
				token, err := decoder.Token()
				if err != nil {
					break
				}

				if _, ok := token.(xml.EndElement); ok {
					break
				}

				innerStart, ok := token.(xml.StartElement)
				if !ok {
					continue
				}

				switch innerStart.Name.Local {
				case "Id", "id", "VlanId":
					if err := decoder.DecodeElement(&vlan.ID, &innerStart); err == nil {
					}
				case "Name", "name":
					if err := decoder.DecodeElement(&vlan.Name, &innerStart); err == nil {
					}
				case "Description", "description":
					if err := decoder.DecodeElement(&vlan.Description, &innerStart); err == nil {
					}
				case "Type", "type":
					if err := decoder.DecodeElement(&vlan.Type, &innerStart); err == nil {
					}
				case "AdminStatus", "admin-status":
					if err := decoder.DecodeElement(&vlan.AdminStatus, &innerStart); err == nil {
					}
				case "BroadcastDiscard", "broadcast-discard":
					if err := decoder.DecodeElement(&vlan.BroadcastDiscard, &innerStart); err == nil {
					}
				case "UnknownMulticastDiscard", "unknown-multicast-discard":
					if err := decoder.DecodeElement(&vlan.UnknownMulticastDiscard, &innerStart); err == nil {
					}
				case "MacLearning", "mac-learning":
					if err := decoder.DecodeElement(&vlan.MacLearning, &innerStart); err == nil {
					}
				case "MacAgingTime", "mac-aging-time":
					if err := decoder.DecodeElement(&vlan.MacAgingTime, &innerStart); err == nil {
					}
				case "StatisticEnable", "statistic-enable":
					if err := decoder.DecodeElement(&vlan.StatisticEnable, &innerStart); err == nil {
					}
				case "StatisticDiscard", "statistic-discard":
					if err := decoder.DecodeElement(&vlan.StatisticDiscard, &innerStart); err == nil {
					}
				case "SuperVlan", "super-vlan":
					var sv uint16
					if err := decoder.DecodeElement(&sv, &innerStart); err == nil {
						vlan.SuperVlan = &sv
					}
				case "UnkownUnicastDiscard", "unknown-unicast-discard":
					// Parse nested container
					for {
						token, err := decoder.Token()
						if err != nil {
							break
						}
						if _, ok := token.(xml.EndElement); ok {
							break
						}
						nestedStart, ok := token.(xml.StartElement)
						if !ok {
							continue
						}
						switch nestedStart.Name.Local {
						case "Discard", "discard":
							if err := decoder.DecodeElement(&vlan.UnkownUnicastDiscard.Discard, &nestedStart); err == nil {
							}
						case "MacLearningEnable", "mac-learning-enable":
							if err := decoder.DecodeElement(&vlan.UnkownUnicastDiscard.MacLearningEnable, &nestedStart); err == nil {
							}
						default:
							_ = decoder.Skip()
						}
					}
				case "Suppression", "suppression":
					// Parse nested container
					for {
						token, err := decoder.Token()
						if err != nil {
							break
						}
						if _, ok := token.(xml.EndElement); ok {
							break
						}
						nestedStart, ok := token.(xml.StartElement)
						if !ok {
							continue
						}
						switch nestedStart.Name.Local {
						case "Inbound", "inbound":
							if err := decoder.DecodeElement(&vlan.Suppression.Inbound, &nestedStart); err == nil {
							}
						case "Outbound", "outbound":
							if err := decoder.DecodeElement(&vlan.Suppression.Outbound, &nestedStart); err == nil {
							}
						default:
							_ = decoder.Skip()
						}
					}
				default:
					_ = decoder.Skip()
				}
			}

			if vlan.ID > 0 {
				vlans[vlan.ID] = vlan
			}
		}
	}

	return vlans, nil
}

// HuaweiInterfaceTestData represents interface test data from Huawei IFM model.
type HuaweiInterfaceTestData struct {
	Name        string
	Description string
	Index       uint32
	Number      string
	Position    string
	ParentName  string
	AdminStatus int
	Type        int
	Class       int
	LinkProtocol int
	RouterType  int
	ServiceType int
	Mtu         uint32
	MacAddress  string
	Bandwidth   uint32
	BandwidthKbps uint32
	VrfName     string
	VsName      string
	AggregationName string
	DownDelayTime uint32
	ProtocolUpDelayTime uint32
	ClearIpDf   bool
	IsL2Switch  bool
	L2ModeEnable bool
	LinkUpDownTrapEnable bool
	StatisticEnable bool
	SpreadMtuFlag bool
	StatisticInterval uint32
	StatisticMode int
	// Nested containers
	ControlFlap struct {
		Ceiling         uint32
		ControlFlapCount uint32
		DecayNg         uint32
		DecayOk         uint32
		Reuse           uint32
		Suppress        uint32
	}
	Damp struct {
		TxOff bool
		Auto struct {
			Level int
		}
		Manual struct {
			HalfLifePeriod   uint16
			MaxSuppressTime  uint16
			Reuse            uint32
			Suppress         uint32
		}
	}
}

// ExtractHuaweiInterfaces extracts Huawei IFM interface data from running configuration.
func (d *Datastore) ExtractHuaweiInterfaces() (map[string]*HuaweiInterfaceTestData, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	xmlStr := string(d.running)
	interfaces := make(map[string]*HuaweiInterfaceTestData)

	decoder := xml.NewDecoder(bytes.NewReader([]byte(xmlStr)))
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		// Look for interface XML elements
		if strings.Contains(start.Name.Local, "Interface") || strings.Contains(start.Name.Local, "interface") {
			iface := &HuaweiInterfaceTestData{}

			for {
				token, err := decoder.Token()
				if err != nil {
					break
				}

				if _, ok := token.(xml.EndElement); ok {
					break
				}

				innerStart, ok := token.(xml.StartElement)
				if !ok {
					continue
				}

				switch innerStart.Name.Local {
				case "Name", "name":
					if err := decoder.DecodeElement(&iface.Name, &innerStart); err == nil {
					}
				case "Description", "description":
					if err := decoder.DecodeElement(&iface.Description, &innerStart); err == nil {
					}
				case "Index", "index":
					if err := decoder.DecodeElement(&iface.Index, &innerStart); err == nil {
					}
				case "Number", "number":
					if err := decoder.DecodeElement(&iface.Number, &innerStart); err == nil {
					}
				case "Position", "position":
					if err := decoder.DecodeElement(&iface.Position, &innerStart); err == nil {
					}
				case "ParentName", "parent-name":
					if err := decoder.DecodeElement(&iface.ParentName, &innerStart); err == nil {
					}
				case "AdminStatus", "admin-status":
					if err := decoder.DecodeElement(&iface.AdminStatus, &innerStart); err == nil {
					}
				case "Type", "type":
					if err := decoder.DecodeElement(&iface.Type, &innerStart); err == nil {
					}
				case "Class", "class":
					if err := decoder.DecodeElement(&iface.Class, &innerStart); err == nil {
					}
				case "LinkProtocol", "link-protocol":
					if err := decoder.DecodeElement(&iface.LinkProtocol, &innerStart); err == nil {
					}
				case "RouterType", "router-type":
					if err := decoder.DecodeElement(&iface.RouterType, &innerStart); err == nil {
					}
				case "ServiceType", "service-type":
					if err := decoder.DecodeElement(&iface.ServiceType, &innerStart); err == nil {
					}
				case "Mtu", "mtu":
					if err := decoder.DecodeElement(&iface.Mtu, &innerStart); err == nil {
					}
				case "MacAddress", "mac-address":
					if err := decoder.DecodeElement(&iface.MacAddress, &innerStart); err == nil {
					}
				case "Bandwidth", "bandwidth":
					if err := decoder.DecodeElement(&iface.Bandwidth, &innerStart); err == nil {
					}
				case "BandwidthKbps", "bandwidth-kbps":
					if err := decoder.DecodeElement(&iface.BandwidthKbps, &innerStart); err == nil {
					}
				case "VrfName", "vrf-name":
					if err := decoder.DecodeElement(&iface.VrfName, &innerStart); err == nil {
					}
				case "VsName", "vs-name":
					if err := decoder.DecodeElement(&iface.VsName, &innerStart); err == nil {
					}
				case "AggregationName", "aggregation-name":
					if err := decoder.DecodeElement(&iface.AggregationName, &innerStart); err == nil {
					}
				case "DownDelayTime", "down-delay-time":
					if err := decoder.DecodeElement(&iface.DownDelayTime, &innerStart); err == nil {
					}
				case "ProtocolUpDelayTime", "protocol-up-delay-time":
					if err := decoder.DecodeElement(&iface.ProtocolUpDelayTime, &innerStart); err == nil {
					}
				case "ClearIpDf", "clear-ip-df":
					if err := decoder.DecodeElement(&iface.ClearIpDf, &innerStart); err == nil {
					}
				case "IsL2Switch", "is-l2-switch":
					if err := decoder.DecodeElement(&iface.IsL2Switch, &innerStart); err == nil {
					}
				case "L2ModeEnable", "l2-mode-enable":
					if err := decoder.DecodeElement(&iface.L2ModeEnable, &innerStart); err == nil {
					}
				case "LinkUpDownTrapEnable", "link-up-down-trap-enable":
					if err := decoder.DecodeElement(&iface.LinkUpDownTrapEnable, &innerStart); err == nil {
					}
				case "StatisticEnable", "statistic-enable":
					if err := decoder.DecodeElement(&iface.StatisticEnable, &innerStart); err == nil {
					}
				case "SpreadMtuFlag", "spread-mtu-flag":
					if err := decoder.DecodeElement(&iface.SpreadMtuFlag, &innerStart); err == nil {
					}
				case "StatisticInterval", "statistic-interval":
					if err := decoder.DecodeElement(&iface.StatisticInterval, &innerStart); err == nil {
					}
				case "StatisticMode", "statistic-mode":
					if err := decoder.DecodeElement(&iface.StatisticMode, &innerStart); err == nil {
					}
				case "ControlFlap", "control-flap":
					// Parse nested container
					for {
						token, err := decoder.Token()
						if err != nil {
							break
						}
						if _, ok := token.(xml.EndElement); ok {
							break
						}
						nestedStart, ok := token.(xml.StartElement)
						if !ok {
							continue
						}
						switch nestedStart.Name.Local {
						case "Ceiling", "ceiling":
							if err := decoder.DecodeElement(&iface.ControlFlap.Ceiling, &nestedStart); err == nil {
							}
						case "ControlFlapCount", "control-flap-count":
							if err := decoder.DecodeElement(&iface.ControlFlap.ControlFlapCount, &nestedStart); err == nil {
							}
						case "DecayNg", "decay-ng":
							if err := decoder.DecodeElement(&iface.ControlFlap.DecayNg, &nestedStart); err == nil {
							}
						case "DecayOk", "decay-ok":
							if err := decoder.DecodeElement(&iface.ControlFlap.DecayOk, &nestedStart); err == nil {
							}
						case "Reuse", "reuse":
							if err := decoder.DecodeElement(&iface.ControlFlap.Reuse, &nestedStart); err == nil {
							}
						case "Suppress", "suppress":
							if err := decoder.DecodeElement(&iface.ControlFlap.Suppress, &nestedStart); err == nil {
							}
						default:
							_ = decoder.Skip()
						}
					}
				case "Damp", "damp":
					// Parse nested Damp container
					for {
						token, err := decoder.Token()
						if err != nil {
							break
						}
						if _, ok := token.(xml.EndElement); ok {
							break
						}
						nestedStart, ok := token.(xml.StartElement)
						if !ok {
							continue
						}
						switch nestedStart.Name.Local {
						case "TxOff", "tx-off":
							if err := decoder.DecodeElement(&iface.Damp.TxOff, &nestedStart); err == nil {
							}
						case "Auto", "auto":
							// Parse Auto sub-container
							for {
								token, err := decoder.Token()
								if err != nil {
									break
								}
								if _, ok := token.(xml.EndElement); ok {
									break
								}
								autoStart, ok := token.(xml.StartElement)
								if !ok {
									continue
								}
								if autoStart.Name.Local == "Level" || autoStart.Name.Local == "level" {
									if err := decoder.DecodeElement(&iface.Damp.Auto.Level, &autoStart); err == nil {
									}
								} else {
									_ = decoder.Skip()
								}
							}
						case "Manual", "manual":
							// Parse Manual sub-container
							for {
								token, err := decoder.Token()
								if err != nil {
									break
								}
								if _, ok := token.(xml.EndElement); ok {
									break
								}
								manualStart, ok := token.(xml.StartElement)
								if !ok {
									continue
								}
								switch manualStart.Name.Local {
								case "HalfLifePeriod", "half-life-period":
									if err := decoder.DecodeElement(&iface.Damp.Manual.HalfLifePeriod, &manualStart); err == nil {
									}
								case "MaxSuppressTime", "max-suppress-time":
									if err := decoder.DecodeElement(&iface.Damp.Manual.MaxSuppressTime, &manualStart); err == nil {
									}
								case "Reuse", "reuse":
									if err := decoder.DecodeElement(&iface.Damp.Manual.Reuse, &manualStart); err == nil {
									}
								case "Suppress", "suppress":
									if err := decoder.DecodeElement(&iface.Damp.Manual.Suppress, &manualStart); err == nil {
									}
								default:
									_ = decoder.Skip()
								}
							}
						default:
							_ = decoder.Skip()
						}
					}
				default:
					_ = decoder.Skip()
				}
			}

			if iface.Name != "" {
				interfaces[iface.Name] = iface
			}
		}
	}

	return interfaces, nil
}

// HuaweiSystemTestData represents system configuration test data.
type HuaweiSystemTestData struct {
	SysName     string
	SysContact  string
	SysLocation string
}

// ExtractHuaweiSystem extracts Huawei system configuration from running config.
func (d *Datastore) ExtractHuaweiSystem() (*HuaweiSystemTestData, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	xmlStr := string(d.running)
	sys := &HuaweiSystemTestData{}

	decoder := xml.NewDecoder(strings.NewReader(xmlStr))
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		// Look for system-info elements
		switch start.Name.Local {
		case "SysName", "sys-name":
			if err := decoder.DecodeElement(&sys.SysName, &start); err == nil {
			}
		case "SysContact", "sys-contact":
			if err := decoder.DecodeElement(&sys.SysContact, &start); err == nil {
			}
		case "SysLocation", "sys-location":
			if err := decoder.DecodeElement(&sys.SysLocation, &start); err == nil {
			}
		default:
			// Check if this is a system-info container - recursively process children
			if strings.Contains(strings.ToLower(start.Name.Local), "system") {
				// Continue parsing - the inner elements will be processed
			}
		}
	}

	return sys, nil
}
