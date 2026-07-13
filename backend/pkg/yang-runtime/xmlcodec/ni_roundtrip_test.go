package xmlcodec

import (
	"reflect"
	"strings"
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
)

// network-instance（/ni:network-instance）是**容器根 + 嵌套 list**：root 无直属标量，
// 两个子容器 global（纯标量）/instances（内含 instance list）。这是 XC-05 容器根从未走
// 过的路径（BGP 容器根其子容器均为纯标量集）——本测试文件用往返真值 + schema 驱动完备
// 枚举拦截「装了容器根引擎但嵌套 list 静默不通」。且 instance struct 是多模块 augment 的
// 共享合并点（Bgp=huawei-bgp、Afs/Parameter=huawei-l3vpn），完备枚举须按 module tag 过滤
// 到原生 huawei-network-instance 字段（design D2），断言原生 config-true 标量恰 5 个。

const niNS = "urn:huawei:yang:huawei-network-instance"

func niSpec() *Spec {
	return &Spec{
		Namespace: niNS,
		Schema:    func() *yang.Entry { return huawei.SchemaTree["HuaweiNetworkInstance_NetworkInstance"] },
	}
}

// 原生 config-true 标量 leaf 计数：global{cfg-router-id, as-notation-plain,
// route-distinguisher-auto-ip}=3 + instance{name(key), description}=2 = 5。
// config-false（sys-router-id, vrf-id）与 augment 子树（Bgp/Afs/Parameter）不计。
// 模型加原生字段会使计数变化而触发复审。
const niNativeConfigTrueLeaves = 5

const niModule = "huawei-network-instance"

func moduleTag(f reflect.StructField) string {
	return f.Tag.Get("module")
}

// populateNativeConfigTrue 按 schema config 继承 + module tag 过滤，给 sv 下每个
// **原生（module=huawei-network-instance）config-true 标量 leaf** 赋唯一值；augment
// 字段（他模块）、config-false、list（Map，另路径覆盖）跳过。返回赋值 leaf 数。
func populateNativeConfigTrue(t *testing.T, sv reflect.Value, e *yang.Entry, parentCfg bool, n *int) {
	t.Helper()
	st := sv.Type()
	for i := 0; i < st.NumField(); i++ {
		f := st.Field(i)
		tag := pathTag(f)
		if tag == "" {
			continue
		}
		if m := moduleTag(f); m != "" && m != niModule {
			continue // augment 字段（huawei-bgp / huawei-l3vpn）：不属本期原生面
		}
		var child *yang.Entry
		if e != nil {
			child = e.Dir[tag]
		}
		cfg := parentCfg
		if child != nil {
			switch child.Config {
			case yang.TSTrue:
				cfg = true
			case yang.TSFalse:
				cfg = false
			}
		}
		fv := sv.Field(i)
		switch {
		case fv.Kind() == reflect.Ptr && fv.Type().Elem().Kind() == reflect.Struct:
			if !cfg {
				continue
			}
			n0 := *n
			fv.Set(reflect.New(fv.Type().Elem()))
			populateNativeConfigTrue(t, fv.Elem(), child, cfg, n)
			if *n == n0 {
				fv.Set(reflect.Zero(fv.Type()))
			}
		case fv.Kind() == reflect.Ptr:
			if !cfg {
				continue
			}
			setScalarLeaf(fv, *n)
			*n++
		case fv.Kind() == reflect.Int64 && fv.Type().Implements(goEnumType):
			if !cfg {
				continue
			}
			fv.SetInt(1)
			*n++
		case fv.Kind() == reflect.Slice:
			if !cfg || fv.Type().Elem().Kind() == reflect.Uint8 {
				continue
			}
			s := reflect.MakeSlice(fv.Type(), 2, 2)
			setBareScalar(s.Index(0), *n*10)
			setBareScalar(s.Index(1), *n*10+1)
			fv.Set(s)
			*n++
		case fv.Kind() == reflect.Map:
			continue // list：由 instance 直枚举 + 集成用例覆盖
		}
	}
}

// TestNiAllNativeConfigTrueLeaves_Roundtrip 完备性主防线：global 容器 + 一条 instance
// list 条目，覆盖全部原生 config-true 标量（global 3 + instance name/description 2），
// 编码→解码→整体 DeepEqual，并断言原生标量计数恰 5。
func TestNiAllNativeConfigTrueLeaves_Roundtrip(t *testing.T) {
	root := huawei.SchemaTree["HuaweiNetworkInstance_NetworkInstance"]
	if root == nil {
		t.Fatal("HuaweiNetworkInstance_NetworkInstance schema 未解析")
	}
	n := 0

	// global：枚举原生 config-true 标量（期望 3）
	orig := &huawei.HuaweiNetworkInstance_NetworkInstance{}
	gv := reflect.ValueOf(orig).Elem().FieldByName("Global")
	gv.Set(reflect.New(gv.Type().Elem()))
	populateNativeConfigTrue(t, gv.Elem(), root.Dir["global"], true, &n)
	if n != 3 {
		t.Fatalf("global 原生 config-true 标量 = %d，期望 3（模型变更？须复审）", n)
	}

	// instance：枚举原生 config-true 标量（期望 name + description = 2）
	instEntry := root.Dir["instances"].Dir["instance"]
	inst := &huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance{}
	beforeInst := n
	populateNativeConfigTrue(t, reflect.ValueOf(inst).Elem(), instEntry, true, &n)
	if n-beforeInst != 2 {
		t.Fatalf("instance 原生 config-true 标量 = %d，期望 2（name+description；config-false/augment 应排除）", n-beforeInst)
	}
	if inst.Name == nil {
		t.Fatal("name（list key）未被枚举赋值")
	}
	orig.Instances = &huawei.HuaweiNetworkInstance_NetworkInstance_Instances{
		Instance: map[string]*huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance{*inst.Name: inst},
	}

	if n != niNativeConfigTrueLeaves {
		t.Fatalf("原生 config-true 标量总数 = %d，期望 %d", n, niNativeConfigTrueLeaves)
	}

	// 往返真值：容器根 + 嵌套 list 编码→解码后整体等价（字段级丢失兜底）
	xml, err := Encode(niSpec(), orig)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	got := &huawei.HuaweiNetworkInstance_NetworkInstance{}
	if err := Decode(niSpec(), []byte(xml), got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !reflect.DeepEqual(orig, got) {
		t.Fatalf("容器根+嵌套 list 往返不等价（字段级丢失）\nXML: %s\n原: %#v\n得: %#v", xml, orig, got)
	}
}

// TestNiEncode_NestedListShape 断言嵌套 list 的 XML 形态：容器根下 global 标量 +
// instances/instance 多条目，携带根 namespace，list 条目正确展开。
func TestNiEncode_NestedListShape(t *testing.T) {
	v := &huawei.HuaweiNetworkInstance_NetworkInstance{
		Global: &huawei.HuaweiNetworkInstance_NetworkInstance_Global{
			CfgRouterId:     ygot.String("1.1.1.1"),
			AsNotationPlain: ygot.Bool(true),
		},
		Instances: &huawei.HuaweiNetworkInstance_NetworkInstance_Instances{
			Instance: map[string]*huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance{
				"vpn-a": {Name: ygot.String("vpn-a"), Description: ygot.String("first")},
				"vpn-b": {Name: ygot.String("vpn-b"), Description: ygot.String("second")},
			},
		},
	}
	out, err := Encode(niSpec(), v)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	for _, want := range []string{
		`<network-instance xmlns="urn:huawei:yang:huawei-network-instance">`,
		"<global>", "<cfg-router-id>1.1.1.1</cfg-router-id>", "<as-notation-plain>true</as-notation-plain>",
		"<instances>", "<instance>", "<name>vpn-a</name>", "<description>first</description>",
		"<name>vpn-b</name>", "<description>second</description>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("输出缺 %q\n实际: %s", want, out)
		}
	}
}

// TestNiConfigFalseAndAugment_NotInEditConfig 负路径：config-false 只读态与 augment
// 子树都不得出现在下发报文（design D2/D5b、NI-03 负路径、NI-06）。
func TestNiConfigFalseAndAugment_NotInEditConfig(t *testing.T) {
	v := &huawei.HuaweiNetworkInstance_NetworkInstance{
		Global: &huawei.HuaweiNetworkInstance_NetworkInstance_Global{CfgRouterId: ygot.String("2.2.2.2")},
		Instances: &huawei.HuaweiNetworkInstance_NetworkInstance_Instances{
			Instance: map[string]*huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance{
				"vrf1": {Name: ygot.String("vrf1"), Description: ygot.String("d")},
			},
		},
	}
	out, err := Encode(niSpec(), v)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	// reconciler 只 set 原生字段，故 augment（bgp/afs/parameter）与 config-false
	// （sys-router-id/vrf-id）本就 nil、不应出现
	for _, bad := range []string{"<sys-router-id>", "<vrf-id>", "<bgp>", "<afs>", "<parameter>", "<traffic-statistic-enable>"} {
		if strings.Contains(out, bad) {
			t.Errorf("非本期字段 %q 不应出现在 edit-config: %s", bad, out)
		}
	}
	if !strings.HasPrefix(out, `<network-instance xmlns="urn:huawei:yang:huawei-network-instance">`) {
		t.Errorf("根 namespace 形态错: %s", out[:min(90, len(out))])
	}
}

// TestNiDecode_WrappedAndPrefixed 真实 get-config 回包（rpc-reply/data 包裹 + 前缀）
// 下嵌套 list 的解码穿透。
func TestNiDecode_WrappedAndPrefixed(t *testing.T) {
	raw := []byte(`<rpc-reply><data>` +
		`<ni:network-instance xmlns:ni="urn:huawei:yang:huawei-network-instance">` +
		`<ni:global><ni:cfg-router-id>3.3.3.3</ni:cfg-router-id></ni:global>` +
		`<ni:instances><ni:instance><ni:name>vpn-x</ni:name><ni:description>dx</ni:description></ni:instance></ni:instances>` +
		`</ni:network-instance></data></rpc-reply>`)
	got := &huawei.HuaweiNetworkInstance_NetworkInstance{}
	if err := Decode(niSpec(), raw, got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Global == nil || got.Global.CfgRouterId == nil || *got.Global.CfgRouterId != "3.3.3.3" {
		t.Fatalf("global 前缀回包未解码: %#v", got.Global)
	}
	inst := got.Instances.Instance["vpn-x"]
	if inst == nil || inst.Description == nil || *inst.Description != "dx" {
		t.Fatalf("嵌套 list 前缀回包未解码: %#v", got.Instances)
	}
}
