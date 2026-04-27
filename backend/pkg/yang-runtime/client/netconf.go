package client

import (
	"context"
	"encoding/xml"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

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
