package schema

import (
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
)

// TestLeafLengthBounds（D9）：字符串 `length` 提取的表格驱动防线——
// 正常双界 / 多段取首末 / nil 与空（无 length 语句）/ 溢出侧省略（R08 不 panic）。
func TestLeafLengthBounds(t *testing.T) {
	num := func(v uint64) yang.Number { return yang.FromUint(v) }
	tests := []struct {
		name    string
		yt      *yang.YangType
		wantMin int
		hasMin  bool
		wantMax int
		hasMax  bool
	}{
		{"nil type", nil, 0, false, 0, false},
		{"no length statement", &yang.YangType{Kind: yang.Ystring}, 0, false, 0, false},
		{
			"explicit 1..31",
			&yang.YangType{Kind: yang.Ystring, Length: yang.YangRange{{Min: num(1), Max: num(31)}}},
			1, true, 31, true,
		},
		{
			"multi-segment takes outer bounds",
			&yang.YangType{Kind: yang.Ystring, Length: yang.YangRange{{Min: num(1), Max: num(8)}, {Min: num(16), Max: num(64)}}},
			1, true, 64, true,
		},
		{
			"overflow max omitted (R08)",
			&yang.YangType{Kind: yang.Ystring, Length: yang.YangRange{{Min: num(1), Max: yang.FromUint(^uint64(0))}}},
			1, true, 0, false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mn, hasMn, mx, hasMx := leafLengthBounds(tt.yt)
			if mn != tt.wantMin || hasMn != tt.hasMin || mx != tt.wantMax || hasMx != tt.hasMax {
				t.Errorf("got [%d(%v), %d(%v)], want [%d(%v), %d(%v)]",
					mn, hasMn, mx, hasMx, tt.wantMin, tt.hasMin, tt.wantMax, tt.hasMax)
			}
		})
	}
}

// TestDefaultLeafLengthAccessors：LengthMin/LengthMax 访问器语义（无界=false）。
func TestDefaultLeafLengthAccessors(t *testing.T) {
	l := &defaultLeaf{lengthMin: 1, hasLenMin: true, lengthMax: 31, hasLenMax: true}
	if mn, ok := l.LengthMin(); !ok || mn != 1 {
		t.Errorf("LengthMin = %d,%v want 1,true", mn, ok)
	}
	if mx, ok := l.LengthMax(); !ok || mx != 31 {
		t.Errorf("LengthMax = %d,%v want 31,true", mx, ok)
	}
	empty := &defaultLeaf{}
	if _, ok := empty.LengthMin(); ok {
		t.Error("empty leaf must report no length lower bound")
	}
	if _, ok := empty.LengthMax(); ok {
		t.Error("empty leaf must report no length upper bound")
	}
}
