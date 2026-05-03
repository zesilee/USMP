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

	// 华为路由配置是通过 CLI 或 huawei-ip YANG 模型下发
	// 这里返回 map 格式供后续处理
	config := map[string]interface{}{
		"destination": routeSpec.DestinationCIDR,
		"type":        routeSpec.Type,
	}

	if routeSpec.NextHopIP != "" {
		config["nextHop"] = routeSpec.NextHopIP
	}
	if routeSpec.OutInterface != "" {
		config["outInterface"] = routeSpec.OutInterface
	}
	if routeSpec.Preference > 0 {
		config["preference"] = routeSpec.Preference
	}
	if routeSpec.Tag > 0 {
		config["tag"] = routeSpec.Tag
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

		// 华为静态路由最大支持 255 条等价路由
		// 优先级范围 1-255
		if routeSpec.Preference > 255 {
			return NewValidationError(h.vendor, ConfigTypeRoute, "preference",
				"优先级必须在 1-255 范围内")
		}

		// 验证 CIDR 格式
		if err := ValidateCIDR(routeSpec.DestinationCIDR); err != nil {
			return NewValidationError(h.vendor, ConfigTypeRoute, "destinationCIDR",
				err.Error())
		}

		// 验证下一跳 IP
		if routeSpec.NextHopIP != "" {
			if err := ValidateIP(routeSpec.NextHopIP); err != nil {
				return NewValidationError(h.vendor, ConfigTypeRoute, "nextHopIP",
					err.Error())
			}
		}

		return nil

	default:
		return h.BaseTranslator.Validate(configType, spec)
	}
}
