package xmlcodec

import (
	"strings"
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
)

// container-mode（XC-01/02 plain-container 根，如 /bgp:bgp）：既有 list 模式
// (vlan/ifm) 之外，容器根模块的编解码。真值断言 + 真实 get-config 回包形态
// (rpc-reply/data 包裹、namespace 前缀)，防「装了通用引擎但容器根静默不通」。

const bgpNS = "urn:huawei:yang:huawei-bgp"

func bgpSpec() *Spec {
	return &Spec{
		Namespace: bgpNS,
		Schema:    func() *yang.Entry { return huawei.SchemaTree["HuaweiBgp_Bgp"] },
	}
}

func TestEncodeContainer_ScalarsAndNesting(t *testing.T) {
	v := &huawei.HuaweiBgp_Bgp{
		BaseProcess: &huawei.HuaweiBgp_Bgp_BaseProcess{
			Enable:      ygot.Bool(true),
			As:          ygot.String("100"),
			AsPathLimit: ygot.Uint16(50),
			GracefulRestart: &huawei.HuaweiBgp_Bgp_BaseProcess_GracefulRestart{
				Enable:      ygot.Bool(true),
				RestartTime: ygot.Uint16(120),
			},
		},
	}
	out, err := Encode(bgpSpec(), v)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	// 根带 namespace、无 list 包裹、标量真值、多层嵌套容器
	for _, want := range []string{
		`<bgp xmlns="urn:huawei:yang:huawei-bgp">`,
		"<base-process>", "<enable>true</enable>", "<as>100</as>", "<as-path-limit>50</as-path-limit>",
		"<graceful-restart><enable>true</enable><restart-time>120</restart-time></graceful-restart>",
		"</base-process></bgp>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("输出缺 %q\n实际: %s", want, out)
		}
	}
	// 容器根不得有 list 键包裹形态（无 <vlans>/<interfaces> 式列表元素）
	if strings.Contains(out, "<bgp><bgp>") {
		t.Errorf("容器根被误当 list 处理: %s", out)
	}
}

func TestEncodeContainer_EmptySelfClosing(t *testing.T) {
	out, err := Encode(bgpSpec(), &huawei.HuaweiBgp_Bgp{})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if out != `<bgp xmlns="urn:huawei:yang:huawei-bgp"/>` {
		t.Errorf("空容器应自闭合, got %q", out)
	}
}

func TestDecodeContainer_Roundtrip(t *testing.T) {
	in := &huawei.HuaweiBgp_Bgp{
		BaseProcess: &huawei.HuaweiBgp_Bgp_BaseProcess{
			Enable:       ygot.Bool(true),
			As:           ygot.String("65000"),
			CheckFirstAs: ygot.Bool(false),
			GracefulRestart: &huawei.HuaweiBgp_Bgp_BaseProcess_GracefulRestart{
				RestartTime: ygot.Uint16(300),
			},
		},
	}
	xml, err := Encode(bgpSpec(), in)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	got := &huawei.HuaweiBgp_Bgp{}
	if err := Decode(bgpSpec(), []byte(xml), got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	bp := got.BaseProcess
	if bp == nil || bp.Enable == nil || !*bp.Enable || bp.As == nil || *bp.As != "65000" ||
		bp.CheckFirstAs == nil || *bp.CheckFirstAs != false {
		t.Fatalf("回读标量不等价: %#v", bp)
	}
	if bp.GracefulRestart == nil || bp.GracefulRestart.RestartTime == nil || *bp.GracefulRestart.RestartTime != 300 {
		t.Fatalf("回读嵌套容器不等价: %#v", bp.GracefulRestart)
	}
}

// 真实 get-config 回包裹在 <rpc-reply><data>…，且常带 namespace 前缀——
// 引擎须按 local 名穿透包裹层解码（否则回读整体丢失，是静默漂移的高发形态）。
func TestDecodeContainer_WrappedAndPrefixed(t *testing.T) {
	raw := []byte(`<rpc-reply><data>` +
		`<bgp:bgp xmlns:bgp="urn:huawei:yang:huawei-bgp">` +
		`<bgp:base-process><bgp:enable>true</bgp:enable><bgp:as>200</bgp:as></bgp:base-process>` +
		`</bgp:bgp></data></rpc-reply>`)
	got := &huawei.HuaweiBgp_Bgp{}
	if err := Decode(bgpSpec(), raw, got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.BaseProcess == nil || got.BaseProcess.As == nil || *got.BaseProcess.As != "200" ||
		got.BaseProcess.Enable == nil || !*got.BaseProcess.Enable {
		t.Fatalf("包裹/前缀回包未正确解码: %#v", got.BaseProcess)
	}
}

func TestDecodeContainer_Empty(t *testing.T) {
	got := &huawei.HuaweiBgp_Bgp{}
	if err := Decode(bgpSpec(), nil, got); err != nil {
		t.Fatalf("空输入应无错: %v", err)
	}
	if got.BaseProcess != nil {
		t.Errorf("空输入不应填充字段")
	}
}
