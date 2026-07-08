package api

import (
	"encoding/json"
	"strings"

	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
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

	var parsed ygot.GoStruct
	switch {
	case strings.Contains(path, "ifm:interfaces"):
		if p, err := client.ParseHuaweiIfmInterfacesXML(raw); err == nil {
			parsed = p
		}
	case strings.Contains(path, "vlan:vlans"):
		if p, err := client.ParseHuaweiVlanVlansXML(raw); err == nil {
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

// ygotTarget describes how to decode a config path into a strongly-typed ygot
// GoStruct: a constructor for the destination struct and the vendor Unmarshal
// (which consumes RFC7951 JSON).
type ygotTarget struct {
	newStruct func() ygot.GoStruct
	unmarshal func([]byte, ygot.GoStruct, ...ytypes.UnmarshalOpt) error
}

// ygotRegistry maps a YANG config path (matched by substring, as the existing
// handler does) to its ygot target. One line per module keeps this generic:
// the decode itself is a single ygot.Unmarshal, replacing the hand-written
// per-module map converters (R04). Add a module by adding a registry entry.
var ygotRegistry = []struct {
	match  func(path string) bool
	target ygotTarget
}{
	{
		match:  func(p string) bool { return strings.Contains(p, "ifm:ifm") && strings.Contains(p, "interfaces") },
		target: ygotTarget{func() ygot.GoStruct { return &huawei.HuaweiIfm_Ifm_Interfaces{} }, huawei.Unmarshal},
	},
	{
		match:  func(p string) bool { return strings.Contains(p, "vlan:") && strings.Contains(p, "vlan") },
		target: ygotTarget{func() ygot.GoStruct { return &huawei.HuaweiVlan_Vlan_Vlans{} }, huawei.Unmarshal},
	},
	{
		match:  func(p string) bool { return strings.Contains(p, "system:") },
		target: ygotTarget{func() ygot.GoStruct { return &huawei.HuaweiSystem_System{} }, huawei.Unmarshal},
	},
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
// given path via a single ygot.Unmarshal. It returns (value, matched, err):
// matched is false when no registry entry covers the path (caller falls back).
func encodeToYgot(path string, data map[string]interface{}) (interface{}, bool, error) {
	var t *ygotTarget
	for i := range ygotRegistry {
		if ygotRegistry[i].match(path) {
			t = &ygotRegistry[i].target
			break
		}
	}
	if t == nil {
		return nil, false, nil
	}
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, true, err
	}
	dest := t.newStruct()
	if err := t.unmarshal(jsonBytes, dest); err != nil {
		return nil, true, err
	}
	return dest, true, nil
}
