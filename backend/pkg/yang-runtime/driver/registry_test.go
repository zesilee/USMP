package driver

import (
	"strings"
	"sync"
	"testing"
)

func testDescriptor(vendor, module, token string) Descriptor {
	return Descriptor{
		Vendor:          vendor,
		Module:          module,
		ControllerToken: token,
		MatchRoute:      func(p string) bool { return strings.Contains(p, module+":") },
	}
}

// DR-01: 注册后按路径查得（注册序决定优先级）。
func TestRegistry_RegisterAndRoute(t *testing.T) {
	r := NewRegistry()
	r.Register(testDescriptor("huawei", "vlan", "vlan"))
	r.Register(testDescriptor("huawei", "ifm", "ifm"))

	d, ok := r.Route("/vlan:vlan/vlan:vlans")
	if !ok || d.Module != "vlan" || d.ControllerToken != "vlan" {
		t.Fatalf("vlan 路径应命中 vlan 描述符, got %+v ok=%v", d, ok)
	}
	if _, ok := r.Route("/route:routing"); ok {
		t.Fatal("无描述符覆盖的路径应返回 ok=false（调用方降级，R08）")
	}
}

// DR-01: 编解码查找只返回携带对应能力的描述符。
func TestRegistry_CodecLookupSkipsIncapable(t *testing.T) {
	r := NewRegistry()
	// 只有路由能力、无编解码。
	r.Register(testDescriptor("huawei", "vlan", "vlan"))

	if _, ok := r.DecoderFor("/vlan:vlan/vlan:vlans"); ok {
		t.Fatal("无 DecodeXML 的描述符不应作为解码器命中")
	}
	if _, ok := r.EncoderFor("/vlan:vlan/vlan:vlans"); ok {
		t.Fatal("无 NewStruct 的描述符不应作为编码器命中")
	}
}

// DR-01: 注册序即匹配序（先注册先匹配，对拍 if-链语义）。
func TestRegistry_FirstMatchWins(t *testing.T) {
	r := NewRegistry()
	all := Descriptor{Vendor: "huawei", Module: "first", ControllerToken: "first",
		MatchRoute: func(string) bool { return true }}
	r.Register(all)
	r.Register(testDescriptor("huawei", "vlan", "vlan"))

	if d, ok := r.Route("/vlan:vlan"); !ok || d.Module != "first" {
		t.Fatalf("先注册的全匹配描述符应先命中, got %+v", d)
	}
}

// R09: 并发注册与查询无数据竞态（-race 锁定）。
func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := NewRegistry()
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			r.Register(testDescriptor("huawei", "vlan", "vlan"))
		}()
		go func() {
			defer wg.Done()
			_, _ = r.Route("/vlan:vlan")
			_, _ = r.DecoderFor("/vlan:vlan")
			_, _ = r.EncoderFor("/vlan:vlan")
		}()
	}
	wg.Wait()
}

// DR-01: 全局缺省注册表可用（生产接线用 driver.Register/Route 包级门面）。
func TestRegistry_PackageLevelFacade(t *testing.T) {
	Register(testDescriptor("huawei", "pkgfacade-mod", "pkgfacade"))
	if _, ok := Route("/pkgfacade-mod:x"); !ok {
		t.Fatal("包级门面应操作全局缺省注册表")
	}
}
