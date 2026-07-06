package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 设备 get-config 回读的 VLAN XML 必须能解析进 ygot map（key=uint16 VLAN id）。
// 与 IFM 同源：ygot 结构体无 xml tag，encoding/xml 无法填 map → actual 恒空 → VLAN 永久漂移。
func TestParseHuaweiVlanVlansXML_Flat(t *testing.T) {
	data := []byte(`<data><vlans xmlns="urn:huawei:params:xml:ns:yang:huawei-vlan">` +
		`<vlan><id>100</id><name>v100</name><description>uplink</description>` +
		`<admin-status>2</admin-status><type>1</type><mac-learning>1</mac-learning></vlan>` +
		`</vlans></data>`)

	vlans, err := ParseHuaweiVlanVlansXML(data)
	require.NoError(t, err)
	require.Contains(t, vlans.Vlan, uint16(100))
	v := vlans.Vlan[100]
	require.NotNil(t, v.Id)
	assert.Equal(t, uint16(100), *v.Id)
	require.NotNil(t, v.Name)
	assert.Equal(t, "v100", *v.Name)
	require.NotNil(t, v.Description)
	assert.Equal(t, "uplink", *v.Description)
	assert.Equal(t, int64(2), int64(v.AdminStatus))
	assert.Equal(t, int64(1), int64(v.Type))
	assert.Equal(t, int64(1), int64(v.MacLearning))
}

func TestParseHuaweiVlanVlansXML_MemberPorts(t *testing.T) {
	data := []byte(`<vlans xmlns="urn:huawei:params:xml:ns:yang:huawei-vlan">` +
		`<vlan><id>200</id><member-ports>` +
		`<member-port><interface-name>GE0/0/1</interface-name><access-type>2</access-type></member-port>` +
		`<member-port><interface-name>GE0/0/2</interface-name></member-port>` +
		`</member-ports></vlan></vlans>`)

	vlans, err := ParseHuaweiVlanVlansXML(data)
	require.NoError(t, err)
	v := vlans.Vlan[200]
	require.NotNil(t, v)
	require.NotNil(t, v.MemberPorts)
	require.Contains(t, v.MemberPorts.MemberPort, "GE0/0/1")
	require.Contains(t, v.MemberPorts.MemberPort, "GE0/0/2")
	assert.Equal(t, int64(2), int64(v.MemberPorts.MemberPort["GE0/0/1"].AccessType))
}

func TestParseHuaweiVlanVlansXML_Empty(t *testing.T) {
	for _, data := range [][]byte{nil, []byte(``), []byte(`<data></data>`)} {
		vlans, err := ParseHuaweiVlanVlansXML(data)
		require.NoError(t, err)
		require.NotNil(t, vlans)
		assert.Empty(t, vlans.Vlan)
	}
}
