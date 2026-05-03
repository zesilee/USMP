package translator

import (
	"fmt"

	bizv1 "github.com/leezesi/usmp/backend/api/v1"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
)

// HuaweiInterfaceTranslator 华为接口配置翻译器
type HuaweiInterfaceTranslator struct {
	BaseTranslator
}

func NewHuaweiInterfaceTranslator() *HuaweiInterfaceTranslator {
	return &HuaweiInterfaceTranslator{
		BaseTranslator: BaseTranslator{vendor: VendorHuawei},
	}
}

// Translate 转换 BusinessInterfaceSpec 到 Huawei IFM YANG 模型
func (t *HuaweiInterfaceTranslator) Translate(spec bizv1.BusinessInterfaceSpec) (*huawei.HuaweiIfm_Ifm_Interfaces, error) {
	ifName := spec.InterfaceName
	if ifName == "" {
		return nil, NewValidationError(t.vendor, ConfigTypeInterface, "interfaceName",
			"接口名称不能为空")
	}

	// 创建接口配置
	ifaces := &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: make(map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface),
	}

	iface := &huawei.HuaweiIfm_Ifm_Interfaces_Interface{
		Name: &ifName,
	}

	// 配置描述
	if spec.Description != "" {
		iface.Description = &spec.Description
	}

	// 配置管理状态
	if spec.AdminStatus == bizv1.InterfaceAdminStatusDown {
		iface.AdminStatus = 2 // Down
	} else {
		iface.AdminStatus = 1 // Up
	}

	// 配置 MTU
	if spec.MTU > 0 {
		if spec.MTU < 64 || spec.MTU > 9216 {
			return nil, NewValidationError(t.vendor, ConfigTypeInterface, "mtu",
				fmt.Sprintf("MTU 必须在 64-9216 范围内，当前值: %d", spec.MTU))
		}
		iface.Mtu = &spec.MTU
	}

	// 配置接口模式（二层/三层）
	if spec.Mode == bizv1.InterfaceModeL2 {
		isL2 := true
		iface.IsL2Switch = &isL2
	} else if spec.Mode == bizv1.InterfaceModeL3 {
		isL2 := false
		iface.IsL2Switch = &isL2
	}

	// 配置服务类型
	if spec.Mode == bizv1.InterfaceModeL2 || spec.Mode == bizv1.InterfaceModeAccess ||
		spec.Mode == bizv1.InterfaceModeTrunk || spec.Mode == bizv1.InterfaceModeHybrid {
		iface.ServiceType = 2 // L2
	} else if spec.Mode == bizv1.InterfaceModeL3 {
		iface.ServiceType = 1 // L3
	}

	ifaces.Interface[ifName] = iface

	// 注意：VLAN 相关配置需要在其他 YANG 模块中配置（如 huawei-vlan 的 port-vlan）
	// 这里只做基本接口配置

	return ifaces, nil
}

// Validate 验证接口配置在华为交换机上的可行性
func (t *HuaweiInterfaceTranslator) Validate(spec bizv1.BusinessInterfaceSpec) error {
	// 接口名称不能为空
	if spec.InterfaceName == "" {
		return NewValidationError(t.vendor, ConfigTypeInterface, "interfaceName",
			"接口名称不能为空")
	}

	// MTU 范围验证
	if spec.MTU > 0 && (spec.MTU < 64 || spec.MTU > 9216) {
		return NewValidationError(t.vendor, ConfigTypeInterface, "mtu",
			fmt.Sprintf("MTU 必须在 64-9216 范围内，当前值: %d", spec.MTU))
	}

	// 三层接口必须配置 IP
	if spec.Mode == bizv1.InterfaceModeL3 && spec.IpAddress == "" {
		return NewValidationError(t.vendor, ConfigTypeInterface, "ipAddress",
			"三层接口必须配置 IP 地址")
	}

	// Trunk 模式不能配置 accessVlan
	if spec.Mode == bizv1.InterfaceModeTrunk && spec.AccessVlan > 0 {
		return NewValidationError(t.vendor, ConfigTypeInterface, "accessVlan",
			"Trunk 模式不能配置 Access VLAN")
	}

	// Access 模式不能配置 trunkAllowedVlans
	if spec.Mode == bizv1.InterfaceModeAccess && len(spec.TrunkAllowedVlans) > 0 {
		return NewValidationError(t.vendor, ConfigTypeInterface, "trunkAllowedVlans",
			"Access 模式不能配置 Trunk 允许通过的 VLAN")
	}

	// Access VLAN 范围验证
	if spec.AccessVlan > 0 && (spec.AccessVlan < 1 || spec.AccessVlan > 4094) {
		return NewValidationError(t.vendor, ConfigTypeInterface, "accessVlan",
			fmt.Sprintf("Access VLAN 必须在 1-4094 范围内，当前值: %d", spec.AccessVlan))
	}

	// Native VLAN 范围验证
	if spec.NativeVlan > 0 && (spec.NativeVlan < 1 || spec.NativeVlan > 4094) {
		return NewValidationError(t.vendor, ConfigTypeInterface, "nativeVlan",
			fmt.Sprintf("Native VLAN 必须在 1-4094 范围内，当前值: %d", spec.NativeVlan))
	}

	// Trunk 允许 VLAN 范围验证
	for _, v := range spec.TrunkAllowedVlans {
		if v.VlanID < 1 || v.VlanID > 4094 {
			return NewValidationError(t.vendor, ConfigTypeInterface, "trunkAllowedVlans",
				fmt.Sprintf("Trunk VLAN ID 必须在 1-4094 范围内，当前值: %d", v.VlanID))
		}
	}

	return nil
}
