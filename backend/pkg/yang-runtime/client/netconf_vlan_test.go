package client

import (
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildHuaweiVlanVlansXML_NilInput(t *testing.T) {
	result, err := buildHuaweiVlanVlansXML(nil)
	require.NoError(t, err)
	assert.Equal(t, `<vlan xmlns="urn:huawei:params:xml:ns:yang:huawei-vlan"><vlans/></vlan>`, result)
}

func TestBuildHuaweiVlanVlansXML_EmptyInput(t *testing.T) {
	vlans := &huawei.HuaweiVlan_Vlan_Vlans{}
	result, err := buildHuaweiVlanVlansXML(vlans)
	require.NoError(t, err)
	assert.Equal(t, `<vlan xmlns="urn:huawei:params:xml:ns:yang:huawei-vlan"><vlans/></vlan>`, result)
}

func TestBuildHuaweiVlanVlansXML_SingleVlan(t *testing.T) {
	vlans := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: make(map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan),
	}

	vlanID := uint16(100)
	name := "TestVLAN"
	desc := "Test Description"

	vlans.Vlan[vlanID] = &huawei.HuaweiVlan_Vlan_Vlans_Vlan{
		Id:               &vlanID,
		Name:             &name,
		Description:      &desc,
		AdminStatus:      huawei.E_HuaweiVlan_AdminStatus(2),  // up
		Type:             huawei.E_HuaweiVlan_VlanType(1),     // common
		BroadcastDiscard: huawei.E_HuaweiVlan_EnableStatus(2), // disable
		MacLearning:      huawei.E_HuaweiVlan_EnableStatus(1), // enable
		StatisticEnable:  huawei.E_HuaweiVlan_EnableStatus(1), // enable
	}

	result, err := buildHuaweiVlanVlansXML(vlans)
	require.NoError(t, err)

	// Verify the result contains expected elements
	assert.Contains(t, result, `<vlan xmlns="urn:huawei:params:xml:ns:yang:huawei-vlan"><vlans>`)
	assert.Contains(t, result, `<vlan>`)
	assert.Contains(t, result, `<id>100</id>`)
	assert.Contains(t, result, `<name>TestVLAN</name>`)
	assert.Contains(t, result, `<description>Test Description</description>`)
	assert.Contains(t, result, `<admin-status>2</admin-status>`)
	assert.Contains(t, result, `<type>1</type>`)
	assert.Contains(t, result, `<broadcast-discard>2</broadcast-discard>`)
	assert.Contains(t, result, `<mac-learning>1</mac-learning>`)
	assert.Contains(t, result, `<statistic-enable>1</statistic-enable>`)
	assert.Contains(t, result, `</vlan>`)
	assert.Contains(t, result, `</vlans>`)
}

func TestBuildHuaweiVlanVlansXML_MultipleVlans(t *testing.T) {
	vlans := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: make(map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan),
	}

	id100 := uint16(100)
	name100 := "VLAN100"
	id200 := uint16(200)
	name200 := "VLAN200"

	vlans.Vlan[id100] = &huawei.HuaweiVlan_Vlan_Vlans_Vlan{Id: &id100, Name: &name100}
	vlans.Vlan[id200] = &huawei.HuaweiVlan_Vlan_Vlans_Vlan{Id: &id200, Name: &name200}

	result, err := buildHuaweiVlanVlansXML(vlans)
	require.NoError(t, err)

	assert.Contains(t, result, `<id>100</id>`)
	assert.Contains(t, result, `<name>VLAN100</name>`)
	assert.Contains(t, result, `<id>200</id>`)
	assert.Contains(t, result, `<name>VLAN200</name>`)
}

func TestBuildHuaweiVlanVlansXML_WithMemberPorts(t *testing.T) {
	vlans := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: make(map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan),
	}

	vlanID := uint16(100)
	ifName := "GigabitEthernet0/0/1"

	vlans.Vlan[vlanID] = &huawei.HuaweiVlan_Vlan_Vlans_Vlan{
		Id: &vlanID,
		MemberPorts: &huawei.HuaweiVlan_Vlan_Vlans_Vlan_MemberPorts{
			MemberPort: make(map[string]*huawei.HuaweiVlan_Vlan_Vlans_Vlan_MemberPorts_MemberPort),
		},
	}

	vlans.Vlan[vlanID].MemberPorts.MemberPort[ifName] = &huawei.HuaweiVlan_Vlan_Vlans_Vlan_MemberPorts_MemberPort{
		InterfaceName: &ifName,
		AccessType:    huawei.E_HuaweiVlan_AccessType(1), // tagged
		TagMode:       huawei.E_HuaweiVlan_TagMode(1),    // untagged
	}

	result, err := buildHuaweiVlanVlansXML(vlans)
	require.NoError(t, err)

	assert.Contains(t, result, `<member-ports>`)
	assert.Contains(t, result, `<member-port>`)
	assert.Contains(t, result, `<interface-name>GigabitEthernet0/0/1</interface-name>`)
	assert.Contains(t, result, `<access-type>1</access-type>`)
	assert.Contains(t, result, `<tag-mode>1</tag-mode>`)
	assert.Contains(t, result, `</member-port>`)
	assert.Contains(t, result, `</member-ports>`)
}

func TestBuildHuaweiVlanVlansXML_XMLEscaping(t *testing.T) {
	vlans := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: make(map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan),
	}

	vlanID := uint16(100)
	name := `VLAN<>&'"Test`
	desc := `Description <special> chars & test`

	vlans.Vlan[vlanID] = &huawei.HuaweiVlan_Vlan_Vlans_Vlan{
		Id:          &vlanID,
		Name:        &name,
		Description: &desc,
	}

	result, err := buildHuaweiVlanVlansXML(vlans)
	require.NoError(t, err)

	// Verify special characters are escaped
	assert.Contains(t, result, `VLAN&lt;&gt;&amp;&apos;&quot;Test`)
	assert.Contains(t, result, `Description &lt;special&gt; chars &amp; test`)
}

func TestMarshalChange_HuaweiVlan(t *testing.T) {
	vlans := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: make(map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan),
	}

	vlanID := uint16(100)
	name := "TestVLAN"
	vlans.Vlan[vlanID] = &huawei.HuaweiVlan_Vlan_Vlans_Vlan{
		Id:   &vlanID,
		Name: &name,
	}

	change := Change{
		NewValue: vlans,
	}

	nc := &NETCONFClient{}
	result, err := nc.marshalChange(change)
	require.NoError(t, err)

	assert.Contains(t, result, `<vlan xmlns="urn:huawei:params:xml:ns:yang:huawei-vlan"><vlans>`)
	assert.Contains(t, result, `<id>100</id>`)
	assert.Contains(t, result, `<name>TestVLAN</name>`)
}
