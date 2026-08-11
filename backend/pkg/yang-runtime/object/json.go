// json.go — 生成式 RFC7951 编解码的运行时小工具（YN-02，codegen-conventions §8）。
// 语义规则全部烘焙在生成代码里；本文件只提供机械原语（无 YANG 知识）。
package object

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// RawJSON marshals a plain scalar/base64 value（标量 Marshal 不可失败；仅供
// 生成代码对已知安全类型使用）。
func RawJSON(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		// 不可达（调用面为标量闭集）；保底不 panic（R08）。
		return json.RawMessage("null")
	}
	return b
}

// JSONArray joins pre-marshaled elements into a JSON array.
func JSONArray(parts []json.RawMessage) json.RawMessage {
	var b bytes.Buffer
	b.WriteByte('[')
	for i, p := range parts {
		if i > 0 {
			b.WriteByte(',')
		}
		b.Write(p)
	}
	b.WriteByte(']')
	return b.Bytes()
}

// StripModule removes an RFC7951 module qualifier（"huawei-vlan:vlan" → "vlan"）。
func StripModule(key string) string {
	if i := strings.IndexByte(key, ':'); i >= 0 {
		return key[i+1:]
	}
	return key
}

// ParseUint64JSON parses an RFC7951 uint64 leaf：编码规范为字符串（§6.1），
// 解码兼容数字形态。
func ParseUint64JSON(raw json.RawMessage) (uint64, error) {
	s := strings.TrimSpace(string(raw))
	if len(s) > 1 && s[0] == '"' {
		var q string
		if err := json.Unmarshal(raw, &q); err != nil {
			return 0, err
		}
		s = q
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid uint64 value %s", raw)
	}
	return v, nil
}

// ParseInt64JSON parses an RFC7951 int64 leaf（同 ParseUint64JSON 双形态）。
func ParseInt64JSON(raw json.RawMessage) (int64, error) {
	s := strings.TrimSpace(string(raw))
	if len(s) > 1 && s[0] == '"' {
		var q string
		if err := json.Unmarshal(raw, &q); err != nil {
			return 0, err
		}
		s = q
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid int64 value %s", raw)
	}
	return v, nil
}

// EnumValueByName resolves a YANG value name to its enum numeric value using a
// generation-wide EnumMaps table（线性扫描——单类型值域小，热路径可后续索引化）。
func EnumValueByName(table map[string]map[int64]EnumDefinition, typeName, name string) (int64, bool) {
	for v, def := range table[typeName] {
		if def.Name == name {
			return v, true
		}
	}
	return 0, false
}

// EmptyJSON is the RFC7951 encoding of a present `empty` leaf（§6.9）。
var EmptyJSON = json.RawMessage("[null]")

// IsEmptyJSON reports whether raw is the RFC7951 empty-leaf form [null]。
func IsEmptyJSON(raw json.RawMessage) bool {
	var arr []interface{}
	if err := json.Unmarshal(raw, &arr); err != nil {
		return false
	}
	return len(arr) == 1 && arr[0] == nil
}
