package client

import (
	"context"
	"fmt"
	"time"

	ygot "github.com/openconfig/ygot/ygot"
	"github.com/scrapli/scrapligo/driver/netconf"
	"github.com/scrapli/scrapligo/driver/options"
	"github.com/scrapli/scrapligo/util"
)

// NETCONFDefaultPort is the default NETCONF port
const NETCONFDefaultPort = 830

// NETCONFClient implements Client interface for NETCONF protocol
type NETCONFClient struct {
	info   DeviceConnectionInfo
	driver *netconf.Driver
	connected bool
}

// NewNETCONFClient creates a new NETCONF client
func NewNETCONFClient(info DeviceConnectionInfo) Client {
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
		// We leave disconnected and let the pool handle reconnect
		return c
	}

	return c
}

func (c *NETCONFClient) connect() error {
	opts := []util.Option{
		options.WithAuthUsername(c.info.Username),
		options.WithAuthPassword(c.info.Password),
		options.WithPort(c.info.Port),
		options.WithTimeoutSocket(c.info.Timeout),
	}

	// TODO: add TLS config option support
	// if c.info.TLSConfig != nil {
	// 	opts = append(opts, options.WithTLSConfig(c.info.TLSConfig))
	// }

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
	if !c.connected || c.driver == nil {
		return &GetResult{
			Error: fmt.Errorf("not connected"),
		}, fmt.Errorf("not connected")
	}

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
	resp, err := c.driver.GetConfig(getOpts.Datastore, withFilter)
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
	if !c.connected || c.driver == nil {
		return nil, fmt.Errorf("not connected")
	}

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

		_, err = c.driver.EditConfig(setOpts.Datastore, xmlConfig)
		if err != nil {
			result.Changes[i] = ChangeResult{
				Change:  change,
				Success: false,
				Error:   err,
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
		_, err := c.driver.Commit()
		if err != nil {
			result.Success = false
			result.Message = fmt.Sprintf("partial success: failed to commit: %v", err)
			return result, nil
		}
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
	if !c.connected || c.driver == nil {
		return nil
	}

	err := c.driver.Close()
	if err != nil {
		c.connected = false
		c.driver = nil
		return err
	}

	c.connected = false
	c.driver = nil
	return nil
}

// IsConnected implements Client interface
func (c *NETCONFClient) IsConnected() bool {
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
		return fmt.Sprintf(`<delete operation="delete" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">`), nil
	}

	// If the value is already a byte slice/string, use it directly
	switch v := change.NewValue.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	}

	// If it's a ygot struct, marshal it to JSON (RFC7951) which can be translated
	// For NETCONF/XML we could use existing xpath/xml construction but this works for now
	output, err := ygot.Marshal7951(change.NewValue)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config to JSON: %w", err)
	}

	return string(output), nil
}
