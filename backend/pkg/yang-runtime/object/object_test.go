package object

import (
	"strings"
	"testing"

	"github.com/openconfig/ygot/ygot"
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

// ygot 参照物：同一映射的 ygot.GoEnum 孪生，契约测试与其对拍。
var ygotTwinMaps = map[string]map[int64]ygot.EnumDefinition{
	"E_YgotTwin_AdminStatus": {
		1: {Name: "down"},
		2: {Name: "up"},
	},
}

type E_YgotTwin_AdminStatus int64

func (E_YgotTwin_AdminStatus) IsYANGGoEnum() {}
func (e E_YgotTwin_AdminStatus) ΛMap() map[string]map[int64]ygot.EnumDefinition {
	return ygotTwinMaps
}
func (e E_YgotTwin_AdminStatus) String() string {
	return ygot.EnumLogString(e, int64(e), "E_YgotTwin_AdminStatus")
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

// TestEnumNameEquivalence：EnumName 与 ygot.EnumName 在合法值/unset/越界三形态
// 上行为一致（YN-01 枚举映射等价契约）。
func TestEnumNameEquivalence(t *testing.T) {
	cases := []struct {
		name string
		val  int64
	}{
		{"valid down", 1},
		{"valid up", 2},
		{"unset", 0},
		{"out-of-range", 99},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotName, gotErr := EnumName(E_Sample_AdminStatus(tc.val))
			wantName, wantErr := ygot.EnumName(E_YgotTwin_AdminStatus(tc.val))
			if gotName != wantName {
				t.Fatalf("EnumName(%d) = %q, ygot twin = %q", tc.val, gotName, wantName)
			}
			if (gotErr == nil) != (wantErr == nil) {
				t.Fatalf("EnumName(%d) err = %v, ygot twin err = %v", tc.val, gotErr, wantErr)
			}
		})
	}
}

// TestEnumStringEquivalence：String()（EnumLogString 路径）与 ygot 一致，
// 含越界的 out-of-range 文案形状。
func TestEnumStringEquivalence(t *testing.T) {
	for _, v := range []int64{1, 2, 99} {
		got := E_Sample_AdminStatus(v).String()
		want := ygot.EnumLogString(E_YgotTwin_AdminStatus(v), v, "E_YgotTwin_AdminStatus")
		// 越界文案含类型名（必然不同），只比形状：前缀与值。
		if v == 99 {
			if !strings.HasPrefix(got, "out-of-range ") || !strings.HasSuffix(got, ": 99") {
				t.Fatalf("String(99) = %q, want out-of-range shape like %q", got, want)
			}
			continue
		}
		if got != want {
			t.Fatalf("String(%d) = %q, ygot twin = %q", v, got, want)
		}
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
