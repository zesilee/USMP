// naming.go — yanggen 命名转换（codegen-conventions.md §1/§4 冻结约定）。
// 字段/路径段的 CamelCase 直接复用 goyang 实现（ygot 用的同一函数），
// 保证与既有生成物逐字对齐；枚举值净化把 genfix 的 |→_OR_ 一并内建。
package main

import (
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

// StructName builds a generated struct type name: CamelCase(module) + "_" +
// CamelCase(path segment)... joined with underscores（choice/case 段不入名，
// augment 子树用宿主路径——调用方负责传入正确的段序列）。
func StructName(module string, segs []string) string {
	parts := make([]string, 0, len(segs)+1)
	parts = append(parts, yang.CamelCase(module))
	for _, s := range segs {
		parts = append(parts, yang.CamelCase(s))
	}
	return strings.Join(parts, "_")
}

// FieldName converts a YANG leaf/node name to its Go field name.
func FieldName(name string) string {
	return yang.CamelCase(name)
}

// safeEnumReplacer 冻结自 ygot gogen/helpers.go 的净化表 + genfix 的 | 修补。
// 大小写原样保留（ARP/ENC_JSON_IETF 不动）。
var safeEnumReplacer = strings.NewReplacer(
	"+", "_PLUS", ",", "_COMMA", "@", "_AT", "$", "_DOLLAR",
	"*", "_ASTERISK", ":", "_COLON", "|", "_OR_",
	".", "_", "-", "_", "/", "_", " ", "_",
)

// SafeEnumValue sanitizes a YANG enum value name into a Go identifier suffix.
func SafeEnumValue(name string) string {
	return safeEnumReplacer.Replace(name)
}

// EnumConstName builds the Go constant name for an enum value:
// <type name without E_ prefix> + "_" + SafeEnumValue(value).
func EnumConstName(enumTypeName, valueName string) string {
	return strings.TrimPrefix(enumTypeName, "E_") + "_" + SafeEnumValue(valueName)
}
