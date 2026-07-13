package client

import (
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// diff 引擎对 ygot 的 interfaces/interface map 字段产出的 Change：
//
//	Path     = "Interface"（Go 字段名）
//	NewValue = map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface（内层 map）
//
// marshalChange 必须识别该 IFM 类型 map（镜像 VLAN 的专用断言），走 buildHuaweiIfmInterfacesXML
// 产出合法的华为 ifm edit-config XML；否则落到通用 <list> 兜底 = 畸形 XML，
// 接口不会真正下发到设备（表现为「新建接口后配置里看不到」）。
func TestMarshalChange_IfmTypedInnerMap(t *testing.T) {
	desc := "uplink"
	ifaceMap := map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{
		"GigabitEthernet0/0/9": {
			Name:        strPtr("GigabitEthernet0/0/9"),
			Description: &desc,
		},
	}

	change := Change{
		Type:       ModifyChange,
		Path:       "Interface", // diff 用 Go 字段名
		NewValue:   ifaceMap,
		SchemaPath: "Interface",
	}

	c := &NETCONFClient{}
	out, err := c.marshalChange(change)
	require.NoError(t, err)

	// 合法 ifm XML 的特征
	assert.Contains(t, out, `<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm"><interfaces>`)
	assert.Contains(t, out, `<interface>`)
	assert.Contains(t, out, `<name>GigabitEthernet0/0/9</name>`)
	assert.Contains(t, out, `<description>uplink</description>`)

	// 绝不能落到畸形兜底：无 <list> 容器、无 PascalCase 结构体标签
	assert.NotContains(t, out, "<list>")
	assert.NotContains(t, out, "HuaweiIfm_Ifm_Interfaces_Interface")
}
