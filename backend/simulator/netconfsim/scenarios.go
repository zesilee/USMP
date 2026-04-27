package netconfsim

import (
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/openconfig"
	"github.com/stretchr/testify/assert"
)

// ScenarioConfig configures fault injection for testing.
type ScenarioConfig struct {
	// ErrorOnRPC injects an error for specific RPC operations (keys: "get-config", "edit-config", "commit").
	ErrorOnRPC map[string]error
	// RejectAuth causes all authentication attempts to be rejected.
	RejectAuth bool
}

// NewScenarioConfig creates a new scenario with defaults (no errors).
func NewScenarioConfig() *ScenarioConfig {
	return &ScenarioConfig{
		ErrorOnRPC: make(map[string]error),
	}
}

// AssertVlanExists asserts that a VLAN with the given ID exists in running config.
func (s *Simulator) AssertVlanExists(t *testing.T, vlanID uint16) {
	t.Helper()
	vlans, err := s.datastore.ExtractVLANs()
	assert.NoError(t, err)
	assert.NotNil(t, vlans)
	assert.Contains(t, vlans.Vlan, vlanID)
}

// AssertVlanName asserts that a VLAN has the expected name.
func (s *Simulator) AssertVlanName(t *testing.T, vlanID uint16, expected string) {
	t.Helper()
	vlans, err := s.datastore.ExtractVLANs()
	assert.NoError(t, err)
	vlan, ok := vlans.Vlan[vlanID]
	assert.True(t, ok)
	assert.NotNil(t, vlan.Config)
	assert.Equal(t, expected, *vlan.Config.Name)
}

// AssertVlanCount asserts the total number of VLANs in running config.
func (s *Simulator) AssertVlanCount(t *testing.T, expected int) {
	t.Helper()
	vlans, err := s.datastore.ExtractVLANs()
	assert.NoError(t, err)
	if vlans == nil {
		assert.Equal(t, expected, 0)
		return
	}
	assert.Len(t, vlans.Vlan, expected)
}

// Force import to satisfy Go's unused check
var _ openconfig.OpenconfigVlan_Vlans

