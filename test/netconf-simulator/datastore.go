package netconfsim

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
	"sync"

	"github.com/leezesi/usmp/internal/generated/openconfig"
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

// SetRunningFromDevice sets the running configuration from an openconfig Device.
// This also updates candidate to match.
func (d *Datastore) SetRunningFromDevice(dev *openconfig.Device) {
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
func (d *Datastore) ExtractVLANs() (*openconfig.OpenconfigVlan_Vlans, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Client converts names from camelCase to kebab-case when sending:
	// VlanId -> vlan-id, Vlan -> vlan, Name -> name, Config -> config etc...
	// We need to convert back to camelCase for xml.Unmarshal to match Go struct fields
	xmlStr := string(d.running)
	xmlStr = fixXMLTagNames(xmlStr)

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

// fixXMLTagNames converts kebab-case tag names back to camel-case that Go xml.Unmarshal expects
// based on the struct field names from ygot generated code.
// - vlan-id (kebab from client) → VlanId (struct field name)
// - vlan → Vlan (struct field name)
// - name → Name (struct field name)
// - status → Status (struct field name)
// - vlans → vlans (struct field VLans has XML tag xml:"vlans", so keep it as vlans)
// - config → config (struct field Config has XML tag xml:"config", so keep it as config)
func fixXMLTagNames(xml string) string {
	repl := strings.NewReplacer(
		"<vlan-id>", "<VlanId>",
		"</vlan-id>", "</VlanId>",
		"<vlan>", "<Vlan>",
		"</vlan>", "</Vlan>",
		"<name>", "<Name>",
		"</name>", "</Name>",
		"<status>", "<Status>",
		"</status>", "</Status>",
	)
	return repl.Replace(xml)
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
