// family.go — 值接口断言助手（object 接口族；S2 双族过渡层的 ygot 分支已随
// S4 删除，历史对拍见 change 提交记录）。
package xmlcodec

import (
	"fmt"
	"reflect"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/object"
)

var objectEnumType = reflect.TypeOf((*object.Enum)(nil)).Elem()

// isEnumType 枚举类型判定（生成枚举恒为 int64 底层）。
func isEnumType(t reflect.Type) bool {
	return t.Kind() == reflect.Int64 && t.Implements(objectEnumType)
}

// enumNameOf 枚举值→YANG 值域名（unset 返回空串 nil err）。
func enumNameOf(v reflect.Value) (string, error) {
	if e, ok := v.Interface().(object.Enum); ok {
		return object.EnumName(e)
	}
	return "", fmt.Errorf("xmlcodec: %s is not an enum", v.Type())
}

// enumValueByName 值域名→int 反查（XC-08 解码）；未命中 ok=false。
func enumValueByName(v reflect.Value, name string) (int64, bool) {
	if e, ok := v.Interface().(object.Enum); ok {
		return object.EnumValueByName(e.EnumMaps(), v.Type().Name(), name)
	}
	return 0, false
}

// listKeyMapOf 取 list 条目 key map（YANG 叶名→值）；取键失败（nil key 叶）
// 时 ok=false，调用方走各自的回退语义。
func listKeyMapOf(v interface{}) (map[string]interface{}, bool) {
	if t, ok := v.(object.KeyedObject); ok {
		if km, err := t.ListKeyMap(); err == nil {
			return km, true
		}
	}
	return nil, false
}
