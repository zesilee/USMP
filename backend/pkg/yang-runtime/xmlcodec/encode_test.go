package xmlcodec

import (
	"strings"
	"testing"

	"github.com/openconfig/goyang/pkg/yang"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/internal/testutil/hwfix"
	"github.com/openconfig/ygot/ygot"
)

const (
	vlanNS = "urn:huawei:params:xml:ns:yang:huawei-vlan"
	ifmNS  = "urn:huawei:params:xml:ns:yang:huawei-ifm"
)

func vlanSpec() *Spec {
	return &Spec{
		Namespace: vlanNS,
		Schema:    func() *yang.Entry { return huawei.SchemaTree["HuaweiVlan_Vlan_Vlans"] },
	}
}

func ifmSpec() *Spec {
	return &Spec{
		Namespace: ifmNS,
		Schema:    func() *yang.Entry { return huawei.SchemaTree["HuaweiIfm_Ifm_Interfaces"] },
	}
}

// TestEncodeMatchesGolden 对拍任务 1.3 冻结的 legacy builder golden（XC-01 核心等价断言）。
func TestEncodeMatchesGolden(t *testing.T) {
	tests := []struct {
		golden string
		spec   *Spec
		value  ygot.GoStruct
	}{
		{"vlan_full", vlanSpec(), hwfix.VlanFull()},
		{"vlan_minimal", vlanSpec(), hwfix.VlanMinimal()},
		{"vlan_empty", vlanSpec(), hwfix.VlanEmpty()},
		{"vlan_escape", vlanSpec(), hwfix.VlanEscape()},
		{"ifm_full", ifmSpec(), hwfix.IfmFull()},
		{"ifm_minimal", ifmSpec(), hwfix.IfmMinimal()},
		{"ifm_empty", ifmSpec(), hwfix.IfmEmpty()},
	}
	for _, tt := range tests {
		t.Run(tt.golden, func(t *testing.T) {
			out, err := Encode(tt.spec, tt.value)
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			canon, err := Canonicalize([]byte(out))
			if err != nil {
				t.Fatalf("canonicalize engine output: %v\nraw: %s", err, out)
			}
			if *hwfix.Update {
				hwfix.WriteGolden(t, tt.golden, canon)
				return
			}
			if want := hwfix.Golden(t, tt.golden); canon != want {
				t.Errorf("engine output != golden %s\n got: %s\nwant: %s\nraw: %s", tt.golden, canon, want, out)
			}
		})
	}
}

// TestEncodeRawShape 在原始输出上断言规范化会抹掉的性质：根 namespace、
// 空容器自闭合、key 叶为条目首元素（D3：定向断言补 golden 全排序盲区）。
func TestEncodeRawShape(t *testing.T) {
	t.Run("minimal exact", func(t *testing.T) {
		out, err := Encode(vlanSpec(), hwfix.VlanMinimal())
		if err != nil {
			t.Fatal(err)
		}
		want := `<vlan xmlns="` + vlanNS + `"><vlans><vlan><id>30</id></vlan></vlans></vlan>`
		if out != want {
			t.Errorf("got %s\nwant %s", out, want)
		}
	})
	t.Run("empty self-closing", func(t *testing.T) {
		out, err := Encode(ifmSpec(), hwfix.IfmEmpty())
		if err != nil {
			t.Fatal(err)
		}
		if want := `<ifm xmlns="` + ifmNS + `"><interfaces/></ifm>`; out != want {
			t.Errorf("got %s want %s", out, want)
		}
	})
	t.Run("key first in every entry", func(t *testing.T) {
		out, err := Encode(vlanSpec(), hwfix.VlanFull())
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(out, "<vlan>") != strings.Count(out, "<vlan><id>") {
			t.Errorf("some <vlan> entry does not lead with key <id>: %s", out)
		}
		ifmOut, err := Encode(ifmSpec(), hwfix.IfmFull())
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(ifmOut, "<interface>") != strings.Count(ifmOut, "<interface><name>") {
			t.Errorf("some <interface> entry does not lead with key <name>: %s", ifmOut)
		}
	})
	t.Run("suppression emitted exactly once", func(t *testing.T) {
		out, err := Encode(vlanSpec(), hwfix.VlanFull())
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Count(out, "<suppression>"); got != 1 {
			t.Errorf("suppression emitted %d times, want 1 (legacy duplicate bug must not be reproduced)", got)
		}
	})
	t.Run("map key fallback when key leaf nil", func(t *testing.T) {
		out, err := Encode(ifmSpec(), &huawei.HuaweiIfm_Ifm_Interfaces{
			Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{
				"GE0/0/9": {Mtu: hwuint32(1500)},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, "<interface><name>GE0/0/9</name>") {
			t.Errorf("map key not used as key-leaf fallback: %s", out)
		}
	})
	t.Run("escape", func(t *testing.T) {
		out, err := Encode(vlanSpec(), hwfix.VlanEscape())
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out, `a<b`) || !strings.Contains(out, "&lt;") {
			t.Errorf("special characters not escaped: %s", out)
		}
	})
}

// --- 不支持形态与坏输入（XC-01 负路径，R08 明确报错不 panic）---

type fakeRoot struct {
	Entry map[string]*fakeEntry `path:"entry"`
}

func (*fakeRoot) IsYANGGoStruct() {}

type fakeEntry struct {
	Name *string  `path:"name"`
	Bad  chan int `path:"bad"`
}

func (*fakeEntry) IsYANGGoStruct() {}

func fakeSpec() *Spec {
	return &Spec{
		Namespace: "urn:fake",
		Schema: func() *yang.Entry {
			return &yang.Entry{
				Name: "fakes",
				Dir: map[string]*yang.Entry{
					"entry": {Name: "entry", Key: "name", ListAttr: &yang.ListAttr{}},
				},
			}
		},
	}
}

func TestEncodeErrors(t *testing.T) {
	name := "x"
	t.Run("unsupported field form", func(t *testing.T) {
		v := &fakeRoot{Entry: map[string]*fakeEntry{"x": {Name: &name, Bad: make(chan int)}}}
		if _, err := Encode(fakeSpec(), v); err == nil || !strings.Contains(err.Error(), "bad") {
			t.Errorf("want explicit unsupported-form error naming the field, got %v", err)
		}
	})
	t.Run("nil value", func(t *testing.T) {
		if _, err := Encode(vlanSpec(), (*huawei.HuaweiVlan_Vlan_Vlans)(nil)); err == nil {
			t.Error("want error for nil container")
		}
	})
	t.Run("nil spec schema", func(t *testing.T) {
		s := &Spec{Namespace: "urn:x", Schema: func() *yang.Entry { return nil }}
		if _, err := Encode(s, hwfix.VlanMinimal()); err == nil {
			t.Error("want error for nil schema entry")
		}
	})
	t.Run("missing namespace", func(t *testing.T) {
		s := &Spec{Schema: func() *yang.Entry { return huawei.SchemaTree["HuaweiVlan_Vlan_Vlans"] }}
		if _, err := Encode(s, hwfix.VlanMinimal()); err == nil {
			t.Error("want error for missing namespace")
		}
	})
}

// TestEncodeConcurrent 并发编码共享 Spec 与 fixture（R09，-race 验证）。
func TestEncodeConcurrent(t *testing.T) {
	spec, v := vlanSpec(), hwfix.VlanFull()
	done := make(chan error, 8)
	for i := 0; i < 8; i++ {
		go func() {
			_, err := Encode(spec, v)
			done <- err
		}()
	}
	for i := 0; i < 8; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
}

func hwuint32(v uint32) *uint32 { return &v }
