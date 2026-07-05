package translator

import (
	"fmt"

	bizv1 "github.com/leezesi/usmp/backend/api/biz/v1"
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

// Translate 转换 api/biz/v1.BusinessInterfaceSpec 到 Huawei IFM YANG 模型。
// biz/v1 接口为 L2 模型（access/trunk/hybrid），无 L3/IP 字段。
func (t *HuaweiInterfaceTranslator) Translate(spec bizv1.BusinessInterfaceSpec) (*huawei.HuaweiIfm_Ifm_Interfaces, error) {
	ifName := spec.IfName
	if ifName == "" {
		return nil, NewValidationError(t.vendor, ConfigTypeInterface, "ifName",
			"接口名称不能为空")
	}

	ifaces := &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: make(map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface),
	}

	iface := &huawei.HuaweiIfm_Ifm_Interfaces_Interface{
		Name: &ifName,
	}

	if spec.Description != "" {
		desc := spec.Description
		iface.Description = &desc
	}

	// 管理状态
	if spec.AdminStatus == bizv1.InterfaceAdminStatusDown {
		iface.AdminStatus = 2 // Down
	} else {
		iface.AdminStatus = 1 // Up
	}

	// MTU
	if spec.MTU > 0 {
		if spec.MTU < 64 || spec.MTU > 9216 {
			return nil, NewValidationError(t.vendor, ConfigTypeInterface, "mtu",
				fmt.Sprintf("MTU 必须在 64-9216 范围内，当前值: %d", spec.MTU))
		}
		mtu := spec.MTU
		iface.Mtu = &mtu
	}

	// biz/v1 接口均为二层交换（access/trunk/hybrid）
	isL2 := true
	iface.IsL2Switch = &isL2
	iface.ServiceType = 2 // L2

	ifaces.Interface[ifName] = iface

	// 注意：VLAN 成员/端口划分需在 huawei-vlan 的 port-vlan 中配置，此处仅基本接口配置。
	return ifaces, nil
}

// Validate 验证接口配置在华为交换机上的可行性
func (t *HuaweiInterfaceTranslator) Validate(spec bizv1.BusinessInterfaceSpec) error {
	if spec.IfName == "" {
		return NewValidationError(t.vendor, ConfigTypeInterface, "ifName",
			"接口名称不能为空")
	}
	if spec.MTU > 0 && (spec.MTU < 64 || spec.MTU > 9216) {
		return NewValidationError(t.vendor, ConfigTypeInterface, "mtu",
			fmt.Sprintf("MTU 必须在 64-9216 范围内，当前值: %d", spec.MTU))
	}
	// Trunk 模式不配 accessVlan
	if spec.Mode == bizv1.InterfaceModeTrunk && spec.AccessVlan > 0 {
		return NewValidationError(t.vendor, ConfigTypeInterface, "accessVlan",
			"Trunk 模式不能配置 Access VLAN")
	}
	// Access 模式不配 trunkVlans
	if spec.Mode == bizv1.InterfaceModeAccess && len(spec.TrunkVlans) > 0 {
		return NewValidationError(t.vendor, ConfigTypeInterface, "trunkVlans",
			"Access 模式不能配置 Trunk 允许通过的 VLAN")
	}
	if spec.AccessVlan > 0 && (spec.AccessVlan < 1 || spec.AccessVlan > 4094) {
		return NewValidationError(t.vendor, ConfigTypeInterface, "accessVlan",
			fmt.Sprintf("Access VLAN 必须在 1-4094 范围内，当前值: %d", spec.AccessVlan))
	}
	for _, v := range spec.TrunkVlans {
		if v < 1 || v > 4094 {
			return NewValidationError(t.vendor, ConfigTypeInterface, "trunkVlans",
				fmt.Sprintf("Trunk VLAN ID 必须在 1-4094 范围内，当前值: %d", v))
		}
	}
	return nil
}
