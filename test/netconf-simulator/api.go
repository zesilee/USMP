package netsim

import (
	"testing"

	"github.com/leezesi/usmp/internal/generated/openconfig"
	"github.com/stretchr/testify/assert"
)

// Simulator is a high-level API for the NETCONF simulator for testing
type Simulator struct {
	server *Server
}

// NewSimulator creates a new simulator with automatic port selection
func NewSimulator() *Simulator {
	cfg := &ServerConfig{
		Addr:     "127.0.0.1",
		Port:     0, // random port
		Username: "admin",
		Password: "admin",
	}
	return &Simulator{
		server: New(cfg),
	}
}

// NewSimulatorWithConfig creates a new simulator with custom configuration
func NewSimulatorWithConfig(cfg *ServerConfig) *Simulator {
	return &Simulator{
		server: New(cfg),
	}
}

// Start starts the simulator
func (s *Simulator) Start() error {
	return s.server.Start()
}

// Stop stops the simulator
func (s *Simulator) Stop() error {
	return s.server.Stop()
}

// Addr returns the actual address the server is listening on
func (s *Simulator) Addr() string {
	return s.server.config.Addr
}

// Port returns the actual port the server is listening on
func (s *Simulator) Port() int {
	return s.server.Port()
}

// Username returns the configured username
func (s *Simulator) Username() string {
	return s.server.config.Username
}

// Password returns the configured password
func (s *Simulator) Password() string {
	return s.server.config.Password
}

// SetScenario sets the scenario configuration for error injection
func (s *Simulator) SetScenario(sc *ScenarioConfig) {
	s.server.SetScenario(sc)
}

// GetDatastore returns the datastore
func (s *Simulator) GetDatastore() *Datastore {
	return s.server.GetDatastore()
}

// SetRunningConfig sets the initial running configuration
func (s *Simulator) SetRunningConfig(dev *openconfig.Device) {
	s.server.SetRunningConfig(dev)
}

// GetRunningConfig gets the current running configuration
func (s *Simulator) GetRunningConfig() *openconfig.Device {
	return s.server.GetRunningConfig()
}

// AssertVlanExists asserts that a VLAN with the given ID exists in running config
func (s *Simulator) AssertVlanExists(t *testing.T, vlanID uint16) {
	dev := s.GetRunningConfig()
	assert.NotNil(t, dev.Vlans)
	assert.NotNil(t, dev.Vlans.Vlan[vlanID])
}

// AssertVlanName asserts that a VLAN has the expected name
func (s *Simulator) AssertVlanName(t *testing.T, vlanID uint16, expectedName string) {
	dev := s.GetRunningConfig()
	assert.NotNil(t, dev.Vlans)
	assert.NotNil(t, dev.Vlans.Vlan[vlanID])
	assert.NotNil(t, dev.Vlans.Vlan[vlanID].Config)
	assert.Equal(t, expectedName, *dev.Vlans.Vlan[vlanID].Config.Name)
}

// AssertVlanCount asserts the number of VLANs in running config
func (s *Simulator) AssertVlanCount(t *testing.T, expected int) {
	dev := s.GetRunningConfig()
	if dev.Vlans == nil {
		assert.Equal(t, expected, 0)
		return
	}
	assert.Equal(t, expected, len(dev.Vlans.Vlan))
}
