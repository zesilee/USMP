package api

import (
	"encoding/json"
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/stretchr/testify/assert"
)

func TestConvertToTypedStruct_HuaweiIfm(t *testing.T) {
	// Use JSON to properly unmarshal to map[string]interface{}
	// This simulates actual API request flow
	jsonStr := `{
			"interface": [
				{
					"name": "GigabitEthernet0/0/1",
					"description": "Test",
					"admin-status": 2,
					"mtu": 1500
				}
			]
		}`

	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &data)
	assert.NoError(t, err)

	t.Logf("data: %+v", data)

	result, err := convertToTypedStruct("/ifm:ifm/ifm:interfaces", data)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	ifm, ok := result.(*huawei.HuaweiIfm_Ifm_Interfaces)
	assert.True(t, ok, "Should be HuaweiIfm_Ifm_Interfaces")
	t.Logf("ifm.Interface: %+v", ifm.Interface)
	assert.Len(t, ifm.Interface, 1)

	iface := ifm.Interface["GigabitEthernet0/0/1"]
	assert.NotNil(t, iface)
	assert.Equal(t, "GigabitEthernet0/0/1", *iface.Name)
	assert.Equal(t, "Test", *iface.Description)
	assert.Equal(t, huawei.E_HuaweiIfm_PortStatus(2), iface.AdminStatus)
	assert.Equal(t, uint32(1500), *iface.Mtu)
}

func TestConvertToTypedStruct_HuaweiVlan(t *testing.T) {
	jsonStr := `{
		"vlans": [
			{
				"id": 10,
				"name": "VLAN10",
				"description": "Test VLAN",
				"type": 1,
				"admin-status": 1,
				"broadcast-discard": 1,
				"unknown-multicast-discard": 0,
				"mac-learning": 1,
				"mac-aging-time": 300,
				"statistic-enable": 0,
				"super-vlan": 100
			}
		]
	}`

	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &data)
	assert.NoError(t, err)

	result, err := convertToTypedStruct("/vlan:vlan/vlan:vlans", data)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	vlans, ok := result.(*huawei.HuaweiVlan_Vlan_Vlans)
	assert.True(t, ok, "Should be HuaweiVlan_Vlan_Vlans")
	assert.Len(t, vlans.Vlan, 1)

	vlan := vlans.Vlan[10]
	assert.NotNil(t, vlan)
	assert.Equal(t, uint16(10), *vlan.Id)
	assert.Equal(t, "VLAN10", *vlan.Name)
	assert.Equal(t, "Test VLAN", *vlan.Description)
	assert.Equal(t, huawei.E_HuaweiVlan_VlanType(1), vlan.Type)
	assert.Equal(t, huawei.E_HuaweiVlan_AdminStatus(1), vlan.AdminStatus)
	assert.Equal(t, huawei.E_HuaweiVlan_EnableStatus(1), vlan.BroadcastDiscard)
	assert.Equal(t, huawei.E_HuaweiVlan_EnableStatus(0), vlan.UnknownMulticastDiscard)
	assert.Equal(t, huawei.E_HuaweiVlan_EnableStatus(1), vlan.MacLearning)
	assert.Equal(t, uint32(300), *vlan.MacAgingTime)
	assert.Equal(t, uint16(100), *vlan.SuperVlan)
}

func TestConvertToTypedStruct_HuaweiVlan_Nested(t *testing.T) {
	jsonStr := `{
		"vlans": [
			{
				"id": 20,
				"name": "VLAN20",
				"unknown-unicast-discard": {
					"discard": 1,
					"mac-learning-enable": 0
				},
				"suppression": {
					"inbound": 1,
					"outbound": 1
				}
			}
		]
	}`

	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &data)
	assert.NoError(t, err)

	result, err := convertToTypedStruct("/vlan:vlan/vlan:vlans", data)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	vlans, ok := result.(*huawei.HuaweiVlan_Vlan_Vlans)
	assert.True(t, ok)

	vlan := vlans.Vlan[20]
	assert.NotNil(t, vlan)

	// Test unknown-unicast-discard
	assert.NotNil(t, vlan.UnkownUnicastDiscard)
	assert.Equal(t, huawei.E_HuaweiVlan_EnableStatus(1), vlan.UnkownUnicastDiscard.Discard)
	assert.Equal(t, huawei.E_HuaweiVlan_EnableStatus(0), vlan.UnkownUnicastDiscard.MacLearningEnable)

	// Test suppression
	assert.NotNil(t, vlan.Suppression)
	assert.Equal(t, huawei.E_HuaweiVlan_EnableStatus(1), vlan.Suppression.Inbound)
	assert.Equal(t, huawei.E_HuaweiVlan_EnableStatus(1), vlan.Suppression.Outbound)
}

func TestValueToUint(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected uint64
		ok       bool
	}{
		{float64(42), 42, true},
		{42, 42, true},
		{int64(42), 42, true},
		{uint(42), 42, true},
		{uint32(42), 42, true},
		{uint64(42), 42, true},
		{"42", 42, true},
		{"invalid", 0, false},
		{nil, 0, false},
	}

	for _, tt := range tests {
		result, ok := valueToUint(tt.input)
		assert.Equal(t, tt.ok, ok)
		if tt.ok {
			assert.Equal(t, tt.expected, result)
		}
	}
}
