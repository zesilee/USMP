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

// Huawei model assertion methods for VLAN testing

// AssertHuaweiVlanExists asserts that a VLAN with the given ID exists using Huawei model parsing.
func (s *Simulator) AssertHuaweiVlanExists(t *testing.T, vlanID uint16) {
	t.Helper()
	vlans, err := s.datastore.ExtractHuaweiVLANs()
	assert.NoError(t, err)
	assert.NotNil(t, vlans)
	assert.Contains(t, vlans, vlanID)
}

// AssertHuaweiVlanName asserts that a VLAN has the expected name using Huawei model parsing.
func (s *Simulator) AssertHuaweiVlanName(t *testing.T, vlanID uint16, expected string) {
	t.Helper()
	vlans, err := s.datastore.ExtractHuaweiVLANs()
	assert.NoError(t, err)
	name, ok := vlans[vlanID]
	assert.True(t, ok)
	assert.Equal(t, expected, name)
}

// AssertHuaweiVlanCount asserts the total number of VLANs in running config using Huawei model parsing.
func (s *Simulator) AssertHuaweiVlanCount(t *testing.T, expected int) {
	t.Helper()
	vlans, err := s.datastore.ExtractHuaweiVLANs()
	assert.NoError(t, err)
	if vlans == nil {
		assert.Equal(t, expected, 0)
		return
	}
	assert.Len(t, vlans, expected)
}

// ============================================
// Full VLAN attribute assertion methods
// ============================================

// AssertHuaweiVlanDescription asserts the description of a VLAN.
func (s *Simulator) AssertHuaweiVlanDescription(t *testing.T, vlanID uint16, expected string) {
	t.Helper()
	vlans, err := s.datastore.ExtractHuaweiVLANsFull()
	assert.NoError(t, err)
	vlan, ok := vlans[vlanID]
	assert.True(t, ok, "VLAN %d not found", vlanID)
	assert.Equal(t, expected, vlan.Description, "VLAN %d description mismatch", vlanID)
}

// AssertHuaweiVlanType asserts the type of a VLAN.
func (s *Simulator) AssertHuaweiVlanType(t *testing.T, vlanID uint16, expected int) {
	t.Helper()
	vlans, err := s.datastore.ExtractHuaweiVLANsFull()
	assert.NoError(t, err)
	vlan, ok := vlans[vlanID]
	assert.True(t, ok, "VLAN %d not found", vlanID)
	assert.Equal(t, expected, vlan.Type, "VLAN %d type mismatch", vlanID)
}

// AssertHuaweiVlanAdminStatus asserts the admin status of a VLAN.
func (s *Simulator) AssertHuaweiVlanAdminStatus(t *testing.T, vlanID uint16, expected int) {
	t.Helper()
	vlans, err := s.datastore.ExtractHuaweiVLANsFull()
	assert.NoError(t, err)
	vlan, ok := vlans[vlanID]
	assert.True(t, ok, "VLAN %d not found", vlanID)
	assert.Equal(t, expected, vlan.AdminStatus, "VLAN %d admin status mismatch", vlanID)
}

// AssertHuaweiVlanBroadcastDiscard asserts broadcast discard setting.
func (s *Simulator) AssertHuaweiVlanBroadcastDiscard(t *testing.T, vlanID uint16, expected int) {
	t.Helper()
	vlans, err := s.datastore.ExtractHuaweiVLANsFull()
	assert.NoError(t, err)
	vlan, ok := vlans[vlanID]
	assert.True(t, ok, "VLAN %d not found", vlanID)
	assert.Equal(t, expected, vlan.BroadcastDiscard, "VLAN %d broadcast discard mismatch", vlanID)
}

// AssertHuaweiVlanUnknownMulticastDiscard asserts unknown multicast discard setting.
func (s *Simulator) AssertHuaweiVlanUnknownMulticastDiscard(t *testing.T, vlanID uint16, expected int) {
	t.Helper()
	vlans, err := s.datastore.ExtractHuaweiVLANsFull()
	assert.NoError(t, err)
	vlan, ok := vlans[vlanID]
	assert.True(t, ok, "VLAN %d not found", vlanID)
	assert.Equal(t, expected, vlan.UnknownMulticastDiscard, "VLAN %d unknown multicast discard mismatch", vlanID)
}

// AssertHuaweiVlanMacLearning asserts MAC learning enable status.
func (s *Simulator) AssertHuaweiVlanMacLearning(t *testing.T, vlanID uint16, expected int) {
	t.Helper()
	vlans, err := s.datastore.ExtractHuaweiVLANsFull()
	assert.NoError(t, err)
	vlan, ok := vlans[vlanID]
	assert.True(t, ok, "VLAN %d not found", vlanID)
	assert.Equal(t, expected, vlan.MacLearning, "VLAN %d MAC learning mismatch", vlanID)
}

// AssertHuaweiVlanMacAgingTime asserts MAC aging time.
func (s *Simulator) AssertHuaweiVlanMacAgingTime(t *testing.T, vlanID uint16, expected uint32) {
	t.Helper()
	vlans, err := s.datastore.ExtractHuaweiVLANsFull()
	assert.NoError(t, err)
	vlan, ok := vlans[vlanID]
	assert.True(t, ok, "VLAN %d not found", vlanID)
	assert.Equal(t, expected, vlan.MacAgingTime, "VLAN %d MAC aging time mismatch", vlanID)
}

// AssertHuaweiVlanStatisticEnable asserts statistic enable status.
func (s *Simulator) AssertHuaweiVlanStatisticEnable(t *testing.T, vlanID uint16, expected int) {
	t.Helper()
	vlans, err := s.datastore.ExtractHuaweiVLANsFull()
	assert.NoError(t, err)
	vlan, ok := vlans[vlanID]
	assert.True(t, ok, "VLAN %d not found", vlanID)
	assert.Equal(t, expected, vlan.StatisticEnable, "VLAN %d statistic enable mismatch", vlanID)
}

// AssertHuaweiVlanStatisticDiscard asserts statistic discard status.
func (s *Simulator) AssertHuaweiVlanStatisticDiscard(t *testing.T, vlanID uint16, expected int) {
	t.Helper()
	vlans, err := s.datastore.ExtractHuaweiVLANsFull()
	assert.NoError(t, err)
	vlan, ok := vlans[vlanID]
	assert.True(t, ok, "VLAN %d not found", vlanID)
	assert.Equal(t, expected, vlan.StatisticDiscard, "VLAN %d statistic discard mismatch", vlanID)
}

// AssertHuaweiVlanSuperVlan asserts super VLAN ID.
func (s *Simulator) AssertHuaweiVlanSuperVlan(t *testing.T, vlanID uint16, expectedSuperVlanID uint16) {
	t.Helper()
	vlans, err := s.datastore.ExtractHuaweiVLANsFull()
	assert.NoError(t, err)
	vlan, ok := vlans[vlanID]
	assert.True(t, ok, "VLAN %d not found", vlanID)
	assert.NotNil(t, vlan.SuperVlan, "VLAN %d super VLAN should be set", vlanID)
	assert.Equal(t, expectedSuperVlanID, *vlan.SuperVlan, "VLAN %d super VLAN mismatch", vlanID)
}

// AssertHuaweiVlanUnkownUnicastDiscard asserts unknown unicast discard settings.
func (s *Simulator) AssertHuaweiVlanUnkownUnicastDiscard(t *testing.T, vlanID uint16, discard int, macLearningEnable int) {
	t.Helper()
	vlans, err := s.datastore.ExtractHuaweiVLANsFull()
	assert.NoError(t, err)
	vlan, ok := vlans[vlanID]
	assert.True(t, ok, "VLAN %d not found", vlanID)
	assert.Equal(t, discard, vlan.UnkownUnicastDiscard.Discard, "VLAN %d unknown unicast discard mismatch", vlanID)
	assert.Equal(t, macLearningEnable, vlan.UnkownUnicastDiscard.MacLearningEnable, "VLAN %d unknown unicast MAC learning mismatch", vlanID)
}

// AssertHuaweiVlanSuppression asserts traffic suppression settings.
func (s *Simulator) AssertHuaweiVlanSuppression(t *testing.T, vlanID uint16, inbound int, outbound int) {
	t.Helper()
	vlans, err := s.datastore.ExtractHuaweiVLANsFull()
	assert.NoError(t, err)
	vlan, ok := vlans[vlanID]
	assert.True(t, ok, "VLAN %d not found", vlanID)
	assert.Equal(t, inbound, vlan.Suppression.Inbound, "VLAN %d inbound suppression mismatch", vlanID)
	assert.Equal(t, outbound, vlan.Suppression.Outbound, "VLAN %d outbound suppression mismatch", vlanID)
}

// Force import to satisfy Go's unused check
var _ openconfig.OpenconfigVlan_Vlans

