package translator

import (
	"fmt"
	"net"
)

// ValidateIP 验证 IP 地址格式
func ValidateIP(ip string) error {
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("无效的 IP 地址: %s", ip)
	}
	return nil
}

// ValidateCIDR 验证 CIDR 格式
func ValidateCIDR(cidr string) error {
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("无效的 CIDR: %s (%w)", cidr, err)
	}
	return nil
}

// ValidateVlanID 验证 VLAN ID 范围
func ValidateVlanID(vlanID uint16) error {
	if vlanID < 1 || vlanID > 4094 {
		return fmt.Errorf("VLAN ID 必须在 1-4094 范围内，当前值: %d", vlanID)
	}
	return nil
}

// ValidateVlanRange 验证 VLAN 范围
func ValidateVlanRange(start, end uint16) error {
	if start < 1 || start > 4094 {
		return fmt.Errorf("VLAN 起始值必须在 1-4094 范围内，当前值: %d", start)
	}
	if end < 1 || end > 4094 {
		return fmt.Errorf("VLAN 结束值必须在 1-4094 范围内，当前值: %d", end)
	}
	if start > end {
		return fmt.Errorf("VLAN 范围无效: %d-%d，起始值不能大于结束值", start, end)
	}
	return nil
}

// ValidateInterfaceName 验证接口名称格式（简化版）
func ValidateInterfaceName(name string) error {
	if name == "" {
		return fmt.Errorf("接口名称不能为空")
	}
	if len(name) > 64 {
		return fmt.Errorf("接口名称长度不能超过 64 个字符")
	}
	return nil
}

// ValidateMTU 验证 MTU 范围
func ValidateMTU(mtu uint32) error {
	if mtu < 64 || mtu > 9216 {
		return fmt.Errorf("MTU 必须在 64-9216 范围内，当前值: %d", mtu)
	}
	return nil
}

// ValidateSpeed 验证速率配置（Mbps）
func ValidateSpeed(speed uint32) error {
	validSpeeds := []uint32{0, 10, 100, 1000, 10000, 25000, 40000, 100000}
	for _, s := range validSpeeds {
		if s == speed {
			return nil
		}
	}
	return fmt.Errorf("不支持的速率配置: %d Mbps，支持: 0(auto), 10, 100, 1000, 10000, 25000, 40000, 100000", speed)
}

// StringPtr 返回字符串指针
func StringPtr(s string) *string {
	return &s
}

// BoolPtr 返回布尔指针
func BoolPtr(b bool) *bool {
	return &b
}

// Uint16Ptr 返回 uint16 指针
func Uint16Ptr(u uint16) *uint16 {
	return &u
}

// Uint32Ptr 返回 uint32 指针
func Uint32Ptr(u uint32) *uint32 {
	return &u
}

// Uint64Ptr 返回 uint64 指针
func Uint64Ptr(u uint64) *uint64 {
	return &u
}

// SafeDeref 安全解引用字符串指针
func SafeDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// SafeDerefBool 安全解引用布尔指针
func SafeDerefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// SafeDerefUint16 安全解引用 uint16 指针
func SafeDerefUint16(u *uint16) uint16 {
	if u == nil {
		return 0
	}
	return *u
}

// SafeDerefUint32 安全解引用 uint32 指针
func SafeDerefUint32(u *uint32) uint32 {
	if u == nil {
		return 0
	}
	return *u
}
