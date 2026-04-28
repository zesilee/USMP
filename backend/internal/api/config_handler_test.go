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
