package xmlcodec

import (
	"strings"
	"testing"

	"github.com/openconfig/goyang/pkg/yang"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/internal/testutil/hwfix"
	"github.com/openconfig/ygot/ygot"
)

// TestEncodeDeleteMatchesGolden 对拍任务 1.3 冻结的 legacy marshalDeleteChange
// golden（XC-03 / DP-07 等价断言，含 nil-Name 条目的 map-key 回退形态）。
func TestEncodeDeleteMatchesGolden(t *testing.T) {
	tests := []struct {
		golden string
		spec   *Spec
		value  ygot.GoStruct
	}{
		{"delete_vlan", vlanSpec(), hwfix.VlanDeleteSet()},
		{"delete_ifm", ifmSpec(), hwfix.IfmDeleteSet()},
	}
	for _, tt := range tests {
		t.Run(tt.golden, func(t *testing.T) {
			out, err := EncodeDelete(tt.spec, tt.value)
			if err != nil {
				t.Fatalf("EncodeDelete: %v", err)
			}
			canon, err := Canonicalize([]byte(out))
			if err != nil {
				t.Fatalf("canonicalize: %v\nraw: %s", err, out)
			}
			if want := hwfix.Golden(t, tt.golden); canon != want {
				t.Errorf("delete encoding != legacy golden %s\n got: %s\nwant: %s\nraw: %s", tt.golden, canon, want, out)
			}
		})
	}
}

// TestEncodeDeleteRawShape：operation 属性带 NETCONF base 前缀、仅 key 叶、
// key 为首（唯一）子元素（DP-07 断言在原始输出上）。
func TestEncodeDeleteRawShape(t *testing.T) {
	out, err := EncodeDelete(vlanSpec(), hwfix.VlanDeleteSet())
	if err != nil {
		t.Fatal(err)
	}
	want := `<vlan nc:operation="delete" xmlns:nc="` + NetconfBaseNS + `"><id>10</id></vlan>`
	if !strings.Contains(out, want) {
		t.Errorf("missing keyed delete entry\n got: %s\nwant substring: %s", out, want)
	}
	if strings.Contains(out, "<name>") || strings.Contains(out, "<description>") {
		t.Errorf("delete encoding must carry key leaves only: %s", out)
	}
	if !strings.HasPrefix(out, `<vlans xmlns="`+vlanNS+`">`) {
		t.Errorf("missing namespaced model container: %s", out)
	}
}

func TestEncodeDeleteErrors(t *testing.T) {
	t.Run("empty set", func(t *testing.T) {
		if _, err := EncodeDelete(vlanSpec(), hwfix.VlanEmpty()); err == nil {
			t.Error("want error for empty delete target")
		}
	})
	t.Run("nil container", func(t *testing.T) {
		if _, err := EncodeDelete(vlanSpec(), (*huawei.HuaweiVlan_Vlan_Vlans)(nil)); err == nil {
			t.Error("want error for nil container")
		}
	})
	t.Run("no key information", func(t *testing.T) {
		// fake schema 的 list child 在 EncodeDelete 场景下抹掉 Key，且 fakeEntry
		// 不实现 ΛListKeyMap → 必须明确报错，绝不发送裸 delete（R08）。
		s := fakeSpec()
		base := s.Schema
		s.Schema = func() *yang.Entry {
			e := base()
			e.Dir["entry"].Key = ""
			return e
		}
		name := "x"
		v := &fakeRoot{Entry: map[string]*fakeEntry{"x": {Name: &name}}}
		if _, err := EncodeDelete(s, v); err == nil {
			t.Error("want error when key cannot be determined")
		}
	})
}

func TestEncodeDeleteConcurrent(t *testing.T) {
	spec, v := ifmSpec(), hwfix.IfmDeleteSet()
	done := make(chan error, 8)
	for i := 0; i < 8; i++ {
		go func() {
			_, err := EncodeDelete(spec, v)
			done <- err
		}()
	}
	for i := 0; i < 8; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
}
