package xmlcodec

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/openconfig/goyang/pkg/yang"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
)

// 反射 + schema 驱动的完备性测试：对 /bgp:bgp 下**每一个** config-true 标量 leaf
// 自动赋值（不可能漏字段），编码→解码→整体 DeepEqual，并断言恰好 29 个（模型加字段
// 会使计数变化而触发复审）。这是防「全属性可配声称但某字段静默丢」的核心防线——
// VLAN 交付时同类矩阵暴露过真机高危 bug。config-false 子树与 list 本期不入配置面，
// 按 schema config 继承跳过。

// 本期 config-true 标量 leaf 计数（global 2 + base-process 直属 13 + confederation 3
// + graceful-restart 4 + reference-period 3 + timer 4）。模型变更须同步复审此常量。
const bgpConfigTrueScalarLeaves = 29

// 本期不入配置面的 config-false 状态容器（按 schema config 继承判定，非人工挑选）。
var bgpConfigFalseContainers = []string{
	"default-parameter", "error-discard-info", "graceful-restart-status",
	"vpn-brief-infos", "remote-prefix-sid-states",
}

// populateConfigTrue 按 schema config 继承，给 sv 下每个 config-true 标量 leaf 赋唯一值；
// config-false 子树与 list 跳过；只含 list 的 config-true 容器赋值后为空则回收为 nil
// （不产生空容器污染）。返回赋值的 leaf 数。
func populateConfigTrue(t *testing.T, sv reflect.Value, e *yang.Entry, parentCfg bool, n *int) {
	t.Helper()
	st := sv.Type()
	for i := 0; i < st.NumField(); i++ {
		tag := pathTag(st.Field(i))
		if tag == "" {
			continue
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
				continue // config-false 子容器：不入配置面
			}
			n0 := *n
			fv.Set(reflect.New(fv.Type().Elem()))
			populateConfigTrue(t, fv.Elem(), child, cfg, n)
			if *n == n0 {
				fv.Set(reflect.Zero(fv.Type())) // 内部无 config-true leaf（如仅含 list）→ 回收
			}
		case fv.Kind() == reflect.Ptr: // 标量指针叶
			if !cfg {
				continue
			}
			setScalarLeaf(fv, *n)
			*n++
		case fv.Kind() == reflect.Int64 && fv.Type().Implements(goEnumType): // 枚举叶
			if !cfg {
				continue
			}
			fv.SetInt(1)
			*n++
		case fv.Kind() == reflect.Slice: // leaf-list（如 confederation/as 多个子 AS）
			if !cfg || fv.Type().Elem().Kind() == reflect.Uint8 {
				continue // binary leaf 引擎不支持，跳过
			}
			s := reflect.MakeSlice(fv.Type(), 2, 2)
			setBareScalar(s.Index(0), *n*10)
			setBareScalar(s.Index(1), *n*10+1)
			fv.Set(s)
			*n++
		case fv.Kind() == reflect.Map: // list：本期不入配置面
			continue
		}
	}
}

func setScalarLeaf(fv reflect.Value, n int) {
	p := reflect.New(fv.Type().Elem())
	setBareScalar(p.Elem(), n)
	fv.Set(p)
}

// setBareScalar 给非指针标量（指针叶的 Elem 或 leaf-list 元素）赋唯一值。
func setBareScalar(v reflect.Value, n int) {
	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(n%2 == 0)
	case reflect.String:
		v.SetString(fmt.Sprintf("v%d", n))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(uint64(n%250 + 1)) // 稳落 uint8 范围
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(int64(n%250 + 1))
	}
}

func TestBgpAllConfigTrueLeaves_Roundtrip(t *testing.T) {
	e := huawei.SchemaTree["HuaweiBgp_Bgp"]
	if e == nil {
		t.Fatal("HuaweiBgp_Bgp schema 未解析")
	}
	orig := &huawei.HuaweiBgp_Bgp{}
	n := 0
	populateConfigTrue(t, reflect.ValueOf(orig).Elem(), e, true, &n)

	// 完备性：恰好覆盖全部 config-true 标量 leaf（漏/多都失败）
	if n != bgpConfigTrueScalarLeaves {
		t.Fatalf("config-true 标量 leaf 覆盖数 = %d，期望 %d（模型变更？须复审范围）", n, bgpConfigTrueScalarLeaves)
	}

	xml, err := Encode(bgpSpec(), orig)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	got := &huawei.HuaweiBgp_Bgp{}
	if err := Decode(bgpSpec(), []byte(xml), got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	// 每个赋值字段编码→解码后原值等价：DeepEqual 兜住任何字段级丢失
	if !reflect.DeepEqual(orig, got) {
		t.Fatalf("全字段往返不等价（字段级丢失）\n原: %s\nXML: %s", mustJSON(orig), xml)
	}
}

func TestBgpConfigFalse_NotInEditConfig(t *testing.T) {
	e := huawei.SchemaTree["HuaweiBgp_Bgp"]
	orig := &huawei.HuaweiBgp_Bgp{}
	n := 0
	populateConfigTrue(t, reflect.ValueOf(orig).Elem(), e, true, &n)
	xml, err := Encode(bgpSpec(), orig)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	for _, c := range bgpConfigFalseContainers {
		if strings.Contains(xml, "<"+c+">") || strings.Contains(xml, "<"+c+"/>") {
			t.Errorf("config-false 容器 %q 不应出现在 edit-config: %s", c, xml)
		}
	}
	// 且根命名空间正确
	if !strings.HasPrefix(xml, `<bgp xmlns="urn:huawei:yang:huawei-bgp">`) {
		t.Errorf("根 namespace 形态错: %s", xml[:min(80, len(xml))])
	}
}

func mustJSON(v interface{}) string { return fmt.Sprintf("%#v", v) }

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
