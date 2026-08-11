// family.go — 值接口双族兼容层（S2，switch-plan）：引擎同时接受自研 object
// 接口族与 ygot 接口族的生成结构体，行为一致——XML 通道对拍（任务4.2）在同一
// 引擎上跑双族即字节级可比。S4 切换完成后删除 ygot 分支与本注释。
package xmlcodec

import (
	"fmt"
	"reflect"

	"github.com/openconfig/ygot/ygot"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/object"
)

var (
	goEnumType     = reflect.TypeOf((*ygot.GoEnum)(nil)).Elem()
	objectEnumType = reflect.TypeOf((*object.Enum)(nil)).Elem()
)

// isEnumType 双族枚举类型判定（生成枚举恒为 int64 底层）。
func isEnumType(t reflect.Type) bool {
	return t.Kind() == reflect.Int64 && (t.Implements(objectEnumType) || t.Implements(goEnumType))
}

// enumNameOf 枚举值→YANG 值域名（unset 返回空串 nil err，两族语义一致）。
func enumNameOf(v reflect.Value) (string, error) {
	switch e := v.Interface().(type) {
	case object.Enum:
		return object.EnumName(e)
	case ygot.GoEnum:
		return ygot.EnumName(e)
	}
	return "", fmt.Errorf("xmlcodec: %s is not an enum", v.Type())
}

// enumValueByName 值域名→int 反查（XC-08 解码）；未命中 ok=false。
func enumValueByName(v reflect.Value, name string) (int64, bool) {
	switch e := v.Interface().(type) {
	case object.Enum:
		return object.EnumValueByName(e.EnumMaps(), v.Type().Name(), name)
	case ygot.GoEnum:
		for val, def := range e.ΛMap()[v.Type().Name()] {
			if def.Name == name {
				return val, true
			}
		}
	}
	return 0, false
}

// listKeyMapOf 取 list 条目 key map（YANG 叶名→值）；条目不属任一族或取键
// 失败（nil key 叶）时 ok=false，调用方走各自的回退语义。
func listKeyMapOf(v interface{}) (map[string]interface{}, bool) {
	switch t := v.(type) {
	case object.KeyedObject:
		if km, err := t.ListKeyMap(); err == nil {
			return km, true
		}
	case ygot.KeyHelperGoStruct:
		if km, err := t.ΛListKeyMap(); err == nil {
			return km, true
		}
	}
	return nil, false
}
