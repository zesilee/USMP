package api

import (
	"encoding/json"
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/stretchr/testify/assert"
)

// BR-05/BR-06 全属性解码矩阵（T02b 等价，RFC7951 单一契约）：
// YANG 真名 + 枚举名字符串 → convertConfig → 强类型 ygot 结构逐字段断言。

func decodeJSON(t *testing.T, jsonStr string) map[string]interface{} {
	t.Helper()
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	return data
}

func TestConvertConfig_HuaweiIfm_FullAttributes(t *testing.T) {
	data := decodeJSON(t, `{
		"interface": [
			{
				"name": "GigabitEthernet0/0/1",
				"description": "Test Interface with full attributes",
				"index": 12345,
				"number": "0/0/1",
				"position": "0/0/1",
				"parent-name": "GigabitEthernet0",
				"admin-status": "up",
				"type": "GigabitEthernet",
				"class": "main-interface",
				"router-type": "broadcast",
				"service-type": "none",
				"mtu": 1500,
				"mac-address": "00:11:22:33:44:55",
				"bandwidth": 1000,
				"bandwidth-kbps": 1000000,
				"vrf-name": "public",
				"vs-name": "vs1"
			}
		]
	}`)

	result, err := convertConfig("/ifm:ifm/ifm:interfaces", data)
	assert.NoError(t, err)

	ifm, ok := result.(*huawei.HuaweiIfm_Ifm_Interfaces)
	assert.True(t, ok, "Should be HuaweiIfm_Ifm_Interfaces, got %T", result)
	assert.Len(t, ifm.Interface, 1)

	iface := ifm.Interface["GigabitEthernet0/0/1"]
	if !assert.NotNil(t, iface) {
		return
	}
	assert.Equal(t, "GigabitEthernet0/0/1", *iface.Name)
	assert.Equal(t, "Test Interface with full attributes", *iface.Description)
	assert.Equal(t, uint32(12345), *iface.Index)
	assert.Equal(t, "0/0/1", *iface.Number)
	assert.Equal(t, "0/0/1", *iface.Position)
	assert.Equal(t, "GigabitEthernet0", *iface.ParentName)
	assert.Equal(t, "up", iface.AdminStatus.String())
	assert.Equal(t, "GigabitEthernet", iface.Type.String())
	assert.Equal(t, "main-interface", iface.Class.String())
	assert.Equal(t, "broadcast", iface.RouterType.String())
	assert.Equal(t, "none", iface.ServiceType.String())
	assert.Equal(t, uint32(1500), *iface.Mtu)
	assert.Equal(t, "00:11:22:33:44:55", *iface.MacAddress)
	assert.Equal(t, uint32(1000), *iface.Bandwidth)
	assert.Equal(t, uint32(1000000), *iface.BandwidthKbps)
	assert.Equal(t, "public", *iface.VrfName)
	assert.Equal(t, "vs1", *iface.VsName)
}

func TestConvertConfig_HuaweiVlan_FullAttributes(t *testing.T) {
	data := decodeJSON(t, `{
		"vlan": [
			{
				"id": 10,
				"name": "VLAN10",
				"description": "Test VLAN",
				"type": "common",
				"admin-status": "up",
				"broadcast-discard": "enable",
				"unknown-multicast-discard": "disable",
				"mac-learning": "enable",
				"mac-aging-time": 300,
				"statistic-enable": "disable",
				"super-vlan": 100
			}
		]
	}`)

	result, err := convertConfig("/vlan:vlan/vlan:vlans", data)
	assert.NoError(t, err)

	vlans, ok := result.(*huawei.HuaweiVlan_Vlan_Vlans)
	assert.True(t, ok, "Should be HuaweiVlan_Vlan_Vlans, got %T", result)
	assert.Len(t, vlans.Vlan, 1)

	vlan := vlans.Vlan[10]
	if !assert.NotNil(t, vlan) {
		return
	}
	assert.Equal(t, uint16(10), *vlan.Id)
	assert.Equal(t, "VLAN10", *vlan.Name)
	assert.Equal(t, "Test VLAN", *vlan.Description)
	assert.Equal(t, "common", vlan.Type.String())
	assert.Equal(t, "up", vlan.AdminStatus.String())
	assert.Equal(t, "enable", vlan.BroadcastDiscard.String())
	assert.Equal(t, "disable", vlan.UnknownMulticastDiscard.String())
	assert.Equal(t, "enable", vlan.MacLearning.String())
	assert.Equal(t, uint32(300), *vlan.MacAgingTime)
	assert.Equal(t, uint16(100), *vlan.SuperVlan)
}

// 嵌套容器（suppression / unkown-unicast-discard，模型内即此拼写）随子树解码。
func TestConvertConfig_HuaweiVlan_NestedContainers(t *testing.T) {
	data := decodeJSON(t, `{
		"vlan": [
			{
				"id": 20,
				"name": "VLAN20",
				"unkown-unicast-discard": {
					"discard": "enable",
					"mac-learning-enable": "disable"
				},
				"suppression": {
					"inbound": "enable",
					"outbound": "enable"
				}
			}
		]
	}`)

	result, err := convertConfig("/vlan:vlan/vlan:vlans", data)
	assert.NoError(t, err)

	vlans := result.(*huawei.HuaweiVlan_Vlan_Vlans)
	vlan := vlans.Vlan[20]
	if !assert.NotNil(t, vlan) {
		return
	}
	if assert.NotNil(t, vlan.UnkownUnicastDiscard) {
		assert.Equal(t, "enable", vlan.UnkownUnicastDiscard.Discard.String())
		assert.Equal(t, "disable", vlan.UnkownUnicastDiscard.MacLearningEnable.String())
	}
	if assert.NotNil(t, vlan.Suppression) {
		assert.Equal(t, "enable", vlan.Suppression.Inbound.String())
		assert.Equal(t, "enable", vlan.Suppression.Outbound.String())
	}
}

// system：模块根整树与子路径扁平（锚点相对包裹）两种形状等价解码。
func TestConvertConfig_HuaweiSystem(t *testing.T) {
	nested, err := convertConfig("/system:system",
		decodeJSON(t, `{"system-info": {"sys-name": "TestRouter", "sys-contact": "admin@example.com", "sys-location": "Beijing"}}`))
	assert.NoError(t, err)
	sys, ok := nested.(*huawei.HuaweiSystem_System)
	assert.True(t, ok, "got %T", nested)
	if assert.NotNil(t, sys.SystemInfo) {
		assert.Equal(t, "TestRouter", *sys.SystemInfo.SysName)
		assert.Equal(t, "admin@example.com", *sys.SystemInfo.SysContact)
		assert.Equal(t, "Beijing", *sys.SystemInfo.SysLocation)
	}

	flat, err := convertConfig("/system:system/system:system-info",
		decodeJSON(t, `{"sys-name": "TestRouter", "sys-contact": "admin@example.com"}`))
	assert.NoError(t, err)
	sys2 := flat.(*huawei.HuaweiSystem_System)
	if assert.NotNil(t, sys2.SystemInfo) {
		assert.Equal(t, "TestRouter", *sys2.SystemInfo.SysName)
	}
}
