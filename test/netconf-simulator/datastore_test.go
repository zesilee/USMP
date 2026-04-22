package netsim

import (
	"testing"

	"github.com/leezesi/usmp/internal/generated/openconfig"
	"github.com/stretchr/testify/assert"
)

func TestDatastore_New(t *testing.T) {
	ds := NewDatastore()
	assert.NotNil(t, ds.GetRunning())
	assert.NotNil(t, ds.GetCandidate())
}

func TestDatastore_SetGetRunning(t *testing.T) {
	ds := NewDatastore()

	dev := &openconfig.Device{}
	vlans := &openconfig.OpenconfigVlan_Vlans{}
	vlan, _ := vlans.NewVlan(100)
	vlan.Config = &openconfig.OpenconfigVlan_Vlans_Vlan_Config{}
	name := "VLAN100"
	vlan.Config.Name = &name
	dev.Vlans = vlans

	ds.SetRunning(dev)
	result := ds.GetRunning()
	assert.NotNil(t, result)
	assert.NotNil(t, result.Vlans)
	assert.NotNil(t, result.Vlans.Vlan[100])
	assert.Equal(t, "VLAN100", *result.Vlans.Vlan[100].Config.Name)
}

func TestDatastore_SetGetCandidate(t *testing.T) {
	ds := NewDatastore()

	dev := &openconfig.Device{}
	vlans := &openconfig.OpenconfigVlan_Vlans{}
	vlan, _ := vlans.NewVlan(200)
	vlan.Config = &openconfig.OpenconfigVlan_Vlans_Vlan_Config{}
	name := "VLAN200"
	vlan.Config.Name = &name
	dev.Vlans = vlans

	ds.SetCandidate(dev)
	result := ds.GetCandidate()
	assert.NotNil(t, result)
	assert.NotNil(t, result.Vlans)
	assert.NotNil(t, result.Vlans.Vlan[200])
	assert.Equal(t, "VLAN200", *result.Vlans.Vlan[200].Config.Name)
}

func TestDatastore_Commit(t *testing.T) {
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

func TestDatastore_RenderConfigXML(t *testing.T) {
	ds := NewDatastore()

	dev := &openconfig.Device{}
	vlans := &openconfig.OpenconfigVlan_Vlans{}
	vlan, _ := vlans.NewVlan(100)
	vlan.Config = &openconfig.OpenconfigVlan_Vlans_Vlan_Config{}
	name := "VLAN100"
	vlan.Config.Name = &name
	vlanID := uint16(100)
	vlan.Config.VlanId = &vlanID
	dev.Vlans = vlans

	xml, err := ds.RenderConfigXML(dev, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, xml)
	xmlStr := string(xml)
	assert.Contains(t, xmlStr, "vlans")
	assert.Contains(t, xmlStr, "vlan")
	assert.Contains(t, xmlStr, "100")
	assert.Contains(t, xmlStr, "VLAN100")
}
