package client

import (
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func strPtr(s string) *string {
	return &s
}

func TestBuildHuaweiIfmInterfacesXML_NilInput(t *testing.T) {
	result, err := buildHuaweiIfmInterfacesXML(nil)
	require.NoError(t, err)
	assert.Equal(t, `<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm"><interfaces/></ifm>`, result)
}

func TestBuildHuaweiIfmInterfacesXML_EmptyInput(t *testing.T) {
	ifaces := &huawei.HuaweiIfm_Ifm_Interfaces{}
	result, err := buildHuaweiIfmInterfacesXML(ifaces)
	require.NoError(t, err)
	assert.Equal(t, `<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm"><interfaces/></ifm>`, result)
}

func TestBuildHuaweiIfmInterfacesXML_SingleInterface(t *testing.T) {
	ifaces := &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: make(map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface),
	}

	mtu := uint32(1500)
	bandwidth := uint32(1000000)
	desc := "Test Interface"
	mac := "00:11:22:33:44:55"

	ifaces.Interface["GigabitEthernet0/0/1"] = &huawei.HuaweiIfm_Ifm_Interfaces_Interface{
		Name:        strPtr("GigabitEthernet0/0/1"),
		Description: &desc,
		AdminStatus: huawei.E_HuaweiIfm_PortStatus(2), // up
		Mtu:         &mtu,
		Bandwidth:   &bandwidth,
		MacAddress:  &mac,
		Type:        huawei.E_HuaweiIfm_PortType(6),  // ethernet-csmacd
		Class:       huawei.E_HuaweiIfm_ClassType(1), // physical
	}

	result, err := buildHuaweiIfmInterfacesXML(ifaces)
	require.NoError(t, err)

	assert.Contains(t, result, `<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm"><interfaces>`)
	assert.Contains(t, result, `<interface>`)
	assert.Contains(t, result, `<name>GigabitEthernet0/0/1</name>`)
	assert.Contains(t, result, `<description>Test Interface</description>`)
	assert.Contains(t, result, `<admin-status>up</admin-status>`)
	assert.Contains(t, result, `<mtu>1500</mtu>`)
	assert.Contains(t, result, `<bandwidth>1000000</bandwidth>`)
	assert.Contains(t, result, `<mac-address>00:11:22:33:44:55</mac-address>`)
	assert.Contains(t, result, `<type>Ip-Trunk</type>`)
	assert.Contains(t, result, `<class>main-interface</class>`)
	assert.Contains(t, result, `</interface>`)
	assert.Contains(t, result, `</interfaces>`)
}

func TestBuildHuaweiIfmInterfacesXML_MultipleInterfaces(t *testing.T) {
	ifaces := &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: make(map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface),
	}

	ifaces.Interface["GigabitEthernet0/0/1"] = &huawei.HuaweiIfm_Ifm_Interfaces_Interface{
		Name:        strPtr("GigabitEthernet0/0/1"),
		Description: strPtr("Uplink to Core"),
	}
	ifaces.Interface["GigabitEthernet0/0/2"] = &huawei.HuaweiIfm_Ifm_Interfaces_Interface{
		Name:        strPtr("GigabitEthernet0/0/2"),
		Description: strPtr("Uplink to Access"),
	}

	result, err := buildHuaweiIfmInterfacesXML(ifaces)
	require.NoError(t, err)

	assert.Contains(t, result, `<name>GigabitEthernet0/0/1</name>`)
	assert.Contains(t, result, `<description>Uplink to Core</description>`)
	assert.Contains(t, result, `<name>GigabitEthernet0/0/2</name>`)
	assert.Contains(t, result, `<description>Uplink to Access</description>`)
}

func TestBuildHuaweiIfmInterfacesXML_BooleanFlags(t *testing.T) {
	ifaces := &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: make(map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface),
	}

	statisticEnable := true
	isL2Switch := true
	l2ModeEnable := true

	ifaces.Interface["GigabitEthernet0/0/1"] = &huawei.HuaweiIfm_Ifm_Interfaces_Interface{
		Name:            strPtr("GigabitEthernet0/0/1"),
		StatisticEnable: &statisticEnable,
		IsL2Switch:      &isL2Switch,
		L2ModeEnable:    &l2ModeEnable,
	}

	result, err := buildHuaweiIfmInterfacesXML(ifaces)
	require.NoError(t, err)

	assert.Contains(t, result, `<statistic-enable>true</statistic-enable>`)
	assert.Contains(t, result, `<is-l2-switch>true</is-l2-switch>`)
	assert.Contains(t, result, `<l2-mode-enable>true</l2-mode-enable>`)
}

func TestBuildHuaweiIfmInterfacesXML_DampContainer(t *testing.T) {
	ifaces := &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: make(map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface),
	}

	halfLife := uint16(10)
	maxSuppress := uint16(30)
	reuse := uint32(1000)
	suppress := uint32(2000)

	ifaces.Interface["GigabitEthernet0/0/1"] = &huawei.HuaweiIfm_Ifm_Interfaces_Interface{
		Name: strPtr("GigabitEthernet0/0/1"),
		Damp: &huawei.HuaweiIfm_Ifm_Interfaces_Interface_Damp{
			Auto: &huawei.HuaweiIfm_Ifm_Interfaces_Interface_Damp_Auto{
				Level: huawei.E_HuaweiIfm_DampLevelType(2), // medium
			},
			Manual: &huawei.HuaweiIfm_Ifm_Interfaces_Interface_Damp_Manual{
				HalfLifePeriod:  &halfLife,
				MaxSuppressTime: &maxSuppress,
				Reuse:           &reuse,
				Suppress:        &suppress,
			},
		},
	}

	result, err := buildHuaweiIfmInterfacesXML(ifaces)
	require.NoError(t, err)

	assert.Contains(t, result, `<damp>`)
	assert.Contains(t, result, `<auto>`)
	assert.Contains(t, result, `<level>middle</level>`)
	assert.Contains(t, result, `</auto>`)
	assert.Contains(t, result, `<manual>`)
	assert.Contains(t, result, `<half-life-period>10</half-life-period>`)
	assert.Contains(t, result, `<max-suppress-time>30</max-suppress-time>`)
	assert.Contains(t, result, `<reuse>1000</reuse>`)
	assert.Contains(t, result, `<suppress>2000</suppress>`)
	assert.Contains(t, result, `</manual>`)
	assert.Contains(t, result, `</damp>`)
}

func TestBuildHuaweiIfmInterfacesXML_ErrorDownContainer(t *testing.T) {
	ifaces := &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: make(map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface),
	}

	recoveryTime := uint32(300)
	remainderTime := uint32(150)

	ifaces.Interface["GigabitEthernet0/0/1"] = &huawei.HuaweiIfm_Ifm_Interfaces_Interface{
		Name: strPtr("GigabitEthernet0/0/1"),
		ErrorDown: &huawei.HuaweiIfm_Ifm_Interfaces_Interface_ErrorDown{
			Cause:         huawei.E_HuaweiIfm_ErrorDownType(1), // auto-negotiation-failed
			RecoveryTime:  &recoveryTime,
			RemainderTime: &remainderTime,
		},
	}

	result, err := buildHuaweiIfmInterfacesXML(ifaces)
	require.NoError(t, err)

	assert.Contains(t, result, `<error-down>`)
	assert.Contains(t, result, `<cause>bpdu-protection</cause>`)
	assert.Contains(t, result, `<recovery-time>300</recovery-time>`)
	assert.Contains(t, result, `<remainder-time>150</remainder-time>`)
	assert.Contains(t, result, `</error-down>`)
}

func TestBuildHuaweiIfmInterfacesXML_ControlFlapContainer(t *testing.T) {
	ifaces := &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: make(map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface),
	}

	ceiling := uint32(10000)
	reuse := uint32(1000)
	suppress := uint32(2000)
	decayOk := uint32(30)
	decayNg := uint32(60)
	flapCount := uint32(5)

	ifaces.Interface["GigabitEthernet0/0/1"] = &huawei.HuaweiIfm_Ifm_Interfaces_Interface{
		Name: strPtr("GigabitEthernet0/0/1"),
		ControlFlap: &huawei.HuaweiIfm_Ifm_Interfaces_Interface_ControlFlap{
			Ceiling:          &ceiling,
			Reuse:            &reuse,
			Suppress:         &suppress,
			DecayOk:          &decayOk,
			DecayNg:          &decayNg,
			ControlFlapCount: &flapCount,
		},
	}

	result, err := buildHuaweiIfmInterfacesXML(ifaces)
	require.NoError(t, err)

	assert.Contains(t, result, `<control-flap>`)
	assert.Contains(t, result, `<ceiling>10000</ceiling>`)
	assert.Contains(t, result, `<reuse>1000</reuse>`)
	assert.Contains(t, result, `<suppress>2000</suppress>`)
	assert.Contains(t, result, `<decay-ok>30</decay-ok>`)
	assert.Contains(t, result, `<decay-ng>60</decay-ng>`)
	assert.Contains(t, result, `<control-flap-count>5</control-flap-count>`)
	assert.Contains(t, result, `</control-flap>`)
}

func TestBuildHuaweiIfmInterfacesXML_XMLEscaping(t *testing.T) {
	ifaces := &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: make(map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface),
	}

	desc := `Test <interface> with "special" chars & 'quotes'`
	ifaces.Interface["GigabitEthernet0/0/1"] = &huawei.HuaweiIfm_Ifm_Interfaces_Interface{
		Name:        strPtr("GigabitEthernet0/0/1"),
		Description: &desc,
	}

	result, err := buildHuaweiIfmInterfacesXML(ifaces)
	require.NoError(t, err)

	assert.Contains(t, result, "&lt;interface&gt;")
	assert.Contains(t, result, "&quot;special&quot;")
	assert.Contains(t, result, "&amp;")
}
