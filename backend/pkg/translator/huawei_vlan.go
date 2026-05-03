package translator

import (
	"fmt"

	bizv1 "github.com/leezesi/usmp/backend/api/v1"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
)

// HuaweiVlanTranslator 华为 VLAN 配置翻译器
type HuaweiVlanTranslator struct {
	BaseTranslator
}

func NewHuaweiVlanTranslator() *HuaweiVlanTranslator {
	return &HuaweiVlanTranslator{
		BaseTranslator: BaseTranslator{vendor: VendorHuawei},
	}
}

// Translate 转换 BusinessVlanSpec 到 Huawei VLAN YANG 模型
func (t *HuaweiVlanTranslator) Translate(spec bizv1.BusinessVlanSpec) (*huawei.HuaweiVlan_Vlan_Vlans, error) {
	vlanID := spec.VlanID

	// 验证 VLAN ID 范围
	if vlanID < 1 || vlanID > 4094 {
		return nil, NewValidationError(t.vendor, ConfigTypeVlan, "vlanID",
			fmt.Sprintf("VLAN ID 必须在 1-4094 范围内，当前值: %d", vlanID))
	}

	// 创建 VLAN 配置
	vlans := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: make(map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan),
	}

	// 创建 VLAN 条目
	vlanName := fmt.Sprintf("Vlan%d", vlanID)
	if spec.Name != "" {
		vlanName = spec.Name
	}

	vlan := &huawei.HuaweiVlan_Vlan_Vlans_Vlan{
		Id:   &vlanID,
		Name: &vlanName,
	}

	// 配置描述
	if spec.Description != "" {
		vlan.Description = &spec.Description
	}

	// 配置管理状态 - 使用数值类型
	if spec.AdminStatus == bizv1.VlanAdminStatusDown {
		vlan.AdminStatus = 2 // Down
	} else {
		vlan.AdminStatus = 1 // Up
	}

	// 配置 VLAN 类型
	vlanType, err := t.convertVlanType(spec.Type)
	if err != nil {
		return nil, err
	}
	vlan.Type = vlanType

	// MAC 地址学习
	if spec.MacLearningEnabled != nil {
		if *spec.MacLearningEnabled {
			vlan.MacLearning = 1 // Enable
		} else {
			vlan.MacLearning = 2 // Disable
		}
	}

	// 统计功能
	if spec.StatisticEnabled != nil {
		if *spec.StatisticEnabled {
			vlan.StatisticEnable = 1 // Enable
		} else {
			vlan.StatisticEnable = 2 // Disable
		}
	}

	// 广播丢弃
	if spec.BroadcastDiscardEnabled != nil && *spec.BroadcastDiscardEnabled {
		vlan.BroadcastDiscard = 1 // Enable
	}

	// Super VLAN ID
	if spec.Type == bizv1.VlanTypeSub && spec.VlanID > 0 {
		// 这里需要根据实际业务配置 Super VLAN
		// 暂时注释，后续完善
	}

	vlans.Vlan[vlanID] = vlan

	return vlans, nil
}

// convertVlanType 转换 VLAN 类型
func (t *HuaweiVlanTranslator) convertVlanType(vlanType bizv1.VlanType) (huawei.E_HuaweiVlan_VlanType, error) {
	switch vlanType {
	case bizv1.VlanTypeCommon, "":
		return 1, nil // Common
	case bizv1.VlanTypeSuper:
		return 2, nil // Super
	case bizv1.VlanTypeSub:
		return 3, nil // Sub
	default:
		// Principal/Separate/Group 是 MUX VLAN 类型，华为需要额外配置
		return 1,
			NewUnsupportedError(t.vendor, ConfigTypeVlan, "type",
				fmt.Sprintf("华为交换机暂不支持 VLAN 类型 '%s'", vlanType))
	}
}

// Validate 验证 VLAN 配置在华为交换机上的可行性
func (t *HuaweiVlanTranslator) Validate(spec bizv1.BusinessVlanSpec) error {
	// VLAN ID 范围验证
	if spec.VlanID < 1 || spec.VlanID > 4094 {
		return NewValidationError(t.vendor, ConfigTypeVlan, "vlanID",
			fmt.Sprintf("VLAN ID 必须在 1-4094 范围内，当前值: %d", spec.VlanID))
	}

	// VLAN 名称长度限制
	if len(spec.Name) > 31 {
		return NewValidationError(t.vendor, ConfigTypeVlan, "name",
			"VLAN 名称长度不能超过 31 个字符")
	}

	// VLAN 描述长度限制
	if len(spec.Description) > 255 {
		return NewValidationError(t.vendor, ConfigTypeVlan, "description",
			"VLAN 描述长度不能超过 255 个字符")
	}

	// Super VLAN 和 Sub VLAN 不能同时配置物理端口
	if spec.Type == bizv1.VlanTypeSuper && len(spec.TaggedPorts) > 0 {
		return NewValidationError(t.vendor, ConfigTypeVlan, "type",
			"Super VLAN 不能配置物理端口")
	}

	return nil
}
