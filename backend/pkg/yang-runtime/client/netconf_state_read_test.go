package client

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/openconfig/ygot/ygot"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	yangdriver "github.com/leezesi/usmp/backend/pkg/yang-runtime/driver"
)

// DP-09 NETCONF <get> 状态读：WithStateData 置位发 <get>（携由 path 构造的
// subtree filter），缺省仍 get-config；断线自愈语义与既有读一致。

const testIfmStateXML = `<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm">
  <interfaces>
    <interface>
      <name>GE0/0/1</name>
      <dynamic><oper-status>1</oper-status><mac-address>00:aa:bb:cc:dd:ee</mac-address></dynamic>
    </interface>
  </interfaces>
</ifm>`

// WithStateData 置位 GetOptions.IncludeState；缺省 false。
func TestWithStateDataOption(t *testing.T) {
	opts := &GetOptions{}
	if opts.IncludeState {
		t.Fatal("IncludeState must default to false")
	}
	WithStateData().Apply(opts)
	if !opts.IncludeState {
		t.Fatal("WithStateData must set IncludeState")
	}
}

// path → subtree filter 体：模块命名空间经驱动注册表解析（测试二进制已注册
// huawei 描述符），未注册模块降级为无命名空间通配；list 谓词剥除。
func TestConstructSubtreeFilter(t *testing.T) {
	cases := []struct {
		name string
		path string
		want string
	}{
		{
			"registered module nested path",
			"/ifm:ifm/ifm:interfaces",
			`<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm"><interfaces/></ifm>`,
		},
		{
			"anchor root only",
			"/ifm:ifm",
			`<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm"/>`,
		},
		{
			"unregistered module falls back to no namespace",
			"/nosuch:nosuch/things",
			`<nosuch><things/></nosuch>`,
		},
		{
			"predicate stripped",
			"/ifm:ifm/ifm:interfaces/interface[name='GE0/0/1']",
			`<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm"><interfaces><interface/></interfaces></ifm>`,
		},
		{"empty path", "/", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := constructSubtreeFilter(tc.path); got != tc.want {
				t.Fatalf("constructSubtreeFilter(%q)\n got: %s\nwant: %s", tc.path, got, tc.want)
			}
		})
	}
}

// WithStateData → <get> 回读含状态子树；缺省 Get 不含（get-config 隔离）。
func TestNETCONFClient_GetWithStateData(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim := startSim(t)
	if err := sim.SetStateDataXML([]byte(testIfmStateXML)); err != nil {
		t.Fatalf("SetStateDataXML: %v", err)
	}
	c := newSimClient(t, sim)
	ctx := context.Background()

	res, err := c.Get(ctx, "/ifm:ifm/ifm:interfaces", WithStateData())
	if err != nil {
		t.Fatalf("Get with state: %v", err)
	}
	data := fmt.Sprintf("%s", res.Data)
	if !strings.Contains(data, "<oper-status>1</oper-status>") {
		t.Fatalf("state get missing merged state leaf: %.500s", data)
	}
	if !strings.Contains(data, "<mtu>1500</mtu>") {
		t.Fatalf("state get missing config leaf: %.500s", data)
	}

	plain, err := c.Get(ctx, "/ifm:ifm/ifm:interfaces")
	if err != nil {
		t.Fatalf("plain get: %v", err)
	}
	if strings.Contains(fmt.Sprintf("%s", plain.Data), "dynamic") {
		t.Fatalf("plain get must stay get-config (no state data): %.500s", plain.Data)
	}
}

// <get> 回读经通用 xmlcodec 解码：Dynamic 状态容器落入 ygot struct（2.3）。
func TestNETCONFClient_GetWithStateData_DecodesIntoYgot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim := startSim(t)
	if err := sim.SetStateDataXML([]byte(testIfmStateXML)); err != nil {
		t.Fatalf("SetStateDataXML: %v", err)
	}
	c := newSimClient(t, sim)

	res, err := c.Get(context.Background(), "/ifm:ifm/ifm:interfaces", WithStateData())
	if err != nil {
		t.Fatalf("Get with state: %v", err)
	}
	d, ok := yangdriver.DecoderFor("/ifm:ifm/ifm:interfaces")
	if !ok {
		t.Fatal("ifm decoder not registered")
	}
	raw, ok := res.Data.([]byte)
	if !ok {
		t.Fatalf("expected []byte data, got %T", res.Data)
	}
	gs, err := d.DecodeXML(raw)
	if err != nil {
		t.Fatalf("DecodeXML: %v", err)
	}
	ifaces, ok := gs.(*huawei.HuaweiIfm_Ifm_Interfaces)
	if !ok {
		t.Fatalf("unexpected decode type %T", gs)
	}
	entry := ifaces.Interface["GE0/0/1"]
	if entry == nil {
		t.Fatalf("interface GE0/0/1 missing after decode: %v", ifaces.Interface)
	}
	if entry.Dynamic == nil {
		t.Fatal("Dynamic state container not decoded")
	}
	if entry.Dynamic.MacAddress == nil || *entry.Dynamic.MacAddress != "00:aa:bb:cc:dd:ee" {
		t.Fatalf("mac-address state leaf not decoded: %v", entry.Dynamic.MacAddress)
	}
	if entry.Dynamic.OperStatus == huawei.HuaweiIfm_PortStatus_UNSET {
		t.Fatal("oper-status state leaf not decoded")
	}
	// 配置叶不受影响
	if entry.Mtu == nil || *entry.Mtu != 1500 {
		t.Fatalf("config leaf mtu lost: %v", entry.Mtu)
	}
	_ = ygot.GoStruct(ifaces)
}

// <get> 路径断线自愈：死连接 → 标记失效 → 重连重试一次（DP-05 语义对齐）。
func TestNETCONFClient_GetWithStateData_SelfHeal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim := startSim(t)
	if err := sim.SetStateDataXML([]byte(testIfmStateXML)); err != nil {
		t.Fatalf("SetStateDataXML: %v", err)
	}
	c := newSimClient(t, sim)
	ctx := context.Background()

	if _, err := c.Get(ctx, "/ifm:ifm/ifm:interfaces", WithStateData()); err != nil {
		t.Fatalf("initial state get must succeed: %v", err)
	}

	// 制造「死连接但自认在线」（复用 ReconnectAfterConnectionLoss 手法）。
	c.mu.Lock()
	deadDriver := c.driver
	c.mu.Unlock()
	if deadDriver == nil {
		t.Fatal("expected live driver after successful get")
	}
	if err := deadDriver.Close(); err != nil {
		t.Fatalf("close underlying driver: %v", err)
	}

	res, err := c.Get(ctx, "/ifm:ifm/ifm:interfaces", WithStateData())
	if err != nil {
		t.Fatalf("state get must self-heal after connection loss: %v", err)
	}
	if !strings.Contains(fmt.Sprintf("%s", res.Data), "oper-status") {
		t.Fatalf("healed state get missing state data: %.500s", res.Data)
	}
}
