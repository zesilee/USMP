package object

import (
	"strings"
	"testing"
)

// ---- 生成器将产出的形状（手写样本，冻结生成约定）----

// sampleEnumMaps 模拟 enum_map.go 的包级映射（与 ygot ΛEnum 同构）。
var sampleEnumMaps = map[string]map[int64]EnumDefinition{
	"E_Sample_AdminStatus": {
		1: {Name: "down"},
		2: {Name: "up"},
	},
}

type E_Sample_AdminStatus int64

func (E_Sample_AdminStatus) IsYangEnum() {}
func (e E_Sample_AdminStatus) EnumMaps() map[string]map[int64]EnumDefinition {
	return sampleEnumMaps
}
func (e E_Sample_AdminStatus) String() string {
	return EnumLogString(e, int64(e), "E_Sample_AdminStatus")
}

// sampleEntry 模拟生成的 list 条目（KeyedObject 形状）。
type sampleEntry struct {
	Id   *uint16
	Name *string
}

func (*sampleEntry) IsYangObject() {}
func (e *sampleEntry) ListKeyMap() (map[string]interface{}, error) {
	if e.Id == nil {
		return nil, errMissingKey("Id")
	}
	return map[string]interface{}{"id": *e.Id}, nil
}

func errMissingKey(f string) error { return &missingKeyError{f} }

type missingKeyError struct{ field string }

func (e *missingKeyError) Error() string { return "nil value for key " + e.field }

// ---- 契约测试 ----

// TestEnumNameContract：EnumName 三形态契约（字面期望冻结自 ygot.EnumName
// 孪生对拍，2026-08-11 迁移期逐形态实证一致后孪生随 ygot 退役）。
func TestEnumNameContract(t *testing.T) {
	cases := []struct {
		name     string
		val      int64
		wantName string
		wantErr  bool
	}{
		{"valid down", 1, "down", false},
		{"valid up", 2, "up", false},
		{"unset", 0, "", false},
		{"out-of-range", 99, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotName, gotErr := EnumName(E_Sample_AdminStatus(tc.val))
			if gotName != tc.wantName {
				t.Fatalf("EnumName(%d) = %q, want %q", tc.val, gotName, tc.wantName)
			}
			if (gotErr != nil) != tc.wantErr {
				t.Fatalf("EnumName(%d) err = %v, wantErr=%v", tc.val, gotErr, tc.wantErr)
			}
		})
	}
}

// TestEnumStringContract：String()（EnumLogString 路径）三形态契约
// （文案形状冻结自 ygot.EnumLogString 对拍）。
func TestEnumStringContract(t *testing.T) {
	if got := E_Sample_AdminStatus(1).String(); got != "down" {
		t.Fatalf("String(1) = %q", got)
	}
	if got := E_Sample_AdminStatus(2).String(); got != "up" {
		t.Fatalf("String(2) = %q", got)
	}
	if got := E_Sample_AdminStatus(99).String(); !strings.HasPrefix(got, "out-of-range ") || !strings.HasSuffix(got, ": 99") {
		t.Fatalf("String(99) = %q, want out-of-range shape", got)
	}
}

// TestKeyedObjectShape：KeyedObject 契约（正常 + 缺键负路径）。
func TestKeyedObjectShape(t *testing.T) {
	var ko KeyedObject = &sampleEntry{Id: Uint16(10)}
	m, err := ko.ListKeyMap()
	if err != nil {
		t.Fatalf("ListKeyMap: %v", err)
	}
	if got, ok := m["id"].(uint16); !ok || got != 10 {
		t.Fatalf("ListKeyMap = %#v, want id=10", m)
	}
	if _, err := (&sampleEntry{}).ListKeyMap(); err == nil {
		t.Fatal("nil key must error (mirror ygot generated behavior)")
	}
}

// TestPointerHelpers：全部标量 helper 往返。
func TestPointerHelpers(t *testing.T) {
	if *String("x") != "x" || *Bool(true) != true ||
		*Int8(-1) != -1 || *Int16(-2) != -2 || *Int32(-3) != -3 || *Int64(-4) != -4 ||
		*Uint8(1) != 1 || *Uint16(2) != 2 || *Uint32(3) != 3 || *Uint64(4) != 4 ||
		*Float64(1.5) != 1.5 {
		t.Fatal("pointer helper round-trip mismatch")
	}
}

// TestEmptyAndBinaryKinds：Empty 底层是 bool、Binary 底层是 []byte——xmlcodec
// 与 diff 引擎按 reflect.Kind 识别这两个形态（约定冻结）。
func TestEmptyAndBinaryKinds(t *testing.T) {
	var e Empty = true
	if bool(e) != true {
		t.Fatal("Empty must be a bool kind")
	}
	var b Binary = []byte{1, 2}
	if len([]byte(b)) != 2 {
		t.Fatal("Binary must be a []byte kind")
	}
}

// TestObjectMarker：Object 标记接口可被指针类型实现（生成结构体恒以指针实现）。
func TestObjectMarker(t *testing.T) {
	var o Object = &sampleEntry{}
	if o == nil {
		t.Fatal("marker interface not satisfiable")
	}
}
