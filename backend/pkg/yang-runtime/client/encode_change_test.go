package client

import (
	"errors"
	"sync"
	"testing"

	_ "github.com/leezesi/usmp/backend/internal/drivers" // 注册 huawei 描述符：真实注册表分发
	"github.com/leezesi/usmp/backend/internal/testutil/hwfix"
)

// TestEncodeChangeXMLParityWithMarshalChange：导出纯函数与既有 marshalChange
// 在注册表命中路径上输出字节一致（CS-01 回归锚点：提取不改行为）。
// 覆盖 add/modify 两种变更类型与 容器/内层 map 两种值形态。
func TestEncodeChangeXMLParityWithMarshalChange(t *testing.T) {
	c := &NETCONFClient{}
	tests := []struct {
		name  string
		typ   ChangeType
		value interface{}
	}{
		{"add vlan container", AddChange, hwfix.VlanFull()},
		{"modify vlan inner map", ModifyChange, hwfix.VlanFull().Vlan},
		{"add ifm container", AddChange, hwfix.IfmFull()},
		{"modify ifm inner map", ModifyChange, hwfix.IfmFull().Interface},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := Change{Type: tt.typ, Path: "irrelevant", NewValue: tt.value}
			want, err := c.marshalChange(ch)
			if err != nil {
				t.Fatalf("marshalChange baseline: %v", err)
			}
			got, err := EncodeChangeXML(ch)
			if err != nil {
				t.Fatalf("EncodeChangeXML: %v", err)
			}
			if got != want {
				t.Errorf("EncodeChangeXML != marshalChange\n got: %s\nwant: %s", got, want)
			}
		})
	}
}

// TestEncodeChangeXMLDeleteParity：DeleteChange 走注册表删除编码，与
// marshalDeleteChange 字节一致（CS-01/CS-05 删除通道同源）。
func TestEncodeChangeXMLDeleteParity(t *testing.T) {
	for _, tt := range []struct {
		name   string
		target interface{}
	}{
		{"vlan delete set", hwfix.VlanDeleteSet()},
		{"ifm delete set", hwfix.IfmDeleteSet()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			want, err := marshalDeleteChange(tt.target)
			if err != nil {
				t.Fatalf("marshalDeleteChange baseline: %v", err)
			}
			got, err := EncodeChangeXML(Change{Type: DeleteChange, OldValue: tt.target})
			if err != nil {
				t.Fatalf("EncodeChangeXML delete: %v", err)
			}
			if got != want {
				t.Errorf("delete encode mismatch\n got: %s\nwant: %s", got, want)
			}
		})
	}
}

// TestEncodeChangeXMLNoEncoder：无 XML 通道的值必须返回可判别的
// ErrNoXMLEncoder（CS-03——预览层据此如实降级，绝不走脏兜底伪造报文）。
func TestEncodeChangeXMLNoEncoder(t *testing.T) {
	type notRegistered struct{ X int }
	_, err := EncodeChangeXML(Change{Type: ModifyChange, NewValue: &notRegistered{1}})
	if err == nil {
		t.Fatal("want error for unregistered model, got nil")
	}
	if !errors.Is(err, ErrNoXMLEncoder) {
		t.Fatalf("want ErrNoXMLEncoder, got %v", err)
	}

	_, err = EncodeChangeXML(Change{Type: DeleteChange, OldValue: &notRegistered{1}})
	if !errors.Is(err, ErrNoXMLEncoder) {
		t.Fatalf("delete: want ErrNoXMLEncoder, got %v", err)
	}
}

// TestEncodeChangeXMLNilValue：非删除变更缺 NewValue 明确报错（R08，
// 与 marshalChange 既有语义一致）。
func TestEncodeChangeXMLNilValue(t *testing.T) {
	if _, err := EncodeChangeXML(Change{Type: ModifyChange, Path: "/x"}); err == nil {
		t.Fatal("want error for nil NewValue, got nil")
	}
}

// TestEncodeChangeXMLPassthrough：string/[]byte 值原样透传（与 marshalChange
// 一致，供上层已编码片段直发）。
func TestEncodeChangeXMLPassthrough(t *testing.T) {
	for _, tt := range []struct {
		name  string
		value interface{}
	}{
		{"string", "<x/>"},
		{"bytes", []byte("<x/>")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out, err := EncodeChangeXML(Change{Type: ModifyChange, NewValue: tt.value})
			if err != nil {
				t.Fatalf("passthrough: %v", err)
			}
			if out != "<x/>" {
				t.Errorf("passthrough got %q", out)
			}
		})
	}
}

// TestEncodeChangeXMLConcurrent：并发编码无竞态（B1 race 防线——注册表
// 只读分发必须协程安全）。
func TestEncodeChangeXMLConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				if _, err := EncodeChangeXML(Change{Type: ModifyChange, NewValue: hwfix.VlanFull()}); err != nil {
					t.Errorf("concurrent encode: %v", err)
					return
				}
				if _, err := EncodeChangeXML(Change{Type: DeleteChange, OldValue: hwfix.VlanDeleteSet()}); err != nil {
					t.Errorf("concurrent delete encode: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
}
