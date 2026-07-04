package netconfsim

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
	"sync"
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
}

// HuaweiSystemTestData represents system configuration test data.
type HuaweiSystemTestData struct {
	SysName     string
	SysContact  string
	SysLocation string
}
