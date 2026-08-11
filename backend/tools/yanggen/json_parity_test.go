package main

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/openconfig/ygot/ygot"

	ygothuawei "github.com/leezesi/usmp/backend/internal/generated/huawei"
	native "github.com/leezesi/usmp/backend/internal/generated/native/huawei"
)

// JSON 通道对拍（任务3.2，YN-06）：同一 RFC7951 输入 → ygot（ytypes.Unmarshal→
// EmitJSON）与 native（生成方法）双路径 解码→编码，输出**语义等价**断言
// （EmitJSON 带缩进且 map 序实现相关，字节比对无意义；两侧输出各自 re-parse 后
// DeepEqual——消费方 config_codec 本就 json.Unmarshal 进 map 再用）。
func TestJSONParityWithYgot(t *testing.T) {
	samples := map[string]string{
		"vlan list+enum": `{
			"huawei-vlan:vlan": {"vlans": {"vlan": [
				{"id": 10, "name": "v10", "description": "ten"},
				{"id": 20, "name": "v20"}
			]}}
		}`,
		"ifm nested containers": `{
			"huawei-ifm:ifm": {"interfaces": {"interface": [
				{"name": "GE0/0/1", "description": "up-link", "admin-status": "up", "mtu": 1500}
			]}}
		}`,
		"bgp container root": `{
			"huawei-bgp:bgp": {"base-process": {"as": "65001"}}
		}`,
		"system scalars": `{
			"huawei-system:system": {"system-info": {"sys-name": "sw-01"}}
		}`,
	}
	for name, in := range samples {
		t.Run(name, func(t *testing.T) {
			// ygot 路径
			yd := &ygothuawei.Device{}
			if err := ygothuawei.Unmarshal([]byte(in), yd); err != nil {
				t.Fatalf("ygot Unmarshal: %v", err)
			}
			yjs, err := ygot.EmitJSON(yd, &ygot.EmitJSONConfig{Format: ygot.RFC7951, SkipValidation: true})
			if err != nil {
				t.Fatalf("ygot EmitJSON: %v", err)
			}
			// native 路径
			nd := &native.Device{}
			if err := nd.UnmarshalJSON([]byte(in)); err != nil {
				t.Fatalf("native Unmarshal: %v", err)
			}
			njs, err := nd.MarshalJSON()
			if err != nil {
				t.Fatalf("native Marshal: %v", err)
			}
			var yv, nv interface{}
			if err := json.Unmarshal([]byte(yjs), &yv); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(njs, &nv); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(yv, nv) {
				t.Fatalf("语义不等:\nygot:   %s\nnative: %s", yjs, njs)
			}
		})
	}
}

// TestJSONParityNegative：非法输入两侧都拒绝（错误文案不要求一致，拒绝性一致）。
func TestJSONParityNegative(t *testing.T) {
	samples := map[string]string{
		"unknown field":  `{"huawei-vlan:vlan": {"vlans": {"vlan": [{"id": 1, "nope": 2}]}}}`,
		"enum bad value": `{"huawei-ifm:ifm": {"interfaces": {"interface": [{"name": "x", "admin-status": "sideways"}]}}}`,
	}
	for name, in := range samples {
		t.Run(name, func(t *testing.T) {
			yerr := ygothuawei.Unmarshal([]byte(in), &ygothuawei.Device{})
			nerr := (&native.Device{}).UnmarshalJSON([]byte(in))
			if (yerr == nil) != (nerr == nil) {
				t.Fatalf("拒绝性不一致: ygot=%v native=%v", yerr, nerr)
			}
		})
	}
}
