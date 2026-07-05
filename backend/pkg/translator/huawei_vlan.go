package translator

import (
	"fmt"

	bizv1 "github.com/leezesi/usmp/backend/api/biz/v1"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
)

// huawei-vlan enable-status 枚举: 1=Enable, 2=Disable
const (
	huaweiEnable  = 1
	huaweiDisable = 2
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

// Translate 转换 api/biz/v1.BusinessVlanSpec 到 Huawei VLAN YANG 模型
func (t *HuaweiVlanTranslator) Translate(spec bizv1.BusinessVlanSpec) (*huawei.HuaweiVlan_Vlan_Vlans, error) {
	vlanID := spec.VlanID
	if vlanID < 1 || vlanID > 4094 {
		return nil, NewValidationError(t.vendor, ConfigTypeVlan, "vlanID",
			fmt.Sprintf("VLAN ID 必须在 1-4094 范围内，当前值: %d", vlanID))
	}

	vlans := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: make(map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan),
	}

	vlanName := fmt.Sprintf("Vlan%d", vlanID)
	if spec.Name != "" {
		vlanName = spec.Name
	}

	vlan := &huawei.HuaweiVlan_Vlan_Vlans_Vlan{
		Id:   &vlanID,
		Name: &vlanName,
	}

	if spec.Description != "" {
		desc := spec.Description
		vlan.Description = &desc
	}

	// 管理状态 (up/down → 1/2)
	if spec.AdminStatus == bizv1.VlanAdminStatusDown {
		vlan.AdminStatus = 2 // Down
	} else {
		vlan.AdminStatus = 1 // Up
	}

	// MAC 地址学习 (enabled/disabled)
	switch spec.MacLearning {
	case bizv1.MacLearningEnabled:
		vlan.MacLearning = huaweiEnable
	case bizv1.MacLearningDisabled:
		vlan.MacLearning = huaweiDisable
	}

	// 广播丢弃
	if spec.BroadcastDiscard {
		vlan.BroadcastDiscard = huaweiEnable
	}

	// 未知组播丢弃
	if spec.UnknownMulticastDiscard {
		vlan.UnknownMulticastDiscard = huaweiEnable
	}

	vlans.Vlan[vlanID] = vlan
	return vlans, nil
}

// Validate 验证 VLAN 配置在华为交换机上的可行性
func (t *HuaweiVlanTranslator) Validate(spec bizv1.BusinessVlanSpec) error {
	if spec.VlanID < 1 || spec.VlanID > 4094 {
		return NewValidationError(t.vendor, ConfigTypeVlan, "vlanID",
			fmt.Sprintf("VLAN ID 必须在 1-4094 范围内，当前值: %d", spec.VlanID))
	}
	if len(spec.Name) > 31 {
		return NewValidationError(t.vendor, ConfigTypeVlan, "name",
			"VLAN 名称长度不能超过 31 个字符")
	}
	if len(spec.Description) > 255 {
		return NewValidationError(t.vendor, ConfigTypeVlan, "description",
			"VLAN 描述长度不能超过 255 个字符")
	}
	return nil
}
