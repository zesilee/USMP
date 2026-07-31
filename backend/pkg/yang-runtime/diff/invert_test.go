package diff

import (
	"reflect"
	"testing"
)

// TestInvert：回滚反算（CS-02）——ADD↔DELETE 互换、MODIFY 新旧值互换，
// Path/SchemaPath 保持，顺序保持，输入不被改动。
func TestInvert(t *testing.T) {
	tests := []struct {
		name string
		in   []Change
		want []Change
	}{
		{
			name: "add becomes delete of added entry",
			in:   []Change{{Type: AddChange, Path: "/vlan/vlans/vlan[id=10]", NewValue: "v10", SchemaPath: "/vlan"}},
			want: []Change{{Type: DeleteChange, Path: "/vlan/vlans/vlan[id=10]", OldValue: "v10", SchemaPath: "/vlan"}},
		},
		{
			name: "delete becomes add rebuilding baseline",
			in:   []Change{{Type: DeleteChange, Path: "/vlan/vlans/vlan[id=20]", OldValue: "v20"}},
			want: []Change{{Type: AddChange, Path: "/vlan/vlans/vlan[id=20]", NewValue: "v20"}},
		},
		{
			name: "modify swaps old and new",
			in:   []Change{{Type: ModifyChange, Path: "/ifm/interfaces/interface[name=ge0]/description", OldValue: "old", NewValue: "new"}},
			want: []Change{{Type: ModifyChange, Path: "/ifm/interfaces/interface[name=ge0]/description", OldValue: "new", NewValue: "old"}},
		},
		{
			name: "mixed set keeps order",
			in: []Change{
				{Type: AddChange, Path: "/a", NewValue: 1},
				{Type: ModifyChange, Path: "/b", OldValue: 2, NewValue: 3},
				{Type: DeleteChange, Path: "/c", OldValue: 4},
			},
			want: []Change{
				{Type: DeleteChange, Path: "/a", OldValue: 1},
				{Type: ModifyChange, Path: "/b", OldValue: 3, NewValue: 2},
				{Type: AddChange, Path: "/c", NewValue: 4},
			},
		},
		{
			name: "empty stays empty",
			in:   []Change{},
			want: []Change{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inCopy := make([]Change, len(tt.in))
			copy(inCopy, tt.in)

			got := Invert(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Invert mismatch\n got: %+v\nwant: %+v", got, tt.want)
			}
			if !reflect.DeepEqual(tt.in, inCopy) {
				t.Errorf("Invert mutated input: %+v", tt.in)
			}
		})
	}
}

// TestInvertResult：DiffResult 级反算——Summary 的 Adds/Deletes 计数互换，
// Modifies/Total 不变。
func TestInvertResult(t *testing.T) {
	r := NewDiffResult()
	r.AddChange(Change{Type: AddChange, Path: "/a", NewValue: 1})
	r.AddChange(Change{Type: AddChange, Path: "/b", NewValue: 2})
	r.AddChange(Change{Type: DeleteChange, Path: "/c", OldValue: 3})
	r.AddChange(Change{Type: ModifyChange, Path: "/d", OldValue: 4, NewValue: 5})

	inv := InvertResult(r)
	if inv.Summary.Adds != 1 || inv.Summary.Deletes != 2 || inv.Summary.Modifies != 1 || inv.Summary.Total != 4 {
		t.Errorf("summary not inverted: %+v", inv.Summary)
	}
	// 原结果不被改动
	if r.Summary.Adds != 2 || r.Summary.Deletes != 1 {
		t.Errorf("InvertResult mutated source summary: %+v", r.Summary)
	}
	if r.Changes[0].Type != AddChange {
		t.Errorf("InvertResult mutated source changes: %+v", r.Changes[0])
	}
}

// TestInvertIdempotentRoundTrip：两次反算回到原集（幂等回环防线）。
func TestInvertIdempotentRoundTrip(t *testing.T) {
	in := []Change{
		{Type: AddChange, Path: "/a", NewValue: 1},
		{Type: DeleteChange, Path: "/c", OldValue: 4},
		{Type: ModifyChange, Path: "/b", OldValue: 2, NewValue: 3},
	}
	if got := Invert(Invert(in)); !reflect.DeepEqual(got, in) {
		t.Errorf("double invert != original\n got: %+v\nwant: %+v", got, in)
	}
}
