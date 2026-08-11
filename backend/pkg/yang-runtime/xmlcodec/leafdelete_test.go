package xmlcodec

import (
	"strings"
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/native/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/object"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

func vlanEntryOnlyKey(id uint16) *huawei.HuaweiVlan_Vlan_Vlans {
	return &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
			id: {Id: object.Uint16(id)},
		},
	}
}

// TestEncodeLeafDeleteShape：叶级删除（CS-05）——条目定位键 + 目标叶带
// nc:operation="delete" 自闭合；条目元素本身绝不携带 operation（不误删条目）。
func TestEncodeLeafDeleteShape(t *testing.T) {
	out, err := EncodeLeafDelete(vlanSpec(), vlanEntryOnlyKey(5), []string{"description", "name"})
	if err != nil {
		t.Fatalf("EncodeLeafDelete: %v", err)
	}
	if !strings.HasPrefix(out, `<vlan xmlns="`+vlanNS+`"><vlans>`) {
		t.Errorf("missing namespaced module wrapper: %s", out)
	}
	if !strings.Contains(out, "<vlan><id>5</id>") {
		t.Errorf("entry must open plain with key leaf first: %s", out)
	}
	for _, leaf := range []string{"description", "name"} {
		want := `<` + leaf + ` nc:operation="delete" xmlns:nc="` + NetconfBaseNS + `"/>`
		if !strings.Contains(out, want) {
			t.Errorf("missing leaf delete %q\n got: %s\nwant substring: %s", leaf, out, want)
		}
	}
	if strings.Contains(out, `<vlan nc:operation`) {
		t.Errorf("entry element must NOT carry delete operation: %s", out)
	}
}

// TestEncodeLeafDeleteMultiEntry：多条目共用同一叶集合，逐条目生成。
func TestEncodeLeafDeleteMultiEntry(t *testing.T) {
	v := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
			5:  {Id: object.Uint16(5)},
			10: {Id: object.Uint16(10)},
		},
	}
	out, err := EncodeLeafDelete(vlanSpec(), v, []string{"description"})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(out, `<description nc:operation="delete"`); got != 2 {
		t.Errorf("want 2 leaf deletes (one per entry), got %d: %s", got, out)
	}
	for _, key := range []string{"<id>5</id>", "<id>10</id>"} {
		if !strings.Contains(out, key) {
			t.Errorf("missing entry key %s: %s", key, out)
		}
	}
}

// TestEncodeLeafDeleteIfmStringKey：字符串键模块（ifm）同形态。
func TestEncodeLeafDeleteIfmStringKey(t *testing.T) {
	v := &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{
			"GE0/0/1": {Name: object.String("GE0/0/1")},
		},
	}
	out, err := EncodeLeafDelete(ifmSpec(), v, []string{"description"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<ifm xmlns="` + ifmNS + `"><interfaces>`,
		"<interface><name>GE0/0/1</name>",
		`<description nc:operation="delete" xmlns:nc="` + NetconfBaseNS + `"/>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in: %s", want, out)
		}
	}
}

// TestEncodeLeafDeleteIdempotent：同输入两次调用输出一致（CS-05 幂等）。
func TestEncodeLeafDeleteIdempotent(t *testing.T) {
	a, err := EncodeLeafDelete(vlanSpec(), vlanEntryOnlyKey(7), []string{"description"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := EncodeLeafDelete(vlanSpec(), vlanEntryOnlyKey(7), []string{"description"})
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Errorf("not idempotent:\n a: %s\n b: %s", a, b)
	}
}

// TestEncodeLeafDeleteErrors：空叶集合、空条目、未知叶、非叶节点均为明确
// 错误——绝不发送无目标或指错目标的删除（R08）。
func TestEncodeLeafDeleteErrors(t *testing.T) {
	t.Run("empty leaves", func(t *testing.T) {
		if _, err := EncodeLeafDelete(vlanSpec(), vlanEntryOnlyKey(5), nil); err == nil {
			t.Error("want error for empty leaf set")
		}
	})
	t.Run("empty target", func(t *testing.T) {
		if _, err := EncodeLeafDelete(vlanSpec(), &huawei.HuaweiVlan_Vlan_Vlans{}, []string{"description"}); err == nil {
			t.Error("want error for empty entry set")
		}
	})
	t.Run("unknown leaf", func(t *testing.T) {
		if _, err := EncodeLeafDelete(vlanSpec(), vlanEntryOnlyKey(5), []string{"no-such-leaf"}); err == nil {
			t.Error("want error for leaf not in schema")
		}
	})
	t.Run("non-leaf node", func(t *testing.T) {
		// ip-subnet-vlans 是嵌套 list 容器而非叶——按叶删除必须拒绝。
		if _, err := EncodeLeafDelete(vlanSpec(), vlanEntryOnlyKey(5), []string{"ip-subnet-vlans"}); err == nil {
			t.Error("want error for non-leaf target")
		}
	})
	t.Run("key leaf itself", func(t *testing.T) {
		// 删除定位键本身没有意义且危险——明确拒绝。
		if _, err := EncodeLeafDelete(vlanSpec(), vlanEntryOnlyKey(5), []string{"id"}); err == nil {
			t.Error("want error when deleting the key leaf")
		}
	})
	t.Run("nil container", func(t *testing.T) {
		if _, err := EncodeLeafDelete(vlanSpec(), (*huawei.HuaweiVlan_Vlan_Vlans)(nil), []string{"description"}); err == nil {
			t.Error("want error for nil container")
		}
	})
	t.Run("no key information", func(t *testing.T) {
		s := fakeSpec()
		s.Schema = func() schema.Node { return fakeSchemaNode(false) }
		name := "x"
		v := &fakeRoot{Entry: map[string]*fakeEntry{"x": {Name: &name}}}
		if _, err := EncodeLeafDelete(s, v, []string{"name"}); err == nil {
			t.Error("want error when key cannot be determined")
		}
	})
}

// TestEncodeLeafDeleteConcurrent：并发编码共享 Spec（R09，-race 验证）。
func TestEncodeLeafDeleteConcurrent(t *testing.T) {
	spec := vlanSpec()
	done := make(chan error, 8)
	for i := 0; i < 8; i++ {
		go func() {
			_, err := EncodeLeafDelete(spec, vlanEntryOnlyKey(9), []string{"description"})
			done <- err
		}()
	}
	for i := 0; i < 8; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
}
