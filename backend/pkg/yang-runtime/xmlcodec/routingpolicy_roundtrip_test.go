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

// routing-policy（/rtp:routing-policy，容器根，同 /bgp:bgp 走 plain-container）的编解码矩阵。
// 本波次接入 BGP import/export route-policy leafref 目标子树 policy-definitions/
// policy-definition 的标量边界（name + address-family-mismatch-deny）。深层 nodes 子树
// （conditions/actions，~1490 行）与其他 rtp filter 显式推迟，此文件锁死推迟状态。

const rtpNS = "urn:huawei:yang:huawei-routing-policy"

func rtpSpec() *Spec {
	return &Spec{
		Namespace: rtpNS,
		Schema:    func() *yang.Entry { return huawei.SchemaTree["HuaweiRoutingPolicy_RoutingPolicy"] },
	}
}

// policy-definition 直属 config-true 标量 leaf 计数（name + address-family-mismatch-deny）。
const rtpPolicyDefScalarLeaves = 2

// TestRoutingPolicy_PolicyDef_Shape：schema 驱动锁死接入形状——policy-definition 直属
// config-true 标量恰好 2，深层 nodes 仍为推迟容器（防悄悄扩面）。
func TestRoutingPolicy_PolicyDef_Shape(t *testing.T) {
	root := huawei.SchemaTree["HuaweiRoutingPolicy_RoutingPolicy"]
	if root == nil {
		t.Fatal("HuaweiRoutingPolicy_RoutingPolicy schema 未解析")
	}
	pdEntry := root.Dir["policy-definitions"].Dir["policy-definition"]
	if pdEntry == nil {
		t.Fatal("policy-definition schema 未解析")
	}
	pd := &huawei.HuaweiRoutingPolicy_RoutingPolicy_PolicyDefinitions_PolicyDefinition{}
	scalars, containers := countDirectConfigTrueScalars(t, reflect.ValueOf(pd).Elem(), pdEntry)
	if scalars != rtpPolicyDefScalarLeaves {
		t.Fatalf("policy-definition 直属 config-true 标量 = %d，期望 %d（模型变更？须复审范围）", scalars, rtpPolicyDefScalarLeaves)
	}
	foundNodes := false
	for _, c := range containers {
		if c == "nodes" {
			foundNodes = true
		}
	}
	if !foundNodes {
		t.Errorf("推迟子容器 nodes 未在 schema 中（模型漂移？）")
	}
}

// TestRoutingPolicy_PolicyDef_Roundtrip：标量边界全字段编码→解码 DeepEqual + 根 namespace。
func TestRoutingPolicy_PolicyDef_Roundtrip(t *testing.T) {
	orig := &huawei.HuaweiRoutingPolicy_RoutingPolicy{
		PolicyDefinitions: &huawei.HuaweiRoutingPolicy_RoutingPolicy_PolicyDefinitions{},
	}
	pd, err := orig.PolicyDefinitions.NewPolicyDefinition("RP1")
	if err != nil {
		t.Fatalf("NewPolicyDefinition: %v", err)
	}
	pd.AddressFamilyMismatchDeny = ygot.Bool(true)

	xml, err := Encode(rtpSpec(), orig)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !strings.HasPrefix(xml, `<routing-policy xmlns="urn:huawei:yang:huawei-routing-policy">`) {
		t.Errorf("根 namespace 形态错: %s", xml[:min(90, len(xml))])
	}
	got := &huawei.HuaweiRoutingPolicy_RoutingPolicy{}
	if err := Decode(rtpSpec(), []byte(xml), got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !reflect.DeepEqual(orig, got) {
		t.Fatalf("标量边界往返不等价（字段级丢失）\n原: %s\nXML: %s", mustJSON(orig), xml)
	}
}

// TestRoutingPolicy_NegativePath_NoPanic：畸形/空 XML 解码不 panic（R08）。
func TestRoutingPolicy_NegativePath_NoPanic(t *testing.T) {
	for _, raw := range []string{"", "<routing-policy", "<other/>", "not xml"} {
		got := &huawei.HuaweiRoutingPolicy_RoutingPolicy{}
		_ = Decode(rtpSpec(), []byte(raw), got)
	}
}

// TestRoutingPolicy_Concurrent_EncodeDecode：并发 Encode/Decode 无数据竞态（R09，-race）。
func TestRoutingPolicy_Concurrent_EncodeDecode(t *testing.T) {
	orig := &huawei.HuaweiRoutingPolicy_RoutingPolicy{
		PolicyDefinitions: &huawei.HuaweiRoutingPolicy_RoutingPolicy_PolicyDefinitions{},
	}
	pd, _ := orig.PolicyDefinitions.NewPolicyDefinition("RP1")
	pd.AddressFamilyMismatchDeny = ygot.Bool(true)

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			xml, err := Encode(rtpSpec(), orig)
			if err != nil {
				t.Errorf("Encode: %v", err)
				return
			}
			got := &huawei.HuaweiRoutingPolicy_RoutingPolicy{}
			if err := Decode(rtpSpec(), []byte(xml), got); err != nil {
				t.Errorf("Decode: %v", err)
			}
		}()
	}
	wg.Wait()
}
