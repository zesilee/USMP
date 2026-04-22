package client

import (
	"context"
	"crypto/tls"
	"time"
)

// Protocol represents the network management protocol
type Protocol string

const (
	// ProtocolNETCONF represents NETCONF protocol
	ProtocolNETCONF Protocol = "netconf"
	// ProtocolGNMI represents gNMI protocol
	ProtocolGNMI Protocol = "gnmi"
	// ProtocolAUTO automatically detects protocol
	ProtocolAUTO Protocol = "auto"
)

// GetOption is an option for Get operations
type GetOption interface {
	// Apply applies the option to the get options
	Apply(*GetOptions)
}

// GetOptions contains options for Get operations
type GetOptions struct {
	// Datastore is the datastore to read from (running/candidate)
	Datastore string
	// Timeout is the request timeout
	Timeout time.Duration
}

// WithDatastore sets the datastore for get operations
func WithDatastore(datastore string) GetOption {
	return &getOptionDatastore{datastore: datastore}
}

type getOptionDatastore struct {
	datastore string
}

func (o *getOptionDatastore) Apply(opts *GetOptions) {
	opts.Datastore = o.datastore
}

// WithTimeout sets the timeout for get operations
func WithTimeout(timeout time.Duration) GetOption {
	return &getOptionTimeout{timeout: timeout}
}

type getOptionTimeout struct {
	timeout time.Duration
}

func (o *getOptionTimeout) Apply(opts *GetOptions) {
	opts.Timeout = o.timeout
}

// SetOption is an option for Set operations
type SetOption interface {
	// Apply applies the option to the set options
	Apply(*SetOptions)
}

// SetOptions contains options for Set operations
type SetOptions struct {
	// Datastore is the datastore to write to
	Datastore string
	// Timeout is the request timeout
	Timeout time.Duration
	// Commit indicates whether to commit after applying changes
	Commit bool
}

// WithCommit sets whether to commit after applying changes
func WithCommit(commit bool) SetOption {
	return &setOptionCommit{commit: commit}
}

type setOptionCommit struct {
	commit bool
}

func (o *setOptionCommit) Apply(opts *SetOptions) {
	opts.Commit = o.commit
}

// DeviceConnectionInfo contains all information needed to connect to a device
type DeviceConnectionInfo struct {
	// IP is the device IP address
	IP string
	// Port is the protocol port (830 for NETCONF, 9339 for gNMI by default)
	Port int
	// Username is the authentication username
	Username string
	// Password is the authentication password
	Password string
	// TLSConfig contains TLS configuration for encrypted connections
	TLSConfig *tls.Config
	// Protocol specifies which protocol to use
	Protocol Protocol
	// Timeout is the connection timeout
	Timeout time.Duration
}

// Client is the unified interface for device configuration clients
type Client interface {
	// Get retrieves configuration at the specified path
	Get(ctx context.Context, path string, opts ...GetOption) (*GetResult, error)
	// Set applies configuration changes to the device
	Set(ctx context.Context, changes []Change, opts ...SetOption) (*SetResult, error)
	// Subscribe subscribes to state change notifications from the device
	Subscribe(ctx context.Context, path string, handler func(Notification)) error
	// Close closes the client connection
	Close() error
	// IsConnected checks if the client is currently connected
	IsConnected() bool
}
