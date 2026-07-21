// Package netconfsim provides a NETCONF simulator for integration testing.
package netconfsim

import (
	"bytes"
	"encoding/xml"
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
	store     *treeDatastore
	scenario  *ScenarioConfig
	extraCaps []string

	listenPort int // 0 = random free port

	mu      sync.Mutex
	running bool
	done    chan struct{}
	wg      sync.WaitGroup

	// conns tracks live client connections so Stop can force-close them.
	// handleSession 阻塞在 readMessage（bufio 读）时感知不到 done channel，
	// 不主动断开连接 Stop 会在 wg.Wait 上永久挂起。
	connMu sync.Mutex
	conns  map[net.Conn]struct{}
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
	return &Simulator{
		username: defaultUsername,
		password: defaultPassword,
		addr:     defaultAddr,
		store:    newTreeDatastore(),
		scenario: NewScenarioConfig(),
		done:     make(chan struct{}),
		conns:    make(map[net.Conn]struct{}),
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
		store:     s.store,
		scenario:  s.scenario,
		extraCaps: s.extraCaps,
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

	// Force-close live sessions: readMessage blocks on the socket and only
	// notices done between messages, so an idle client would hang Stop forever.
	s.connMu.Lock()
	for conn := range s.conns {
		_ = conn.Close()
	}
	s.connMu.Unlock()

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
// Accepts any device struct (e.g., huawei.Device).
func (s *Simulator) SetRunningConfig(dev interface{}) {
	_ = s.store.SetRunning(deviceToConfigXML(dev))
}

// SetRunningConfigXML sets the initial running configuration directly from XML bytes.
func (s *Simulator) SetRunningConfigXML(xmlBytes []byte) {
	_ = s.store.SetRunning(xmlBytes)
}

// deviceToConfigXML marshals a device struct to XML, wrapping it in <config> when
// needed. Mirrors the legacy Datastore.SetRunningFromDevice marshaling so seeded
// running config is byte-for-byte equivalent to the pre-switch behavior.
func deviceToConfigXML(dev interface{}) []byte {
	buf, err := xml.Marshal(dev)
	if err != nil {
		return []byte(`<config/>`)
	}
	if !bytes.Contains(buf, []byte("<config")) {
		buf = []byte(fmt.Sprintf("<config>%s</config>", buf))
	}
	return buf
}

// SetStateDataXML sets the config-false state overlay from XML bytes (NS-08):
// merged into <get> responses, invisible to <get-config>, untouched by config
// writes. Returns the parse error, leaving prior state intact on failure.
func (s *Simulator) SetStateDataXML(xmlBytes []byte) error {
	return s.store.SetState(xmlBytes)
}

// SetCapabilities sets extra YANG-module capabilities to advertise in the NETCONF
// hello (in addition to base:1.0/:candidate/:writable-running). Must be called
// before Start. Used to exercise per-device capability-based module narrowing.
func (s *Simulator) SetCapabilities(caps []string) {
	s.extraCaps = caps
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

		// Register under connMu with a done re-check: Stop closes done before
		// sweeping conns, so a conn registered after the sweep must observe done
		// here and close itself — otherwise it would outlive Stop and hang wg.Wait.
		s.connMu.Lock()
		select {
		case <-s.done:
			s.connMu.Unlock()
			_ = conn.Close()
			return
		default:
		}
		s.conns[conn] = struct{}{}
		s.connMu.Unlock()

		s.wg.Add(1)
		go func() {
			s.server.handleConnection(conn)
			s.connMu.Lock()
			delete(s.conns, conn)
			s.connMu.Unlock()
			s.wg.Done()
		}()
	}
}
