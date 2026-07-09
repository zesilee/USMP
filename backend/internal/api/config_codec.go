package api

import (
	"encoding/json"

	"github.com/openconfig/ygot/ygot"

	// 空白导入触发 huawei 驱动描述符注册（DR-01）：本包编解码与 manager 路由
	// 均从 driver 注册表查表。
	_ "github.com/leezesi/usmp/backend/internal/drivers"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/driver"
)

// decodeRunningConfig turns a raw NETCONF XML readback into an RFC7951-shaped
// map (yang-named keys, list-as-array) so config-read endpoints return the same
// structure the frontend submits and lists — e.g. {"interface":[{"name":...}]}.
// Without this the handler returned opaque XML bytes (a base64 string over JSON)
// from which the "接口配置" list could extract no rows. Unrecognised paths or
// already-decoded (non-[]byte) data pass through unchanged.
func decodeRunningConfig(path string, data interface{}) interface{} {
	raw, ok := data.([]byte)
	if !ok || len(raw) == 0 || raw[0] != '<' {
		return data
	}

	// 解码器按驱动描述符注册表查表（DR-03）——不再散落路径字符串匹配。
	var parsed ygot.GoStruct
	if d, ok := driver.DecoderFor(path); ok {
		if p, err := d.DecodeXML(raw); err == nil {
			parsed = p
		}
	}
	if parsed == nil {
		return data
	}

	// SkipValidation：回读是「展示设备真值」，设备侧值不合本地 pattern（如子接口
	// number 带小数点）不应让整个回读降级成不透明 XML（R08）；写路径校验不受影响。
	js, err := ygot.EmitJSON(parsed, &ygot.EmitJSONConfig{Format: ygot.RFC7951, SkipValidation: true})
	if err != nil {
		return data
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(js), &out); err != nil {
		return data
	}
	return out
}

// convertConfig decodes request data into a typed desired config. It prefers the
// generic ygot codec (RFC7951 input) and falls back to the legacy hard-coded
// converters for the legacy (non-RFC7951, e.g. integer-enum) input shape. The
// legacy path is removed once the frontend emits RFC7951 (task 3.6 / group 5).
func convertConfig(path string, data map[string]interface{}) (interface{}, error) {
	if v, matched, err := encodeToYgot(path, data); matched && err == nil {
		return v, nil
	}
	return convertToTypedStruct(path, data)
}

// encodeToYgot decodes RFC7951-shaped request data into the ygot GoStruct for the
// given path via a single ygot.Unmarshal. The path→(struct, unmarshal) mapping
// comes from the driver descriptor registry (DR-03)——加模块 = 注册一条描述符。
// It returns (value, matched, err): matched is false when no descriptor covers
// the path (caller falls back to the legacy converters).
func encodeToYgot(path string, data map[string]interface{}) (interface{}, bool, error) {
	d, ok := driver.EncoderFor(path)
	if !ok {
		return nil, false, nil
	}
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, true, err
	}
	dest := d.NewStruct()
	if err := d.Unmarshal(jsonBytes, dest); err != nil {
		return nil, true, err
	}
	return dest, true, nil
}
