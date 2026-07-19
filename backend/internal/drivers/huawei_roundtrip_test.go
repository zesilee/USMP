package drivers

import (
	"reflect"
	"strings"
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/xmlcodec"
)

// populateFirstScalar 经 schema 引导在容器根下寻找首个 config-true 的
// string/uint/bool 标量叶并赋样例值（递归进入首个 config-true 子容器，深度≤4）。
// 返回是否成功赋值。enum/union/identityref/leafref 等复杂类型跳过——参数化矩阵
// 只保证「每模块至少一条真值走通编解码」，深层类型面按需求波次补（design D5）。
func populateFirstScalar(sv reflect.Value, e *yang.Entry, depth int) bool {
	if depth > 4 || e == nil || sv.Kind() != reflect.Ptr || sv.IsNil() {
		return false
	}
	el := sv.Elem()
	st := el.Type()
	// 先试本层标量
	for i := 0; i < st.NumField(); i++ {
		tag := pathOf(st.Field(i))
		child := childEntry(e, tag)
		if child == nil || !child.IsLeaf() || child.Config == yang.TSFalse {
			continue
		}
		f := el.Field(i)
		if f.Kind() != reflect.Ptr || !f.CanSet() {
			continue
		}
		switch f.Type().Elem().Kind() {
		case reflect.String:
			// pattern 约束未知，用保守字母样例
			v := reflect.New(f.Type().Elem())
			v.Elem().SetString("usmp")
			f.Set(v)
			return true
		case reflect.Bool:
			v := reflect.New(f.Type().Elem())
			v.Elem().SetBool(true)
			f.Set(v)
			return true
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			v := reflect.New(f.Type().Elem())
			v.Elem().SetUint(1)
			f.Set(v)
			return true
		}
	}
	// 次试本层 keyed list：建一行、填单个简单键（string/uint）
	for i := 0; i < st.NumField(); i++ {
		tag := pathOf(st.Field(i))
		child := childEntry(e, tag)
		if child == nil || !child.IsList() || child.Config == yang.TSFalse {
			continue
		}
		f := el.Field(i)
		if f.Kind() != reflect.Map || !f.CanSet() {
			continue
		}
		if populateListRow(f, child) {
			return true
		}
	}
	// 再递归子容器
	for i := 0; i < st.NumField(); i++ {
		tag := pathOf(st.Field(i))
		child := childEntry(e, tag)
		if child == nil || child.IsLeaf() || child.IsList() || child.Config == yang.TSFalse {
			continue
		}
		f := el.Field(i)
		if f.Kind() != reflect.Ptr || f.Type().Elem().Kind() != reflect.Struct || !f.CanSet() {
			continue
		}
		f.Set(reflect.New(f.Type().Elem()))
		if populateFirstScalar(f, child, depth+1) {
			return true
		}
		f.Set(reflect.Zero(f.Type())) // 没找到标量则回滚，避免空容器噪声
	}
	return false
}

// populateListRow 为 map 形态的 keyed list 建一行：仅支持单个 string/uint 简单键
// （enum/union/多键跳过——按需求波次补，design D5）。
func populateListRow(f reflect.Value, list *yang.Entry) bool {
	keyNames := list.Key
	if keyNames == "" || len(splitFields(keyNames)) != 1 {
		return false
	}
	keyName := splitFields(keyNames)[0]
	keyType := f.Type().Key()
	elemType := f.Type().Elem() // *Struct
	if elemType.Kind() != reflect.Ptr || elemType.Elem().Kind() != reflect.Struct {
		return false
	}
	var kv reflect.Value
	switch keyType.Kind() {
	case reflect.String:
		kv = reflect.ValueOf("usmp").Convert(keyType)
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		kv = reflect.ValueOf(uint64(1)).Convert(keyType)
	default:
		return false // enum/union/struct 键跳过
	}
	elem := reflect.New(elemType.Elem())
	set := false
	est := elemType.Elem()
	for i := 0; i < est.NumField(); i++ {
		if pathOf(est.Field(i)) != keyName {
			continue
		}
		ef := elem.Elem().Field(i)
		if ef.Kind() != reflect.Ptr || ef.Type().Elem() != keyType {
			return false
		}
		p := reflect.New(keyType)
		p.Elem().Set(kv)
		ef.Set(p)
		set = true
		break
	}
	if !set {
		return false
	}
	m := reflect.MakeMap(f.Type())
	m.SetMapIndex(kv, elem)
	f.Set(m)
	return true
}

func splitFields(s string) []string {
	var out []string
	for _, w := range strings.Fields(s) {
		out = append(out, w)
	}
	return out
}

func pathOf(f reflect.StructField) string {
	return f.Tag.Get("path")
}

// childEntry 在 entry 下按名查子节点，穿透 choice/case（ygot 结构体字段拍平
// choice/case，schema 树保留其层级）。
func childEntry(e *yang.Entry, name string) *yang.Entry {
	if name == "" || e == nil || e.Dir == nil {
		return nil
	}
	if c, ok := e.Dir[name]; ok {
		return c
	}
	for _, c := range e.Dir {
		if c.IsChoice() || c.IsCase() {
			if got := childEntry(c, name); got != nil {
				return got
			}
		}
	}
	return nil
}

// TestFullOnboardingEncodeDecodeRoundtrip 对每个表行模块构造最小真值实例，
// Encode→Decode 断言相等（T02b 参数化矩阵之编解码往返；无可赋值标量的模块
// 走空容器往返，保证 namespace/根元素管线不缺）。
func TestFullOnboardingEncodeDecodeRoundtrip(t *testing.T) {
	for _, pm := range plainModules {
		pm := pm
		t.Run(pm.module, func(t *testing.T) {
			src := pm.newFn()
			spec := &xmlcodec.Spec{Namespace: pm.ns, Schema: specSchemaOf(t, pm)}
			entry := spec.Schema()
			if entry == nil {
				t.Fatalf("SchemaTree 入口缺失")
			}
			populated := populateFirstScalar(reflect.ValueOf(src), entry, 0)

			xml, err := xmlcodec.Encode(spec, src)
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			dst := pm.newFn()
			if err := xmlcodec.Decode(spec, []byte(xml), dst); err != nil {
				t.Fatalf("Decode: %v", err)
			}
			eq, err := ygotDiffEmpty(src, dst)
			if err != nil {
				t.Fatalf("diff: %v", err)
			}
			if !eq {
				t.Fatalf("往返不相等（populated=%v）\nXML: %s", populated, xml)
			}
		})
	}
}

func specSchemaOf(t *testing.T, pm plainModule) func() *yang.Entry {
	t.Helper()
	key := schemaKeyOf(pm.newFn)
	return func() *yang.Entry { return huawei.SchemaTree[key] }
}

func ygotDiffEmpty(a, b ygot.GoStruct) (bool, error) {
	n, err := ygot.Diff(a, b)
	if err != nil {
		return false, err
	}
	return len(n.GetUpdate()) == 0 && len(n.GetDelete()) == 0, nil
}
