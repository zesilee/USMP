package client

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
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
	// opMu 串行化整段写事务（edit-config…commit/discard）：scrapligo driver 单通道
	// 不承受并发 RPC，且两个并发 Set 交错会把彼此的变更混进同一 candidate（2PC 原子性
	// 破坏，R09）。读走 get-config 单 RPC，同样经 opMu 防通道交错。
	opMu sync.Mutex
}

// NewNETCONFClient creates a new NETCONF client and connects immediately
func NewNETCONFClient(info DeviceConnectionInfo) (*NETCONFClient, error) {
	if info.Port == 0 {
		info.Port = NETCONFDefaultPort
	}
	if info.Timeout == 0 {
		info.Timeout = 10 * time.Second
	}
	// Credentials come from the shared DeviceStore (resolved by callers). No
	// admin/admin fallback here: an unregistered device connects with empty
	// credentials and SSH fails cleanly, rather than silently masking a missing
	// registration.

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
	c.opMu.Lock()
	defer c.opMu.Unlock()
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
	c.opMu.Lock()
	defer c.opMu.Unlock()
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

// ServerCapabilities returns the NETCONF capabilities the device advertised in
// its hello, or nil if not connected. Used by the hybrid schema resolver to
// narrow the usable YANG module set per device.
func (c *NETCONFClient) ServerCapabilities() []string {
	c.mu.RLock()
	driver := c.driver
	c.mu.RUnlock()
	if driver == nil {
		return nil
	}
	return driver.ServerCapabilities()
}

// DiscardCandidate discards the candidate configuration on the device.
// This is used to abort a 2PC transaction before commit.
func (c *NETCONFClient) DiscardCandidate(ctx context.Context) error {
	c.mu.RLock()
	if !c.connected || c.driver == nil {
		c.mu.RUnlock()
		c.mu.Lock()
		defer c.mu.Unlock()
		if err := c.connect(); err != nil {
			return err
		}
	} else {
		c.mu.RUnlock()
	}

	c.mu.RLock()
	driver := c.driver
	c.mu.RUnlock()

	// scrapligo's Discard method discards the candidate config
	resp, err := driver.Discard()
	if err != nil {
		return fmt.Errorf("failed to discard candidate: %w", err)
	}

	if resp.Failed != nil {
		return fmt.Errorf("discard candidate failed: %w", resp.Failed)
	}

	return nil
}

func (c *NETCONFClient) constructFilter(path string) string {
	// For simplicity, we use an XPath filter for the path
	// Convert /interfaces/interface[name='eth0'] to XPath notation
	return fmt.Sprintf(`<filter xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" select="%s"/>`, path)
}

func (c *NETCONFClient) marshalChange(change Change) (string, error) {
	if change.Type == DeleteChange {
		return marshalDeleteChange(change.OldValue)
	}
	if change.NewValue == nil {
		// 非删除变更缺 NewValue 无从编码——明确报错优于发送无目标的裸元素（R08）。
		return "", fmt.Errorf("marshal change: nil NewValue for %s change at %s", change.Type, change.Path)
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

	// Special case: *huawei.HuaweiVlan_Vlan_Vlans - Huawei VLAN model
	if vlans, ok := change.NewValue.(*huawei.HuaweiVlan_Vlan_Vlans); ok && vlans != nil {
		return buildHuaweiVlanVlansXML(vlans)
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
		// Special case: Huawei VLAN entries map — the generic per-entry xml.Marshal
		// chokes on the nested member-port map (xml 不支持 map). Route through the
		// dedicated builder which serializes member-ports correctly.
		if vlanMap, ok := change.NewValue.(map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan); ok {
			return buildHuaweiVlanVlansXML(&huawei.HuaweiVlan_Vlan_Vlans{Vlan: vlanMap})
		}

		// Special case: Huawei IFM interfaces map — the diff engine emits the interfaces/interface
		// list (ygot renders it as map[string]*..._Interface) as a single change whose NewValue is
		// this typed inner map and whose Path is the Go field name "Interface". The path-based
		// detection below (change.Path contains "ifm:ifm") never matches that, so without this
		// dedicated type assertion IFM falls through to the malformed generic <list> builder and the
		// interface is never actually pushed to the device（表现为「新建接口后配置里看不到」）。
		// 镜像上面 VLAN 的处理，路由到专用 builder。
		if ifaceMap, ok := change.NewValue.(map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface); ok {
			return buildHuaweiIfmInterfacesXML(&huawei.HuaweiIfm_Ifm_Interfaces{Interface: ifaceMap})
		}

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
		return fmt.Sprintf(`<interfaces xmlns="%s"/>`, HuaweiIfmNS), nil
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf(`<interfaces xmlns="%s">`, HuaweiIfmNS))

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

		// Bandwidth
		if iface.Bandwidth != nil {
			builder.WriteString(fmt.Sprintf("<bandwidth>%d</bandwidth>", *iface.Bandwidth))
		}

		// BandwidthKbps
		if iface.BandwidthKbps != nil {
			builder.WriteString(fmt.Sprintf("<bandwidth-kbps>%d</bandwidth-kbps>", *iface.BandwidthKbps))
		}

		// MAC Address
		if iface.MacAddress != nil {
			builder.WriteString(fmt.Sprintf("<mac-address>%s</mac-address>", xmlEscape(*iface.MacAddress)))
		}

		// Class (enum)
		if iface.Class != 0 {
			builder.WriteString(fmt.Sprintf("<class>%d</class>", iface.Class))
		}

		// ServiceType (enum)
		if iface.ServiceType != 0 {
			builder.WriteString(fmt.Sprintf("<service-type>%d</service-type>", iface.ServiceType))
		}

		// LinkProtocol (enum)
		if iface.LinkProtocol != 0 {
			builder.WriteString(fmt.Sprintf("<link-protocol>%d</link-protocol>", iface.LinkProtocol))
		}

		// EncapsulationType (enum)
		if iface.EncapsulationType != 0 {
			builder.WriteString(fmt.Sprintf("<encapsulation-type>%d</encapsulation-type>", iface.EncapsulationType))
		}

		// RouterType (enum)
		if iface.RouterType != 0 {
			builder.WriteString(fmt.Sprintf("<router-type>%d</router-type>", iface.RouterType))
		}

		// NetworkLayerStatus (enum)
		if iface.NetworkLayerStatus != 0 {
			builder.WriteString(fmt.Sprintf("<network-layer-status>%d</network-layer-status>", iface.NetworkLayerStatus))
		}

		// Index
		if iface.Index != nil {
			builder.WriteString(fmt.Sprintf("<index>%d</index>", *iface.Index))
		}

		// Number
		if iface.Number != nil {
			builder.WriteString(fmt.Sprintf("<number>%s</number>", xmlEscape(*iface.Number)))
		}

		// ParentName
		if iface.ParentName != nil {
			builder.WriteString(fmt.Sprintf("<parent-name>%s</parent-name>", xmlEscape(*iface.ParentName)))
		}

		// Position
		if iface.Position != nil {
			builder.WriteString(fmt.Sprintf("<position>%s</position>", xmlEscape(*iface.Position)))
		}

		// VrfName
		if iface.VrfName != nil {
			builder.WriteString(fmt.Sprintf("<vrf-name>%s</vrf-name>", xmlEscape(*iface.VrfName)))
		}

		// VsName
		if iface.VsName != nil {
			builder.WriteString(fmt.Sprintf("<vs-name>%s</vs-name>", xmlEscape(*iface.VsName)))
		}

		// AggregationName
		if iface.AggregationName != nil {
			builder.WriteString(fmt.Sprintf("<aggregation-name>%s</aggregation-name>", xmlEscape(*iface.AggregationName)))
		}

		// Boolean flags
		if iface.ClearIpDf != nil {
			builder.WriteString(fmt.Sprintf("<clear-ip-df>%t</clear-ip-df>", *iface.ClearIpDf))
		}
		if iface.IsL2Switch != nil {
			builder.WriteString(fmt.Sprintf("<is-l2-switch>%t</is-l2-switch>", *iface.IsL2Switch))
		}
		if iface.L2ModeEnable != nil {
			builder.WriteString(fmt.Sprintf("<l2-mode-enable>%t</l2-mode-enable>", *iface.L2ModeEnable))
		}
		if iface.LinkUpDownTrapEnable != nil {
			builder.WriteString(fmt.Sprintf("<link-up-down-trap-enable>%t</link-up-down-trap-enable>", *iface.LinkUpDownTrapEnable))
		}
		if iface.SpreadMtuFlag != nil {
			builder.WriteString(fmt.Sprintf("<spread-mtu-flag>%t</spread-mtu-flag>", *iface.SpreadMtuFlag))
		}
		if iface.StatisticEnable != nil {
			builder.WriteString(fmt.Sprintf("<statistic-enable>%t</statistic-enable>", *iface.StatisticEnable))
		}

		// StatisticInterval
		if iface.StatisticInterval != nil {
			builder.WriteString(fmt.Sprintf("<statistic-interval>%d</statistic-interval>", *iface.StatisticInterval))
		}

		// StatisticMode (enum)
		if iface.StatisticMode != 0 {
			builder.WriteString(fmt.Sprintf("<statistic-mode>%d</statistic-mode>", iface.StatisticMode))
		}

		// L2SwitchPortIndex
		if iface.L2SwitchPortIndex != nil {
			builder.WriteString(fmt.Sprintf("<l2-switch-port-index>%d</l2-switch-port-index>", *iface.L2SwitchPortIndex))
		}

		// DownDelayTime
		if iface.DownDelayTime != nil {
			builder.WriteString(fmt.Sprintf("<down-delay-time>%d</down-delay-time>", *iface.DownDelayTime))
		}

		// ProtocolUpDelayTime
		if iface.ProtocolUpDelayTime != nil {
			builder.WriteString(fmt.Sprintf("<protocol-up-delay-time>%d</protocol-up-delay-time>", *iface.ProtocolUpDelayTime))
		}

		// Damp container
		if iface.Damp != nil {
			builder.WriteString("<damp>")
			if iface.Damp.Auto != nil {
				builder.WriteString("<auto>")
				if iface.Damp.Auto.Level != 0 {
					builder.WriteString(fmt.Sprintf("<level>%d</level>", iface.Damp.Auto.Level))
				}
				builder.WriteString("</auto>")
			}
			if iface.Damp.Manual != nil {
				builder.WriteString("<manual>")
				if iface.Damp.Manual.HalfLifePeriod != nil {
					builder.WriteString(fmt.Sprintf("<half-life-period>%d</half-life-period>", *iface.Damp.Manual.HalfLifePeriod))
				}
				if iface.Damp.Manual.MaxSuppressTime != nil {
					builder.WriteString(fmt.Sprintf("<max-suppress-time>%d</max-suppress-time>", *iface.Damp.Manual.MaxSuppressTime))
				}
				if iface.Damp.Manual.Reuse != nil {
					builder.WriteString(fmt.Sprintf("<reuse>%d</reuse>", *iface.Damp.Manual.Reuse))
				}
				if iface.Damp.Manual.Suppress != nil {
					builder.WriteString(fmt.Sprintf("<suppress>%d</suppress>", *iface.Damp.Manual.Suppress))
				}
				builder.WriteString("</manual>")
			}
			builder.WriteString("</damp>")
		}

		// ErrorDown container
		if iface.ErrorDown != nil {
			builder.WriteString("<error-down>")
			if iface.ErrorDown.Cause != 0 {
				builder.WriteString(fmt.Sprintf("<cause>%d</cause>", iface.ErrorDown.Cause))
			}
			if iface.ErrorDown.RecoveryTime != nil {
				builder.WriteString(fmt.Sprintf("<recovery-time>%d</recovery-time>", *iface.ErrorDown.RecoveryTime))
			}
			if iface.ErrorDown.RemainderTime != nil {
				builder.WriteString(fmt.Sprintf("<remainder-time>%d</remainder-time>", *iface.ErrorDown.RemainderTime))
			}
			builder.WriteString("</error-down>")
		}

		// ControlFlap container
		if iface.ControlFlap != nil {
			builder.WriteString("<control-flap>")
			if iface.ControlFlap.Ceiling != nil {
				builder.WriteString(fmt.Sprintf("<ceiling>%d</ceiling>", *iface.ControlFlap.Ceiling))
			}
			if iface.ControlFlap.Reuse != nil {
				builder.WriteString(fmt.Sprintf("<reuse>%d</reuse>", *iface.ControlFlap.Reuse))
			}
			if iface.ControlFlap.Suppress != nil {
				builder.WriteString(fmt.Sprintf("<suppress>%d</suppress>", *iface.ControlFlap.Suppress))
			}
			if iface.ControlFlap.DecayOk != nil {
				builder.WriteString(fmt.Sprintf("<decay-ok>%d</decay-ok>", *iface.ControlFlap.DecayOk))
			}
			if iface.ControlFlap.DecayNg != nil {
				builder.WriteString(fmt.Sprintf("<decay-ng>%d</decay-ng>", *iface.ControlFlap.DecayNg))
			}
			if iface.ControlFlap.ControlFlapCount != nil {
				builder.WriteString(fmt.Sprintf("<control-flap-count>%d</control-flap-count>", *iface.ControlFlap.ControlFlapCount))
			}
			builder.WriteString("</control-flap>")
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

const HuaweiVlanNS = "urn:huawei:params:xml:ns:yang:huawei-vlan"
const HuaweiIfmNS = "urn:huawei:params:xml:ns:yang:huawei-ifm"

// NetconfBaseNS is the NETCONF base namespace carrying the edit-config
// `operation` attribute (RFC 6241 §7.2).
const NetconfBaseNS = "urn:ietf:params:xml:ns:netconf:base:1.0"

// marshalDeleteChange builds a keyed edit-config delete for the model entries in
// target (DP-07)：外层模型容器 + 条目元素带 nc:operation="delete" + 仅 key 叶
// （key 为首元素，对齐 RFC 键匹配惯例；真机与 netconfsim 均按此匹配条目）。
// 未知模型返回明确错误，绝不发送无目标的裸 delete 元素（R08）。
func marshalDeleteChange(target interface{}) (string, error) {
	switch v := target.(type) {
	case *huawei.HuaweiVlan_Vlan_Vlans:
		if v == nil || len(v.Vlan) == 0 {
			return "", fmt.Errorf("marshal delete: empty vlan target")
		}
		var b strings.Builder
		fmt.Fprintf(&b, `<vlans xmlns="%s">`, HuaweiVlanNS)
		for id, vlan := range v.Vlan {
			key := id
			if vlan != nil && vlan.Id != nil {
				key = *vlan.Id
			}
			fmt.Fprintf(&b, `<vlan nc:operation="delete" xmlns:nc="%s"><id>%d</id></vlan>`, NetconfBaseNS, key)
		}
		b.WriteString(`</vlans>`)
		return b.String(), nil
	case *huawei.HuaweiIfm_Ifm_Interfaces:
		if v == nil || len(v.Interface) == 0 {
			return "", fmt.Errorf("marshal delete: empty ifm target")
		}
		var b strings.Builder
		fmt.Fprintf(&b, `<interfaces xmlns="%s">`, HuaweiIfmNS)
		for name, iface := range v.Interface {
			key := name
			if iface != nil && iface.Name != nil {
				key = *iface.Name
			}
			fmt.Fprintf(&b, `<interface nc:operation="delete" xmlns:nc="%s"><name>%s</name></interface>`, NetconfBaseNS, xmlEscape(key))
		}
		b.WriteString(`</interfaces>`)
		return b.String(), nil
	}
	return "", fmt.Errorf("marshal delete: unsupported model %T", target)
}

// ifmInterfaceXML is a plain intermediate struct for decoding a single <interface>
// element from a device get-config reply. ygot-generated structs render YANG lists as
// Go maps and carry no `xml:` tags, so encoding/xml cannot unmarshal into them directly
// —— that is why the actual config was silently empty and the reconciler永远算出 diff
// （前端「一直漂移」）。We decode into this struct, then build the ygot map by hand.
// 覆盖 UI 可配置字段（与 mapEntryToHuaweiInterface 对齐：name/description/admin-status/mtu/type），
// 这些正是对账 diff 会比较的字段，足以让设备落盘 desired 后收敛。
type ifmInterfaceXML struct {
	Name        string  `xml:"name"`
	Description *string `xml:"description"`
	AdminStatus *uint64 `xml:"admin-status"`
	Mtu         *uint32 `xml:"mtu"`
	Type        *uint64 `xml:"type"`
	// 标识/呈现叶（通用控制台表格列）：此前回读不透出 → 前端列恒空、
	// 种子/真机数据无法展示（同「回读解析恒空」根因谱系）。
	Class        *uint64 `xml:"class"`
	ParentName   *string `xml:"parent-name"`
	Number       *string `xml:"number"`
	LinkProtocol *uint64 `xml:"link-protocol"`
	RouterType   *uint64 `xml:"router-type"`
}

// vlanMemberPortXML / vlanEntryXML are plain intermediate structs for decoding a device
// get-config <vlan> element. Same rationale as ifmInterfaceXML: ygot renders the VLAN list
// (and its member-port list) as Go maps with no xml tags, so encoding/xml cannot unmarshal
// into them — actual 恒空 → VLAN 永久漂移。覆盖 UI 可配置的扁平叶子 + 嵌套 member-port，
// 与 buildHuaweiVlanVlansXML 下发字段对齐，保证设备落盘 desired 后对账收敛。
type vlanMemberPortXML struct {
	InterfaceName string  `xml:"interface-name"`
	AccessType    *uint64 `xml:"access-type"`
	TagMode       *uint64 `xml:"tag-mode"`
}

type vlanEntryXML struct {
	Id               *uint16             `xml:"id"`
	Name             *string             `xml:"name"`
	Description      *string             `xml:"description"`
	AdminStatus      *uint64             `xml:"admin-status"`
	Type             *uint64             `xml:"type"`
	BroadcastDiscard *uint64             `xml:"broadcast-discard"`
	MacLearning      *uint64             `xml:"mac-learning"`
	StatisticEnable  *uint64             `xml:"statistic-enable"`
	MemberPorts      []vlanMemberPortXML `xml:"member-ports>member-port"`
}

// ParseHuaweiVlanVlansXML parses a NETCONF get-config reply (raw XML bytes, wrapped or bare)
// into a *huawei.HuaweiVlan_Vlan_Vlans with its Vlan map (key = VLAN id) populated. Robust to
// namespace prefixes and outer wrappers via token scanning. Returns an empty (non-nil) container
// when no vlans are present.
func ParseHuaweiVlanVlansXML(data []byte) (*huawei.HuaweiVlan_Vlan_Vlans, error) {
	result := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: make(map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan),
	}
	if len(data) == 0 {
		return result, nil
	}

	dec := xml.NewDecoder(bytes.NewReader(data))
	idx := 0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse vlan vlans xml: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "vlan" {
			continue
		}

		var x vlanEntryXML
		if err := dec.DecodeElement(&x, &se); err != nil {
			return nil, fmt.Errorf("decode <vlan>: %w", err)
		}

		entry := &huawei.HuaweiVlan_Vlan_Vlans_Vlan{}
		if x.Id != nil {
			entry.Id = x.Id
		}
		if x.Name != nil {
			entry.Name = x.Name
		}
		if x.Description != nil {
			entry.Description = x.Description
		}
		if x.AdminStatus != nil {
			entry.AdminStatus = huawei.E_HuaweiVlan_AdminStatus(*x.AdminStatus)
		}
		if x.Type != nil {
			entry.Type = huawei.E_HuaweiVlan_VlanType(*x.Type)
		}
		if x.BroadcastDiscard != nil {
			entry.BroadcastDiscard = huawei.E_HuaweiVlan_EnableStatus(*x.BroadcastDiscard)
		}
		if x.MacLearning != nil {
			entry.MacLearning = huawei.E_HuaweiVlan_EnableStatus(*x.MacLearning)
		}
		if x.StatisticEnable != nil {
			entry.StatisticEnable = huawei.E_HuaweiVlan_EnableStatus(*x.StatisticEnable)
		}
		if len(x.MemberPorts) > 0 {
			mp := &huawei.HuaweiVlan_Vlan_Vlans_Vlan_MemberPorts{
				MemberPort: make(map[string]*huawei.HuaweiVlan_Vlan_Vlans_Vlan_MemberPorts_MemberPort),
			}
			for _, p := range x.MemberPorts {
				port := &huawei.HuaweiVlan_Vlan_Vlans_Vlan_MemberPorts_MemberPort{}
				if p.InterfaceName != "" {
					name := p.InterfaceName
					port.InterfaceName = &name
				}
				if p.AccessType != nil {
					port.AccessType = huawei.E_HuaweiVlan_AccessType(*p.AccessType)
				}
				if p.TagMode != nil {
					port.TagMode = huawei.E_HuaweiVlan_TagMode(*p.TagMode)
				}
				key := p.InterfaceName
				if key == "" {
					key = fmt.Sprintf("port-%d", len(mp.MemberPort))
				}
				mp.MemberPort[key] = port
			}
			entry.MemberPorts = mp
		}

		var key uint16
		if x.Id != nil {
			key = *x.Id
		} else {
			key = uint16(idx)
		}
		idx++
		result.Vlan[key] = entry
	}

	return result, nil
}

// ParseHuaweiIfmInterfacesXML parses a NETCONF get-config reply (raw XML bytes, whether
// wrapped in <data>/<rpc-reply> or bare <interfaces>) into a *huawei.HuaweiIfm_Ifm_Interfaces
// with its Interface map populated. It scans the token stream for <interface> elements so it
// is robust to namespace prefixes and outer wrapper tags. Returns an empty (non-nil) container
// when the reply carries no interfaces.
func ParseHuaweiIfmInterfacesXML(data []byte) (*huawei.HuaweiIfm_Ifm_Interfaces, error) {
	result := &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: make(map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface),
	}
	if len(data) == 0 {
		return result, nil
	}

	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse ifm interfaces xml: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "interface" {
			continue
		}

		var x ifmInterfaceXML
		if err := dec.DecodeElement(&x, &se); err != nil {
			return nil, fmt.Errorf("decode <interface>: %w", err)
		}

		entry := &huawei.HuaweiIfm_Ifm_Interfaces_Interface{}
		if x.Name != "" {
			name := x.Name
			entry.Name = &name
		}
		if x.Description != nil {
			entry.Description = x.Description
		}
		if x.Mtu != nil {
			entry.Mtu = x.Mtu
		}
		if x.AdminStatus != nil {
			entry.AdminStatus = huawei.E_HuaweiIfm_PortStatus(*x.AdminStatus)
		}
		if x.Type != nil {
			entry.Type = huawei.E_HuaweiIfm_PortType(*x.Type)
		}
		if x.Class != nil {
			entry.Class = huawei.E_HuaweiIfm_ClassType(*x.Class)
		}
		if x.ParentName != nil {
			entry.ParentName = x.ParentName
		}
		if x.Number != nil {
			entry.Number = x.Number
		}
		if x.LinkProtocol != nil {
			entry.LinkProtocol = huawei.E_HuaweiIfm_LinkProtocol(*x.LinkProtocol)
		}
		if x.RouterType != nil {
			entry.RouterType = huawei.E_HuaweiIfm_RouterType(*x.RouterType)
		}

		key := x.Name
		if key == "" {
			key = fmt.Sprintf("iface-%d", len(result.Interface))
		}
		result.Interface[key] = entry
	}

	return result, nil
}

// buildHuaweiVlanVlansXML generates Huawei VLAN standard XML for VLAN configuration.
func buildHuaweiVlanVlansXML(vlans *huawei.HuaweiVlan_Vlan_Vlans) (string, error) {
	if vlans == nil || len(vlans.Vlan) == 0 {
		return fmt.Sprintf(`<vlans xmlns="%s"/>`, HuaweiVlanNS), nil
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf(`<vlans xmlns="%s">`, HuaweiVlanNS))

	for vlanID, vlan := range vlans.Vlan {
		if vlan == nil {
			continue
		}

		builder.WriteString("<vlan>")

		// VLAN ID (required)
		if vlan.Id != nil {
			builder.WriteString(fmt.Sprintf("<id>%d</id>", *vlan.Id))
		} else {
			// Use the map key as ID
			builder.WriteString(fmt.Sprintf("<id>%d</id>", vlanID))
		}

		// Name
		if vlan.Name != nil {
			builder.WriteString(fmt.Sprintf("<name>%s</name>", xmlEscape(*vlan.Name)))
		}

		// Description
		if vlan.Description != nil {
			builder.WriteString(fmt.Sprintf("<description>%s</description>", xmlEscape(*vlan.Description)))
		}

		// AdminStatus (enum)
		if vlan.AdminStatus != 0 {
			builder.WriteString(fmt.Sprintf("<admin-status>%d</admin-status>", vlan.AdminStatus))
		}

		// Type (enum)
		if vlan.Type != 0 {
			builder.WriteString(fmt.Sprintf("<type>%d</type>", vlan.Type))
		}

		// BroadcastDiscard (enum)
		if vlan.BroadcastDiscard != 0 {
			builder.WriteString(fmt.Sprintf("<broadcast-discard>%d</broadcast-discard>", vlan.BroadcastDiscard))
		}

		// MacLearning (enum)
		if vlan.MacLearning != 0 {
			builder.WriteString(fmt.Sprintf("<mac-learning>%d</mac-learning>", vlan.MacLearning))
		}

		// StatisticEnable (enum)
		if vlan.StatisticEnable != 0 {
			builder.WriteString(fmt.Sprintf("<statistic-enable>%d</statistic-enable>", vlan.StatisticEnable))
		}

		// StatisticDiscard (enum)
		if vlan.StatisticDiscard != 0 {
			builder.WriteString(fmt.Sprintf("<statistic-discard>%d</statistic-discard>", vlan.StatisticDiscard))
		}

		// UnknownMulticastDiscard (enum)
		if vlan.UnknownMulticastDiscard != 0 {
			builder.WriteString(fmt.Sprintf("<unknown-multicast-discard>%d</unknown-multicast-discard>", vlan.UnknownMulticastDiscard))
		}

		// UnkownUnicastDiscard (nested container)
		if vlan.UnkownUnicastDiscard != nil {
			builder.WriteString("<unkown-unicast-discard>")
			if vlan.UnkownUnicastDiscard.Discard != 0 {
				builder.WriteString(fmt.Sprintf("<discard>%d</discard>", vlan.UnkownUnicastDiscard.Discard))
			}
			if vlan.UnkownUnicastDiscard.MacLearningEnable != 0 {
				builder.WriteString(fmt.Sprintf("<mac-learning-enable>%d</mac-learning-enable>", vlan.UnkownUnicastDiscard.MacLearningEnable))
			}
			builder.WriteString("</unkown-unicast-discard>")
		}

		// Suppression (nested container)
		if vlan.Suppression != nil {
			builder.WriteString("<suppression>")
			if vlan.Suppression.Inbound != 0 {
				builder.WriteString(fmt.Sprintf("<inbound>%d</inbound>", vlan.Suppression.Inbound))
			}
			if vlan.Suppression.Outbound != 0 {
				builder.WriteString(fmt.Sprintf("<outbound>%d</outbound>", vlan.Suppression.Outbound))
			}
			builder.WriteString("</suppression>")
		}

		// MacAgingTime
		if vlan.MacAgingTime != nil {
			builder.WriteString(fmt.Sprintf("<mac-aging-time>%d</mac-aging-time>", *vlan.MacAgingTime))
		}

		// SuperVlan
		if vlan.SuperVlan != nil {
			builder.WriteString(fmt.Sprintf("<super-vlan>%d</super-vlan>", *vlan.SuperVlan))
		}

		// MemberPorts (container with port list)
		if vlan.MemberPorts != nil && len(vlan.MemberPorts.MemberPort) > 0 {
			builder.WriteString("<member-ports>")
			for portKey, port := range vlan.MemberPorts.MemberPort {
				if port == nil {
					continue
				}
				builder.WriteString("<member-port>")
				// Interface name
				if port.InterfaceName != nil {
					builder.WriteString(fmt.Sprintf("<interface-name>%s</interface-name>", xmlEscape(*port.InterfaceName)))
				} else {
					builder.WriteString(fmt.Sprintf("<interface-name>%s</interface-name>", xmlEscape(portKey)))
				}
				// AccessType
				if port.AccessType != 0 {
					builder.WriteString(fmt.Sprintf("<access-type>%d</access-type>", port.AccessType))
				}
				// TagMode
				if port.TagMode != 0 {
					builder.WriteString(fmt.Sprintf("<tag-mode>%d</tag-mode>", port.TagMode))
				}
				builder.WriteString("</member-port>")
			}
			builder.WriteString("</member-ports>")
		}

		// Suppression container
		if vlan.Suppression != nil {
			builder.WriteString("<suppression>")
			if vlan.Suppression.Inbound != 0 {
				builder.WriteString(fmt.Sprintf("<inbound>%d</inbound>", vlan.Suppression.Inbound))
			}
			if vlan.Suppression.Outbound != 0 {
				builder.WriteString(fmt.Sprintf("<outbound>%d</outbound>", vlan.Suppression.Outbound))
			}
			builder.WriteString("</suppression>")
		}

		builder.WriteString("</vlan>")
	}

	builder.WriteString("</vlans>")
	return builder.String(), nil
}
