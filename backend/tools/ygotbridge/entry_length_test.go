package ygotbridge

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
