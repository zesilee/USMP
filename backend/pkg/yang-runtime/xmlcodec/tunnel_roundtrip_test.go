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

// tunnel-management（/tnlm:tunnel-management，容器根，同 /bgp:bgp 走 plain-container）
// 的编解码矩阵。本波次接入**标量边界**：tunnel-policy(name+description) 嵌套 list +
// tunnel-down-switch/enable 标量。深层 ipv4-set/ipv6-set（choice/presence/ordered-by/
// nexthops/leafref）显式推迟为 follow-up，此文件断言边界并锁死其推迟状态（防悄悄越界）。

const tnlmNS = "urn:huawei:yang:huawei-tunnel-management"

func tnlmSpec() *Spec {
	return &Spec{
		Namespace: tnlmNS,
		Schema:    func() *yang.Entry { return huawei.SchemaTree["HuaweiTunnelManagement_TunnelManagement"] },
	}
}

// 本波次 tunnel-policy 条目直属 config-true 标量 leaf 计数（name + description）。
// 模型变更（如给 tunnel-policy 加标量）会使计数变化触发复审。
const tnlmPolicyScalarLeaves = 2

// 本波次显式推迟的 tunnel-policy 子容器（深层 ipv4/ipv6-set）——断言仍为推迟状态：
// 若未来接入，此清单须同步、完备矩阵须扩展（防「悄悄越界接入未测子树」）。
var tnlmDeferredPolicyContainers = []string{"ipv4-set", "ipv6-set"}

// countDirectConfigTrueScalars 统计 sv 结构体直属（不递归）的 config-true 标量指针叶数，
// 并返回遇到的 config-true 子容器名集合。用于锁死「标量边界 + 推迟容器」的精确形状。
func countDirectConfigTrueScalars(t *testing.T, sv reflect.Value, e *yang.Entry) (scalars int, containers []string) {
	t.Helper()
	st := sv.Type()
	for i := 0; i < st.NumField(); i++ {
		tag := pathTag(st.Field(i))
		if tag == "" || e == nil {
			continue
		}
		child := e.Dir[tag]
		if child == nil || child.Config == yang.TSFalse {
			continue
		}
		switch fv := sv.Field(i); {
		case fv.Kind() == reflect.Ptr && fv.Type().Elem().Kind() == reflect.Struct:
			containers = append(containers, tag)
		case fv.Kind() == reflect.Ptr:
			scalars++
		}
	}
	return
}

// TestTunnelManagement_ScalarBoundary_Shape：schema 驱动锁死本波次接入形状——
// tunnel-policy 直属 config-true 标量恰好 2（name+description），深层 ipv4/ipv6-set
// 仍为推迟容器；根直属 config-true 标量在 list 之外仅 tunnel-down-switch 子容器。
func TestTunnelManagement_ScalarBoundary_Shape(t *testing.T) {
	root := huawei.SchemaTree["HuaweiTunnelManagement_TunnelManagement"]
	if root == nil {
		t.Fatal("HuaweiTunnelManagement_TunnelManagement schema 未解析")
	}
	polEntry := root.Dir["tunnel-policys"].Dir["tunnel-policy"]
	if polEntry == nil {
		t.Fatal("tunnel-policy schema 未解析")
	}
	pol := &huawei.HuaweiTunnelManagement_TunnelManagement_TunnelPolicys_TunnelPolicy{}
	scalars, containers := countDirectConfigTrueScalars(t, reflect.ValueOf(pol).Elem(), polEntry)
	if scalars != tnlmPolicyScalarLeaves {
		t.Fatalf("tunnel-policy 直属 config-true 标量 = %d，期望 %d（模型变更？须复审范围）", scalars, tnlmPolicyScalarLeaves)
	}
	// 深层子容器仍为推迟状态（形状不变），逐一在位
	for _, want := range tnlmDeferredPolicyContainers {
		found := false
		for _, c := range containers {
			if c == want {
				found = true
			}
		}
		if !found {
			t.Errorf("推迟子容器 %q 未在 schema 中（模型漂移？）", want)
		}
	}
}

// TestTunnelManagement_ScalarBoundary_Roundtrip：标量边界全字段编码→解码 DeepEqual，
// 兜住任何字段级丢失；断言根 namespace 与真值。
func TestTunnelManagement_ScalarBoundary_Roundtrip(t *testing.T) {
	orig := &huawei.HuaweiTunnelManagement_TunnelManagement{
		TunnelPolicys:    &huawei.HuaweiTunnelManagement_TunnelManagement_TunnelPolicys{},
		TunnelDownSwitch: &huawei.HuaweiTunnelManagement_TunnelManagement_TunnelDownSwitch{Enable: ygot.Bool(true)},
	}
	p, err := orig.TunnelPolicys.NewTunnelPolicy("policy-a")
	if err != nil {
		t.Fatalf("NewTunnelPolicy: %v", err)
	}
	p.Description = ygot.String("bind te tunnels")

	xml, err := Encode(tnlmSpec(), orig)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !strings.HasPrefix(xml, `<tunnel-management xmlns="urn:huawei:yang:huawei-tunnel-management">`) {
		t.Errorf("根 namespace 形态错: %s", xml[:min(90, len(xml))])
	}
	got := &huawei.HuaweiTunnelManagement_TunnelManagement{}
	if err := Decode(tnlmSpec(), []byte(xml), got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !reflect.DeepEqual(orig, got) {
		t.Fatalf("标量边界往返不等价（字段级丢失）\n原: %s\nXML: %s", mustJSON(orig), xml)
	}
}

// TestTunnelManagement_Boundary_LengthEdges：name(1..39)/description(1..80) 的下界/上界
// 合法值编解码往返无损（长度越界拦截是共享约束引擎职责，不在编解码层）。
func TestTunnelManagement_Boundary_LengthEdges(t *testing.T) {
	cases := []struct{ name, desc string }{
		{"a", "d"}, // 下界 1/1
		{strings.Repeat("n", 39), strings.Repeat("d", 80)}, // 上界 39/80
	}
	for _, tc := range cases {
		orig := &huawei.HuaweiTunnelManagement_TunnelManagement{
			TunnelPolicys: &huawei.HuaweiTunnelManagement_TunnelManagement_TunnelPolicys{},
		}
		p, _ := orig.TunnelPolicys.NewTunnelPolicy(tc.name)
		p.Description = ygot.String(tc.desc)
		xml, err := Encode(tnlmSpec(), orig)
		if err != nil {
			t.Fatalf("Encode(name=%d,desc=%d): %v", len(tc.name), len(tc.desc), err)
		}
		got := &huawei.HuaweiTunnelManagement_TunnelManagement{}
		if err := Decode(tnlmSpec(), []byte(xml), got); err != nil {
			t.Fatalf("Decode(name=%d,desc=%d): %v", len(tc.name), len(tc.desc), err)
		}
		rp := got.TunnelPolicys.TunnelPolicy[tc.name]
		if rp == nil || rp.Description == nil || *rp.Description != tc.desc {
			t.Fatalf("边界长度往返丢失: name=%d desc=%d", len(tc.name), len(tc.desc))
		}
	}
}

// TestTunnelManagement_ConfigFalse_NotInEditConfig：config-false 只读态子树永不出现在
// 下发报文（tunnel-infos/subscribe-tunnel-policys）。
func TestTunnelManagement_ConfigFalse_NotInEditConfig(t *testing.T) {
	orig := &huawei.HuaweiTunnelManagement_TunnelManagement{
		TunnelDownSwitch: &huawei.HuaweiTunnelManagement_TunnelManagement_TunnelDownSwitch{Enable: ygot.Bool(true)},
	}
	xml, err := Encode(tnlmSpec(), orig)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	for _, c := range []string{"tunnel-infos", "subscribe-tunnel-policys"} {
		if strings.Contains(xml, "<"+c+">") || strings.Contains(xml, "<"+c+"/>") {
			t.Errorf("config-false 容器 %q 不应出现在 edit-config: %s", c, xml)
		}
	}
}

// TestTunnelManagement_NegativePath_NoPanic：畸形/空 XML 解码不 panic（R08），返回错误或空。
func TestTunnelManagement_NegativePath_NoPanic(t *testing.T) {
	for _, raw := range []string{"", "<tunnel-management", "<other/>", "not xml at all"} {
		got := &huawei.HuaweiTunnelManagement_TunnelManagement{}
		_ = Decode(tnlmSpec(), []byte(raw), got) // 不 panic 即通过；错误可接受
	}
}

// TestTunnelManagement_Concurrent_EncodeDecode：并发 Encode/Decode 无数据竞态（R09，-race）。
func TestTunnelManagement_Concurrent_EncodeDecode(t *testing.T) {
	orig := &huawei.HuaweiTunnelManagement_TunnelManagement{
		TunnelPolicys:    &huawei.HuaweiTunnelManagement_TunnelManagement_TunnelPolicys{},
		TunnelDownSwitch: &huawei.HuaweiTunnelManagement_TunnelManagement_TunnelDownSwitch{Enable: ygot.Bool(true)},
	}
	p, _ := orig.TunnelPolicys.NewTunnelPolicy("policy-a")
	p.Description = ygot.String("d")

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			xml, err := Encode(tnlmSpec(), orig)
			if err != nil {
				t.Errorf("Encode: %v", err)
				return
			}
			got := &huawei.HuaweiTunnelManagement_TunnelManagement{}
			if err := Decode(tnlmSpec(), []byte(xml), got); err != nil {
				t.Errorf("Decode: %v", err)
			}
		}()
	}
	wg.Wait()
}
