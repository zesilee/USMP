package schema

import "testing"

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
