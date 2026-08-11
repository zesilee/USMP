// Package object 是自研 YANG 类型系统运行库（retire-ygot-runtime YN-01，参考
// K8s runtime.Object 范式）：极小标记接口 + 枚举映射 + 指针 helper，替代
// ygot.GoStruct/KeyHelperGoStruct/GoEnum 三接口族。生成器（tools/yanggen）产出的
// 结构体实现本包接口；结构约定与 ygot 生成物字节级冻结（字段名、`path`+`module`
// tag、list=map[K]*T、`...Key` 复合键、枚举常量名），使 diff/xmlcodec 消费方免改。
//
// 本包零第三方依赖——发布二进制的类型系统地基（YN-05）。
package object

import (
	"fmt"
	"reflect"
)

// Object is the marker interface implemented by every generated YANG container
// or list-member struct（镜像 ygot.GoStruct：仅标记，类型保持「哑」）。
type Object interface {
	// IsYangObject is a marker method indicating the struct is a generated
	// YANG object whose fields carry `path`/`module` tags.
	IsYangObject()
}

// KeyedObject is implemented by generated list members, exposing their YANG
// list keys（镜像 ygot.KeyHelperGoStruct.ΛListKeyMap）。
type KeyedObject interface {
	Object
	// ListKeyMap returns the list keys as a map of YANG key-leaf name to
	// value. A nil key leaf is an error (mirror generated ygot behavior).
	ListKeyMap() (map[string]interface{}, error)
}

// EnumDefinition describes one enumerated value（镜像 ygot.EnumDefinition）。
type EnumDefinition struct {
	// Name is the YANG string name of the enumerated value.
	Name string
	// DefiningModule is set for identity values only（RFC7951 §6.8 序列化用）。
	DefiningModule string
	// Value is the numeric value where populated by generation.
	Value int
}

// Enum is implemented by every generated enumeration type（镜像 ygot.GoEnum）。
type Enum interface {
	// IsYangEnum is a marker method indicating a generated YANG enum.
	IsYangEnum()
	// EnumMaps returns the generation-wide map of enum type name → value →
	// definition（镜像 ΛMap；生成器按包产出一张静态表）。
	EnumMaps() map[string]map[int64]EnumDefinition
	// String returns the YANG name when in range（经 EnumLogString）。
	String() string
}

// EnumName returns the YANG string name of e: "" for unset (zero), the defined
// name when in range, and an error for non-zero out-of-range values（与
// ygot.EnumName 行为契约一致，见 object_test.go 对拍）。
func EnumName(e Enum) (string, error) {
	if e == nil {
		return "", fmt.Errorf("object: nil enum")
	}
	v := reflect.ValueOf(e)
	if v.Kind() != reflect.Int64 && !v.CanInt() {
		return "", fmt.Errorf("object: enum %T is not an integer type", e)
	}
	val := v.Int()
	if val == 0 {
		return "", nil // unset
	}
	typeName := v.Type().Name()
	def, ok := e.EnumMaps()[typeName][val]
	if !ok {
		return "", fmt.Errorf("object: value %d out of range for enum %s", val, typeName)
	}
	return def.Name, nil
}

// EnumLogString renders a log-friendly name for val of the named enum type:
// the YANG name when mapped, an out-of-range note otherwise（镜像
// ygot.EnumLogString，生成的 String() 调用它）。
func EnumLogString(e Enum, val int64, enumTypeName string) string {
	def, ok := e.EnumMaps()[enumTypeName][val]
	if !ok {
		return fmt.Sprintf("out-of-range %s enum value: %v", enumTypeName, val)
	}
	return def.Name
}

// Empty is the YANG `empty` leaf type：非指针 bool，true=present、false=跳发
// （镜像 ygot.YANGEmpty；xmlcodec 按 reflect bool kind 识别）。
type Empty bool

// Binary is the YANG `binary` leaf type（镜像 ygot.Binary）。
type Binary []byte

// 指针 helper（镜像 ygot.String/Uint16 等）：生成代码与调用方构造可选叶用。

// String returns a pointer to s.
func String(s string) *string { return &s }

// Bool returns a pointer to b.
func Bool(b bool) *bool { return &b }

// Int8 returns a pointer to v.
func Int8(v int8) *int8 { return &v }

// Int16 returns a pointer to v.
func Int16(v int16) *int16 { return &v }

// Int32 returns a pointer to v.
func Int32(v int32) *int32 { return &v }

// Int64 returns a pointer to v.
func Int64(v int64) *int64 { return &v }

// Uint8 returns a pointer to v.
func Uint8(v uint8) *uint8 { return &v }

// Uint16 returns a pointer to v.
func Uint16(v uint16) *uint16 { return &v }

// Uint32 returns a pointer to v.
func Uint32(v uint32) *uint32 { return &v }

// Uint64 returns a pointer to v.
func Uint64(v uint64) *uint64 { return &v }

// Float64 returns a pointer to v.
func Float64(v float64) *float64 { return &v }
