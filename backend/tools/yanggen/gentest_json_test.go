package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/object"
	"github.com/leezesi/usmp/backend/tools/yanggen/gentest"
)

// 生成式 RFC7951 JSON 方法行为断言（任务3.1）：形状规则逐条验证 + 往返恒等。

func fullBox() *gentest.UsmpTest_Box {
	return &gentest.UsmpTest_Box{
		Host:   object.String("sw1"),
		Mtu:    object.Uint16(1500),
		On:     object.Bool(true),
		Big:    object.Uint64(18446744073709551615),
		Mark:   true,
		Status: gentest.UsmpTest_EnableStatus_enable,
		Mode:   gentest.E_UsmpTest_Box_Mode(1),
		Port:   &gentest.UsmpTest_Box_Port_Union_Uint16{Uint16: 8080},
		Tags:   []string{"a", "b"},
		Levels: []uint32{1, 2},
		Feature: &gentest.UsmpTest_Box_Feature{
			Rate: object.Uint32(100),
		},
		Extras: &gentest.UsmpTest_Box_Extras{Note: object.String("hi")},
		Vlans: &gentest.UsmpTest_Box_Vlans{
			Vlan: map[uint16]*gentest.UsmpTest_Box_Vlans_Vlan{
				20: {Id: object.Uint16(20), Name: object.String("v20")},
				10: {Id: object.Uint16(10), Name: object.String("v10")},
			},
		},
		Routes: &gentest.UsmpTest_Box_Routes{
			Route: map[gentest.UsmpTest_Box_Routes_Route_Key]*gentest.UsmpTest_Box_Routes_Route{
				{Vrf: "v1", Prefix: "10.0.0.0/8"}: {
					Vrf: object.String("v1"), Prefix: object.String("10.0.0.0/8"),
					Metric: object.Uint32(5),
				},
			},
		},
	}
}

func TestJSONMarshalShapes(t *testing.T) {
	b, err := fullBox().MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	s := string(b)
	for _, want := range []string{
		`"big":"18446744073709551615"`, // uint64 字符串化（§6.1）
		`"mark":[null]`,                // empty（§6.9）
		`"status":"enable"`,            // 枚举值域名
		`"usmp-test-ext:extras"`,       // augment 跨模块键限定
		`"port":8080`,                  // union uint16 成员
		`"vlan":[{`,                    // list → 数组
		`"tags":["a","b"]`,             // leaf-list
	} {
		if !strings.Contains(s, want) {
			t.Errorf("输出缺少 %s\nJSON: %s", want, s)
		}
	}
	// list 数组按 key 排序：v10 在 v20 前
	if strings.Index(s, `"v10"`) > strings.Index(s, `"v20"`) {
		t.Error("list 未按 key 排序")
	}
}

func TestJSONRoundTrip(t *testing.T) {
	src := fullBox()
	b, err := src.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	dst := &gentest.UsmpTest_Box{}
	if err := dst.UnmarshalJSON(b); err != nil {
		t.Fatalf("UnmarshalJSON: %v\nJSON: %s", err, b)
	}
	b2, err := dst.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != string(b2) {
		t.Fatalf("往返不恒等:\n1: %s\n2: %s", b, b2)
	}
	if *dst.Big != 18446744073709551615 || !bool(dst.Mark) || dst.Status != gentest.UsmpTest_EnableStatus_enable {
		t.Fatal("字段还原不完整")
	}
	if dst.Vlans.Vlan[10].Name == nil || *dst.Vlans.Vlan[10].Name != "v10" {
		t.Fatal("list map 还原失败")
	}
	if len(dst.Routes.Route) != 1 {
		t.Fatal("复合键 list 还原失败")
	}
}

func TestJSONDecodeForms(t *testing.T) {
	// 64 位兼容数字形态；模块限定键兼容
	in := `{"usmp-test:big": 42, "host": "h"}`
	dst := &gentest.UsmpTest_Box{}
	if err := dst.UnmarshalJSON([]byte(in)); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if *dst.Big != 42 || *dst.Host != "h" {
		t.Fatalf("decoded = %+v", dst)
	}
}

func TestJSONNegative(t *testing.T) {
	cases := map[string]string{
		"unknown field": `{"nope": 1}`,
		"bad enum":      `{"status": "nonsense"}`,
		"bad empty":     `{"mark": true}`,
		"missing key":   `{"vlans": {"vlan": [{"name": "x"}]}}`,
		"type mismatch": `{"mtu": "abc"}`,
	}
	for name, in := range cases {
		if err := (&gentest.UsmpTest_Box{}).UnmarshalJSON([]byte(in)); err == nil {
			t.Errorf("%s: 未报错（输入 %s）", name, in)
		}
	}
}

func TestJSONViaEncodingJSON(t *testing.T) {
	// json.Marshal 路径也走生成方法（标准库集成面）
	b, err := json.Marshal(fullBox())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"status":"enable"`) {
		t.Fatal("json.Marshal 未走 MarshalJSON")
	}
}
