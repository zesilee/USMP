package translator

import (
	bizv1 "github.com/leezesi/usmp/backend/api/biz/v1"
)

// HuaweiTranslator 华为交换机完整翻译器实现
type HuaweiTranslator struct {
	*BaseTranslator
	vlanTrans  *HuaweiVlanTranslator
	ifaceTrans *HuaweiInterfaceTranslator
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

// TranslateRoute 翻译路由配置
func (h *HuaweiTranslator) TranslateRoute(spec interface{}) (interface{}, error) {
	routeSpec, ok := spec.(bizv1.BusinessRouteSpec)
	if !ok {
		ptr, ok := spec.(*bizv1.BusinessRouteSpec)
		if !ok {
			return nil, NewValidationError(h.vendor, ConfigTypeRoute, "spec",
				"类型错误，需要 bizv1.BusinessRouteSpec")
		}
		routeSpec = *ptr
	}

	// 华为路由配置经 CLI 或 huawei-ip YANG 模型下发；此处返回 map 供后续处理
	// （Route 尚未 ygot 化，为 stub，后续完善）。
	config := map[string]interface{}{
		"destination": routeSpec.Destination,
	}

	if routeSpec.NextHop != "" {
		config["nextHop"] = routeSpec.NextHop
	}
	if routeSpec.Preference > 0 {
		config["preference"] = routeSpec.Preference
	}
	if routeSpec.BfdEnabled {
		config["bfdEnabled"] = true
	}
	if routeSpec.Description != "" {
		config["description"] = routeSpec.Description
	}

	return config, nil
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

	case ConfigTypeRoute:
		// 路由基础验证
		routeSpec, ok := spec.(bizv1.BusinessRouteSpec)
		if !ok {
			ptr, ok := spec.(*bizv1.BusinessRouteSpec)
			if !ok {
				return NewValidationError(h.vendor, ConfigTypeRoute, "spec",
					"类型错误，需要 bizv1.BusinessRouteSpec")
			}
			routeSpec = *ptr
		}

		// 验证 CIDR 格式
		if err := ValidateCIDR(routeSpec.Destination); err != nil {
			return NewValidationError(h.vendor, ConfigTypeRoute, "destination",
				err.Error())
		}

		// 验证下一跳 IP
		if routeSpec.NextHop != "" {
			if err := ValidateIP(routeSpec.NextHop); err != nil {
				return NewValidationError(h.vendor, ConfigTypeRoute, "nextHop",
					err.Error())
			}
		}

		return nil

	default:
		return h.BaseTranslator.Validate(configType, spec)
	}
}
