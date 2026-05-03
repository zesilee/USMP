// Package translator 提供业务 CRD Spec 到厂商 YANG 模型的统一转换层
package translator

import (
	"fmt"
	"reflect"
)

// VendorType 厂商类型
type VendorType string

const (
	VendorHuawei   VendorType = "Huawei"
	VendorCisco    VendorType = "Cisco"
	VendorH3C      VendorType = "H3C"
	VendorJuniper  VendorType = "Juniper"
	VendorUnknown  VendorType = "Unknown"
)

// ConfigType 配置类型
type ConfigType string

const (
	ConfigTypeVlan      ConfigType = "Vlan"
	ConfigTypeInterface ConfigType = "Interface"
	ConfigTypeRoute     ConfigType = "Route"
	ConfigTypeSystem    ConfigType = "System"
)

// Translator 翻译器接口：定义所有配置类型的转换方法
type Translator interface {
	// Vendor 返回厂商名称
	Vendor() VendorType

	// TranslateVlan 转换 VLAN 配置
	// spec: 业务 VLAN Spec (bizv1.BusinessVlanSpec)
	// 返回: 厂商 YANG 结构体，如 *huawei.HuaweiVlan_Vlan_Vlans
	TranslateVlan(spec interface{}) (interface{}, error)

	// TranslateInterface 转换接口配置
	// spec: 业务 Interface Spec (bizv1.BusinessInterfaceSpec)
	// 返回: 厂商 YANG 结构体，如 *huawei.HuaweiIfm_Ifm_Interfaces
	TranslateInterface(spec interface{}) (interface{}, error)

	// TranslateRoute 转换路由配置（预留）
	TranslateRoute(spec interface{}) (interface{}, error)

	// TranslateSystem 转换系统配置（预留）
	TranslateSystem(spec interface{}) (interface{}, error)

	// Validate 验证配置合法性（厂商层面验证）
	Validate(configType ConfigType, spec interface{}) error
}

// TranslateError 翻译错误
type TranslateError struct {
	Vendor      VendorType
	ConfigType  ConfigType
	Field       string
	Reason      string
	Unsupported bool // 是否是不支持的特性
}

func (e *TranslateError) Error() string {
	if e.Unsupported {
		return fmt.Sprintf("[%s] %s 配置不支持字段 '%s': %s",
			e.Vendor, e.ConfigType, e.Field, e.Reason)
	}
	return fmt.Sprintf("[%s] %s 配置字段 '%s' 错误: %s",
		e.Vendor, e.ConfigType, e.Field, e.Reason)
}

// NewUnsupportedError 创建不支持特性错误
func NewUnsupportedError(vendor VendorType, configType ConfigType, field, reason string) error {
	return &TranslateError{
		Vendor:      vendor,
		ConfigType:  configType,
		Field:       field,
		Reason:      reason,
		Unsupported: true,
	}
}

// NewValidationError 创建验证错误
func NewValidationError(vendor VendorType, configType ConfigType, field, reason string) error {
	return &TranslateError{
		Vendor:     vendor,
		ConfigType: configType,
		Field:      field,
		Reason:     reason,
	}
}

// IsUnsupportedError 判断是否是不支持特性错误
func IsUnsupportedError(err error) bool {
	if e, ok := err.(*TranslateError); ok {
		return e.Unsupported
	}
	return false
}

// BaseTranslator 基础翻译器，提供通用工具方法
type BaseTranslator struct {
	vendor VendorType
}

// Vendor 实现 Translator 接口
func (b *BaseTranslator) Vendor() VendorType {
	return b.vendor
}

// assertType 类型断言辅助函数
func (b *BaseTranslator) assertType(spec interface{}, expected reflect.Type) (interface{}, error) {
	actual := reflect.TypeOf(spec)
	if actual == nil {
		return nil, fmt.Errorf("spec 不能为空")
	}
	if actual != expected && actual.Kind() == reflect.Ptr && actual.Elem() == expected {
		return spec, nil
	}
	if actual != expected && actual.Kind() != reflect.Ptr {
		ptr := reflect.New(expected)
		ptr.Elem().Set(reflect.ValueOf(spec))
		return ptr.Interface(), nil
	}
	return spec, nil
}

// TranslateRoute 默认实现（未支持）
func (b *BaseTranslator) TranslateRoute(spec interface{}) (interface{}, error) {
	return nil, NewUnsupportedError(b.vendor, ConfigTypeRoute, "*", "该厂商暂不支持路由配置转换")
}

// TranslateSystem 默认实现（未支持）
func (b *BaseTranslator) TranslateSystem(spec interface{}) (interface{}, error) {
	return nil, NewUnsupportedError(b.vendor, ConfigTypeSystem, "*", "该厂商暂不支持系统配置转换")
}

// Validate 默认实现（通过）
func (b *BaseTranslator) Validate(configType ConfigType, spec interface{}) error {
	return nil
}
