package netsim

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"sync"

	"github.com/leezesi/usmp/internal/generated/openconfig"
)

// Device is the root OpenConfig device structure
type Device = openconfig.Device

// Datastore manages running and candidate configuration stores
type Datastore struct {
	running   *Device
	candidate *Device
	mu        sync.RWMutex
}

// NewDatastore creates a new empty datastore
func NewDatastore() *Datastore {
	return &Datastore{
		running:   &Device{},
		candidate: &Device{},
	}
}

// GetRunning returns the current running configuration
func (d *Datastore) GetRunning() *Device {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.running
}

// GetCandidate returns the current candidate configuration
func (d *Datastore) GetCandidate() *Device {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.candidate
}

// SetRunning sets the running configuration
func (d *Datastore) SetRunning(dev *Device) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.running = dev
}

// SetCandidate sets the candidate configuration
func (d *Datastore) SetCandidate(dev *Device) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.candidate = dev
}

// Commit copies candidate configuration to running configuration
func (d *Datastore) Commit() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Create a new device and copy populated fields
	d.running = &Device{}
	if d.candidate.Vlans != nil {
		d.running.Vlans = d.candidate.Vlans
	}
	// Add more fields as more modules are added

	return nil
}

// RenderConfigXML renders the configuration to XML
func (d *Datastore) RenderConfigXML(device *Device, filter []byte) ([]byte, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var buf bytes.Buffer

	// Start config
	buf.WriteString("<config xmlns=\"http://tail-f.com/ns/netconf\">\n")

	// Render VLANs if present
	if device.Vlans != nil {
		buf.WriteString("  <vlans xmlns=\"http://openconfig.net/yang/vlan\">\n")
		for _, vlan := range device.Vlans.Vlan {
			buf.WriteString("    <vlan>\n")
			if vlan.VlanId != nil {
				fmt.Fprintf(&buf, "      <vlan-id>%d</vlan-id>\n", *vlan.VlanId)
			}
			if vlan.Config != nil {
				buf.WriteString("      <config>\n")
				if vlan.Config.Name != nil {
					fmt.Fprintf(&buf, "        <name>%s</name>\n", *vlan.Config.Name)
				}
				if vlan.Config.VlanId != nil {
					fmt.Fprintf(&buf, "        <vlan-id>%d</vlan-id>\n", *vlan.Config.VlanId)
				}
				buf.WriteString("      </config>\n")
			}
			buf.WriteString("    </vlan>\n")
		}
		buf.WriteString("  </vlans>\n")
	}

	// End config
	buf.WriteString("</config>\n")

	return buf.Bytes(), nil
}

// VLANConfigXML represents parsed VLAN configuration from XML
type vlanXML struct {
	XMLName xml.Name `xml:"vlans"`
	Vlan    []struct {
		VlanId  string  `xml:"vlan-id"`
		Config  *struct {
			Name    *string `xml:"name"`
			VlanId  *string `xml:"vlan-id"`
			Status  *string `xml:"status"`
		} `xml:"config"`
	} `xml:"vlan"`
}

// ParseConfigXML parses configuration XML and merges it into the candidate device
func (d *Datastore) ParseConfigXML(configXML []byte, device *Device) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Parse VLANs if present in the config
	var vlansXML vlanXML
	err := xml.Unmarshal(configXML, &vlansXML)
	if err == nil {
		// Successfully parsed directly
	} else {
		// Try to parse if wrapped in another <config>
		// Need to parse from the original bytes since Unmarshal consumes the reader
		var wrapped struct {
			Vlans vlanXML `xml:"vlans"`
		}
		if err2 := xml.Unmarshal(configXML, &wrapped); err2 == nil {
			vlansXML = wrapped.Vlans
			err = nil
		} else {
			var wrappedConfig struct {
				Config struct {
					Vlans vlanXML `xml:"vlans"`
				} `xml:"config"`
			}
			if err3 := xml.Unmarshal(configXML, &wrappedConfig); err3 == nil {
				vlansXML = wrappedConfig.Config.Vlans
				err = nil
			} else {
				return fmt.Errorf("failed to parse config XML: %w", err)
			}
		}
	}

	// Initialize Vlans if needed
	if device.Vlans == nil {
		device.Vlans = &openconfig.OpenconfigVlan_Vlans{}
		device.Vlans.Vlan = make(map[uint16]*openconfig.OpenconfigVlan_Vlans_Vlan)
	}

	// Merge each VLAN
	for _, vlanXML := range vlansXML.Vlan {
		// Parse VLAN ID
		var id uint16
		var idStr string
		if vlanXML.VlanId != "" {
			idStr = vlanXML.VlanId
		} else if vlanXML.Config != nil && vlanXML.Config.VlanId != nil {
			idStr = *vlanXML.Config.VlanId
		} else {
			continue
		}

		parsedID, err := strconv.Atoi(idStr)
		if err != nil {
			continue
		}
		id = uint16(parsedID)

		// Get or create VLAN
		vlan, exists := device.Vlans.Vlan[id]
		if !exists {
			vlan, _ = device.Vlans.NewVlan(id)
		}

		// Ensure config exists
		if vlan.Config == nil {
			vlan.Config = &openconfig.OpenconfigVlan_Vlans_Vlan_Config{}
		}

		// Merge config
		if vlanXML.Config != nil {
			if vlanXML.Config.Name != nil {
				name := *vlanXML.Config.Name
				vlan.Config.Name = &name
			}
			if vlanXML.Config.VlanId != nil {
				parsedConfigID, err := strconv.Atoi(*vlanXML.Config.VlanId)
				if err == nil {
					configID := uint16(parsedConfigID)
					vlan.Config.VlanId = &configID
				}
			}
		}

		// Set VlanId on the VLAN itself
		vlan.VlanId = &id
	}

	return nil
}
