package netsim

import (
	"testing"

	"github.com/scrapli/scrapligo/driver/netconf"
	"github.com/scrapligo/scrapligo/driver/options"
	"github.com/stretchr/testify/assert"

	"github.com/leezesi/usmp/internal/generated/openconfig"
)

// TestE2E_FullWorkflow tests the full NETCONF workflow:
// 1. Start simulator
// 2. Connect with scrapligo NETCONF client
// 3. edit-config to add VLANs
// 4. commit
// 5. get-config and verify the result
func TestE2E_FullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Start simulator with random port
	sim := NewSimulator()
	err := sim.Start()
	assert.NoError(t, err)
	defer sim.Stop()

	// 2. Connect with NETCONF client
	connectErr := make(chan error, 1)
	var c *netconf.Driver
	go func() {
		var err error
		c, err = netconf.Connect(
			sim.Addr(),
			sim.Port(),
			sim.Username(),
			sim.Password(),
			options.WithTransportTimeout(5),
		)
		connectErr <- err
	}()

	// Wait for connection
	err = <-connectErr
	assert.NoError(t, err)
	defer c.Close()

	// 3. edit-config to add a new VLAN
	config := `<config>
  <vlans xmlns="http://openconfig.net/yang/vlan">
    <vlan>
      <vlan-id>100</vlan-id>
      <config>
        <vlan-id>100</vlan-id>
        <name>TestVLAN100</name>
      </config>
    </vlan>
    <vlan>
      <vlan-id>200</vlan-id>
      <config>
        <vlan-id>200</vlan-id>
        <name>TestVLAN200</name>
      </config>
    </vlan>
  </vlans>
</config>`

	// Edit candidate config
	result, err := c.EditConfig(
		netconf.WithTargetCandidate(),
		netconf.WithDefaultOperationMerge(),
		netconf.WithConfig(config),
	)
	assert.NoError(t, err)
	assert.True(t, result.IsOk())

	// 4. Commit
	result, err = c.Commit()
	assert.NoError(t, err)
	assert.True(t, result.IsOk())

	// 5. Verify in simulator
	sim.AssertVlanCount(t, 2)
	sim.AssertVlanExists(t, 100)
	sim.AssertVlanExists(t, 200)
	sim.AssertVlanName(t, 100, "TestVLAN100")
	sim.AssertVlanName(t, 200, "TestVLAN200")

	// 6. Get-config from device and verify
	result, err = c.GetConfig(
		netconf.WithSourceRunning(),
		netconf.WithFilterSubtree(`<vlans xmlns="http://openconfig.net/yang/vlan"/>`),
	)
	assert.NoError(t, err)
	assert.True(t, result.IsOk())
	assert.Contains(t, string(result.Data), "TestVLAN100")
	assert.Contains(t, string(result.Data), "TestVLAN200")
}

// TestE2E_GetConfig tests get-config with initial configuration
func TestE2E_GetConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Start simulator
	sim := NewSimulator()
	err := sim.Start()
	assert.NoError(t, err)
	defer sim.Stop()

	// 2. Set initial config
	dev := &openconfig.Device{}
	vlans := &openconfig.OpenconfigVlan_Vlans{}
	vlan100, _ := vlans.NewVlan(100)
	vlan100.Config = &openconfig.OpenconfigVlan_Vlans_Vlan_Config{}
	name := "Preconfigured"
	vlan100.Config.Name = &name
	id := uint16(100)
	vlan100.Config.VlanId = &id
	dev.Vlans = vlans
	sim.SetRunningConfig(dev)

	// 3. Connect and get-config
	c, err := netconf.Connect(
		sim.Addr(),
		sim.Port(),
		sim.Username(),
		sim.Password(),
		options.WithTransportTimeout(5),
	)
	assert.NoError(t, err)
	defer c.Close()

	result, err := c.GetConfig(
		netconf.WithSourceRunning(),
		netconf.WithFilterSubtree(`<vlans xmlns="http://openconfig.net/yang/vlan"/>`),
	)
	assert.NoError(t, err)
	assert.True(t, result.IsOk())
	assert.Contains(t, string(result.Data), "Preconfigured")
	assert.Contains(t, string(result.Data), "100")
}

// TestE2E_AuthenticationFailure tests rejected authentication
func TestE2E_AuthenticationFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	sim := NewSimulator()
	sc := NewScenarioConfig()
	sc.RejectAuth = true
	sim.SetScenario(sc)
	err := sim.Start()
	assert.NoError(t, err)
	defer sim.Stop()

	// Try to connect - should fail
	c, err := netconf.Connect(
		sim.Addr(),
		sim.Port(),
		"wrong",
		"wrong",
		options.WithTransportTimeout(2),
	)
	assert.Error(t, err)
	assert.Nil(t, c)
}
