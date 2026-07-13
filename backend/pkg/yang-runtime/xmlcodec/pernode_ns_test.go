package xmlcodec

import (
	"strings"
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
)

// XC-06 per-node namespace：augment 跨模块树（huawei-bgp peers/afs augment 到
// huawei-network-instance 根下）在模块边界发正确 namespace。正确性靠 encode 输出的
// namespace 真值断言（sim/decode namespace-宽容，往返/集成证不出 namespace 错）。

const niBgpNS = "urn:huawei:yang:huawei-bgp"

// niSpecWithNS 是带 per-node namespace 解析表的 network-instance Spec。
func niSpecWithNS() *Spec {
	return &Spec{
		Namespace: niNS, // urn:huawei:yang:huawei-network-instance（见 ni_roundtrip_test.go）
		Schema:    func() *yang.Entry { return huawei.SchemaTree["HuaweiNetworkInstance_NetworkInstance"] },
		Namespaces: map[string]string{
			"huawei-network-instance": niNS,
			"huawei-bgp":              niBgpNS,
		},
	}
}

// niWithBgpAugment 构造一条含 huawei-bgp augment 子树（bgp/base-process/router-id）
// 的 network-instance——即 2a peers 所在的跨模块结构。
func niWithBgpAugment() *huawei.HuaweiNetworkInstance_NetworkInstance {
	return &huawei.HuaweiNetworkInstance_NetworkInstance{
		Instances: &huawei.HuaweiNetworkInstance_NetworkInstance_Instances{
			Instance: map[string]*huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance{
				"_public_": {
					Name: ygot.String("_public_"),
					Bgp: &huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance_Bgp{
						BaseProcess: &huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance_Bgp_BaseProcess{
							RouterId: ygot.String("1.1.1.1"),
						},
					},
				},
			},
		},
	}
}

func TestXC06_AugmentSubtreeGetsModuleNamespace(t *testing.T) {
	out, err := Encode(niSpecWithNS(), niWithBgpAugment())
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	// 模块边界 <bgp> 必须带 huawei-bgp namespace
	if !strings.Contains(out, `<bgp xmlns="urn:huawei:yang:huawei-bgp">`) {
		t.Errorf("bgp 子树缺 huawei-bgp namespace（真机会拒绝）\n实际: %s", out)
	}
	// 根仍是 ni namespace
	if !strings.HasPrefix(out, `<network-instance xmlns="urn:huawei:yang:huawei-network-instance">`) {
		t.Errorf("根 namespace 错\n实际: %s", out)
	}
	// 同级 ni 字段（name）不另发 xmlns（继承 ni 根）
	if strings.Contains(out, `<name xmlns=`) {
		t.Errorf("ni 原生字段 name 不应另发 xmlns\n实际: %s", out)
	}
	// bgp 子节点（base-process/router-id）继承 bgp namespace，不重复发
	if strings.Contains(out, `<base-process xmlns=`) || strings.Contains(out, `<router-id xmlns=`) {
		t.Errorf("bgp 子节点不应重复发 xmlns（继承 <bgp>）\n实际: %s", out)
	}
}

// XC-06 场景4：混合 namespace 报文解码鲁棒——encode（ni 根 + bgp 子树各自 namespace）
// → decode 按 local 名穿透 namespace，跨模块 augment 字段（bgp/base-process/router-id）
// 真值幸存。
func TestXC06_MixedNamespace_DecodeRoundtrip(t *testing.T) {
	orig := niWithBgpAugment()
	xml, err := Encode(niSpecWithNS(), orig)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	got := &huawei.HuaweiNetworkInstance_NetworkInstance{}
	if err := Decode(niSpecWithNS(), []byte(xml), got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	inst := got.Instances.Instance["_public_"]
	if inst == nil || inst.Bgp == nil || inst.Bgp.BaseProcess == nil ||
		inst.Bgp.BaseProcess.RouterId == nil || *inst.Bgp.BaseProcess.RouterId != "1.1.1.1" {
		t.Fatalf("跨 namespace augment 字段未解码幸存: %#v", inst)
	}
}

// 零回归：单模块树（bgp 容器根，Namespaces 未登记差异模块）编码不新增 xmlns。
func TestXC06_SingleModuleTree_NoExtraXmlns(t *testing.T) {
	// bgpSpec() 无 Namespaces → 现状行为
	out, err := Encode(bgpSpec(), &huawei.HuaweiBgp_Bgp{
		BaseProcess: &huawei.HuaweiBgp_Bgp_BaseProcess{As: ygot.String("100")},
	})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	// 仅根一个 xmlns，子节点无
	if strings.Count(out, "xmlns=") != 1 {
		t.Errorf("单模块树应仅根发 1 个 xmlns，实际 %d 个\n%s", strings.Count(out, "xmlns="), out)
	}
	if want := `<bgp xmlns="urn:huawei:yang:huawei-bgp"><base-process><as>100</as></base-process></bgp>`; out != want {
		t.Errorf("单模块树输出漂移\n期望: %s\n实际: %s", want, out)
	}
}

// 缺省 Namespaces（nil）等价现状：以 ni 根编码不带 bgp 子树时输出与无解析表一致。
func TestXC06_NilNamespaces_LegacyBehavior(t *testing.T) {
	specNil := &Spec{Namespace: niNS, Schema: func() *yang.Entry { return huawei.SchemaTree["HuaweiNetworkInstance_NetworkInstance"] }}
	v := &huawei.HuaweiNetworkInstance_NetworkInstance{
		Global: &huawei.HuaweiNetworkInstance_NetworkInstance_Global{CfgRouterId: ygot.String("9.9.9.9")},
	}
	out, err := Encode(specNil, v)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if strings.Count(out, "xmlns=") != 1 {
		t.Errorf("nil Namespaces 应仅根 1 个 xmlns，实际 %d\n%s", strings.Count(out, "xmlns="), out)
	}
}
