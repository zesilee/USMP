package translator

import (
	"fmt"
	"strings"
	"sync"
)

var (
	translators = make(map[VendorType]Translator)
	// registryMu 保护注册表：注册发生在各厂商实现文件的 init()（TE-01 编译期
	// 自注册），运行期以读为主；加锁防运行期注册与读并发竞态（R09）。
	registryMu sync.RWMutex
)

// RegisterTranslator 注册厂商翻译器（并发安全；厂商实现文件 init() 调用）
func RegisterTranslator(vendor VendorType, translator Translator) {
	registryMu.Lock()
	defer registryMu.Unlock()
	translators[vendor] = translator
}

// GetTranslator 获取指定厂商的翻译器
func GetTranslator(vendor VendorType) (Translator, error) {
	registryMu.RLock()
	translator, ok := translators[vendor]
	registryMu.RUnlock()
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
	registryMu.RLock()
	defer registryMu.RUnlock()
	vendors := make([]VendorType, 0, len(translators))
	for v := range translators {
		vendors = append(vendors, v)
	}
	return vendors
}

// IsVendorSupported 检查厂商是否支持
func IsVendorSupported(vendor VendorType) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	_, ok := translators[vendor]
	return ok
}

// VendorFromString 把大小写无关的厂商标签（如 DeviceStore 里的 "huawei"）映射为
// 枚举规范值（"Huawei"）。未知标签返回 (VendorUnknown, false)，由调用方决定
// 降级或透传原值以获得含厂商名的明确错误（R08）。
func VendorFromString(s string) (VendorType, bool) {
	for _, v := range []VendorType{VendorHuawei, VendorCisco, VendorH3C, VendorJuniper} {
		if strings.EqualFold(string(v), s) {
			return v, true
		}
	}
	return VendorUnknown, false
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
