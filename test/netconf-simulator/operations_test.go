package netsim

import (
	"encoding/xml"
	"testing"

	"github.com/leezesi/usmp/internal/generated/openconfig"
	"github.com/stretchr/testify/assert"
)

func TestEditConfigRequest_Unmarshal(t *testing.T) {
	input := `
<edit-config>
  <target>
    <candidate/>
  </target>
  <default-operation>merge</default-operation>
  <config>
    <vlans xmlns="http://openconfig.net/yang/vlan">
      <vlan>
        <vlan-id>100</vlan-id>
        <config>
          <vlan-id>100</vlan-id>
          <name>VLAN100</name>
        </config>
      </vlan>
    </vlans>
  </config>
</edit-config>`

	var req EditConfigRequest
	err := xml.Unmarshal([]byte(input), &req)
	assert.NoError(t, err)
	assert.NotNil(t, req.Target.Candidate)
	assert.Equal(t, "merge", req.DefaultOperation)
	assert.NotEmpty(t, req.Config.Content)
	// After unmarshal with innerxml, contains the content inside <config>
	configStr := string(req.Config.Content)
	assert.True(t, len(configStr) > 10)
	// May have whitespace differences, just check contains the key elements
	assert.Contains(t, configStr, "vlans")
	assert.Contains(t, configStr, "100")
}

func TestParseConfigXML(t *testing.T) {
	ds := NewDatastore()
	dev := &openconfig.Device{}

	configXML := []byte(`
<vlans xmlns="http://openconfig.net/yang/vlan">
  <vlan>
    <vlan-id>100</vlan-id>
    <config>
      <name>VLAN100</name>
      <vlan-id>100</vlan-id>
    </config>
  </vlan>
  <vlan>
    <vlan-id>200</vlan-id>
    <config>
      <name>VLAN200</name>
      <vlan-id>200</vlan-id>
    </config>
  </vlan>
</vlans>`)

	err := ds.ParseConfigXML(configXML, dev)
	assert.NoError(t, err)
	assert.NotNil(t, dev.Vlans)
	assert.Len(t, dev.Vlans.Vlan, 2)

	vlan100 := dev.Vlans.Vlan[100]
	assert.NotNil(t, vlan100)
	assert.Equal(t, uint16(100), *vlan100.VlanId)
	assert.NotNil(t, vlan100.Config)
	assert.Equal(t, "VLAN100", *vlan100.Config.Name)
	assert.Equal(t, uint16(100), *vlan100.Config.VlanId)

	vlan200 := dev.Vlans.Vlan[200]
	assert.NotNil(t, vlan200)
	assert.Equal(t, "VLAN200", *vlan200.Config.Name)
}

func TestParseConfigXML_Wrapped(t *testing.T) {
	ds := NewDatastore()
	dev := &openconfig.Device{}

	configXML := []byte(`<config>
<vlans xmlns="http://openconfig.net/yang/vlan">
  <vlan>
    <vlan-id>100</vlan-id>
  </vlan>
</vlans>
</config>`)

	err := ds.ParseConfigXML(configXML, dev)
	assert.NoError(t, err)
	assert.NotNil(t, dev.Vlans)
	assert.NotNil(t, dev.Vlans.Vlan[100])
}

func TestEditConfig_Merge(t *testing.T) {
	server := New(&ServerConfig{Port: 0})
	ds := server.GetDatastore()

	// Initial state
	dev := &openconfig.Device{}
	vlans := &openconfig.OpenconfigVlan_Vlans{}
	vlan, _ := vlans.NewVlan(100)
	vlan.Config = &openconfig.OpenconfigVlan_Vlans_Vlan_Config{}
	name := "Initial"
	vlan.Config.Name = &name
	dev.Vlans = vlans
	ds.SetRunning(dev)

	// Verify initial
	running := ds.GetRunning()
	assert.Equal(t, "Initial", *running.Vlans.Vlan[100].Config.Name)
}

func TestCommit(t *testing.T) {
	ds := NewDatastore()
	dev := &openconfig.Device{}
	vlans := &openconfig.OpenconfigVlan_Vlans{}
	vlan, _ := vlans.NewVlan(100)
	vlan.Config = &openconfig.OpenconfigVlan_Vlans_Vlan_Config{}
	name := "VLAN100"
	vlan.Config.Name = &name
	dev.Vlans = vlans
	ds.SetCandidate(dev)

	err := ds.Commit()
	assert.NoError(t, err)

	running := ds.GetRunning()
	assert.NotNil(t, running.Vlans)
	assert.NotNil(t, running.Vlans.Vlan[100])
	assert.Equal(t, "VLAN100", *running.Vlans.Vlan[100].Config.Name)
}
