// Package netconfsim provides a NETCONF simulator for integration testing.
package netconfsim

import (
	"fmt"
	"net"
	"sync"
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/openconfig"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ssh"
)

const (
	defaultUsername = "admin"
	defaultPassword = "admin"
	defaultAddr     = "127.0.0.1"
)

// Simulator is a NETCONF simulator that provides a fake NETCONF server for testing.
type Simulator struct {
	username  string
	password  string
	addr      string
	port      int
	listener  net.Listener
	server    *sshServer
	config    *ssh.ServerConfig
	datastore *Datastore
	scenario  *ScenarioConfig

	mu     sync.Mutex
	running bool
	done   chan struct{}
	wg     sync.WaitGroup
}

// NewSimulator creates a new NETCONF simulator with default credentials.
func NewSimulator() *Simulator {
	ds := NewDatastore()
	return &Simulator{
		username:  defaultUsername,
		password:  defaultPassword,
		addr:      defaultAddr,
		datastore: ds,
		scenario:  NewScenarioConfig(),
		done:      make(chan struct{}),
	}
}

// Start starts the NETCONF simulator.
func (s *Simulator) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("simulator already running")
	}

	// Generate temporary SSH host key
	signer, err := generateSigner()
	if err != nil {
		return fmt.Errorf("generate SSH signer: %w", err)
	}

	// Configure SSH server
	s.config = &ssh.ServerConfig{
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if s.scenario.RejectAuth {
				return nil, fmt.Errorf("authentication rejected")
			}
			if conn.User() == s.username && string(password) == s.password {
				return &ssh.Permissions{}, nil
			}
			return nil, fmt.Errorf("invalid credentials")
		},
	}
	s.config.AddHostKey(signer)

	// Start listening
	listener, err := net.Listen("tcp", net.JoinHostPort(s.addr, "0"))
	if err != nil {
		return fmt.Errorf("start listener: %w", err)
	}
	s.listener = listener
	s.port = listener.Addr().(*net.TCPAddr).Port

	s.server = &sshServer{
		config:    s.config,
		datastore: s.datastore,
		scenario:  s.scenario,
		done:      s.done,
	}

	s.wg.Add(1)
	go s.acceptLoop()

	s.running = true
	return nil
}

// Stop stops the NETCONF simulator.
func (s *Simulator) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.done)
	if s.listener != nil {
		_ = s.listener.Close()
	}
	s.mu.Unlock()
	s.wg.Wait()
}

// Username returns the configured username.
func (s *Simulator) Username() string {
	return s.username
}

// Password returns the configured password.
func (s *Simulator) Password() string {
	return s.password
}

// Addr returns the listening address.
func (s *Simulator) Addr() string {
	return s.addr
}

// Port returns the listening port.
func (s *Simulator) Port() int {
	return s.port
}

// SetRunningConfig sets the initial running configuration from an openconfig Device.
func (s *Simulator) SetRunningConfig(dev *openconfig.Device) {
	s.datastore.SetRunningFromDevice(dev)
}

// SetScenario sets the scenario configuration for error injection testing.
func (s *Simulator) SetScenario(sc *ScenarioConfig) {
	s.scenario = sc
	if s.server != nil {
		s.server.scenario = sc
	}
}

func (s *Simulator) acceptLoop() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				continue
			}
		}

		s.wg.Add(1)
		go func() {
			s.server.handleConnection(conn)
			s.wg.Done()
		}()
	}
}

// GetDatastore returns the underlying datastore for direct assertions.
func (s *Simulator) GetDatastore() *Datastore {
	return s.datastore
}

// AssertInterfaceExists verifies that the interface with the given name exists in the running config.
func (s *Simulator) AssertInterfaceExists(t *testing.T, name string) {
	interfaces, err := s.datastore.ExtractInterfaces()
	assert.NoError(t, err, "failed to extract interfaces from running config")
	assert.NotNil(t, interfaces, "interfaces should not be nil")

	_, exists := interfaces.Interface[name]
	assert.True(t, exists, "interface %q should exist in running config, but got: %v", name, interfaces.Interface)
}

// AssertInterfaceEnabled verifies that the interface exists and has the expected enabled state.
func (s *Simulator) AssertInterfaceEnabled(t *testing.T, name string, expected bool) {
	interfaces, err := s.datastore.ExtractInterfaces()
	assert.NoError(t, err, "failed to extract interfaces from running config")

	iface, exists := interfaces.Interface[name]
	assert.True(t, exists, "interface %q should exist", name)
	if !exists {
		return
	}

	assert.NotNil(t, iface.Config, "interface %q should have Config", name)
	if iface.Config == nil {
		return
	}

	assert.NotNil(t, iface.Config.Enabled, "interface %q should have Enabled field set", name)
	if iface.Config.Enabled != nil {
		assert.Equal(t, expected, *iface.Config.Enabled, "interface %q enabled state should match", name)
	}
}

// AssertInterfaceMtu verifies that the interface exists and has the expected MTU.
func (s *Simulator) AssertInterfaceMtu(t *testing.T, name string, expectedMtu uint16) {
	interfaces, err := s.datastore.ExtractInterfaces()
	assert.NoError(t, err, "failed to extract interfaces from running config")

	iface, exists := interfaces.Interface[name]
	assert.True(t, exists, "interface %q should exist", name)
	if !exists {
		return
	}

	assert.NotNil(t, iface.Config, "interface %q should have Config", name)
	if iface.Config == nil {
		return
	}

	assert.NotNil(t, iface.Config.Mtu, "interface %q should have Mtu field set", name)
	if iface.Config.Mtu != nil {
		assert.Equal(t, expectedMtu, *iface.Config.Mtu, "interface %q MTU should match", name)
	}
}
