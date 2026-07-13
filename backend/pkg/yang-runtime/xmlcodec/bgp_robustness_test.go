package xmlcodec

import (
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/openconfig/ygot/ygot"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
)

// 完备矩阵：并发(R09)、边界、负路径。补齐 yang-config-test-design 对容器根 BGP 的
// 并发-race/边界/负路径三层，防「简单值能跑、极端/异常静默崩」。

// 并发编解码无数据竞态（R09）：引擎无共享可变状态，多协程并发编码同一只读结构、
// 各自解码到独立结构，-race 下必须干净。
func TestBgpCodec_Concurrent(t *testing.T) {
	spec := bgpSpec()
	src := &huawei.HuaweiBgp_Bgp{
		BaseProcess: &huawei.HuaweiBgp_Bgp_BaseProcess{
			Enable: ygot.Bool(true), As: ygot.String("65000"),
			GracefulRestart: &huawei.HuaweiBgp_Bgp_BaseProcess_GracefulRestart{RestartTime: ygot.Uint16(300)},
		},
	}
	want, err := Encode(spec, src)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := Encode(spec, src)
			if err != nil || got != want {
				t.Errorf("并发编码不一致/出错: %v", err)
				return
			}
			dst := &huawei.HuaweiBgp_Bgp{}
			if err := Decode(spec, []byte(got), dst); err != nil {
				t.Errorf("并发解码出错: %v", err)
				return
			}
			if dst.BaseProcess == nil || dst.BaseProcess.As == nil || *dst.BaseProcess.As != "65000" {
				t.Errorf("并发解码值错: %#v", dst.BaseProcess)
			}
		}()
	}
	wg.Wait()
}

// 边界：最大整数值 + leaf-list 多元素 + XML 特殊字符（转义路径）往返无损。
func TestBgpCodec_BoundaryValues(t *testing.T) {
	spec := bgpSpec()
	src := &huawei.HuaweiBgp_Bgp{
		BaseProcess: &huawei.HuaweiBgp_Bgp_BaseProcess{
			LocalIfnetMtu: ygot.Uint16(65535),      // uint16 max
			DelayTime:     ygot.Uint32(4294967295), // uint32 max
			As:            ygot.String(`a<b&c">'`), // 特殊字符（测转义/反转义，非合法 AS，仅验编解码鲁棒）
			Confederation: &huawei.HuaweiBgp_Bgp_BaseProcess_Confederation{
				As: []string{"1", "65535", "4294967295"}, // leaf-list 多元素
			},
		},
	}
	xml, err := Encode(spec, src)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	// 特殊字符必须实体转义（合法 XML）
	if strings.Contains(xml, "a<b&c") {
		t.Errorf("特殊字符未转义，非法 XML: %s", xml)
	}
	dst := &huawei.HuaweiBgp_Bgp{}
	if err := Decode(spec, []byte(xml), dst); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !reflect.DeepEqual(src, dst) {
		t.Fatalf("边界值往返不等价\n原: %#v\n得: %#v\nXML: %s", src.BaseProcess, dst.BaseProcess, xml)
	}
}

// 负路径：畸形 XML 必须返回明确错误、绝不 panic（R08）。
func TestBgpDecode_MalformedXML(t *testing.T) {
	spec := bgpSpec()
	cases := []struct {
		name string
		raw  string
	}{
		{"截断未闭合", `<bgp xmlns="urn:huawei:yang:huawei-bgp"><base-process><enable>true`},
		{"标签错配", `<bgp><base-process><enable>true</as></base-process></bgp>`},
		{"叶内非法嵌套", `<bgp><base-process><enable><x>1</x></enable></base-process></bgp>`},
		{"非数值到 uint 叶", `<bgp><base-process><as-path-limit>notnum</as-path-limit></base-process></bgp>`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("解码 panic（违反 R08）: %v", r)
				}
			}()
			dst := &huawei.HuaweiBgp_Bgp{}
			if err := Decode(spec, []byte(tc.raw), dst); err == nil {
				t.Errorf("畸形 XML 应返回错误")
			}
		})
	}
}

// 负路径：未知元素被跳过（前向兼容），已知字段仍正确解析。
func TestBgpDecode_UnknownElementsSkipped(t *testing.T) {
	raw := []byte(`<bgp xmlns="urn:huawei:yang:huawei-bgp"><base-process>` +
		`<future-unknown-leaf>x</future-unknown-leaf>` +
		`<enable>true</enable>` +
		`<another-unknown><nested>y</nested></another-unknown>` +
		`<as>100</as></base-process></bgp>`)
	dst := &huawei.HuaweiBgp_Bgp{}
	if err := Decode(bgpSpec(), raw, dst); err != nil {
		t.Fatalf("含未知元素不应报错（前向兼容）: %v", err)
	}
	if dst.BaseProcess == nil || dst.BaseProcess.Enable == nil || !*dst.BaseProcess.Enable ||
		dst.BaseProcess.As == nil || *dst.BaseProcess.As != "100" {
		t.Fatalf("已知字段未正确解析: %#v", dst.BaseProcess)
	}
}
