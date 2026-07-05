// Package testsupport provides test-only assertion helpers for the netconfsim
// simulator. Keeping these out of the netconfsim core package lets the core be
// compiled into a standalone deployable binary without a testing/testify dependency.
//
// Assertions query the simulator's structured tree store via its Running* methods
// (no XML string parsing on the test side).
package testsupport

import (
	"testing"

	"github.com/leezesi/usmp/backend/simulator/netconfsim"
	"github.com/stretchr/testify/assert"
)

// Huawei model assertion methods for VLAN testing

// AssertHuaweiVlanExists asserts that a VLAN with the given ID exists using Huawei model parsing.
func AssertHuaweiVlanExists(t *testing.T, sim *netconfsim.Simulator, vlanID uint16) {
	t.Helper()
	vlans := sim.RunningHuaweiVLANs()
	assert.Contains(t, vlans, vlanID)
}

// AssertHuaweiVlanName asserts that a VLAN has the expected name using Huawei model parsing.
func AssertHuaweiVlanName(t *testing.T, sim *netconfsim.Simulator, vlanID uint16, expected string) {
	t.Helper()
	vlans := sim.RunningHuaweiVLANs()
	name, ok := vlans[vlanID]
	assert.True(t, ok)
	assert.Equal(t, expected, name)
}

// AssertHuaweiVlanCount asserts the total number of VLANs in running config using Huawei model parsing.
func AssertHuaweiVlanCount(t *testing.T, sim *netconfsim.Simulator, expected int) {
	t.Helper()
	vlans := sim.RunningHuaweiVLANs()
	assert.Len(t, vlans, expected)
}

// ============================================
// Full VLAN attribute assertion methods
// ============================================

// AssertHuaweiVlanDescription asserts the description of a VLAN.
func AssertHuaweiVlanDescription(t *testing.T, sim *netconfsim.Simulator, vlanID uint16, expected string) {
	t.Helper()
	vlans := sim.RunningHuaweiVLANsFull()
	vlan, ok := vlans[vlanID]
	assert.True(t, ok, "VLAN %d not found", vlanID)
	assert.Equal(t, expected, vlan.Description, "VLAN %d description mismatch", vlanID)
}

// AssertHuaweiVlanType asserts the type of a VLAN.
func AssertHuaweiVlanType(t *testing.T, sim *netconfsim.Simulator, vlanID uint16, expected int) {
	t.Helper()
	vlans := sim.RunningHuaweiVLANsFull()
	vlan, ok := vlans[vlanID]
	assert.True(t, ok, "VLAN %d not found", vlanID)
	assert.Equal(t, expected, vlan.Type, "VLAN %d type mismatch", vlanID)
}

// AssertHuaweiVlanAdminStatus asserts the admin status of a VLAN.
func AssertHuaweiVlanAdminStatus(t *testing.T, sim *netconfsim.Simulator, vlanID uint16, expected int) {
	t.Helper()
	vlans := sim.RunningHuaweiVLANsFull()
	vlan, ok := vlans[vlanID]
	assert.True(t, ok, "VLAN %d not found", vlanID)
	assert.Equal(t, expected, vlan.AdminStatus, "VLAN %d admin status mismatch", vlanID)
}

// AssertHuaweiVlanBroadcastDiscard asserts broadcast discard setting.
func AssertHuaweiVlanBroadcastDiscard(t *testing.T, sim *netconfsim.Simulator, vlanID uint16, expected int) {
	t.Helper()
	vlans := sim.RunningHuaweiVLANsFull()
	vlan, ok := vlans[vlanID]
	assert.True(t, ok, "VLAN %d not found", vlanID)
	assert.Equal(t, expected, vlan.BroadcastDiscard, "VLAN %d broadcast discard mismatch", vlanID)
}

// AssertHuaweiVlanUnknownMulticastDiscard asserts unknown multicast discard setting.
func AssertHuaweiVlanUnknownMulticastDiscard(t *testing.T, sim *netconfsim.Simulator, vlanID uint16, expected int) {
	t.Helper()
	vlans := sim.RunningHuaweiVLANsFull()
	vlan, ok := vlans[vlanID]
	assert.True(t, ok, "VLAN %d not found", vlanID)
	assert.Equal(t, expected, vlan.UnknownMulticastDiscard, "VLAN %d unknown multicast discard mismatch", vlanID)
}

// AssertHuaweiVlanMacLearning asserts MAC learning enable status.
func AssertHuaweiVlanMacLearning(t *testing.T, sim *netconfsim.Simulator, vlanID uint16, expected int) {
	t.Helper()
	vlans := sim.RunningHuaweiVLANsFull()
	vlan, ok := vlans[vlanID]
	assert.True(t, ok, "VLAN %d not found", vlanID)
	assert.Equal(t, expected, vlan.MacLearning, "VLAN %d MAC learning mismatch", vlanID)
}

// AssertHuaweiVlanMacAgingTime asserts MAC aging time.
func AssertHuaweiVlanMacAgingTime(t *testing.T, sim *netconfsim.Simulator, vlanID uint16, expected uint32) {
	t.Helper()
	vlans := sim.RunningHuaweiVLANsFull()
	vlan, ok := vlans[vlanID]
	assert.True(t, ok, "VLAN %d not found", vlanID)
	assert.Equal(t, expected, vlan.MacAgingTime, "VLAN %d MAC aging time mismatch", vlanID)
}

// AssertHuaweiVlanMemberPort asserts a VLAN's member port exists with the given
// access-type and tag-mode (VLAN 端口成员端到端断言).
func AssertHuaweiVlanMemberPort(t *testing.T, sim *netconfsim.Simulator, vlanID uint16, ifName string, accessType, tagMode int) {
	t.Helper()
	vlans := sim.RunningHuaweiVLANsFull()
	vlan, ok := vlans[vlanID]
	assert.True(t, ok, "VLAN %d not found", vlanID)
	for _, p := range vlan.MemberPorts {
		if p.InterfaceName == ifName {
			assert.Equal(t, accessType, p.AccessType, "VLAN %d port %s access-type", vlanID, ifName)
			assert.Equal(t, tagMode, p.TagMode, "VLAN %d port %s tag-mode", vlanID, ifName)
			return
		}
	}
	t.Errorf("VLAN %d member-port %q not found on device (got %d ports)", vlanID, ifName, len(vlan.MemberPorts))
}

// AssertHuaweiVlanStatisticEnable asserts statistic enable status.
func AssertHuaweiVlanStatisticEnable(t *testing.T, sim *netconfsim.Simulator, vlanID uint16, expected int) {
	t.Helper()
	vlans := sim.RunningHuaweiVLANsFull()
	vlan, ok := vlans[vlanID]
	assert.True(t, ok, "VLAN %d not found", vlanID)
	assert.Equal(t, expected, vlan.StatisticEnable, "VLAN %d statistic enable mismatch", vlanID)
}

// AssertHuaweiVlanStatisticDiscard asserts statistic discard status.
func AssertHuaweiVlanStatisticDiscard(t *testing.T, sim *netconfsim.Simulator, vlanID uint16, expected int) {
	t.Helper()
	vlans := sim.RunningHuaweiVLANsFull()
	vlan, ok := vlans[vlanID]
	assert.True(t, ok, "VLAN %d not found", vlanID)
	assert.Equal(t, expected, vlan.StatisticDiscard, "VLAN %d statistic discard mismatch", vlanID)
}

// AssertHuaweiVlanSuperVlan asserts super VLAN ID.
func AssertHuaweiVlanSuperVlan(t *testing.T, sim *netconfsim.Simulator, vlanID uint16, expectedSuperVlanID uint16) {
	t.Helper()
	vlans := sim.RunningHuaweiVLANsFull()
	vlan, ok := vlans[vlanID]
	assert.True(t, ok, "VLAN %d not found", vlanID)
	assert.NotNil(t, vlan.SuperVlan, "VLAN %d super VLAN should be set", vlanID)
	if vlan.SuperVlan != nil {
		assert.Equal(t, expectedSuperVlanID, *vlan.SuperVlan, "VLAN %d super VLAN mismatch", vlanID)
	}
}

// AssertHuaweiVlanUnkownUnicastDiscard asserts unknown unicast discard settings.
func AssertHuaweiVlanUnkownUnicastDiscard(t *testing.T, sim *netconfsim.Simulator, vlanID uint16, discard int, macLearningEnable int) {
	t.Helper()
	vlans := sim.RunningHuaweiVLANsFull()
	vlan, ok := vlans[vlanID]
	assert.True(t, ok, "VLAN %d not found", vlanID)
	assert.Equal(t, discard, vlan.UnkownUnicastDiscard.Discard, "VLAN %d unknown unicast discard mismatch", vlanID)
	assert.Equal(t, macLearningEnable, vlan.UnkownUnicastDiscard.MacLearningEnable, "VLAN %d unknown unicast MAC learning mismatch", vlanID)
}

// AssertHuaweiVlanSuppression asserts traffic suppression settings.
func AssertHuaweiVlanSuppression(t *testing.T, sim *netconfsim.Simulator, vlanID uint16, inbound int, outbound int) {
	t.Helper()
	vlans := sim.RunningHuaweiVLANsFull()
	vlan, ok := vlans[vlanID]
	assert.True(t, ok, "VLAN %d not found", vlanID)
	assert.Equal(t, inbound, vlan.Suppression.Inbound, "VLAN %d inbound suppression mismatch", vlanID)
	assert.Equal(t, outbound, vlan.Suppression.Outbound, "VLAN %d outbound suppression mismatch", vlanID)
}

// ============================================
// IFM Interface assertion methods
// ============================================

// AssertHuaweiInterfaceExists asserts that an interface with the given name exists.
func AssertHuaweiInterfaceExists(t *testing.T, sim *netconfsim.Simulator, ifaceName string) {
	t.Helper()
	ifaces := sim.RunningHuaweiInterfaces()
	assert.Contains(t, ifaces, ifaceName, "Interface %s not found", ifaceName)
}

// AssertHuaweiInterfaceDescription asserts interface description.
func AssertHuaweiInterfaceDescription(t *testing.T, sim *netconfsim.Simulator, ifaceName string, expected string) {
	t.Helper()
	ifaces := sim.RunningHuaweiInterfaces()
	iface, ok := ifaces[ifaceName]
	assert.True(t, ok, "Interface %s not found", ifaceName)
	assert.Equal(t, expected, iface.Description, "Interface %s description mismatch", ifaceName)
}

// AssertHuaweiInterfaceAdminStatus asserts interface admin status.
func AssertHuaweiInterfaceAdminStatus(t *testing.T, sim *netconfsim.Simulator, ifaceName string, expected int) {
	t.Helper()
	ifaces := sim.RunningHuaweiInterfaces()
	iface, ok := ifaces[ifaceName]
	assert.True(t, ok, "Interface %s not found", ifaceName)
	assert.Equal(t, expected, iface.AdminStatus, "Interface %s admin status mismatch", ifaceName)
}

// AssertHuaweiInterfaceMtu asserts interface MTU.
func AssertHuaweiInterfaceMtu(t *testing.T, sim *netconfsim.Simulator, ifaceName string, expected uint32) {
	t.Helper()
	ifaces := sim.RunningHuaweiInterfaces()
	iface, ok := ifaces[ifaceName]
	assert.True(t, ok, "Interface %s not found", ifaceName)
	assert.Equal(t, expected, iface.Mtu, "Interface %s MTU mismatch", ifaceName)
}

// AssertHuaweiInterfaceControlFlap asserts control-flap configuration.
func AssertHuaweiInterfaceControlFlap(t *testing.T, sim *netconfsim.Simulator, ifaceName string, ceiling uint32, decayNg uint32, decayOk uint32, reuse uint32, suppress uint32) {
	t.Helper()
	ifaces := sim.RunningHuaweiInterfaces()
	iface, ok := ifaces[ifaceName]
	assert.True(t, ok, "Interface %s not found", ifaceName)
	assert.Equal(t, ceiling, iface.ControlFlap.Ceiling, "Interface %s control-flap ceiling mismatch", ifaceName)
	assert.Equal(t, decayNg, iface.ControlFlap.DecayNg, "Interface %s control-flap decay-ng mismatch", ifaceName)
	assert.Equal(t, decayOk, iface.ControlFlap.DecayOk, "Interface %s control-flap decay-ok mismatch", ifaceName)
	assert.Equal(t, reuse, iface.ControlFlap.Reuse, "Interface %s control-flap reuse mismatch", ifaceName)
	assert.Equal(t, suppress, iface.ControlFlap.Suppress, "Interface %s control-flap suppress mismatch", ifaceName)
}

// AssertHuaweiInterfaceDampManual asserts damp manual configuration.
func AssertHuaweiInterfaceDampManual(t *testing.T, sim *netconfsim.Simulator, ifaceName string, halfLife uint16, maxSuppress uint16, reuse uint32, suppress uint32) {
	t.Helper()
	ifaces := sim.RunningHuaweiInterfaces()
	iface, ok := ifaces[ifaceName]
	assert.True(t, ok, "Interface %s not found", ifaceName)
	assert.Equal(t, halfLife, iface.Damp.Manual.HalfLifePeriod, "Interface %s damp manual half-life mismatch", ifaceName)
	assert.Equal(t, maxSuppress, iface.Damp.Manual.MaxSuppressTime, "Interface %s damp manual max-suppress mismatch", ifaceName)
	assert.Equal(t, reuse, iface.Damp.Manual.Reuse, "Interface %s damp manual reuse mismatch", ifaceName)
	assert.Equal(t, suppress, iface.Damp.Manual.Suppress, "Interface %s damp manual suppress mismatch", ifaceName)
}

// AssertHuaweiInterfaceTimers asserts interface timer configurations.
func AssertHuaweiInterfaceTimers(t *testing.T, sim *netconfsim.Simulator, ifaceName string, downDelay uint32, upDelay uint32) {
	t.Helper()
	ifaces := sim.RunningHuaweiInterfaces()
	iface, ok := ifaces[ifaceName]
	assert.True(t, ok, "Interface %s not found", ifaceName)
	assert.Equal(t, downDelay, iface.DownDelayTime, "Interface %s down-delay-time mismatch", ifaceName)
	assert.Equal(t, upDelay, iface.ProtocolUpDelayTime, "Interface %s protocol-up-delay-time mismatch", ifaceName)
}

// AssertHuaweiInterfaceFlags asserts interface boolean flag configurations.
func AssertHuaweiInterfaceFlags(t *testing.T, sim *netconfsim.Simulator, ifaceName string, clearIpDf bool, isL2Switch bool, l2ModeEnable bool, linkUpDownTrap bool, statisticEnable bool, spreadMtuFlag bool) {
	t.Helper()
	ifaces := sim.RunningHuaweiInterfaces()
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
func AssertHuaweiInterfaceStatistics(t *testing.T, sim *netconfsim.Simulator, ifaceName string, interval uint32, mode int) {
	t.Helper()
	ifaces := sim.RunningHuaweiInterfaces()
	iface, ok := ifaces[ifaceName]
	assert.True(t, ok, "Interface %s not found", ifaceName)
	assert.Equal(t, interval, iface.StatisticInterval, "Interface %s statistic-interval mismatch", ifaceName)
	assert.Equal(t, mode, iface.StatisticMode, "Interface %s statistic-mode mismatch", ifaceName)
}

// AssertHuaweiInterfaceNetwork asserts interface network configurations.
func AssertHuaweiInterfaceNetwork(t *testing.T, sim *netconfsim.Simulator, ifaceName string, macAddress string, vrfName string, vsName string) {
	t.Helper()
	ifaces := sim.RunningHuaweiInterfaces()
	iface, ok := ifaces[ifaceName]
	assert.True(t, ok, "Interface %s not found", ifaceName)
	assert.Equal(t, macAddress, iface.MacAddress, "Interface %s mac-address mismatch", ifaceName)
	assert.Equal(t, vrfName, iface.VrfName, "Interface %s vrf-name mismatch", ifaceName)
	assert.Equal(t, vsName, iface.VsName, "Interface %s vs-name mismatch", ifaceName)
}

// AssertHuaweiSystem asserts system configuration.
func AssertHuaweiSystem(t *testing.T, sim *netconfsim.Simulator, sysName string, sysContact string, sysLocation string) {
	t.Helper()
	sys := sim.RunningHuaweiSystem()
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
func AssertHuaweiSystemName(t *testing.T, sim *netconfsim.Simulator, expected string) {
	t.Helper()
	sys := sim.RunningHuaweiSystem()
	assert.Equal(t, expected, sys.SysName, "System name mismatch")
}

// ============================================
// OpenConfig interface assertion methods
// ============================================

// AssertInterfaceExists verifies that the interface with the given name exists in the running config.
func AssertInterfaceExists(t *testing.T, sim *netconfsim.Simulator, name string) {
	t.Helper()
	ifaces := sim.RunningOCInterfaces()
	_, exists := ifaces[name]
	assert.True(t, exists, "interface %q should exist in running config, but got: %v", name, ifaces)
}

// AssertInterfaceEnabled verifies that the interface exists and has the expected enabled state.
func AssertInterfaceEnabled(t *testing.T, sim *netconfsim.Simulator, name string, expected bool) {
	t.Helper()
	ifaces := sim.RunningOCInterfaces()
	iface, exists := ifaces[name]
	assert.True(t, exists, "interface %q should exist", name)
	if !exists {
		return
	}
	assert.NotNil(t, iface.Enabled, "interface %q should have Enabled field set", name)
	if iface.Enabled != nil {
		assert.Equal(t, expected, *iface.Enabled, "interface %q enabled state should match", name)
	}
}

// AssertInterfaceMtu verifies that the interface exists and has the expected MTU.
func AssertInterfaceMtu(t *testing.T, sim *netconfsim.Simulator, name string, expectedMtu uint16) {
	t.Helper()
	ifaces := sim.RunningOCInterfaces()
	iface, exists := ifaces[name]
	assert.True(t, exists, "interface %q should exist", name)
	if !exists {
		return
	}
	assert.NotNil(t, iface.Mtu, "interface %q should have Mtu field set", name)
	if iface.Mtu != nil {
		assert.Equal(t, expectedMtu, *iface.Mtu, "interface %q MTU should match", name)
	}
}
