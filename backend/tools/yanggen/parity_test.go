package main

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	ygothuawei "github.com/leezesi/usmp/backend/internal/generated/huawei"
	native "github.com/leezesi/usmp/backend/internal/generated/native/huawei"
)

// 结构对拍（任务2.4，YN-01 冻结约定守护）：从 Device 反射遍历 ygot 与 native
// 两侧可达 struct 类型图，断言类型名集合与逐字段（名/path/module tag/类型形状）
// 等价。ygot 侧是基准；差异=生成器缺陷（或需在 codegen-conventions.md 显式拍板
// 的偏差）。

// collectTypes 从根 struct 反射收集全部可达 struct 类型（名→Type）。
func collectTypes(root reflect.Type) map[string]reflect.Type {
	out := map[string]reflect.Type{}
	var visit func(t reflect.Type)
	visit = func(t reflect.Type) {
		switch t.Kind() {
		case reflect.Ptr, reflect.Slice:
			visit(t.Elem())
		case reflect.Map:
			visit(t.Key())
			visit(t.Elem())
		case reflect.Struct:
			if t.Name() == "" || out[t.Name()] != nil {
				return
			}
			out[t.Name()] = t
			for i := 0; i < t.NumField(); i++ {
				visit(t.Field(i).Type)
			}
		}
	}
	visit(root)
	return out
}

func TestParityTypeSets(t *testing.T) {
	yg := collectTypes(reflect.TypeOf(ygothuawei.Device{}))
	nv := collectTypes(reflect.TypeOf(native.Device{}))

	var missing, extra []string
	for name := range yg {
		if _, ok := nv[name]; !ok {
			missing = append(missing, name)
		}
	}
	for name := range nv {
		if _, ok := yg[name]; !ok {
			extra = append(extra, name)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) > 0 || len(extra) > 0 {
		t.Errorf("类型集不一致: ygot 有而 native 缺 %d 个, native 多出 %d 个", len(missing), len(extra))
		for i, n := range missing {
			if i >= 20 {
				t.Logf("  ... 及其余 %d 个缺失", len(missing)-20)
				break
			}
			t.Logf("  missing: %s", n)
		}
		for i, n := range extra {
			if i >= 20 {
				t.Logf("  ... 及其余 %d 个多出", len(extra)-20)
				break
			}
			t.Logf("  extra: %s", n)
		}
	}
}

func TestParityFields(t *testing.T) {
	yg := collectTypes(reflect.TypeOf(ygothuawei.Device{}))
	nv := collectTypes(reflect.TypeOf(native.Device{}))

	mismatches := 0
	report := func(format string, args ...interface{}) {
		mismatches++
		if mismatches <= 30 {
			t.Errorf(format, args...)
		}
	}

	names := make([]string, 0, len(yg))
	for n := range yg {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		yt := yg[name]
		nt, ok := nv[name]
		if !ok {
			continue // TestParityTypeSets 已报
		}
		yf := fieldMap(yt)
		nf := fieldMap(nt)
		for fn, yfd := range yf {
			nfd, ok := nf[fn]
			if !ok {
				report("%s: 字段 %s 缺失", name, fn)
				continue
			}
			if yfd.Tag.Get("path") != nfd.Tag.Get("path") {
				report("%s.%s: path tag %q != %q", name, fn, nfd.Tag.Get("path"), yfd.Tag.Get("path"))
			}
			// _Key struct 的字段无 module tag（两侧同口径），普通字段须一致。
			if !strings.HasSuffix(name, "_Key") && yfd.Tag.Get("module") != nfd.Tag.Get("module") {
				report("%s.%s: module tag %q != %q", name, fn, nfd.Tag.Get("module"), yfd.Tag.Get("module"))
			}
			if ys, ns := shapeOf(yfd.Type), shapeOf(nfd.Type); ys != ns {
				report("%s.%s: 类型形状 %q != %q", name, fn, ns, ys)
			}
		}
		for fn := range nf {
			if _, ok := yf[fn]; !ok {
				report("%s: native 多出字段 %s", name, fn)
			}
		}
	}
	if mismatches > 30 {
		t.Errorf("字段级差异共 %d 处（仅展示前 30）", mismatches)
	}
}

// TestParityEnumTables：枚举值表逐条对拍（类型名→数值→YANG 原名；identityref
// 含 DefiningModule）。native 表 = EnumMaps，ygot 表 = ΛEnum。
func TestParityEnumTables(t *testing.T) {
	yg := ygothuawei.ΛEnum
	nv := native.EnumMaps
	for name, yvals := range yg {
		nvals, ok := nv[name]
		if !ok {
			t.Errorf("枚举类型 %s 在 native 表缺失", name)
			continue
		}
		if len(yvals) != len(nvals) {
			t.Errorf("%s: 值数 %d != %d", name, len(nvals), len(yvals))
			continue
		}
		for v, ydef := range yvals {
			ndef, ok := nvals[v]
			if !ok {
				t.Errorf("%s: 值 %d 在 native 缺失（ygot Name=%q）", name, v, ydef.Name)
				continue
			}
			if ndef.Name != ydef.Name || ndef.DefiningModule != ydef.DefiningModule {
				t.Errorf("%s[%d]: {%q,%q} != {%q,%q}", name, v, ndef.Name, ndef.DefiningModule, ydef.Name, ydef.DefiningModule)
			}
		}
	}
	for name := range nv {
		if _, ok := yg[name]; !ok {
			t.Errorf("native 多出枚举类型 %s", name)
		}
	}
}

func fieldMap(t reflect.Type) map[string]reflect.StructField {
	out := map[string]reflect.StructField{}
	for i := 0; i < t.NumField(); i++ {
		out[t.Field(i).Name] = t.Field(i)
	}
	return out
}

// shapeOf 把类型规约成两侧可比的形状串：包名剥除、YANGEmpty/Empty 与
// Binary 归一、union 接口按名比较。
func shapeOf(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Ptr:
		return "*" + shapeOf(t.Elem())
	case reflect.Slice:
		if t.Name() == "Binary" { // ygot Binary / object.Binary 均为具名 []byte
			return "binary"
		}
		return "[]" + shapeOf(t.Elem())
	case reflect.Map:
		return "map[" + shapeOf(t.Key()) + "]" + shapeOf(t.Elem())
	case reflect.Interface:
		if t.Name() == "" {
			return "interface{}"
		}
		return t.Name() // union 接口按名
	case reflect.Bool:
		if t.Name() == "YANGEmpty" || t.Name() == "Empty" {
			return "empty"
		}
		if t.Name() != "bool" {
			return t.Name()
		}
		return "bool"
	case reflect.Struct:
		return t.Name()
	default:
		if strings.HasPrefix(t.Name(), "E_") {
			return t.Name()
		}
		return t.Kind().String()
	}
}
