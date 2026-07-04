package api

import (
	"encoding/json"
	"strings"

	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
)

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
