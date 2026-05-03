package translator

import (
	"fmt"
	"sync"
)

var (
	translators = make(map[VendorType]Translator)
	once        sync.Once
)

// RegisterTranslator 注册厂商翻译器
func RegisterTranslator(vendor VendorType, translator Translator) {
	translators[vendor] = translator
}

// GetTranslator 获取指定厂商的翻译器
func GetTranslator(vendor VendorType) (Translator, error) {
	// 延迟初始化，避免循环导入
	once.Do(func() {
		// 默认注册华为翻译器
		RegisterTranslator(VendorHuawei, NewHuaweiTranslator())
	})

	translator, ok := translators[vendor]
	if !ok {
		return nil, fmt.Errorf("未找到厂商 '%s' 的翻译器", vendor)
	}
	return translator, nil
}

// MustGetTranslator 获取翻译器，失败则 panic
func MustGetTranslator(vendor VendorType) Translator {
	translator, err := GetTranslator(vendor)
	if err != nil {
		panic(err)
	}
	return translator
}

// SupportedVendors 返回支持的厂商列表
func SupportedVendors() []VendorType {
	vendors := make([]VendorType, 0, len(translators))
	for v := range translators {
		vendors = append(vendors, v)
	}
	return vendors
}

// IsVendorSupported 检查厂商是否支持
func IsVendorSupported(vendor VendorType) bool {
	_, ok := translators[vendor]
	return ok
}

// TranslateConfig 便捷函数：翻译指定类型的配置
func TranslateConfig(vendor VendorType, configType ConfigType, spec interface{}) (interface{}, error) {
	translator, err := GetTranslator(vendor)
	if err != nil {
		return nil, err
	}

	if err := translator.Validate(configType, spec); err != nil {
		return nil, err
	}

	switch configType {
	case ConfigTypeVlan:
		return translator.TranslateVlan(spec)
	case ConfigTypeInterface:
		return translator.TranslateInterface(spec)
	case ConfigTypeRoute:
		return translator.TranslateRoute(spec)
	case ConfigTypeSystem:
		return translator.TranslateSystem(spec)
	default:
		return nil, fmt.Errorf("未知配置类型: %s", configType)
	}
}
