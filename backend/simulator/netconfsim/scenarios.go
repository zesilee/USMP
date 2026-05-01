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

// ============================================
// IFM Interface assertion methods
// ============================================

// AssertHuaweiInterfaceExists asserts that an interface with the given name exists.
func (s *Simulator) AssertHuaweiInterfaceExists(t *testing.T, ifaceName string) {
	t.Helper()
	ifaces, err := s.datastore.ExtractHuaweiInterfaces()
	assert.NoError(t, err)
	assert.Contains(t, ifaces, ifaceName, "Interface %s not found", ifaceName)
}

// AssertHuaweiInterfaceDescription asserts interface description.
func (s *Simulator) AssertHuaweiInterfaceDescription(t *testing.T, ifaceName string, expected string) {
	t.Helper()
	ifaces, err := s.datastore.ExtractHuaweiInterfaces()
	assert.NoError(t, err)
	iface, ok := ifaces[ifaceName]
	assert.True(t, ok, "Interface %s not found", ifaceName)
	assert.Equal(t, expected, iface.Description, "Interface %s description mismatch", ifaceName)
}

// AssertHuaweiInterfaceAdminStatus asserts interface admin status.
func (s *Simulator) AssertHuaweiInterfaceAdminStatus(t *testing.T, ifaceName string, expected int) {
	t.Helper()
	ifaces, err := s.datastore.ExtractHuaweiInterfaces()
	assert.NoError(t, err)
	iface, ok := ifaces[ifaceName]
	assert.True(t, ok, "Interface %s not found", ifaceName)
	assert.Equal(t, expected, iface.AdminStatus, "Interface %s admin status mismatch", ifaceName)
}

// AssertHuaweiInterfaceMtu asserts interface MTU.
func (s *Simulator) AssertHuaweiInterfaceMtu(t *testing.T, ifaceName string, expected uint32) {
	t.Helper()
	ifaces, err := s.datastore.ExtractHuaweiInterfaces()
	assert.NoError(t, err)
	iface, ok := ifaces[ifaceName]
	assert.True(t, ok, "Interface %s not found", ifaceName)
	assert.Equal(t, expected, iface.Mtu, "Interface %s MTU mismatch", ifaceName)
}

// AssertHuaweiInterfaceControlFlap asserts control-flap configuration.
func (s *Simulator) AssertHuaweiInterfaceControlFlap(t *testing.T, ifaceName string, ceiling uint32, decayNg uint32, decayOk uint32, reuse uint32, suppress uint32) {
	t.Helper()
	ifaces, err := s.datastore.ExtractHuaweiInterfaces()
	assert.NoError(t, err)
	iface, ok := ifaces[ifaceName]
	assert.True(t, ok, "Interface %s not found", ifaceName)
	assert.Equal(t, ceiling, iface.ControlFlap.Ceiling, "Interface %s control-flap ceiling mismatch", ifaceName)
	assert.Equal(t, decayNg, iface.ControlFlap.DecayNg, "Interface %s control-flap decay-ng mismatch", ifaceName)
	assert.Equal(t, decayOk, iface.ControlFlap.DecayOk, "Interface %s control-flap decay-ok mismatch", ifaceName)
	assert.Equal(t, reuse, iface.ControlFlap.Reuse, "Interface %s control-flap reuse mismatch", ifaceName)
	assert.Equal(t, suppress, iface.ControlFlap.Suppress, "Interface %s control-flap suppress mismatch", ifaceName)
}

// AssertHuaweiInterfaceDampManual asserts damp manual configuration.
func (s *Simulator) AssertHuaweiInterfaceDampManual(t *testing.T, ifaceName string, halfLife uint16, maxSuppress uint16, reuse uint32, suppress uint32) {
	t.Helper()
	ifaces, err := s.datastore.ExtractHuaweiInterfaces()
	assert.NoError(t, err)
	iface, ok := ifaces[ifaceName]
	assert.True(t, ok, "Interface %s not found", ifaceName)
	assert.Equal(t, halfLife, iface.Damp.Manual.HalfLifePeriod, "Interface %s damp manual half-life mismatch", ifaceName)
	assert.Equal(t, maxSuppress, iface.Damp.Manual.MaxSuppressTime, "Interface %s damp manual max-suppress mismatch", ifaceName)
	assert.Equal(t, reuse, iface.Damp.Manual.Reuse, "Interface %s damp manual reuse mismatch", ifaceName)
	assert.Equal(t, suppress, iface.Damp.Manual.Suppress, "Interface %s damp manual suppress mismatch", ifaceName)
}

// AssertHuaweiInterfaceTimers asserts interface timer configurations.
func (s *Simulator) AssertHuaweiInterfaceTimers(t *testing.T, ifaceName string, downDelay uint32, upDelay uint32) {
	t.Helper()
	ifaces, err := s.datastore.ExtractHuaweiInterfaces()
	assert.NoError(t, err)
	iface, ok := ifaces[ifaceName]
	assert.True(t, ok, "Interface %s not found", ifaceName)
	assert.Equal(t, downDelay, iface.DownDelayTime, "Interface %s down-delay-time mismatch", ifaceName)
	assert.Equal(t, upDelay, iface.ProtocolUpDelayTime, "Interface %s protocol-up-delay-time mismatch", ifaceName)
}

// AssertHuaweiInterfaceFlags asserts interface boolean flag configurations.
func (s *Simulator) AssertHuaweiInterfaceFlags(t *testing.T, ifaceName string, clearIpDf bool, isL2Switch bool, l2ModeEnable bool, linkUpDownTrap bool, statisticEnable bool, spreadMtuFlag bool) {
	t.Helper()
	ifaces, err := s.datastore.ExtractHuaweiInterfaces()
	assert.NoError(t, err)
	iface, ok := ifaces[ifaceName]
	assert.True(t, ok, "Interface %s not found", ifaceName)
	assert.Equal(t, clearIpDf, iface.ClearIpDf, "Interface %s clear-ip-df mismatch", ifaceName)
	assert.Equal(t, isL2Switch, iface.IsL2Switch, "Interface %s is-l2-switch mismatch", ifaceName)
	assert.Equal(t, l2ModeEnable, iface.L2ModeEnable, "Interface %s l2-mode-enable mismatch", ifaceName)
	assert.Equal(t, linkUpDownTrap, iface.LinkUpDownTrapEnable, "Interface %s link-up-down-trap-enable mismatch", ifaceName)
	assert.Equal(t, statisticEnable, iface.StatisticEnable, "Interface %s statistic-enable mismatch", ifaceName)
	assert.Equal(t, spreadMtuFlag, iface.SpreadMtuFlag, "Interface %s spread-mtu-flag mismatch", ifaceName)
}

// AssertHuaweiInterfaceStatistics asserts interface statistic configurations.
func (s *Simulator) AssertHuaweiInterfaceStatistics(t *testing.T, ifaceName string, interval uint32, mode int) {
	t.Helper()
	ifaces, err := s.datastore.ExtractHuaweiInterfaces()
	assert.NoError(t, err)
	iface, ok := ifaces[ifaceName]
	assert.True(t, ok, "Interface %s not found", ifaceName)
	assert.Equal(t, interval, iface.StatisticInterval, "Interface %s statistic-interval mismatch", ifaceName)
	assert.Equal(t, mode, iface.StatisticMode, "Interface %s statistic-mode mismatch", ifaceName)
}

// AssertHuaweiInterfaceNetwork asserts interface network configurations.
func (s *Simulator) AssertHuaweiInterfaceNetwork(t *testing.T, ifaceName string, macAddress string, vrfName string, vsName string) {
	t.Helper()
	ifaces, err := s.datastore.ExtractHuaweiInterfaces()
	assert.NoError(t, err)
	iface, ok := ifaces[ifaceName]
	assert.True(t, ok, "Interface %s not found", ifaceName)
	assert.Equal(t, macAddress, iface.MacAddress, "Interface %s mac-address mismatch", ifaceName)
	assert.Equal(t, vrfName, iface.VrfName, "Interface %s vrf-name mismatch", ifaceName)
	assert.Equal(t, vsName, iface.VsName, "Interface %s vs-name mismatch", ifaceName)
}

// AssertHuaweiSystem asserts system configuration.
func (s *Simulator) AssertHuaweiSystem(t *testing.T, sysName string, sysContact string, sysLocation string) {
	t.Helper()
	sys, err := s.datastore.ExtractHuaweiSystem()
	assert.NoError(t, err)
	if sysName != "" {
		assert.Equal(t, sysName, sys.SysName, "System name mismatch")
	}
	if sysContact != "" {
		assert.Equal(t, sysContact, sys.SysContact, "System contact mismatch")
	}
	if sysLocation != "" {
		assert.Equal(t, sysLocation, sys.SysLocation, "System location mismatch")
	}
}

// AssertHuaweiSystemName asserts system name.
func (s *Simulator) AssertHuaweiSystemName(t *testing.T, expected string) {
	t.Helper()
	sys, err := s.datastore.ExtractHuaweiSystem()
	assert.NoError(t, err)
	assert.Equal(t, expected, sys.SysName, "System name mismatch")
}

// Force import to satisfy Go's unused check
var _ openconfig.OpenconfigVlan_Vlans

