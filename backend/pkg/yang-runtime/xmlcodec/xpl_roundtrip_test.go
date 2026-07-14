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

// xpl（/xpl:xpl，容器根，同 /bgp:bgp 走 plain-container）的编解码矩阵。本波次接入 BGP
// route-filter leafref 目标子树 route-filters/route-filter(name+content)。xpl 其他策略
// list（as-path/community/prefix/rd/large-community 等）非本 leafref 目标，显式推迟。

const xplNS = "urn:huawei:yang:huawei-xpl"

func xplSpec() *Spec {
	return &Spec{
		Namespace: xplNS,
		Schema:    func() *yang.Entry { return huawei.SchemaTree["HuaweiXpl_Xpl"] },
	}
}

// route-filter 条目直属 config-true 标量 leaf 计数（name + content）。模型变更触发复审。
const xplRouteFilterScalarLeaves = 2

// TestXpl_RouteFilter_Shape：schema 驱动锁死 route-filter 接入形状——直属 config-true
// 标量恰好 2（name+content），无深层子容器（防悄悄扩面）。
func TestXpl_RouteFilter_Shape(t *testing.T) {
	root := huawei.SchemaTree["HuaweiXpl_Xpl"]
	if root == nil {
		t.Fatal("HuaweiXpl_Xpl schema 未解析")
	}
	rfEntry := root.Dir["route-filters"].Dir["route-filter"]
	if rfEntry == nil {
		t.Fatal("route-filter schema 未解析")
	}
	rf := &huawei.HuaweiXpl_Xpl_RouteFilters_RouteFilter{}
	scalars, containers := countDirectConfigTrueScalars(t, reflect.ValueOf(rf).Elem(), rfEntry)
	if scalars != xplRouteFilterScalarLeaves {
		t.Fatalf("route-filter 直属 config-true 标量 = %d，期望 %d（模型变更？须复审范围）", scalars, xplRouteFilterScalarLeaves)
	}
	if len(containers) != 0 {
		t.Fatalf("route-filter 不应有 config-true 子容器，实得 %v", containers)
	}
}

// TestXpl_RouteFilter_Roundtrip：route-filter 全字段编码→解码 DeepEqual + 根 namespace 真值。
func TestXpl_RouteFilter_Roundtrip(t *testing.T) {
	orig := &huawei.HuaweiXpl_Xpl{RouteFilters: &huawei.HuaweiXpl_Xpl_RouteFilters{}}
	rf, err := orig.RouteFilters.NewRouteFilter("RF1")
	if err != nil {
		t.Fatalf("NewRouteFilter: %v", err)
	}
	rf.Content = ygot.String("if-match protocol bgp")

	xml, err := Encode(xplSpec(), orig)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !strings.HasPrefix(xml, `<xpl xmlns="urn:huawei:yang:huawei-xpl">`) {
		t.Errorf("根 namespace 形态错: %s", xml[:min(70, len(xml))])
	}
	got := &huawei.HuaweiXpl_Xpl{}
	if err := Decode(xplSpec(), []byte(xml), got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !reflect.DeepEqual(orig, got) {
		t.Fatalf("route-filter 往返不等价（字段级丢失）\n原: %s\nXML: %s", mustJSON(orig), xml)
	}
}

// TestXpl_Boundary_ContentLengthEdges：content(1..16380) 下界/上界合法值编解码往返无损。
func TestXpl_Boundary_ContentLengthEdges(t *testing.T) {
	for _, content := range []string{"x", strings.Repeat("c", 16380)} {
		orig := &huawei.HuaweiXpl_Xpl{RouteFilters: &huawei.HuaweiXpl_Xpl_RouteFilters{}}
		rf, _ := orig.RouteFilters.NewRouteFilter("RF1")
		rf.Content = ygot.String(content)
		xml, err := Encode(xplSpec(), orig)
		if err != nil {
			t.Fatalf("Encode(content=%d): %v", len(content), err)
		}
		got := &huawei.HuaweiXpl_Xpl{}
		if err := Decode(xplSpec(), []byte(xml), got); err != nil {
			t.Fatalf("Decode(content=%d): %v", len(content), err)
		}
		rrf := got.RouteFilters.RouteFilter["RF1"]
		if rrf == nil || rrf.Content == nil || *rrf.Content != content {
			t.Fatalf("边界长度往返丢失: content=%d", len(content))
		}
	}
}

// TestXpl_NegativePath_NoPanic：畸形/空 XML 解码不 panic（R08）。
func TestXpl_NegativePath_NoPanic(t *testing.T) {
	for _, raw := range []string{"", "<xpl", "<other/>", "not xml"} {
		got := &huawei.HuaweiXpl_Xpl{}
		_ = Decode(xplSpec(), []byte(raw), got)
	}
}

// TestXpl_Concurrent_EncodeDecode：并发 Encode/Decode 无数据竞态（R09，-race）。
func TestXpl_Concurrent_EncodeDecode(t *testing.T) {
	orig := &huawei.HuaweiXpl_Xpl{RouteFilters: &huawei.HuaweiXpl_Xpl_RouteFilters{}}
	rf, _ := orig.RouteFilters.NewRouteFilter("RF1")
	rf.Content = ygot.String("if-match protocol bgp")

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			xml, err := Encode(xplSpec(), orig)
			if err != nil {
				t.Errorf("Encode: %v", err)
				return
			}
			got := &huawei.HuaweiXpl_Xpl{}
			if err := Decode(xplSpec(), []byte(xml), got); err != nil {
				t.Errorf("Decode: %v", err)
			}
		}()
	}
	wg.Wait()
}
