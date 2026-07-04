// Package netconfsim provides a NETCONF simulator for integration testing.
package netconfsim

import (
	"fmt"
	"net"
	"strconv"
	"sync"

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

	listenPort int // 0 = random free port

	mu      sync.Mutex
	running bool
	done    chan struct{}
	wg      sync.WaitGroup
}

// SetListen configures the bind address and port before Start.
// A port of 0 selects a random free port. Intended for the standalone binary;
// tests keep the default random port.
func (s *Simulator) SetListen(addr string, port int) {
	if addr != "" {
		s.addr = addr
	}
	s.listenPort = port
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
	listener, err := net.Listen("tcp", net.JoinHostPort(s.addr, strconv.Itoa(s.listenPort)))
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

// SetRunningConfig sets the initial running configuration from a Device struct.
// Accepts any device struct (e.g., openconfig.Device or huawei.Device).
func (s *Simulator) SetRunningConfig(dev interface{}) {
	s.datastore.SetRunningFromDevice(dev)
}

// SetRunningConfigXML sets the initial running configuration directly from XML bytes.
func (s *Simulator) SetRunningConfigXML(xmlBytes []byte) {
	s.datastore.SetRunningFromXML(xmlBytes)
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
