package translator

import (
	bizv1 "github.com/leezesi/usmp/backend/api/v1"
)

// HuaweiTranslator 华为交换机完整翻译器实现
type HuaweiTranslator struct {
	*BaseTranslator
	vlanTrans      *HuaweiVlanTranslator
	ifaceTrans     *HuaweiInterfaceTranslator
}

// NewHuaweiTranslator 创建华为翻译器
func NewHuaweiTranslator() Translator {
	return &HuaweiTranslator{
		BaseTranslator: &BaseTranslator{vendor: VendorHuawei},
		vlanTrans:      NewHuaweiVlanTranslator(),
		ifaceTrans:     NewHuaweiInterfaceTranslator(),
	}
}

// TranslateVlan 翻译 VLAN 配置
func (h *HuaweiTranslator) TranslateVlan(spec interface{}) (interface{}, error) {
	vlanSpec, ok := spec.(bizv1.BusinessVlanSpec)
	if !ok {
		ptr, ok := spec.(*bizv1.BusinessVlanSpec)
		if !ok {
			return nil, NewValidationError(h.vendor, ConfigTypeVlan, "spec",
				"类型错误，需要 bizv1.BusinessVlanSpec")
		}
		vlanSpec = *ptr
	}

	return h.vlanTrans.Translate(vlanSpec)
}

// TranslateInterface 翻译接口配置
func (h *HuaweiTranslator) TranslateInterface(spec interface{}) (interface{}, error) {
	ifaceSpec, ok := spec.(bizv1.BusinessInterfaceSpec)
	if !ok {
		ptr, ok := spec.(*bizv1.BusinessInterfaceSpec)
		if !ok {
			return nil, NewValidationError(h.vendor, ConfigTypeInterface, "spec",
				"类型错误，需要 bizv1.BusinessInterfaceSpec")
		}
		ifaceSpec = *ptr
	}

	return h.ifaceTrans.Translate(ifaceSpec)
}

// Validate 验证配置
func (h *HuaweiTranslator) Validate(configType ConfigType, spec interface{}) error {
	switch configType {
	case ConfigTypeVlan:
		vlanSpec, ok := spec.(bizv1.BusinessVlanSpec)
		if !ok {
			ptr, ok := spec.(*bizv1.BusinessVlanSpec)
			if !ok {
				return NewValidationError(h.vendor, ConfigTypeVlan, "spec",
					"类型错误，需要 bizv1.BusinessVlanSpec")
			}
			vlanSpec = *ptr
		}
		return h.vlanTrans.Validate(vlanSpec)

	case ConfigTypeInterface:
		ifaceSpec, ok := spec.(bizv1.BusinessInterfaceSpec)
		if !ok {
			ptr, ok := spec.(*bizv1.BusinessInterfaceSpec)
			if !ok {
				return NewValidationError(h.vendor, ConfigTypeInterface, "spec",
					"类型错误，需要 bizv1.BusinessInterfaceSpec")
			}
			ifaceSpec = *ptr
		}
		return h.ifaceTrans.Validate(ifaceSpec)

	default:
		return h.BaseTranslator.Validate(configType, spec)
	}
}
