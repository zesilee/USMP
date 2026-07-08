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

func TestMapEntryToInterface_FullAttributes(t *testing.T) {
	input := map[string]interface{}{
		"name":                     "GigabitEthernet0/0/1",
		"description":              "Test Interface with full attributes",
		"index":                    float64(12345),
		"number":                   "0/0/1",
		"position":                 "0/0/1",
		"parent-name":              "GigabitEthernet0",
		"admin-status":             float64(2),
		"type":                     float64(3),
		"class":                    float64(1),
		"link-protocol":            float64(0),
		"router-type":              float64(3),
		"service-type":             float64(1),
		"mtu":                      float64(1500),
		"mac-address":              "00:11:22:33:44:55",
		"bandwidth":                float64(1000),
		"bandwidth-kbps":           float64(1000000),
		"vrf-name":                 "public",
		"vs-name":                  "vs1",
		"aggregation-name":         "Eth-Trunk1",
		"down-delay-time":          float64(10),
		"protocol-up-delay-time":   float64(20),
		"clear-ip-df":              true,
		"is-l2-switch":             false,
		"l2-mode-enable":           false,
		"link-up-down-trap-enable": true,
		"statistic-enable":         true,
		"spread-mtu-flag":          false,
		"statistic-interval":       float64(300),
		"statistic-mode":           float64(1),
		"control-flap": map[string]interface{}{
			"ceiling":            float64(6000),
			"control-flap-count": float64(5),
			"decay-ng":           float64(10),
			"decay-ok":           float64(120),
			"reuse":              float64(1000),
			"suppress":           float64(3000),
		},
		"damp": map[string]interface{}{
			"tx-off": true,
			"auto": map[string]interface{}{
				"level": float64(2),
			},
			"manual": map[string]interface{}{
				"half-life-period":  float64(10),
				"max-suppress-time": float64(60),
				"reuse":             float64(500),
				"suppress":          float64(4000),
			},
		},
	}

	result := mapEntryToInterface(input)

	// 验证基础属性
	assert.Equal(t, "GigabitEthernet0/0/1", *result.Name)
	assert.Equal(t, "Test Interface with full attributes", *result.Description)
	assert.Equal(t, uint32(12345), *result.Index)
	assert.Equal(t, "0/0/1", *result.Number)
	assert.Equal(t, "0/0/1", *result.Position)
	assert.Equal(t, "GigabitEthernet0", *result.ParentName)

	// 验证类型和状态
	assert.Equal(t, huawei.E_HuaweiIfm_PortStatus(2), result.AdminStatus)
	assert.Equal(t, huawei.E_HuaweiIfm_PortType(3), result.Type)
	assert.Equal(t, huawei.E_HuaweiIfm_ClassType(1), result.Class)
	assert.Equal(t, huawei.E_HuaweiIfm_LinkProtocol(0), result.LinkProtocol)
	assert.Equal(t, huawei.E_HuaweiIfm_RouterType(3), result.RouterType)
	assert.Equal(t, huawei.E_HuaweiIfm_ServiceType(1), result.ServiceType)

	// 验证网络参数
	assert.Equal(t, uint32(1500), *result.Mtu)
	assert.Equal(t, "00:11:22:33:44:55", *result.MacAddress)
	assert.Equal(t, uint32(1000), *result.Bandwidth)
	assert.Equal(t, uint32(1000000), *result.BandwidthKbps)
	assert.Equal(t, "public", *result.VrfName)
	assert.Equal(t, "vs1", *result.VsName)

	// 验证链路聚合
	assert.Equal(t, "Eth-Trunk1", *result.AggregationName)

	// 验证定时器
	assert.Equal(t, uint32(10), *result.DownDelayTime)
	assert.Equal(t, uint32(20), *result.ProtocolUpDelayTime)

	// 验证功能开关
	assert.True(t, *result.ClearIpDf)
	assert.False(t, *result.IsL2Switch)
	assert.False(t, *result.L2ModeEnable)
	assert.True(t, *result.LinkUpDownTrapEnable)
	assert.True(t, *result.StatisticEnable)
	assert.False(t, *result.SpreadMtuFlag)

	// 验证统计配置
	assert.Equal(t, uint32(300), *result.StatisticInterval)
	assert.Equal(t, huawei.E_HuaweiIfm_StatisticMode(1), result.StatisticMode)

	// 验证嵌套容器 - ControlFlap
	assert.NotNil(t, result.ControlFlap)
	assert.Equal(t, uint32(6000), *result.ControlFlap.Ceiling)
	assert.Equal(t, uint32(5), *result.ControlFlap.ControlFlapCount)
	assert.Equal(t, uint32(10), *result.ControlFlap.DecayNg)
	assert.Equal(t, uint32(120), *result.ControlFlap.DecayOk)
	assert.Equal(t, uint32(1000), *result.ControlFlap.Reuse)
	assert.Equal(t, uint32(3000), *result.ControlFlap.Suppress)

	// 验证嵌套容器 - Damp
	assert.NotNil(t, result.Damp)
	assert.True(t, *result.Damp.TxOff)

	// 验证 Damp.Auto
	assert.NotNil(t, result.Damp.Auto)
	assert.Equal(t, huawei.E_HuaweiIfm_DampLevelType(2), result.Damp.Auto.Level)

	// 验证 Damp.Manual
	assert.NotNil(t, result.Damp.Manual)
	assert.Equal(t, uint16(10), *result.Damp.Manual.HalfLifePeriod)
	assert.Equal(t, uint16(60), *result.Damp.Manual.MaxSuppressTime)
	assert.Equal(t, uint32(500), *result.Damp.Manual.Reuse)
	assert.Equal(t, uint32(4000), *result.Damp.Manual.Suppress)
}

func TestConvertToTypedStruct_HuaweiSystem(t *testing.T) {
	jsonStr := `{
		"system-info": {
			"sys-name": "TestRouter",
			"sys-contact": "admin@example.com",
			"sys-location": "Beijing"
		}
	}`

	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &data)
	assert.NoError(t, err)

	result, err := convertToTypedStruct("/system:system", data)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	sys, ok := result.(*huawei.HuaweiSystem_System)
	assert.True(t, ok, "Should be HuaweiSystem_System")
	assert.NotNil(t, sys.SystemInfo)
	assert.Equal(t, "TestRouter", *sys.SystemInfo.SysName)
	assert.Equal(t, "admin@example.com", *sys.SystemInfo.SysContact)
	assert.Equal(t, "Beijing", *sys.SystemInfo.SysLocation)
}

func TestConvertToTypedStruct_HuaweiSystem_FlatData(t *testing.T) {
	jsonStr := `{
		"sys-name": "FlatRouter",
		"sys-contact": "support@example.com",
		"sys-location": "Shanghai"
	}`

	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &data)
	assert.NoError(t, err)

	result, err := convertToTypedStruct("/system:system", data)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	sys, ok := result.(*huawei.HuaweiSystem_System)
	assert.True(t, ok)
	assert.NotNil(t, sys.SystemInfo)
	assert.Equal(t, "FlatRouter", *sys.SystemInfo.SysName)
	assert.Equal(t, "support@example.com", *sys.SystemInfo.SysContact)
	assert.Equal(t, "Shanghai", *sys.SystemInfo.SysLocation)
}
