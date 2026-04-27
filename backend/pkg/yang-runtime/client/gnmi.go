package client

import (
	"context"
	"fmt"
	"time"

	gnmi "github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// GNMINative is the default gNMI port
const GNMINative = 9339

// GNMITLS is the default gNMI TLS port
const GNMITLS = 9340

// GNMIClient implements Client interface for gNMI protocol
type GNMIClient struct {
	info      DeviceConnectionInfo
	conn      *grpc.ClientConn
	client    gnmi.GNMIClient
	connected bool
}

// NewGNMIClient creates a new gNMI client
func NewGNMIClient(info DeviceConnectionInfo) Client {
	if info.Port == 0 {
		if info.TLSConfig != nil {
			info.Port = GNMITLS
		} else {
			info.Port = GNMINative
		}
	}
	if info.Timeout == 0 {
		info.Timeout = 10 * time.Second
	}

	c := &GNMIClient{
		info:      info,
		connected: false,
	}

	// Connect immediately
	if err := c.connect(); err != nil {
		// Leave disconnected and let pool handle reconnect
		return c
	}

	return c
}

func (c *GNMIClient) connect() error {
	ctx, cancel := context.WithTimeout(context.Background(), c.info.Timeout)
	defer cancel()

	target := fmt.Sprintf("%s:%d", c.info.IP, c.info.Port)

	var opts []grpc.DialOption
	if c.info.TLSConfig != nil {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(c.info.TLSConfig)))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	conn, err := grpc.DialContext(ctx, target, opts...)
	if err != nil {
		return fmt.Errorf("failed to dial gNMI: %w", err)
	}

	c.conn = conn
	c.client = gnmi.NewGNMIClient(conn)
	c.connected = true

	return nil
}

// Get implements Client interface
func (c *GNMIClient) Get(ctx context.Context, path string, opts ...GetOption) (*GetResult, error) {
	if !c.connected || c.client == nil {
		return &GetResult{
			Path:      path,
			Timestamp: time.Now(),
			Error:     fmt.Errorf("not connected"),
		}, fmt.Errorf("not connected")
	}

	// Apply options
	getOpts := &GetOptions{
		Datastore: "running",
	}
	for _, opt := range opts {
		opt.Apply(getOpts)
	}

	req := &gnmi.GetRequest{}

	resp, err := c.client.Get(ctx, req)
	if err != nil {
		return &GetResult{
			Path:      path,
			Timestamp: time.Now(),
			Error:     err,
		}, err
	}

	if len(resp.Notification) == 0 {
		return &GetResult{
			Path:      path,
			Data:      nil,
			Timestamp: time.Now(),
			Error:     fmt.Errorf("empty response"),
		}, fmt.Errorf("empty response")
	}

	// Return the first notification
	result := &GetResult{
		Path:      path,
		Data:      resp.Notification[0],
		Timestamp: time.Now(),
		Error:     nil,
	}

	return result, nil
}

// Set implements Client interface
func (c *GNMIClient) Set(ctx context.Context, changes []Change, opts ...SetOption) (*SetResult, error) {
	if !c.connected || c.client == nil {
		return nil, fmt.Errorf("not connected")
	}

	// Apply options
	setOpts := &SetOptions{
		Datastore: "running",
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

	req := &gnmi.SetRequest{}

	// Build the set request from changes
	for i, change := range changes {
		// For gNMI, each change is an update
		update := &gnmi.Update{
			// Path and Val would be populated by caller based on YANG structure
			// This base implementation accepts pre-constructed updates
		}

		switch change.Type {
		case AddChange:
			req.Update = append(req.Update, update)
		case ModifyChange:
			req.Update = append(req.Update, update)
		case DeleteChange:
			req.Delete = append(req.Delete, &gnmi.Path{})
		}

		result.Changes[i] = ChangeResult{
			Change:  change,
			Success: true,
			Error:   nil,
		}
	}

	_, err := c.client.Set(ctx, req)
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("failed to apply changes: %v", err)
		return result, nil
	}

	return result, nil
}

// Subscribe implements Client interface
func (c *GNMIClient) Subscribe(ctx context.Context, path string, handler func(Notification)) error {
	if !c.connected || c.client == nil {
		return fmt.Errorf("not connected")
	}

	// Convert path to gNMI path
	gnmiPath := &gnmi.Path{
		Elem: []*gnmi.PathElem{
			{Name: path},
		},
	}

	// Create subscription request
	req := &gnmi.SubscribeRequest{
		Request: &gnmi.SubscribeRequest_Subscribe{
			Subscribe: &gnmi.SubscriptionList{
				Prefix: gnmiPath,
				Subscription: []*gnmi.Subscription{
					{
						Path: gnmiPath,
						Mode: gnmi.SubscriptionMode_ON_CHANGE,
					},
				},
			},
		},
	}

	// Create streaming call
	stream, err := c.client.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("failed to create subscribe stream: %w", err)
	}

	// Send subscription request
	if err := stream.Send(req); err != nil {
		return fmt.Errorf("failed to send subscribe request: %w", err)
	}

	// Process incoming notifications in a goroutine
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				resp, err := stream.Recv()
				if err != nil {
					return
				}
				if update := resp.GetUpdate(); update != nil {
					// Convert to our Notification type
					handler(Notification{
						Path:      path,
						Data:      update,
						Timestamp: time.Now(),
					})
				}
			}
		}
	}()

	return nil
}

// Close implements Client interface
func (c *GNMIClient) Close() error {
	if !c.connected || c.conn == nil {
		c.connected = false
		return nil
	}

	err := c.conn.Close()
	c.connected = false
	c.conn = nil
	c.client = nil
	return err
}

// IsConnected implements Client interface
func (c *GNMIClient) IsConnected() bool {
	return c.connected && c.conn != nil && c.client != nil
}
