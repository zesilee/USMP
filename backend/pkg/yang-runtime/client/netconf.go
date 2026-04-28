package client

import (
	"context"
	"encoding/xml"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/internal/generated/openconfig"
	"github.com/scrapli/scrapligo/driver/netconf"
	"github.com/scrapli/scrapligo/driver/options"
	"github.com/scrapli/scrapligo/response"
	"github.com/scrapli/scrapligo/transport"
	"github.com/scrapli/scrapligo/util"
)

// NETCONFDefaultPort is the default NETCONF port
const NETCONFDefaultPort = 830

// NETCONFClient implements Client interface for NETCONF protocol
type NETCONFClient struct {
	mu        sync.RWMutex
	info      DeviceConnectionInfo
	driver    *netconf.Driver
	connected bool
}

// NewNETCONFClient creates a new NETCONF client and connects immediately
func NewNETCONFClient(info DeviceConnectionInfo) (*NETCONFClient, error) {
	if info.Port == 0 {
		info.Port = NETCONFDefaultPort
	}
	if info.Timeout == 0 {
		info.Timeout = 10 * time.Second
	}

	c := &NETCONFClient{
		info: info,
	}

	// Connect immediately
	if err := c.connect(); err != nil {
		// Return the client with the error so caller can handle it
		return c, err
	}

	return c, nil
}

func (c *NETCONFClient) connect() error {
	opts := []util.Option{
		options.WithAuthUsername(c.info.Username),
		options.WithAuthPassword(c.info.Password),
		options.WithPort(c.info.Port),
		options.WithTimeoutSocket(c.info.Timeout),
		options.WithAuthNoStrictKey(),
		options.WithTransportType(transport.StandardTransport),
	}

	driver, err := netconf.NewDriver(
		c.info.IP,
		opts...,
	)
	if err != nil {
		return fmt.Errorf("failed to create NETCONF driver: %w", err)
	}

	err = driver.Open()
	if err != nil {
		return fmt.Errorf("failed to open NETCONF connection: %w", err)
	}

	c.driver = driver
	c.connected = true

	return nil
}

// Get implements Client interface
func (c *NETCONFClient) Get(ctx context.Context, path string, opts ...GetOption) (*GetResult, error) {
	c.mu.RLock()
	if !c.connected || c.driver == nil {
		c.mu.RUnlock()
		// Try to reconnect
		c.mu.Lock()
		defer c.mu.Unlock()
		if err := c.connect(); err != nil {
			return &GetResult{
				Error: err,
			}, err
		}
	} else {
		c.mu.RUnlock()
	}

	c.mu.RLock()
	driver := c.driver
	c.mu.RUnlock()

	// Apply options
	getOpts := &GetOptions{
		Datastore: "running",
	}
	for _, opt := range opts {
		opt.Apply(getOpts)
	}

	// Construct filter
	filter := c.constructFilter(path)
	// Create option that sets the filter on the operation
	withFilter := func(o interface{}) error {
		op, ok := o.(*netconf.OperationOptions)
		if !ok {
			return util.ErrIgnoredOption
		}
		op.Filter = filter
		return nil
	}
	resp, err := driver.GetConfig(getOpts.Datastore, withFilter)
	if err != nil {
		return &GetResult{
			Error: err,
		}, err
	}

	if resp == nil || len(resp.Result) == 0 {
		return &GetResult{
			Path:      path,
			Data:      nil,
			Timestamp: time.Now(),
			Error:     fmt.Errorf("empty response"),
		}, fmt.Errorf("empty response")
	}

	result := &GetResult{
		Path:      path,
		Data:      []byte(resp.Result),
		Timestamp: time.Now(),
		Error:     nil,
	}

	return result, nil
}

// Set implements Client interface
func (c *NETCONFClient) Set(ctx context.Context, changes []Change, opts ...SetOption) (*SetResult, error) {
	c.mu.RLock()
	if !c.connected || c.driver == nil {
		c.mu.RUnlock()
		// Try to reconnect
		c.mu.Lock()
		defer c.mu.Unlock()
		if err := c.connect(); err != nil {
			return nil, err
		}
	} else {
		c.mu.RUnlock()
	}

	c.mu.RLock()
	driver := c.driver
	c.mu.RUnlock()

	// Apply options
	setOpts := &SetOptions{
		Datastore: "candidate",
		Commit:    true,
	}
	for _, opt := range opts {
		opt.Apply(setOpts)
	}

	result := &SetResult{
		Success:   true,
		Timestamp: time.Now(),
		Changes:   make([]ChangeResult, len(changes)),
	}

	// Apply each change
	for i, change := range changes {
		// For NETCONF, we need to convert the change to XML
		xmlConfig, err := c.marshalChange(change)
		if err != nil {
			result.Changes[i] = ChangeResult{
				Change:  change,
				Success: false,
				Error:   err,
			}
			result.Success = false
			continue
		}

		var resp *response.NetconfResponse
		resp, err = driver.EditConfig(setOpts.Datastore, xmlConfig)
		if err != nil {
			result.Changes[i] = ChangeResult{
				Change:  change,
				Success: false,
				Error:   err,
			}
			result.Success = false
			continue
		}
		// Check for NETCONF level errors (<rpc-error> in response)
		if resp.Failed != nil {
			result.Changes[i] = ChangeResult{
				Change:  change,
				Success: false,
				Error:   resp.Failed,
			}
			result.Success = false
			continue
		}

		result.Changes[i] = ChangeResult{
			Change:  change,
			Success: true,
			Error:   nil,
		}
	}

	// Commit if requested and all changes succeeded
	if setOpts.Commit && result.Success {
		resp, err := c.driver.Commit()
		if err != nil {
			result.Success = false
			result.Message = fmt.Sprintf("partial success: failed to commit: %v", err)
			return result, err
		}
		// If response contains <rpc-error>, resp.Failed will be non-nil
		if resp.Failed != nil {
			result.Success = false
			result.Message = fmt.Sprintf("partial success: commit failed: %v", resp.Failed)
			return result, resp.Failed
		}
	}

	if !result.Success {
		// Print any errors for debugging
		for _, ch := range result.Changes {
			if !ch.Success && ch.Error != nil {
				fmt.Printf("Change failed: %v\n", ch.Error)
			}
		}
		// If any change failed, return an error to caller
		return result, fmt.Errorf("one or more changes failed to apply")
	}

	return result, nil
}

// Subscribe implements Client interface
func (c *NETCONFClient) Subscribe(ctx context.Context, path string, handler func(Notification)) error {
	// NETCONF doesn't have built-in subscription like gNMI
	// TODO: Implement NETCONF notification subscription
	return fmt.Errorf("subscription not implemented for NETCONF")
}

// Close implements Client interface
func (c *NETCONFClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.driver == nil {
		return nil
	}

	err := c.driver.Close()
	c.connected = false
	c.driver = nil
	return err
}

// IsConnected implements Client interface
func (c *NETCONFClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected && c.driver != nil
}

func (c *NETCONFClient) constructFilter(path string) string {
	// For simplicity, we use an XPath filter for the path
	// Convert /interfaces/interface[name='eth0'] to XPath notation
	return fmt.Sprintf(`<filter xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" select="%s"/>`, path)
}

func (c *NETCONFClient) marshalChange(change Change) (string, error) {
	if change.NewValue == nil {
		// Delete operation
		return fmt.Sprintf(`<delete operation="delete" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"/>`), nil
	}

	// If the value is already a byte slice/string, use it directly
	switch v := change.NewValue.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	}

	// Special case: *openconfig.OpenconfigVlan_Vlans - contains a map field that xml.Marshal can't handle
	// We handle it manually by extracting the map and iterating
	if vlans, ok := change.NewValue.(*openconfig.OpenconfigVlan_Vlans); ok && vlans != nil {
		var builder strings.Builder
		builder.WriteString("<vlans>")
		// Iterate through all VLAN entries in the map
		for _, vlan := range vlans.Vlan {
			if vlan == nil {
				continue
			}
			entryXML, err := xml.Marshal(vlan)
			if err != nil {
				return "", fmt.Errorf("failed to marshal VLAN entry: %w", err)
			}
			builder.Write(entryXML)
		}
		builder.WriteString("</vlans>")
		outputStr := builder.String()
		// Fix XML element naming: convert from Go camelCase to YANG kebab-case
		// We specifically match the full opening and closing tags to avoid accidentally replacing
		// substrings in element content (e.g. "NewName" → "Newname" when replacing "Name" → "name")
		repl := strings.NewReplacer(
			"<VlanId>", "<vlan-id>",
			"</VlanId>", "</vlan-id>",
			"OpenconfigVlan_Vlans_Vlan", "vlan",
			"<Vlan>", "<vlan>",
			"</Vlan>", "</vlan>",
			"<Name>", "<name>",
			"</Name>", "</name>",
			"<Status>", "<status>",
			"</Status>", "</status>",
			"<Config>", "<config>",
			"</Config>", "</config>",
			"<VLans>", "<vlans>",
			"</VLans>", "</vlans>",
		)
		outputStr = repl.Replace(outputStr)
		return outputStr, nil
	}

	// Special case: *openconfig.OpenconfigInterfaces_Interfaces - generates
	// OpenConfig standard XML with proper namespace and YANG-conforming element names
	if interfaces, ok := change.NewValue.(*openconfig.OpenconfigInterfaces_Interfaces); ok && interfaces != nil {
		return buildOpenConfigInterfacesXML(interfaces)
	}

	// Special case: *huawei.HuaweiIfm_Ifm_Interfaces - Huawei IFM model
	if ifaces, ok := change.NewValue.(*huawei.HuaweiIfm_Ifm_Interfaces); ok && ifaces != nil {
		return buildHuaweiIfmInterfacesXML(ifaces)
	}

	// Try xml.Marshal for other types
	output, err := xml.Marshal(change.NewValue)
	if err == nil {
		// Success, fix naming and return
		outputStr := string(output)
		repl := strings.NewReplacer(
			"<VlanId>", "<vlan-id>",
			"</VlanId>", "</vlan-id>",
			"<Vlan>", "<vlan>",
			"</Vlan>", "</vlan>",
			"<VLans>", "<vlans>",
			"</VLans>", "</vlans>",
			"<Name>", "<name>",
			"</Name>", "</name>",
			"<Status>", "<status>",
			"</Status>", "</status>",
			"<Config>", "<config>",
			"</Config>", "</config>",
		)
		outputStr = repl.Replace(outputStr)
		return outputStr, nil
	}

	// If xml.Marshal failed and it's a map, handle manually
	v := reflect.ValueOf(change.NewValue)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() == reflect.Map {
		// Special case: Huawei IFM interface config from JSON map
		if strings.Contains(change.Path, "ifm:ifm") && strings.Contains(change.Path, "interfaces") {
			ifaces, err := mapToHuaweiIfmInterfaces(change.NewValue)
			if err == nil {
				return buildHuaweiIfmInterfacesXML(ifaces)
			}
		}

		var builder strings.Builder

		// Determine container tag based on the path
		containerTag := "vlans"
		if strings.HasSuffix(change.Path, "vlans") {
			containerTag = "vlans"
		} else if strings.HasSuffix(change.Path, "vlan") {
			containerTag = "vlan"
		} else {
			containerTag = "list"
		}
		builder.WriteString(fmt.Sprintf("<%s>", containerTag))

		// Iterate through all map entries and marshal each value individually
		for _, key := range v.MapKeys() {
			entryVal := v.MapIndex(key)
			if entryVal.IsValid() && !entryVal.IsNil() {
				// Each entry is a pointer to a struct that can be marshaled
				entryXML, err2 := xml.Marshal(entryVal.Interface())
				if err2 != nil {
					return "", fmt.Errorf("failed to marshal map entry: %w", err2)
				}
				builder.Write(entryXML)
			}
		}

		builder.WriteString(fmt.Sprintf("</%s>", containerTag))
		outputStr := builder.String()

		// Fix XML element naming: convert from Go camelCase to YANG kebab-case
		repl := strings.NewReplacer(
			"<VlanId>", "<vlan-id>",
			"</VlanId>", "</vlan-id>",
			"OpenconfigVlan_Vlans_Vlan", "vlan",
			"<Vlan>", "<vlan>",
			"</Vlan>", "</vlan>",
			"<Name>", "<name>",
			"</Name>", "</name>",
			"<Status>", "<status>",
			"</Status>", "</status>",
			"<Config>", "<config>",
			"</Config>", "</config>",
		)
		outputStr = repl.Replace(outputStr)
		return outputStr, nil
	}

	// Still failed - return original error
	return "", fmt.Errorf("failed to marshal config to XML: %w", err)
}

// OpenConfig XML namespace constants
const (
	OpenConfigInterfacesNS = "http://openconfig.net/yang/interfaces"
	IanaIfTypeNS           = "urn:ietf:params:xml:ns:yang:iana-if-type"
)

// buildOpenConfigInterfacesXML generates OpenConfig-standard XML for interfaces.
func buildOpenConfigInterfacesXML(interfaces *openconfig.OpenconfigInterfaces_Interfaces) (string, error) {
	if interfaces == nil || len(interfaces.Interface) == 0 {
		return fmt.Sprintf(`<interfaces xmlns="%s"/>`, OpenConfigInterfacesNS), nil
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf(`<interfaces xmlns="%s">`, OpenConfigInterfacesNS))

	// Iterate through all interface entries
	for name, iface := range interfaces.Interface {
		if iface == nil {
			continue
		}

		builder.WriteString("<interface>")

		// Interface name - required, use map key as fallback
		if iface.Name != nil {
			builder.WriteString(fmt.Sprintf("<name>%s</name>", xmlEscape(*iface.Name)))
		} else {
			builder.WriteString(fmt.Sprintf("<name>%s</name>", xmlEscape(name)))
		}

		// Config container - standard YANG pattern
		if iface.Config != nil {
			builder.WriteString("<config>")

			if iface.Config.Name != nil {
				builder.WriteString(fmt.Sprintf("<name>%s</name>", xmlEscape(*iface.Config.Name)))
			}

			// config/type - convert enum integer to IANA standard type name
			switch iface.Config.Type {
			case 1: // ethernetCsmacd
				builder.WriteString(fmt.Sprintf(`<type xmlns:ianaift="%s">ianaift:ethernetCsmacd</type>`, IanaIfTypeNS))
			case 24: // softwareLoopback
				builder.WriteString(fmt.Sprintf(`<type xmlns:ianaift="%s">ianaift:softwareLoopback</type>`, IanaIfTypeNS))
			default:
				builder.WriteString(fmt.Sprintf(`<type xmlns:ianaift="%s">ianaift:ethernetCsmacd</type>`, IanaIfTypeNS))
			}

			if iface.Config.Mtu != nil {
				builder.WriteString(fmt.Sprintf("<mtu>%d</mtu>", *iface.Config.Mtu))
			}

			if iface.Config.Enabled != nil {
				builder.WriteString(fmt.Sprintf("<enabled>%t</enabled>", *iface.Config.Enabled))
			}

			if iface.Config.Description != nil {
				builder.WriteString(fmt.Sprintf("<description>%s</description>", xmlEscape(*iface.Config.Description)))
			}

			builder.WriteString("</config>")
		}

		builder.WriteString("</interface>")
	}

	builder.WriteString("</interfaces>")
	return builder.String(), nil
}

// buildHuaweiIfmInterfacesXML generates Huawei IFM standard XML for interfaces.
func buildHuaweiIfmInterfacesXML(ifaces *huawei.HuaweiIfm_Ifm_Interfaces) (string, error) {
	if ifaces == nil || len(ifaces.Interface) == 0 {
		return `<interfaces xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm"/>`, nil
	}

	var builder strings.Builder
	builder.WriteString(`<interfaces xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm">`)

	for name, iface := range ifaces.Interface {
		if iface == nil {
			continue
		}

		builder.WriteString("<interface>")

		// Interface name (required)
		if iface.Name != nil {
			builder.WriteString(fmt.Sprintf("<name>%s</name>", xmlEscape(*iface.Name)))
		} else {
			builder.WriteString(fmt.Sprintf("<name>%s</name>", xmlEscape(name)))
		}

		// Description
		if iface.Description != nil {
			builder.WriteString(fmt.Sprintf("<description>%s</description>", xmlEscape(*iface.Description)))
		}

		// AdminStatus (enum 1=down, 2=up)
		if iface.AdminStatus != 0 {
			builder.WriteString(fmt.Sprintf("<admin-status>%d</admin-status>", iface.AdminStatus))
		}

		// MTU
		if iface.Mtu != nil {
			builder.WriteString(fmt.Sprintf("<mtu>%d</mtu>", *iface.Mtu))
		}

		// Interface Type (enum value)
		if iface.Type != 0 {
			builder.WriteString(fmt.Sprintf("<type>%d</type>", iface.Type))
		}

		builder.WriteString("</interface>")
	}

	builder.WriteString("</interfaces>")
	return builder.String(), nil
}

// mapToHuaweiIfmInterfaces converts a map (from JSON unmarshal) to Huawei IFM struct
func mapToHuaweiIfmInterfaces(data interface{}) (*huawei.HuaweiIfm_Ifm_Interfaces, error) {
	result := &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: make(map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface),
	}

	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected map[string]interface{}, got %T", data)
	}

	// The "interface" key contains the list of interfaces
	ifacesData, ok := dataMap["interface"]
	if !ok {
		// Try kebab-case "interface" (from frontend schema)
		ifacesData, ok = dataMap["interface"]
		if !ok {
			// Try with "Interface" (PascalCase from marshaled struct)
			ifacesData, ok = dataMap["Interface"]
			if !ok {
				// If no interface field, assume this IS the interface config itself
				// Create a dummy entry with the data
				result.Interface["default"] = mapEntryToHuaweiInterface(dataMap)
				return result, nil
			}
		}
	}

	// ifacesData could be a slice or a map
	v := reflect.ValueOf(ifacesData)
	switch v.Kind() {
	case reflect.Map:
		// Map case: key is interface name, value is interface config
		for _, key := range v.MapKeys() {
			entryVal := v.MapIndex(key)
			if entryMap, ok := entryVal.Interface().(map[string]interface{}); ok {
				ifaceName := key.String()
				result.Interface[ifaceName] = mapEntryToHuaweiInterface(entryMap)
			}
		}
	case reflect.Slice:
		// Slice case: array of interface configs
		for i := 0; i < v.Len(); i++ {
			entryVal := v.Index(i)
			if entryMap, ok := entryVal.Interface().(map[string]interface{}); ok {
				iface := mapEntryToHuaweiInterface(entryMap)
				if iface.Name != nil {
					result.Interface[*iface.Name] = iface
				} else {
					result.Interface[fmt.Sprintf("iface-%d", i)] = iface
				}
			}
		}
	}

	return result, nil
}

// mapEntryToHuaweiInterface converts a single interface config map entry to Huawei struct
func mapEntryToHuaweiInterface(m map[string]interface{}) *huawei.HuaweiIfm_Ifm_Interfaces_Interface {
	result := &huawei.HuaweiIfm_Ifm_Interfaces_Interface{}

	for k, v := range m {
		// Normalize key to handle kebab-case, camelCase, PascalCase variations
		key := strings.ToLower(strings.ReplaceAll(k, "-", ""))

		switch key {
		case "name":
			if s, ok := v.(string); ok {
				result.Name = &s
			}
		case "description":
			if s, ok := v.(string); ok {
				result.Description = &s
			}
		case "adminstatus":
			if num, ok := valueToUint(v); ok {
				result.AdminStatus = huawei.E_HuaweiIfm_PortStatus(num)
			}
		case "mtu":
			if num, ok := valueToUint(v); ok {
				uint32Val := uint32(num)
				result.Mtu = &uint32Val
			}
		case "type":
			if num, ok := valueToUint(v); ok {
				result.Type = huawei.E_HuaweiIfm_PortType(num)
			}
		}
	}

	return result
}

// valueToUint converts various numeric types to uint64
func valueToUint(v interface{}) (uint64, bool) {
	switch val := v.(type) {
	case float64:
		return uint64(val), true
	case int:
		return uint64(val), true
	case int64:
		return uint64(val), true
	case uint:
		return uint64(val), true
	case uint32:
		return uint64(val), true
	case uint64:
		return val, true
	case string:
		if num, err := strconv.ParseUint(val, 10, 64); err == nil {
			return num, true
		}
	}
	return 0, false
}

// xmlEscape escapes XML special characters in a string
func xmlEscape(s string) string {
	var buf strings.Builder
	for _, r := range s {
		switch r {
		case '<':
			buf.WriteString("&lt;")
		case '>':
			buf.WriteString("&gt;")
		case '&':
			buf.WriteString("&amp;")
		case '"':
			buf.WriteString("&quot;")
		case '\'':
			buf.WriteString("&apos;")
		default:
			buf.WriteRune(r)
		}
	}
	return buf.String()
}
