package drivers

import (
	"strings"
	"sync"
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/driver"
)

// full-yang-onboarding（DR-06）参数化矩阵：plain-container 模块全量注册基线。
// 行 = 根容器名（= 模块路由 token = 控制台 /module/<root>）。缩水即红灯——
// 左树叶「可用」以此为后盾（LT-04）。
var fullOnboardRoots = []string{
	"acl",
	"analysis-collector",
	"anyflow",
	"arp",
	"bd",
	"bfd",
	"cfg",
	"devm",
	"driver",
	"dsa",
	"ecc",
	"evpn",
	"fib",
	"ftpc",
	"grpc",
	"hwtacacs",
	"ifm-trunk",
	"l3-multicast",
	"l3vpn",
	"lacp",
	"license",
	"lldp",
	"loadbalance",
	"m-lag",
	"mac-flapping-detect",
	"macsec",
	"microsegmentation",
	"mirror",
	"monitor-link",
	"mstp",
	"multicast",
	"mvpn",
	"nqa",
	"ntp",
	"nvo3",
	"openflow-agent",
	"ospfv2",
	"ospfv3",
	"packetevent",
	"qos",
	"routing",
	"routing-policy",
	"rsa",
	"sflow",
	"sm2",
	"snmp",
	"syslog",
	"system-resources-usage",
	"telemetry-system",
	"time-range",
	"tunnel-management",
	"unicast-forward",
	"vrrp",
	"vty",
	"vxlan-ext",
	"vxlan-path-detect",
	"xpl",
}

// TestFullOnboardingDescriptors 对每个模块断言 route/decode/encode 三查表命中、
// 锚点/命名空间/SchemaTree 入口齐备（T02b 参数化矩阵之注册表不变量）。
func TestFullOnboardingDescriptors(t *testing.T) {
	seenNS := map[string]string{}
	for _, root := range fullOnboardRoots {
		root := root
		t.Run(root, func(t *testing.T) {
			anchor := "/" + root + ":" + root
			d, ok := driver.Route(anchor)
			if !ok || d.Module != root {
				t.Fatalf("Route(%q) 未命中本模块: ok=%v module=%q", anchor, ok, d.Module)
			}
			if _, ok := driver.DecoderFor(anchor); !ok {
				t.Fatalf("DecoderFor(%q) 未命中", anchor)
			}
			enc, ok := driver.EncoderFor(anchor)
			if !ok {
				t.Fatalf("EncoderFor(%q) 未命中", anchor)
			}
			if enc.EncodeAnchor != anchor {
				t.Fatalf("EncodeAnchor=%q, want %q", enc.EncodeAnchor, anchor)
			}
			if enc.XML == nil || enc.XML.Namespace == "" {
				t.Fatalf("XML spec/namespace 缺失")
			}
			if prev, dup := seenNS[enc.XML.Namespace]; dup {
				t.Fatalf("namespace %q 与 %q 重复", enc.XML.Namespace, prev)
			}
			seenNS[enc.XML.Namespace] = root
			v := enc.NewStruct()
			if v == nil {
				t.Fatal("NewStruct 返回 nil")
			}
			if enc.XML.Schema == nil || enc.XML.Schema() == nil {
				t.Fatalf("SchemaTree 入口缺失（构造子类型未在生成闭包中）")
			}
			if got := enc.XML.Schema().Name; got != root {
				t.Fatalf("SchemaTree 入口名=%q, want 根容器 %q", got, root)
			}
		})
	}
}

// TestRootNameConventionRegression 锁定路径口径修复：根名≠YANG prefix 的模块
// 必须按根名命中（此前 tnlm:/rtp: 锚导致控制台写链路不可达）；ni 双口径兼容
// （业务意图编排层仍以 /ni: 调用）。
func TestRootNameConventionRegression(t *testing.T) {
	for _, p := range []string{
		"/tunnel-management:tunnel-management/tunnel-management:tunnel-policys",
		"/routing-policy:routing-policy/routing-policy:policy-definitions",
	} {
		if _, ok := driver.EncoderFor(p); !ok {
			t.Fatalf("EncoderFor(%q) 未命中（根名口径断链未修复）", p)
		}
	}
	for _, p := range []string{"/ni:network-instance", "/network-instance:network-instance"} {
		if _, ok := driver.EncoderFor(p); !ok {
			t.Fatalf("EncoderFor(%q) 未命中（ni 双口径兼容缺失）", p)
		}
	}
}

// TestFullOnboardingSchemaRoots 断言生成闭包把全部模块根容器纳入 fakeroot
// （任务 1.5：schema 面基线）。
func TestFullOnboardingSchemaRoots(t *testing.T) {
	s, err := huawei.Schema()
	if err != nil {
		t.Fatalf("Schema(): %v", err)
	}
	root := s.SchemaTree["Device"]
	if root == nil {
		t.Fatal("fakeroot Device 缺失")
	}
	for _, r := range fullOnboardRoots {
		if root.Dir[r] == nil {
			t.Fatalf("fakeroot 缺根容器 %q", r)
		}
	}
}

// TestFullOnboardingLookupRace 并发查表（R09）。
func TestFullOnboardingLookupRace(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, root := range fullOnboardRoots {
				anchor := "/" + root + ":" + root
				driver.Route(anchor)
				driver.EncoderFor(anchor)
				driver.DecoderFor(strings.ToLower(anchor))
			}
		}()
	}
	wg.Wait()
}
