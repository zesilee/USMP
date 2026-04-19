package netconf

import (
	"fmt"

	"github.com/leezesi/usmp/internal/types"
	"github.com/scrapli/scrapligo/driver/netconf"
	"github.com/scrapli/scrapligo/driver/options"
	"github.com/scrapli/scrapligo/util"
)

// Client is a NETCONF client for a single switch device
type Client struct {
	device    types.DeviceInfo
	session   *netconf.Driver
	connected bool
}

// NewClient creates a new NETCONF client
func NewClient(device types.DeviceInfo) *Client {
	return &Client{
		device:    device,
		session:   nil,
		connected: false,
	}
}

// Connect establishes a NETCONF connection to the device
func (c *Client) Connect() error {
	opts := []util.Option{
		options.WithAuthUsername(c.device.Username),
		options.WithAuthPassword(c.device.Password),
		options.WithPort(c.device.Port),
		options.WithTimeoutSocket(10),
	}

	driver, err := netconf.NewDriver(
		c.device.IP,
		opts...,
	)
	if err != nil {
		return fmt.Errorf("failed to create NETCONF driver: %w", err)
	}

	err = driver.Open()
	if err != nil {
		return fmt.Errorf("failed to open NETCONF connection: %w", err)
	}

	c.session = driver
	c.connected = true

	return nil
}

// Disconnect closes the NETCONF connection
func (c *Client) Disconnect() error {
	if !c.connected || c.session == nil {
		return nil
	}

	err := c.session.Close()
	if err != nil {
		c.connected = false
		c.session = nil
		return err
	}

	c.connected = false
	c.session = nil
	return nil
}

// IsConnected returns the current connection state
func (c *Client) IsConnected() bool {
	return c.connected && c.session != nil
}

// GetSession returns the underlying NETCONF driver session
func (c *Client) GetSession() *netconf.Driver {
	return c.session
}

// GetConfig retrieves configuration for a specific YANG path
func (c *Client) GetConfig(yangPath string) ([]byte, error) {
	if !c.connected || c.session == nil {
		return nil, fmt.Errorf("not connected")
	}

	filter := ConstructGetConfigFilter(yangPath)
	resp, err := c.session.GetConfig(filter)
	if err != nil {
		return nil, fmt.Errorf("get-config failed: %w", err)
	}

	if resp == nil || len(resp.Result) == 0 {
		return nil, fmt.Errorf("empty response from get-config")
	}

	return []byte(resp.Result), nil
}

// EditConfig sends configuration edit to the device
func (c *Client) EditConfig(target string, configXML string) error {
	if !c.connected || c.session == nil {
		return fmt.Errorf("not connected")
	}

	_, err := c.session.EditConfig(target, configXML)
	if err != nil {
		return fmt.Errorf("edit-config failed: %w", err)
	}

	return nil
}

// Commit commits the candidate configuration
func (c *Client) Commit() error {
	if !c.connected || c.session == nil {
		return fmt.Errorf("not connected")
	}

	_, err := c.session.Commit()
	if err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	return nil
}

// EditConfigAndCommit combines edit-config and commit
func (c *Client) EditConfigAndCommit(yangPath string, data interface{}) error {
	configXML, err := ConstructEditConfig(yangPath, data)
	if err != nil {
		return fmt.Errorf("failed to construct edit-config: %w", err)
	}

	if err := c.EditConfig("running", configXML); err != nil {
		return err
	}

	if err := c.Commit(); err != nil {
		return err
	}

	return nil
}
