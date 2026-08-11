package xmlcodec

import (
	"strings"
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/internal/testutil/hwfix"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
	"github.com/openconfig/ygot/ygot"
)

const (
	vlanNS = "urn:huawei:yang:huawei-vlan"
	ifmNS  = "urn:huawei:yang:huawei-ifm"
)

func vlanSpec() *Spec {
	return &Spec{
		Namespace: vlanNS,
		Schema:    irTestNode("/vlan/vlans"),
	}
}

func ifmSpec() *Spec {
	return &Spec{
		Namespace: ifmNS,
		Schema:    irTestNode("/ifm/interfaces"),
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
		Schema:    func() schema.Node { return fakeSchemaNode(true) },
	}
}

// fakeSchemaNode 经 IR DTO 构造合成 schema（IR 是 schema 包唯一公开树构建面）：
// fakes 容器 + entry list；withKey=false 抹掉 key（无 key 信息负路径用）。
func fakeSchemaNode(withKey bool) schema.Node {
	var keys []string
	if withKey {
		keys = []string{"name"}
	}
	m, err := schema.ModuleFromIR(schema.IRModule{
		Name: "fakes",
		Root: &schema.IRNode{Kind: "container", Name: "fakes", Path: "/fakes", Children: []*schema.IRNode{
			{Kind: "list", Name: "entry", Path: "/fakes/entry", Keys: keys, Children: []*schema.IRNode{
				{Kind: "leaf", Name: "name", Path: "/fakes/entry/name", LeafType: "string", IsKey: withKey},
			}},
		}},
	})
	if err != nil {
		panic(err)
	}
	return m.Root()
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
		s := &Spec{Namespace: "urn:x", Schema: func() schema.Node { return nil }}
		if _, err := Encode(s, hwfix.VlanMinimal()); err == nil {
			t.Error("want error for nil schema entry")
		}
	})
	t.Run("missing namespace", func(t *testing.T) {
		s := &Spec{Schema: irTestNode("/vlan/vlans")}
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

// TestEncode_EnumEmitsYANGName 是 XC-08 回归锚点：合法枚举 leaf SHALL 编码为其 YANG
// 值域名（真机据此校验），SHALL NOT 发整数。此缺陷曾潜伏于 vlan/ifm（golden 固化整数、
// 宽容 sim 测不出），由 acl 的 mandatory type 枚举在完备矩阵中暴露。
func TestEncode_EnumEmitsYANGName(t *testing.T) {
	id := uint16(1)
	v := &huawei.HuaweiVlan_Vlan_Vlans{
		Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
			1: {Id: &id, AdminStatus: huawei.HuaweiVlan_AdminStatus_up},
		},
	}
	out, err := Encode(vlanSpec(), v)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !strings.Contains(out, "<admin-status>up</admin-status>") {
		t.Errorf("枚举须发值域名 <admin-status>up</admin-status>\n实际: %s", out)
	}
	if strings.Contains(out, "<admin-status>2</admin-status>") {
		t.Errorf("枚举不得发整数\n实际: %s", out)
	}
}
