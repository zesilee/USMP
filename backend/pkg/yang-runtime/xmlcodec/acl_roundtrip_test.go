package xmlcodec

import (
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
)

// acl（/acl:acl，容器根，同 /bgp:bgp 走 plain-container）的编解码矩阵。本波次接入 BGP
// ACL group leafref 目标 groups/group(IPv4) + group6s/group6(IPv6) 的标量/枚举边界。首次
// 覆盖枚举 leaf（type/match-order）。深层 rule-* 子树与 ip-pools/port-pools 显式推迟。

const aclNS = "urn:huawei:yang:huawei-acl"

func aclSpec() *Spec {
	return &Spec{
		Namespace: aclNS,
		Schema:    func() *yang.Entry { return huawei.SchemaTree["HuaweiAcl_Acl"] },
	}
}

// group 直属 config-true 标量指针叶计数（identity+description+number+step；枚举 type/
// match-order 另计）。模型变更触发复审。
const aclGroupScalarPtrLeaves = 4

// TestAcl_Group_Shape：schema 驱动锁死 group 接入形状——直属 config-true 标量指针叶恰好 4，
// 深层 rule-* 仍为推迟容器（防悄悄扩面）。枚举 type/match-order 由往返测试覆盖。
func TestAcl_Group_Shape(t *testing.T) {
	root := huawei.SchemaTree["HuaweiAcl_Acl"]
	if root == nil {
		t.Fatal("HuaweiAcl_Acl schema 未解析")
	}
	gEntry := root.Dir["groups"].Dir["group"]
	if gEntry == nil {
		t.Fatal("group schema 未解析")
	}
	g := &huawei.HuaweiAcl_Acl_Groups_Group{}
	scalars, containers := countDirectConfigTrueScalars(t, reflect.ValueOf(g).Elem(), gEntry)
	if scalars != aclGroupScalarPtrLeaves {
		t.Fatalf("group 直属 config-true 标量指针叶 = %d，期望 %d（模型变更？须复审范围）", scalars, aclGroupScalarPtrLeaves)
	}
	foundRuleContainer := false
	for _, c := range containers {
		if strings.HasPrefix(c, "rule-") {
			foundRuleContainer = true
		}
	}
	if !foundRuleContainer {
		t.Errorf("推迟子容器 rule-* 未在 schema 中（模型漂移？）containers=%v", containers)
	}
}

// TestAcl_Group_Roundtrip：group+group6 标量/枚举边界全字段编码→解码 DeepEqual + 根
// namespace + 枚举值域名真值。
func TestAcl_Group_Roundtrip(t *testing.T) {
	orig := &huawei.HuaweiAcl_Acl{
		Groups:  &huawei.HuaweiAcl_Acl_Groups{},
		Group6S: &huawei.HuaweiAcl_Acl_Group6S{},
	}
	g, err := orig.Groups.NewGroup("G1")
	if err != nil {
		t.Fatalf("NewGroup: %v", err)
	}
	g.Type = huawei.HuaweiAcl_Group4Type_advance
	g.MatchOrder = huawei.HuaweiAcl_MatchOrder_config
	g.Description = ygot.String("g1 desc")
	g.Number = ygot.Uint32(3001)
	g6, _ := orig.Group6S.NewGroup6("G6")
	g6.Type = huawei.HuaweiAcl_Group6Type_basic

	xml, err := Encode(aclSpec(), orig)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !strings.HasPrefix(xml, `<acl xmlns="urn:huawei:yang:huawei-acl">`) {
		t.Errorf("根 namespace 形态错: %s", xml[:min(60, len(xml))])
	}
	// 枚举须编码为值域名（非整数）
	for _, want := range []string{"<type>advance</type>", "<match-order>config</match-order>", "<type>basic</type>"} {
		if !strings.Contains(xml, want) {
			t.Errorf("枚举编码缺 %q（须值域名非整数）\n实际: %s", want, xml)
		}
	}
	got := &huawei.HuaweiAcl_Acl{}
	if err := Decode(aclSpec(), []byte(xml), got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !reflect.DeepEqual(orig, got) {
		t.Fatalf("标量/枚举边界往返不等价（字段级丢失）\n原: %s\nXML: %s", mustJSON(orig), xml)
	}
}

// TestAcl_NegativePath_NoPanic：畸形/空 XML 解码不 panic（R08）。
func TestAcl_NegativePath_NoPanic(t *testing.T) {
	for _, raw := range []string{"", "<acl", "<other/>", "not xml"} {
		got := &huawei.HuaweiAcl_Acl{}
		_ = Decode(aclSpec(), []byte(raw), got)
	}
}

// TestAcl_Concurrent_EncodeDecode：并发 Encode/Decode 无数据竞态（R09，-race）。
func TestAcl_Concurrent_EncodeDecode(t *testing.T) {
	orig := &huawei.HuaweiAcl_Acl{Groups: &huawei.HuaweiAcl_Acl_Groups{}}
	g, _ := orig.Groups.NewGroup("G1")
	g.Type = huawei.HuaweiAcl_Group4Type_basic

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			xml, err := Encode(aclSpec(), orig)
			if err != nil {
				t.Errorf("Encode: %v", err)
				return
			}
			got := &huawei.HuaweiAcl_Acl{}
			if err := Decode(aclSpec(), []byte(xml), got); err != nil {
				t.Errorf("Decode: %v", err)
			}
		}()
	}
	wg.Wait()
}
