package client

import (
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseHuaweiIfmInterfacesXML_PresentationLeaves 保证回读透出通用控制台
// 表格列所需的标识叶（class/parent-name/number/link-protocol/router-type）。
// 此前解析只覆盖 name/description/admin-status/mtu/type，这些叶回读恒空 →
// 前端列永远空、种子数据无法展示（同「回读解析恒空→漂移」根因谱系）。
func TestParseHuaweiIfmInterfacesXML_PresentationLeaves(t *testing.T) {
	xmlData := []byte(`
<data>
  <ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm">
    <interfaces>
      <interface>
        <name>200GE0/1/0.1</name>
        <class>2</class>
        <type>16</type>
        <parent-name>200GE0/1/0</parent-name>
        <number>0/1/0.1</number>
        <admin-status>1</admin-status>
        <link-protocol>1</link-protocol>
        <router-type>3</router-type>
      </interface>
      <interface>
        <name>200GE0/1/0</name>
        <class>1</class>
        <type>93</type>
        <admin-status>2</admin-status>
      </interface>
    </interfaces>
  </ifm>
</data>`)

	got, err := ParseHuaweiIfmInterfacesXML(xmlData)
	require.NoError(t, err)
	require.Len(t, got.Interface, 2)

	sub := got.Interface["200GE0/1/0.1"]
	require.NotNil(t, sub)
	assert.Equal(t, huawei.E_HuaweiIfm_ClassType(2), sub.Class, "class=sub-interface")
	assert.Equal(t, huawei.E_HuaweiIfm_PortType(16), sub.Type, "type=Vlanif")
	require.NotNil(t, sub.ParentName)
	assert.Equal(t, "200GE0/1/0", *sub.ParentName)
	require.NotNil(t, sub.Number)
	assert.Equal(t, "0/1/0.1", *sub.Number)
	assert.Equal(t, huawei.E_HuaweiIfm_PortStatus(1), sub.AdminStatus, "admin-status=down")
	assert.Equal(t, huawei.E_HuaweiIfm_LinkProtocol(1), sub.LinkProtocol, "link-protocol=ethernet")
	assert.Equal(t, huawei.E_HuaweiIfm_RouterType(3), sub.RouterType)

	// 缺省叶保持零值（不误置）。
	main := got.Interface["200GE0/1/0"]
	require.NotNil(t, main)
	assert.Nil(t, main.ParentName)
	assert.Equal(t, huawei.E_HuaweiIfm_LinkProtocol(0), main.LinkProtocol)
}
